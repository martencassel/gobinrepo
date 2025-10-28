package remote

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
	"github.com/martencassel/gobinrepo/internal/mw"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	"github.com/martencassel/gobinrepo/internal/util/oci"
	"github.com/martencassel/gobinrepo/internal/util/trace"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

type DockerRemoteHandler struct {
	blobs blobs.BlobStore
	store *configstore.RepoConfigStore
	*configstore.RepoConfigStore
	traceEnable bool
}

func NewDockerRemoteHandler(blobs blobs.BlobStore, store *configstore.RepoConfigStore, traceEnable bool) *DockerRemoteHandler {
	return &DockerRemoteHandler{
		blobs:       blobs,
		store:       store,
		traceEnable: traceEnable,
	}
}

// RegisterRoutes registers the Docker Remote Registry API routes
func (h *DockerRemoteHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/v2", func(c *gin.Context) {
		c.Header("Docker-Distribution-API-Version", "registry/2.0")
		c.Status(http.StatusOK)
	})
	// r.GET("/v2/:repoKey/*name/manifests/:ref", h.GetManifest)
	// r.GET("/v2/:repoKey/*name/blobs/:digest", h.GetBlob)

	r.GET("/v2/:repoKey/*path", h.handleV2)
}

func (h *DockerRemoteHandler) handleV2(c *gin.Context) {
	repoKey := c.Param("repoKey")
	rest := strings.TrimPrefix(c.Param("path"), "/")

	switch {
	case strings.Contains(rest, "/manifests/"):
		parts := strings.SplitN(rest, "/manifests/", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manifest path"})
			return
		}
		h.GetManifestWithParams(c, repoKey, parts[0], parts[1])
	case strings.Contains(rest, "/blobs/"):
		parts := strings.SplitN(rest, "/blobs/", 2)
		name, digest := parts[0], parts[1]
		h.GetBlobWithParams(c, repoKey, name, digest)
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported v2 path"})
	}
}

func (h *DockerRemoteHandler) GetManifestWithParams(c *gin.Context, repoKey, name, ref string) {
	c.Set("RepoKey", repoKey)
	c.Set("SubPath", name+"/manifests/"+ref)
	h.GetManifest(c)
}

func (h *DockerRemoteHandler) GetBlobWithParams(c *gin.Context, repoKey, name, digest string) {
	c.Set("RepoKey", repoKey)
	c.Set("SubPath", name+"/blobs/"+digest)
	h.GetBlob(c)
}

// GetManifest handles requests for Docker manifests
func (h *DockerRemoteHandler) GetManifest(c *gin.Context) {
	repoKey, ok := repoKeyFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing repoKey"})
		return
	}
	cfg, ok := h.store.Get(repoKey)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown repoKey: repository configuration not found"})
		return
	}
	log.Infof("Using repo config: %+v", cfg)

	reqURL := c.Request.URL.String()
	log.Infof("Received request for manifest URL: %s", reqURL)

	url, err := oci.ParseOCIURL(reqURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OCI URL"})
		return
	}
	client := newTracedRegistryClient(cfg.RemoteURL, h.traceEnable, &cfg)
	normalizedName := normalizeName(cfg.RemoteURL, url.Name.Rest())

	resp, err := client.GetManifest(c.Request.Context(), normalizedName, url.Reference.String(), c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to get manifest from upstream"})
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()

	c.Status(resp.StatusCode)
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Header(k, vv)
		}
	}
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		log.Errorf("Failed to copy response body: %v", err)
	}
}

// GetBlob handles requests for Docker blobs
func (h *DockerRemoteHandler) GetBlob(c *gin.Context) {
	repoKey, ok := repoKeyFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing repoKey"})
		return
	}
	cfg, ok := h.store.Get(repoKey)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown repoKey"})
		return
	}
	log.Infof("Using repo config: %+v", cfg)

	corrID, _ := c.Get(mw.CorrelationIDHeader)
	ctx := c.Request.Context()
	start := time.Now()

	// Pre‑populate a logger with correlation ID and request basics
	logger := log.WithFields(log.Fields{
		"correlation_id": corrID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.String(),
		"repoKey":        c.Param("repoKey"),
	})

	// Parse and validate
	ociURL, requestDigest, err := oci.ParseDigestURL(c.Request.URL.String(), "registry-1.docker.io")
	if err != nil {
		writeError(c, http.StatusBadRequest, "Invalid blob request", err)
		return
	}

	req := &blobRequest{
		Ctx:    ctx,
		Gin:    c,
		URL:    ociURL,
		Digest: requestDigest,
		Start:  start,
		Logger: logger,
	}
	h.streamBlob(req, &cfg)
}

