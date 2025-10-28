package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func newDefaultTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = 100        // total idle connections
	tr.MaxIdleConnsPerHost = 100 // important for registries
	tr.IdleConnTimeout = 90 * time.Second
	tr.TLSHandshakeTimeout = 10 * time.Second
	tr.ExpectContinueTimeout = 1 * time.Second
	return tr
}

type cacheKey struct {
	service string
	scope   string
}

type cachedToken struct {
	value     string
	expiresAt time.Time
}

type TokenRoundTripper struct {
	Base http.RoundTripper

	Debug           bool
	Client          *http.Client
	Username        string
	Password        string
	cache           map[cacheKey]cachedToken
	mu              sync.RWMutex
	timeout         time.Duration
	stopCh          chan struct{} // to stop cleanup when shutting down
	cleanupInterval time.Duration
	closeOnce       sync.Once
}
type TokenRoundTripperOption func(*TokenRoundTripper)

func WithBasicAuth(username, password string) TokenRoundTripperOption {
	return func(trt *TokenRoundTripper) {
		trt.Username = username
		trt.Password = password
	}
}

func WithHTTPClient(c *http.Client) TokenRoundTripperOption {
	return func(trt *TokenRoundTripper) {
		trt.Client = c
		if c.Transport != nil {
			trt.Base = c.Transport
		}
	}
}

func WithCacheCleanupInterval(d time.Duration) TokenRoundTripperOption {
	return func(trt *TokenRoundTripper) {
		trt.cleanupInterval = d
	}
}

func WithTimeout(d time.Duration) TokenRoundTripperOption {
	return func(trt *TokenRoundTripper) {
		trt.Client.Timeout = d
		trt.timeout = d
	}
}

func WithTransport(t http.RoundTripper) TokenRoundTripperOption {
	return func(trt *TokenRoundTripper) {
		trt.Base = t
		// also update the client if not overridden separately
		trt.Client.Transport = t
	}
}

func NewTokenRoundTripper(debug bool, opts ...TokenRoundTripperOption) *TokenRoundTripper {
	defaultTimeout := 30 * time.Second
	defaultTransport := newDefaultTransport()
	trt := &TokenRoundTripper{
		Base:            defaultTransport,
		Debug:           debug,
		Client:          &http.Client{Timeout: defaultTimeout, Transport: newDefaultTransport()},
		cache:           make(map[cacheKey]cachedToken),
		stopCh:          make(chan struct{}),
		cleanupInterval: 10 * time.Minute,
		timeout:         defaultTimeout,
	}

	if len(opts) == 1 {
		return trt
	}
	for _, opt := range opts {
		opt(trt)
	}
	go trt.cleanupLoop()
	return trt
}

func (trt *TokenRoundTripper) Transport() http.RoundTripper { return trt }

func (trt *TokenRoundTripper) cleanupLoop() {
	ticker := time.NewTicker(trt.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			trt.pruneExpired()
		case <-trt.stopCh:
			return
		}
	}
}

func (trt *TokenRoundTripper) pruneExpired() {
	now := time.Now()
	trt.mu.Lock()
	defer trt.mu.Unlock()
	n := 0
	for k, tok := range trt.cache {
		if now.After(tok.expiresAt) {
			delete(trt.cache, k)
			n++
		}
	}
	if n > 0 {
		log.Debugf("pruned %d expired tokens from cache\n", n)
	}

}

func (trt *TokenRoundTripper) Close() {
	trt.closeOnce.Do(func() { close(trt.stopCh) })
}

type Challenge struct {
	Realm   string
	Service string
	Scopes  []string
}

var ErrMissingRealm = errors.New("missing realm in challenge")

func ParseChallenge(header string) (*Challenge, error) {
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return nil, fmt.Errorf("unsupported auth scheme")
	}
	c := &Challenge{}
	parts := strings.Split(header[len("Bearer "):], ",")
	for _, p := range parts {
		key, val, ok := parseKV(p)
		if !ok {
			continue
		}
		applyKV(c, key, val)
	}
	if c.Realm == "" {
		return nil, ErrMissingRealm
	}
	return c, nil
}

func parseKV(part string) (key, val string, ok bool) {
	kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
	if len(kv) != 2 {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(kv[0]))
	val = strings.TrimSpace(kv[1])
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
	}
	return key, val, true
}

func applyKV(c *Challenge, key, val string) {
	switch key {
	case "realm":
		c.Realm = val
	case "service":
		c.Service = val
	case "scope":
		c.Scopes = strings.Fields(val)
	}
}

func (trt *TokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// First attempt
	resp, err := trt.Base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	// Parse challenge
	var challenge *Challenge
	for _, hdr := range resp.Header.Values("WWW-Authenticate") {
		challenge, _ = ParseChallenge(hdr)
	}
	// We’re retrying, so discard the 401 body
	_ = resp.Body.Close()
	// Fetch token
	token, err := trt.fetchToken(req.Context(), challenge)
	if err != nil {
		return nil, err
	}
	// Clone request and retry with Authorization header
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)
	return trt.Base.RoundTrip(req2)
}

func newCacheKey(service string, scopes []string) cacheKey {
	sorted := append([]string(nil), scopes...)
	sort.Strings(sorted)
	return cacheKey{
		service: strings.ToLower(strings.TrimSpace(service)),
		scope:   strings.Join(sorted, " "),
	}
}

func (trt *TokenRoundTripper) fetchToken(ctx context.Context, ch *Challenge) (string, error) {
	key := newCacheKey(ch.Service, ch.Scopes)
	if token, ok := trt.getCachedToken(key); ok {
		log.Debugf("cache hit for %s\n", key)
		return token, nil
	}
	tokenURL, err := buildTokenURL(ch)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	if trt.Username != "" {
		req.SetBasicAuth(trt.Username, trt.Password)
	}
	resp, err := trt.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("failed to close response body: %v", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}
	token, expiry, err := decodeTokenResponse(resp.Body)
	if err != nil {
		return "", err
	}
	trt.setCachedToken(key, token, expiry)
	return token, nil
}

func decodeTokenResponse(r io.Reader) (string, time.Time, error) {
	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&body); err != nil {
		return "", time.Time{}, err
	}
	token := body.Token
	if token == "" {
		token = body.AccessToken
	}
	if token == "" {
		return "", time.Time{}, fmt.Errorf("no token in response")
	}

	expiry := time.Now().Add(5 * time.Minute)
	if body.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(body.ExpiresIn) * time.Second)
	}
	return token, expiry, nil
}

func buildTokenURL(ch *Challenge) (string, error) {
	u, err := url.Parse(ch.Realm)
	if err != nil {
		return "", fmt.Errorf("invalid realm URL %q: %w", ch.Realm, err)
	}
	q := u.Query()
	if ch.Service != "" {
		q.Set("service", ch.Service)
	}
	for _, scope := range ch.Scopes {
		q.Add("scope", scope)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil

}

func (trt *TokenRoundTripper) getCachedToken(key cacheKey) (string, bool) {
	trt.mu.RLock()
	defer trt.mu.RUnlock()
	tok, ok := trt.cache[key]
	if ok && time.Now().Before(tok.expiresAt) {
		return tok.value, true
	}
	return "", false
}

func (trt *TokenRoundTripper) setCachedToken(key cacheKey, value string, expiry time.Time) {
	trt.mu.Lock()
	defer trt.mu.Unlock()
	trt.cache[key] = cachedToken{value: value, expiresAt: expiry}
}
