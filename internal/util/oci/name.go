package oci

import (
	"fmt"
	"regexp"
	"strings"
)

type RepositoryName struct {
	components []string
}

const (
	compRegExp = `[a-z0-9]+(?:[._-][a-z0-9]+)*`
)

func ParseRepositoryName(s string) (RepositoryName, error) {
	parts := strings.Split(s, "/")
	r := regexp.MustCompile(compRegExp)
	for _, p := range parts {
		if !r.MatchString(p) {
			return RepositoryName{}, fmt.Errorf("invalid repository component: %q", p)
		}
	}
	return RepositoryName{components: parts}, nil
}

func (r RepositoryName) Namespace() string {
	return r.components[0]
}

func (r RepositoryName) Head() string {
	return r.components[0]
}

func (r RepositoryName) Components() []string {
	return r.components
}
func (r RepositoryName) Rest() string {
	return strings.Join(r.components[1:], "/")
}

func (r RepositoryName) IsSingleComponentRest() bool {
	return len(r.Rest()) == 1
}

func (r RepositoryName) String() string {
	return strings.Join(r.components, "/")
}

func (n RepositoryName) NamespaceOrDefault() string {
	ns := n.Namespace()
	if ns == "" {
		return "library"
	}
	return ns
}

func (r RepositoryName) StripNamespace() RepositoryName {
	if len(r.components) <= 1 {
		// No namespace to strip; keep as-is (will be resolved by resolver).
		return RepositoryName{components: r.components}
	}
	return RepositoryName{components: r.components[1:]}
}

// WithNamespace returns a copy of the repository name with the namespace replaced.
// If the repository has only one component, the new namespace is prepended.
func (r RepositoryName) WithNamespace(ns string) RepositoryName {
	ns = strings.ToLower(ns)
	switch len(r.components) {
	case 0:
		return RepositoryName{components: []string{ns}}
	case 1:
		return RepositoryName{components: []string{ns, r.components[0]}}
	default:
		return RepositoryName{components: append([]string{ns}, r.components[1:]...)}
	}
}
