package helm

import (
	"os"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestHelm(t *testing.T) {
	repoKeyMap := make(map[string]string)
	m1, m2, err := BuildMapper("testdata/argo-cd-index.yaml", "https://proxy.example.com/helm/helm-remote", repoKeyMap)
	if err != nil {
		t.Fatalf("failed to build mapper: %v", err)
	}
	data, err := yaml.Marshal(repoKeyMap)
	if err != nil {
		t.Fatalf("failed to marshal repoKeyMap: %v", err)
	}
	os.WriteFile("/tmp/input.yaml", data, 0644)
	data, err = yaml.Marshal(m1)
	if err != nil {
		t.Fatalf("failed to marshal depRepoMap: %v", err)
	}
	os.WriteFile("/tmp/dep-repo-map.yaml", data, 0644)
	data, err = yaml.Marshal(m2)
	if err != nil {
		t.Fatalf("failed to marshal chartUrlMap: %v", err)
	}
	os.WriteFile("/tmp/chart-url-map.yaml", data, 0644)

}
