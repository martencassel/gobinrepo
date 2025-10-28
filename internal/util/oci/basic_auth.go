package oci

import "net/http"

type BasicAuthRoundTripper struct {
	Username string
	Password string
	Base     http.RoundTripper
}

func (rt *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.Username != "" {
		req.SetBasicAuth(rt.Username, rt.Password)
	}
	return rt.Base.RoundTrip(req)
}
