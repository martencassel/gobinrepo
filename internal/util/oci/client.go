package oci

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

type RegistryClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewRegistryClient constructs a client that talks to an upstream registry
// using the provided TokenRoundTripper for Bearer-token auth.
func NewRegistryClient(baseURL string, rt http.RoundTripper, opts ...func(*http.Client)) *RegistryClient {
	if rt == nil {
		defaultRT := http.DefaultTransport
		rt = defaultRT
	}
	client := &http.Client{
		Transport: rt,
		Timeout:   0,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return &RegistryClient{
		httpClient: client,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}

}

// Ping checks if the registry is alive.
func (c *RegistryClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry ping failed: %s", resp.Status)
	}
	return nil
}

// OCIManifest is a minimal struct for OCI/Docker v2 manifests.
// Extend with full schema as needed.
type OCIManifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        Descriptor   `json:"config"`
	Layers        []Descriptor `json:"layers"`
}

type Descriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// FetchManifest retrieves an image manifest from the registry.
// The caller is responsible for closing resp.Body.
func (c *RegistryClient) FetchManifest(ctx context.Context, repo, reference string, hdr http.Header) (*http.Response, error) {
	u := c.baseURL + "/v2/" + repo + "/manifests/" + reference
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if hdr != nil {
		copyForwardHeaders(req.Header, hdr)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", v1.MediaTypeImageManifest)
	}
	return c.httpClient.Do(req)
}

// GetBlob fetches a blob (layer) from the registry and streams it into the provided writer.
// It automatically closes the response body.
// Returns an error if the fetch fails or the copy fails.
func (c *RegistryClient) GetBlob(ctx context.Context, repo, digest string, hdr http.Header) (*http.Response, error) {
	resp, err := c.FetchBlob(ctx, repo, digest, hdr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blob fetch failed (%s): %s", digest, resp.Status)
	}
	return resp, nil
}

// FetchBlob retrieves a blob (layer) from the registry.
// The caller is responsible for closing resp.Body.
func (c *RegistryClient) FetchBlob(ctx context.Context, repo, digest string, hdr http.Header) (*http.Response, error) {
	u := c.baseURL + "/v2/" + repo + "/blobs/" + digest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	copyForwardHeaders(req.Header, hdr)
	return c.httpClient.Do(req)
}

// GetManifest fetches and decodes an image manifest into an OCIManifest struct.
// It closes the response body automatically.
func (c *RegistryClient) GetManifest(ctx context.Context, repo, reference string, hdr http.Header) (*http.Response, error) {
	reqRef := fmt.Sprintf("%s@%s", repo, reference)
	log.Printf("Fetching manifest for %s", reqRef)
	resp, err := c.FetchManifest(ctx, repo, reference, hdr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("manifest fetch failed: %s", resp.Status)
	}
	return resp, nil
}

// HeadManifest performs a HEAD request for the specified manifest.
func (c *RegistryClient) HeadManifest(ctx context.Context, repo, reference string, hdr http.Header) (*http.Response, error) {
	u := c.baseURL + "/v2/" + repo + "/manifests/" + reference
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return nil, err
	}
	copyForwardHeaders(req.Header, hdr)
	return c.httpClient.Do(req)
}

// ForwardRequest is a generic method to forward an arbitrary downstream request to the upstream registry,
// preserving method and headers (with filtering). Body is not reused (for safety) unless provided explicitly.
func (c *RegistryClient) ForwardRequest(ctx context.Context, method, upstreamPath string, body io.Reader, hdr http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+upstreamPath, body)
	if err != nil {
		return nil, err
	}
	copyForwardHeaders(req.Header, hdr)
	return c.httpClient.Do(req)
}

func (c *RegistryClient) StreamAndCache(
	w http.ResponseWriter,
	resp *http.Response,
	cacheWriter io.Writer,
) error {
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()
	// Copy headers/status first
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	// TeeReader duplicates the stream: downstream + cache
	tee := io.TeeReader(resp.Body, cacheWriter)
	// Stream to client while writing to cache
	_, err := io.Copy(w, tee)
	return err
}

// StreamForwardRequest forwards and streams directly into an http.ResponseWriter.
func (c *RegistryClient) StreamForwardRequest(
	ctx context.Context,
	w http.ResponseWriter,
	method, upstreamPath string,
	body io.Reader,
	hdr http.Header,
) error {
	resp, err := c.ForwardRequest(ctx, method, upstreamPath, body, hdr)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()

	// Copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream body
	_, err = io.Copy(w, resp.Body)
	return err
}

// copyForwardHeaders clones end-to-end headers while stripping hop-by-hop and downstream Authorization.
// RFC 7230 §6.1 hop-by-hop headers must not be forwarded by proxies.
func copyForwardHeaders(dst, src http.Header) {
	if src == nil {
		return
	}
	// Lowercased set of hop-by-hop headers to drop.
	hopByHop := map[string]struct{}{
		"connection":          {},
		"keep-alive":          {},
		"proxy-authenticate":  {},
		"proxy-authorization": {},
		"te":                  {},
		"trailer":             {},
		"transfer-encoding":   {},
		"upgrade":             {},
		"authorization":       {}, // handled by TokenRoundTripper upstream
		"host":                {}, // upstream host is set by client/transport
	}

	for k, vv := range src {
		if _, skip := hopByHop[strings.ToLower(k)]; skip {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
