package trace

import (
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	yellowBlock = "\033[43;30m" // yellow background, black text
	resetColor  = "\033[0m"
)

type TracingRoundTripper struct {
	Base http.RoundTripper
}

func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base().RoundTrip(req)
	dur := time.Since(start).Round(time.Millisecond)

	// First line: request summary
	log.Infof("%s UPSTREAM %s %s (%s) %s",
		yellowBlock, req.Method, req.URL.String(), dur, resetColor)

	// Compact request headers
	var reqHdrs []string
	for k, v := range req.Header {
		reqHdrs = append(reqHdrs, k+": "+strings.Join(v, ","))
	}
	if len(reqHdrs) > 0 {
		log.Infof("%s   req: %s %s", yellowBlock, strings.Join(reqHdrs, " | "), resetColor)
	}

	if err != nil {
		log.Warnf("%s   error: %v %s", yellowBlock, err, resetColor)
		return nil, err
	}

	// Compact response headers
	var respHdrs []string
	for k, v := range resp.Header {
		respHdrs = append(respHdrs, k+": "+strings.Join(v, ","))
	}
	if len(respHdrs) > 0 {
		log.Infof("%s   resp: %d %s %s", yellowBlock, resp.StatusCode, strings.Join(respHdrs, " | "), resetColor)
	}

	return resp, nil
}

func (t *TracingRoundTripper) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}