// --- helpers ---

type blobRequest struct {
	Ctx    context.Context
	Gin    *gin.Context
	URL    oci.OCIURL
	Digest digest.Digest
	Start  time.Time
	Logger *log.Entry
}

func (h *DockerRemoteHandler) streamBlob(req *blobRequest, cfg *configstore.RepoConfig) {
	// Try local cache
	exists, err := h.blobs.Exists(req.Ctx, req.Digest)
	if err != nil {
		req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check blob existence"})
		return
	}
	if exists {
		reader, err := h.blobs.Get(req.Ctx, req.Digest)
		if err != nil {
			req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open blob"})
			return
		}
		defer func() {
			if cerr := reader.Close(); cerr != nil {
				log.Warnf("failed to close reader: %v", cerr)
			}
		}()

		written, err := io.Copy(req.Gin.Writer, reader)
		if err != nil {
			req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream blob"})
			return
		}

		req.Logger.WithFields(log.Fields{
			"digest":   req.Digest,
			"size":     written,
			"duration": time.Since(req.Start).Round(time.Millisecond),
		}).Info("Blob served from local store")
		return
	}
	client := newTracedRegistryClient(cfg.RemoteURL, h.traceEnable, cfg)

	normalizedName := normalizeName(cfg.RemoteURL, req.URL.Name.Rest())

	resp, err := client.GetBlob(req.Ctx, normalizedName, req.URL.Reference.String(), req.Gin.Request.Header)
	if err != nil {
		req.Gin.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch blob from upstream"})
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()

	writer, err := h.blobs.Writer(req.Ctx, req.Digest)
	if err != nil {
		req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create blob writer"})
		return
	}
	defer func() {
		if cerr := writer.Close(); cerr != nil {
			log.Warnf("failed to close writer: %v", cerr)
		}
	}()

	written, err := io.Copy(io.MultiWriter(req.Gin.Writer, writer), resp.Body)
	if err != nil {
		if errors.Is(req.Ctx.Err(), context.Canceled) {
			req.Logger.WithFields(log.Fields{
				"digest": req.Digest,
			}).Warn("Streaming aborted due to server shutdown or client disconnect")
		} else {
			req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream blob"})
		}
		return
	}

	req.Logger.WithFields(log.Fields{
		"digest":   req.Digest,
		"size":     written,
		"duration": time.Since(req.Start).Round(time.Millisecond),
		"status":   resp.StatusCode,
	}).Info("Blob streamed from upstream")
}

func writeError(c *gin.Context, status int, msg string, err error) {
	log.WithError(err).Warn(msg)
	c.JSON(status, gin.H{"error": msg})
}

func repoKeyFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get("RepoKey")
	if !ok {
		return "", false
	}
	key, ok := v.(string)
	return key, ok
}

// normalizeName applies registry-specific normalization rules.
// For Docker Hub (registry-1.docker.io), unscoped names are prefixed with "library/".
// For all other registries, the name is returned unchanged.
func normalizeName(remoteURL, name string) string {
	if remoteURL == "https://registry-1.docker.io" {
		// If name already contains a slash, leave it alone
		if strings.Contains(name, "/") {
			return name
		}
		return "library/" + name
	}
	return name
}

func newDefaultTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = 500
	tr.MaxIdleConnsPerHost = 500
	tr.IdleConnTimeout = 90 * time.Second
	tr.TLSHandshakeTimeout = 10 * time.Second
	tr.ExpectContinueTimeout = 1 * time.Second
	tr.ResponseHeaderTimeout = 15 * time.Second
	tr.DisableCompression = true // blobs are already compressed
	return tr
}

func newTracedRegistryClient(remoteURL string, traceUpstream bool, cfg *configstore.RepoConfig) *oci.RegistryClient {
	base := newDefaultTransport()
	var rt http.RoundTripper = base
	if cfg.Username != "" {
		rt = &oci.BasicAuthRoundTripper{
			Username: cfg.Username,
			Password: cfg.Password,
			Base:     rt,
		}
	}
	rt = oci.NewTokenRoundTripper(true, oci.WithTransport(rt))
	if traceUpstream {
		rt = &trace.TracingRoundTripper{Base: rt}
	}
	return oci.NewRegistryClient(remoteURL, rt)
}
