package oci

import (
	"fmt"

	digest "github.com/opencontainers/go-digest"
)

func ParseDigestURL(raw string, defaultHost string) (OCIURL, digest.Digest, error) {
	u, err := ParseOCIURL(raw)
	if err != nil {
		return OCIURL{}, "", fmt.Errorf("invalid OCI URL: %w", err)
	}
	u.RegistryHost = defaultHost

	if u.Reference.IsTag() {
		return OCIURL{}, "", fmt.Errorf("expected digest reference, got tag: %s", u.Reference.String())
	}
	d, err := digest.Parse(u.Reference.String())
	if err != nil {
		return OCIURL{}, "", fmt.Errorf("invalid digest: %w", err)
	}
	return u, d, nil
}
