package oci

// NamespaceResolver decides how to normalize a name for a given registry.
type NamespaceResolver interface {
	ResolveNamespace(n RepositoryName) string
}

// DockerHubResolver applies the "library" default for single-segment names.
type DockerHubResolver struct{}

func (DockerHubResolver) ResolveNamespace(n RepositoryName) string {
	if n.Namespace() == "" {
		return "library"
	}
	return n.Namespace()
}

// DefaultResolver leaves the namespace untouched.
type DefaultResolver struct{}

func (DefaultResolver) ResolveNamespace(n RepositoryName) string {
	return n.Namespace()
}
