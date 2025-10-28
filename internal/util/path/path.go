package path

import "regexp"

var (
	rePackage = regexp.MustCompile(`^/([^/]+)/([^/]+)/(.*)$`)
	reV2      = regexp.MustCompile(`^/v2/([^/]+)/(.*)$`)
)

// PackagePath is one variant of API paths.
type PackagePath struct {
	PackageType string
	RepoKey     string
	Rest        string
}

// RepoAPIPath represents /api/repo
type RepoAPIPath struct{}

// V2Path represents /v2/<namespace>/...
type V2Path struct {
	Namespace string
	Rest      string
}

// Path is the sum type interface
type Path interface {
	Kind() string
}

func (p PackagePath) Kind() string { return "package" }
func (RepoAPIPath) Kind() string   { return "repoAPI" }
func (p V2Path) Kind() string      { return "v2" }

func RepoKeyFromPath(p Path) (string, bool) {
	if pkg, ok := p.(PackagePath); ok {
		return pkg.RepoKey, true
	}
	return "", false
}

func ParsePath(path string) Path {
	switch {
	case path == "/api/repo":
		return RepoAPIPath{}

	case rePackage.MatchString(path):
		m := rePackage.FindStringSubmatch(path)
		return PackagePath{
			PackageType: m[1],
			RepoKey:     m[2],
			Rest:        m[3],
		}

	case reV2.MatchString(path):
		m := reV2.FindStringSubmatch(path)
		return V2Path{
			Namespace: m[1],
			Rest:      m[2],
		}

	default:
		return nil
	}
}
