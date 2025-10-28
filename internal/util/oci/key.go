package oci

import (
	"net/http"
	"path"
)

type FilestoreKey struct {
	Namespace string
	Key       string
	Metadata  map[string]string
}

func (u OCIURL) FilestoreKey(resp *http.Response) FilestoreKey {
	return FilestoreKey{
		Namespace: u.Name.Namespace(),
		Key:       path.Join(u.Name.Rest(), u.Subresource, u.Reference.String()),
		Metadata: map[string]string{
			"digest": resp.Header.Get("Docker-Content-Digest"),
			"tag":    u.Reference.String(),
		},
	}
}
