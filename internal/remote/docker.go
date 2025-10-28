package remote

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/mw"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	"github.com/martencassel/gobinrepo/internal/util/oci"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

type DockerRemoteHandler struct {
	blobs blobs.BlobStore
}

func NewDockerRemoteHandler(blobs blobs.BlobStore) *DockerRemoteHandler {
	return &DockerRemoteHandler{
		blobs: blobs,
	}
}

// RegisterRoutes registers the Docker Remote Registry API routes
func (h *DockerRemoteHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/v2", func(c *gin.Context) {
		c.Header("Docker-Distribution-API-Version", "registry/2.0")
		c.Status(http.StatusOK)
	})
	r.GET("/v2/:repoKey/:name/manifests/:ref", h.GetManifest)
	r.GET("/v2/:repoKey/:name/blobs/:digest", h.GetBlob)
}

// GetManifest handles requests for Docker manifests
func (h *DockerRemoteHandler) GetManifest(c *gin.Context) {
	reqURL := c.Request.URL.String()
	log.Infof("Received request for manifest URL: %s", reqURL)

	url, err := oci.ParseOCIURL(reqURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OCI URL"})
		return
	}
	client := newDockerHubClient()

	resp, err := client.GetManifest(c.Request.Context(), "library/"+url.Name.Rest(), url.Reference.String(), c.Request.Header)
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
	h.streamBlob(req)
}

// --- helpers ---

// newDockerHubClient creates a Docker Hub registry client
func newDockerHubClient() *oci.RegistryClient {
	rt := oci.NewTokenRoundTripper(true, nil)
	return oci.NewRegistryClient("https://registry-1.docker.io", rt)
}

type blobRequest struct {
	Ctx    context.Context
	Gin    *gin.Context
	URL    oci.OCIURL
	Digest digest.Digest
	Start  time.Time
	Logger *log.Entry
}

func (h *DockerRemoteHandler) streamBlob(req *blobRequest) {
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

	// Otherwise fetch from upstream
	client := newDockerHubClient()
	resp, err := client.GetBlob(req.Ctx, "library/"+req.URL.Name.Rest(), req.URL.Reference.String(), req.Gin.Request.Header)
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
		req.Gin.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream blob"})
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
