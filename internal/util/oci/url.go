package oci

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// NormalizeNamespace strips the local namespace and replaces it with the upstream's.
func NormalizeNamespace(localNS, upstreamNS, name string) string {
	// If the name already has a namespace, drop it.
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		name = parts[1]
	}
	return path.Join(upstreamNS, name)
}

func (o OCIURL) UpstreamURL(resolver NamespaceResolver) string {

	// 1) Strip local namespace (repoKey) from the incoming name
	stripped := o.Name.StripNamespace()

	// 2) Resolve upstream namespace policy given the stripped name
	//    For Docker Hub, empty namespace => "library"
	upstreamNS := resolver.ResolveNamespace(stripped)

	// 3) Build normalized upstream path
	normalized := path.Join(upstreamNS, stripped.Rest())

	return fmt.Sprintf("https://%s/v2/%s/%s/%s",
		o.RegistryHost,
		normalized,
		o.Subresource,
		o.Reference,
	)
}

func ResolverForHost(host string) NamespaceResolver {
	switch host {
	case "registry-1.docker.io", "docker.io":
		return DockerHubResolver{}
	default:
		return DefaultResolver{}
	}
}

type OCIURL struct {
	RegistryHost string
	Name         RepositoryName
	Subresource  string
	Reference    Reference
	Digest       string
}

func ParseOCIURL(s string) (OCIURL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return OCIURL{}, err
	}
	if !strings.HasPrefix(u.Path, "/v2/") {
		return OCIURL{}, fmt.Errorf("invalid OCI URL, missing /v2/ prefix: %q", u.Path)
	}
	path := strings.TrimPrefix(u.Path, "/v2/")
	path = strings.TrimPrefix(path, "/")

	var subResource, namePart, refPart string
	switch {
	case strings.Contains(path, "/manifests/"):
		parts := strings.SplitN(path, "/manifests/", 2)
		namePart, refPart = parts[0], parts[1]
		subResource = "manifests"
	case strings.Contains(path, "/blobs/"):
		parts := strings.SplitN(path, "/blobs/", 2)
		namePart, refPart = parts[0], parts[1]
		subResource = "blobs"
	default:
		return OCIURL{}, fmt.Errorf("invalid OCI URL, missing manifests or blobs: %q", path)
	}

	repoName, err := ParseRepositoryName(namePart)
	if err != nil {
		return OCIURL{}, fmt.Errorf("invalid repository name: %v", err)
	}
	ref, err := ParseReference(refPart)
	if err != nil {
		return OCIURL{}, fmt.Errorf("invalid reference: %v", err)
	}

	return OCIURL{
		RegistryHost: u.Host,
		Name:         repoName,
		Subresource:  subResource,
		Reference:    ref,
	}, nil
}

func (o OCIURL) String() string {
	return fmt.Sprintf("https://%s/v2/%s/%s/%s", o.RegistryHost, o.Name.String(), o.Subresource, o.Reference.String())
}

func (o OCIURL) IsManifest() bool {
	return o.Subresource == "manifests"
}

func (o OCIURL) IsBlob() bool {
	return o.Subresource == "blobs"
}
