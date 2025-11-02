package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	log "github.com/sirupsen/logrus"
	repo "helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type HelmRepoHandler struct {
	blobs blobs.BlobStore
	store *configstore.RepoConfigStore

	// Delegates for test injection
	onRedirect func(*gin.Context)
	onIndex    func(*gin.Context)
	onChart    func(*gin.Context)
}

func NewHelmRepoHandler(blobs blobs.BlobStore, store *configstore.RepoConfigStore) *HelmRepoHandler {
	h := &HelmRepoHandler{
		blobs: blobs,
		store: store,
	}
	// default to real methods
	h.onRedirect = h.handleRedirectedChartFile
	h.onIndex = h.handleIndex
	h.onChart = h.handleChartFile
	return h
}

func (h *HelmRepoHandler) Register(c *gin.Engine) {
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
	case strings.Contains(rest, "external/https"):
		h.onRedirect(c)
	case strings.Contains(rest, "index.yaml") && strings.HasSuffix(rest, "index.yaml"):
		h.onIndex(c)
	case strings.HasSuffix(rest, ".tgz") || strings.HasSuffix(rest, ".tar.gz"):
		h.onChart(c)
	default:
		c.String(404, "not found")
	}
}

func (h *HelmRepoHandler) handleRedirectedChartFile(c *gin.Context) {
	repoName := c.Param("repoKey")
	log.Infof("Handling redirected Helm chart request for repoKey: %s", repoName)

	u, err := url.Parse(c.Request.URL.String())
	if err != nil {
		c.String(400, "invalid URL: %v", err)
		return
	}

	// Extract string after external/https/
	externalPath := strings.SplitN(u.Path, "external/", 2)[1]
	externalURL, err := url.QueryUnescape(externalPath)
	if err != nil {
		c.String(400, "invalid external URL: %v", err)
		return
	}

	// Replace https/ with https://
	externalURL = strings.Replace(externalURL, "https/", "https://", 1)
	externalURL = strings.Replace(externalURL, "http/", "http://", 1)

	// Fetch the chart from the external URL

	log.Infof("Fetching Helm chart from external URL: %s", externalURL)
	log.Infof("Forwarding Helm chart file request to external URL: %s", externalURL)

	res, err := http.Get(externalURL)
	if err != nil {
		c.String(500, "failed to fetch from external_url: %v", err)
		return
	}
	defer res.Body.Close()

	// Copy response headers
	for k, v := range res.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Status(res.StatusCode)
	// Copy response body to client
	_, err = io.Copy(c.Writer, res.Body)
	if err != nil {
		c.String(500, "failed to copy response body: %v", err)
		return
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
	}

	// Read the entire body
	data, err := io.ReadAll(res.Body)
	if err != nil {
		c.String(500, "failed to read index.yaml: %v", err)
		return
	}
	index, err := LoadIndexReader(bytes.NewReader(data))
	if err != nil {
		c.String(500, "failed to load index.yaml: %v", err)
		return
	}

	// Rewrite
	RewriteAbsoluteChartURLs(index)

	StripDeprecatedFieldsReflect(index)
	//	StripDeprecatedFields(index)

	// Marshal back
	rewritten, err := yaml.Marshal(index)
	if err != nil {
		c.String(500, "failed to marshal rewritten index.yaml: %v", err)
		return
	}

	// Adjust headers
	c.Writer.Header().Del("Content-Length")
	c.Writer.Header().Set("Content-Type", "application/x-yaml")
	c.Status(res.StatusCode)

	// Write response
	c.Writer.Write(rewritten)
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

// rewriteHelmURL takes an absolute .tgz URL and rewrites it into
// "<absolute-url>" -> "external/<safe-version-of-absolute-url>"
func rewriteHelmURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() {
		return "", fmt.Errorf("URL is not absolute: %s", raw)
	}
	// Encode scheme + host + path into a safe path segment
	safe := strings.ReplaceAll(raw, "://", "/")
	safe = strings.ReplaceAll(safe, "/", "/")
	return "external/" + safe, nil
}

// LoadIndexReader reads all data from an io.Reader and returns an IndexFile
// by writing to a temporary file and delegating to repo.LoadIndexFile.
func LoadIndexReader(r io.Reader) (*repo.IndexFile, error) {
	// Create a temp file
	tmp, err := os.CreateTemp("", "index-*.yaml")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name()) // clean up after ourselves
	defer tmp.Close()

	// Copy the reader into the temp file
	if _, err := io.Copy(tmp, r); err != nil {
		return nil, err
	}

	// Ensure file is flushed
	if err := tmp.Sync(); err != nil {
		return nil, err
	}

	// Now use Helm’s built-in loader
	idx, err := repo.LoadIndexFile(tmp.Name())
	if err != nil {
		return nil, err
	}

	// LoadIndexFile already calls SortEntries internally,
	// so you don’t need to call idx.SortEntries() again.
	return idx, nil
}

// RewriteAbsoluteChartURLs walks through an IndexFile and rewrites all absolute .tgz URLs
// into the shorter proxy form using rewriteHelmURL.
func RewriteAbsoluteChartURLs(index *repo.IndexFile) {
	const (
		colorRed   = "\033[31m"
		colorBlue  = "\033[34m"
		colorReset = "\033[0m"
	)
	for name, versions := range index.Entries {
		for _, ver := range versions {
			for i, u := range ver.URLs {
				parsed, err := url.Parse(u)
				if err != nil {
					continue
				}
				if parsed.IsAbs() && strings.HasSuffix(strings.ToLower(parsed.Path), ".tgz") {
					fmt.Printf("%sAbsolute tgz URL in chart %s: %s%s\n",
						colorRed, name, u, colorReset)

					rewritten, err := rewriteHelmURL(u)
					if err != nil {
						log.Printf("failed to rewrite URL %s: %v", u, err)
						continue
					}
					fmt.Printf("%sRewritten URL: %s%s\n",
						colorBlue, rewritten, colorReset)

					ver.URLs[i] = rewritten
				}
			}
		}
	}
}

// StripDeprecatedFields removes deprecated or unwanted fields from an IndexFile
func StripDeprecatedFields(index *repo.IndexFile) {
	// Example: clear top-level Generated timestamp if you don’t want it
	index.Generated = time.Time{}

	for _, versions := range index.Entries {
		for _, ver := range versions {
			ver.ChecksumDeprecated = ""
			ver.EngineDeprecated = ""
			ver.TillerVersionDeprecated = ""
			ver.URLDeprecated = ""
			// also clear Created/Digest/Removed if you want them gone
			ver.Created = time.Time{}
			ver.Digest = ""
			ver.Removed = false

		}
	}
}

// StripDeprecatedFieldsReflect uses reflection to zero out fields with "Deprecated" in their name.
func StripDeprecatedFieldsReflect(index *repo.IndexFile) {
	for _, versions := range index.Entries {
		for _, ver := range versions {
			v := reflect.ValueOf(ver).Elem()
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				fieldType := v.Type().Field(i)
				if strings.Contains(fieldType.Name, "Deprecated") {
					if field.CanSet() {
						field.Set(reflect.Zero(field.Type()))
					}
				}
			}
			// also clear Created/Digest/Removed if you want
			ver.Created = time.Time{}
			ver.Digest = ""
			ver.Removed = false
		}
	}
}
