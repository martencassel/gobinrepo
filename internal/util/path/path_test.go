package path

import (
	"fmt"
	"testing"
)

func TestPath(t *testing.T) {
	samples := []string{
		"/api/repo",
		"/docker/library/ubuntu/latest",
		"/v2/acme/widgets/blob",
	}

	for _, s := range samples {
		p := ParsePath(s)
		switch v := p.(type) {
		case RepoAPIPath:
			fmt.Println("Repo API endpoint")
		case PackagePath:
			fmt.Printf("PackageType=%s RepoKey=%s Rest=%s\n",
				v.PackageType, v.RepoKey, v.Rest)
		case V2Path:
			fmt.Printf("Namespace=%s Rest=%s\n", v.Namespace, v.Rest)
		default:
			fmt.Println("Unrecognized path:", s)
		}
	}
}
