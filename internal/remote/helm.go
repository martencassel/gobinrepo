package remote

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	log "github.com/sirupsen/logrus"
	repo "helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type HelmRepoHandler struct {
	blobs blobs.BlobStore
	store *configstore.RepoConfigStore
}

func NewHelmRepoHandler(blobs blobs.BlobStore, store *configstore.RepoConfigStore) *HelmRepoHandler {
	return &HelmRepoHandler{
		blobs: blobs,
		store: store,
	}
}

func (h *HelmRepoHandler) Register(c *gin.Engine) {
	//c.GET("/helm/:repoKey/index.yaml", h.handleIndex)
	c.GET("/helm/:repoKey/*path", h.handleHelmRequest)
}

func (h *HelmRepoHandler) handleHelmRequest(c *gin.Context) {
	repoKey := c.Param("repoKey")
	rest := strings.TrimPrefix(c.Param("path"), "/")
	log.Infof("Received Helm request: repoKey=%s, path=%s", repoKey, rest)
	log.WithFields(log.Fields{
		"repo_key": repoKey,
		"path":     rest,
	}).Info("Handling Helm request")
	switch {
	case strings.Contains(rest, "index.yaml") && strings.HasSuffix(rest, "index.yaml"):
		h.handleIndex(c)
	case strings.HasSuffix(rest, ".tgz") || strings.HasSuffix(rest, ".tar.gz"):
		h.handleChartFile(c)
	default:
		c.String(404, "not found")
	}
}

func (h *HelmRepoHandler) handleChartFile(c *gin.Context) {
	repoName := c.Param("repoKey")
	log.Infof("Handling Helm chart request for repoKey: %s", repoName)
	repoConfig, ok := h.store.Get(repoName)
	if !ok {
		c.String(404, "repository not found")
	}
	log.Info("Handling Helm Chart file request")
	log.Infof("repoConfig: %v", repoConfig)
	path := c.Request.URL.Path
	// Normalize path to remove /helm/:repoKey/ prefix
	path = path[len("/helm/"+repoName+"/"):]
	// Forward the request to the remote Helm repo
	log.Infof("Forwarding Helm chart file request to remote: repo=%s, path=%s", repoName, path)
	res := h.forwardRequest(c, repoName, path)
	if res == nil {
		return
	}
	// Copy response headers
	for k, v := range res.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Status(res.StatusCode)
	// Copy response body to client
	_, err := io.Copy(c.Writer, res.Body)
	if err != nil {
		c.String(500, "failed to copy response body: %v", err)
		return
	}
}

func (h *HelmRepoHandler) handleIndex(c *gin.Context) {
	repoName := c.Param("repoKey")
	log.Infof("Handling Helm index request for repoKey: %s", repoName)
	repoConfig, ok := h.store.Get(repoName)
	if !ok {
		c.String(404, "repository not found")
		return
	}
	log.Infof("repoConfig: %v", repoConfig)
	log.Infof("Repo config: %s", repoConfig.String())
	if repoConfig.PackageType != configstore.PackageTypeHelm {
		c.String(400, "not a helm repository")
		return
	}
	path := c.Request.URL.Path
	// Normalize path to remove /helm/:repoKey/ prefix
	path = path[len("/helm/"+repoName+"/"):]
	// Forward the request to the remote Helm repo
	res := h.forwardRequest(c, repoName, path)
	if res == nil {
		return
	}
	// Copy response headers
	for k, v := range res.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	} // Create temp file
	tempFile, err := os.CreateTemp("", "index.yaml")
	if err != nil {
		c.String(500, "failed to create temp file: %v", err)
		return
	}
	defer os.Remove(tempFile.Name())

	// Wrap the response body in a TeeReader
	tee := io.TeeReader(res.Body, tempFile)

	// Now stream to client while also writing to temp file
	c.Status(res.StatusCode)
	for k, v := range res.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	_, err = io.Copy(c.Writer, tee)
	if err != nil {
		log.Errorf("failed to copy response body: %v", err)
		return
	}

	// At this point, tempFile contains a copy of the body
	// You can parse it afterwards if you want:
	if _, err := tempFile.Seek(0, io.SeekStart); err == nil {
		indexFile, err := repo.LoadIndexFile(tempFile.Name())
		if err == nil {
			dump, _ := yaml.Marshal(indexFile)
			log.Infof("Parsed index.yaml:\n%s", string(dump))

			// ANSI color codes
			const (
				colorCyan  = "\033[36m"
				colorGreen = "\033[32m"
				colorReset = "\033[0m"
			)

			// Print with colors
			log.Infof("%sParsed index.yaml:%s\n%s%s%s",
				colorCyan, colorReset,
				colorGreen, string(dump), colorReset,
			)
		}
	}

	// Now just forward the original response to the client
	c.Status(res.StatusCode)
	for k, v := range res.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	_, err = io.Copy(c.Writer, res.Body)
	if err != nil {
		log.Errorf("failed to copy response body: %v", err)
	}

	c.Status(res.StatusCode)
	// Copy response body to client
	_, err = io.Copy(c.Writer, res.Body)
	if err != nil {
		c.String(500, "failed to copy response body: %v", err)
		return
	}
}

func (r *HelmRepoHandler) forwardRequest(c *gin.Context, repoKey, path string) *http.Response {
	// Forward the request to the remote Helm repo.
	repoConfig, ok := r.store.Get(repoKey)
	if !ok {
		log.Infof("Repository not found in store: %s", repoKey)
		c.String(404, "repository not found")
		return nil
	}
	// Normalize the path
	upstreamPath := fmt.Sprintf("%s/%s", repoConfig.RemoteURL, path)
	client := &http.Client{}
	log.Infof("Forwarding request to upstream Helm repo: %s", upstreamPath)
	req, err := http.NewRequest("GET", upstreamPath, nil)
	if err != nil {
		c.String(500, "failed to create request: %v", err)
		return nil
	}
	// Copy headers
	for k, v := range c.Request.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		c.String(500, "failed to fetch from upstream: %v", err)
		return nil
	}
	return resp
}
