package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

type DebianRemoteHandler struct {
	blobs blobs.BlobStore
	store *configstore.RepoConfigStore
	*configstore.RepoConfigStore
	traceEnable bool
}

func NewDebianRemoteHandler(blobs blobs.BlobStore, store *configstore.RepoConfigStore, traceEnable bool) *DebianRemoteHandler {
	return &DebianRemoteHandler{
		blobs:       blobs,
		store:       store,
		traceEnable: traceEnable,
	}
}

// /debian/debian-remote/debian/dists/stable/InRelease
func (r *DebianRemoteHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/debian/:repoKey/*path", r.handleDebianRequest)
}

func (r *DebianRemoteHandler) handleDebianRequest(c *gin.Context) {
	repoKey := c.Param("repoKey")
	rest := strings.TrimPrefix(c.Param("path"), "/")
	log.WithFields(log.Fields{
		"repo_key": repoKey,
		"path":     rest,
	}).Info("Handling Debian request")

	repoConfig, ok := r.store.Get(repoKey)
	if !ok {
		c.JSON(404, gin.H{
			"error": "Repository not found",
		})
		return
	}
	switch {
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "InRelease"):
		r.handleInRelease(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "Release"):
		r.handleRelease(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "Release.gpg"):
		r.handleReleaseGPG(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "Packages"):
		r.handlePackages(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "Packages.gz"):
		r.handlePackagesGz(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "dists/") && strings.HasSuffix(rest, "Packages.xz"):
		r.handlePackagesXz(c, repoKey, rest, &repoConfig)
	case strings.Contains(rest, "pool/"):
		r.handlePool(c, repoKey, rest, &repoConfig)
	default:
		c.JSON(404, gin.H{
			"error": "Not Found",
		})
	}
}

func (r *DebianRemoteHandler) handleInRelease(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling InRelease for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) handleRelease(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling Release for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) handleReleaseGPG(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling ReleaseGPG for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) handlePackages(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling Packages for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) handlePackagesGz(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling PackagesGz for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) handlePackagesXz(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling PackagesXz for repoKey=%s, path=%s", repoKey, path)
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	r.writeResponse(c, resp)
}

func (r *DebianRemoteHandler) getFile(repoKey, path string) (bool, io.ReadCloser, error) {
	digestPath := filepath.Join("/tmp/filestore", repoKey, path)
	// Check if digest file exists
	if _, err := os.Stat(digestPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil, fmt.Errorf("digest file does not exist: %w", err)
		}
		return false, nil, fmt.Errorf("failed to stat digest file: %w", err)
	}
	digestBytes, err := os.ReadFile(digestPath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read digest file: %w", err)
	}
	dgstStr := strings.TrimSpace(string(digestBytes))
	dgst, err := digest.Parse(dgstStr)
	if err != nil {
		return false, nil, fmt.Errorf("invalid digest in digest file: %w", err)
	}
	// Retrieve blob from blob store
	blobReader, err := r.blobs.Get(context.Background(), dgst)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get blob from blob store: %w", err)
	}
	return true, blobReader, nil
}

func (r *DebianRemoteHandler) handlePool(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) {
	log.Infof("Handling Pool for repoKey=%s, path=%s", repoKey, path)
	if repoConfig == nil {
		handleError(c, 404, "repository not found")
		return
	}
	// Check if file exists in local filestore
	found, blobReader, err := r.getFile(repoKey, path)
	if err != nil {
		log.Warnf("Error checking local filestore for file: %v", err)
	}
	if found {
		log.Infof("Serving file from local filestore: repoKey=%s, path=%s", repoKey, path)
		// Stream blob to response
		_, err := io.Copy(c.Writer, blobReader)
		if err != nil {
			log.Errorf("Error streaming blob from local filestore: %v", err)
			handleError(c, 500, fmt.Sprintf("Failed to stream blob: %v", err))
			return
		}
		return
	}
	// File not found locally; fetch from upstream and store
	log.Infof("File not found in local filestore; fetching from upstream: repoKey=%s, path=%s", repoKey, path)

	// 1. Forward the request upstream
	resp := r.forwardRequest(c, repoKey, path, repoConfig)
	if resp == nil {
		log.Errorf("Error forwarding request for blob: repoKey=%s, path=%s", repoKey, path)
		handleError(c, 500, "failed to forward request")
		return
	}
	// 2. Prepare sinks
	h := sha256.New()
	err = os.MkdirAll("/tmp/blobstore/tmp", 0o755)
	if err != nil {
		log.Errorf("Error creating temp directory for blob upload: %v", err)
		handleError(c, 500, "Failed to create temp directory for blob upload")
		return
	}
	tmpFile, err := os.CreateTemp("/tmp/blobstore/tmp", "upload-*")
	if err != nil {
		log.Errorf("Error creating temp file for blob upload: %v", err)
		handleError(c, 500, "Failed to create temp file for blob upload")
		return
	}

	downstream := c.Writer // Gins response writer

	// 3. Multiwriter: temp file + hasher + downstream
	mw := io.MultiWriter(tmpFile, h, downstream)

	// 4. Copy upstream response into all three
	if _, err := io.Copy(mw, resp.Body); err != nil {
		log.Errorf("Error copying upstream response: %v", err)
		handleError(c, 500, fmt.Sprintf("Failed to read upstream response: %v", err))
		return
	}
	// 5. Finalize digest
	dgst := digest.NewDigest(digest.SHA256, h)

	// 6. Commit temp file to blob store
	// Rewind temp file if Put needs to read from it.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		log.Errorf("Error seeking temp file: %v", err)
		handleError(c, 500, fmt.Sprintf("Failed to seek temp file: %v", err))
		return
	}
	if err := r.blobs.Put(c.Request.Context(), dgst, tmpFile); err != nil {
		log.Errorf("Error storing blob in blob store: %v", err)
		handleError(c, 500, fmt.Sprintf("Failed to store blob: %v", err))
		return
	}
	// 7. Delete temp file if needed
	if err := tmpFile.Close(); err != nil {
		log.Warnf("Failed to close temp file: %v", err)
	}
	if err := os.Remove(tmpFile.Name()); err != nil {
		log.Warnf("Failed to remove temp file: %v", err)
	}

	// Save to index file
	digestPath := filepath.Join("/tmp/filestore", repoKey, path)
	err = os.MkdirAll(filepath.Dir(digestPath), 0o755)
	if err != nil {
		log.Errorf("Error creating directories for filestore: %v", err)
		handleError(c, 500, fmt.Sprintf("Failed to create directories for filestore: %v", err))
		return
	}
	err = os.WriteFile(digestPath, []byte(dgst.String()), 0o644)
	if err != nil {
		log.Errorf("Error writing digest file: %v", err)
		handleError(c, 500, fmt.Sprintf("Failed to write digest file: %v", err))
		return
	}
}

func handleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}

func (r *DebianRemoteHandler) forwardRequest(c *gin.Context, repoKey, path string, repoConfig *configstore.RepoConfig) *http.Response {
	reqClone := c.Request.Clone(c.Request.Context())
	// Remove /debian/<repoKey>/ prefix from path
	path = strings.Replace(path, fmt.Sprintf("/debian/%s/", repoKey), "", 1)
	remoteUrl := fmt.Sprintf("%s/%s", repoConfig.RemoteURL, path)
	log.Infof("Forwarding request to upstream Debian repo: %s", remoteUrl)
	req, _ := http.NewRequest(reqClone.Method, remoteUrl, reqClone.Body)
	req.Header = reqClone.Header
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error forwarding request: %v", err)
		c.JSON(502, gin.H{
			"error": "Bad Gateway",
		})
		return nil
	}
	// Copy response headers and status code to Gin context
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	return resp
}

func (r *DebianRemoteHandler) writeResponse(c *gin.Context, resp *http.Response) {
	if resp == nil {
		handleError(c, 500, "failed to forward request")
		return
	}
	// Copy response headers and status code to Gin context
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	// Copy response body to Gin context
	_, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		handleError(c, 500, fmt.Sprintf("Failed to read upstream InRelease response: %v", err))
		return
	}
}

type DebianFilestore struct {
	baseDir string
}

func NewDebianFilestore(basePath string) *DebianFilestore {
	return &DebianFilestore{
		baseDir: basePath,
	}
}

// atomicWrite writes to a temp file and renames it atomically.
// func atomicWrite(targetPath string, data []byte) error {
// 	tmpPath := targetPath + ".tmp"
// 	f, err := os.Create(tmpPath)
// 	if err != nil {
// 		return err
// 	}
// 	if err := f.Close(); err != nil {
// 		return err
// 	}

// 	if _, err := f.Write(data); err != nil {
// 		return err
// 	}
// 	if err := f.Sync(); err != nil {
// 		return err
// 	}
// 	return os.Rename(tmpPath, targetPath)
// }

// CachedEntry represents the digest and metadata for a cached file.
type CacheEntry struct {
	RepoKey   string                 `json:"repoKey"`
	Path      string                 `json:"path"`
	CacheKey  string                 `json:"cacheKey"`  // SHA256 of canonical path
	Digest    string                 `json:"digest"`    // Digest of blob content
	Metadata  map[string]interface{} `json:"metadata"`  // Arbitrary metadata
	Timestamp string                 `json:"timestamp"` // RFC3339
}

// GenerateCacheEntry creates a cache key and metadata for a given repoKey and path.
func GenerateCacheEntry(repoKey, relPath, blobDigest string, headers map[string]string) (*CacheEntry, error) {
	// Canonical path (normalized)
	canonicalPath := filepath.ToSlash(relPath)
	// Cache key = SHA256 of canonical path
	hash := sha256.Sum256([]byte(canonicalPath))
	cacheKey := hex.EncodeToString(hash[:])
	// Metadata extraction
	metadata := map[string]interface{}{
		"status":         headers["Status"],
		"etag":           headers["Etag"],
		"last_modified":  headers["Last-Modified"],
		"content_length": headers["Content-Length"],
		"cache_control":  headers["Cache-Control"],
		"x_cache":        headers["X-Cache"],
		"backend":        headers["Backend"],
	}
	return &CacheEntry{
		RepoKey:   repoKey,
		Path:      canonicalPath,
		CacheKey:  cacheKey,
		Digest:    blobDigest,
		Metadata:  metadata,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// ReadDigestAndMetadata reads and validates the digest and metadata for a given repoKey and path.
func ReadDigestAndMetadata(repoKey, relPath string) (*CacheEntry, error) {
	baseDir := "/tmp/filestore"
	digestPath := filepath.Join(baseDir, repoKey, relPath)
	metaPath := digestPath + ".json"

	// Read digest
	digestBytes, err := os.ReadFile(digestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read digest: %w", err)
	}
	digest := string(digestBytes)

	// Read metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(metaBytes, &metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata JSON: %w", err)
	}
	return &CacheEntry{
		Digest:   digest,
		Metadata: metadata,
	}, nil
}
