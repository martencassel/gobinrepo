package helm

import (
	"path"
	"strings"

	"helm.sh/helm/v3/pkg/repo"
)

func BuildMapper(indexPath, proxyBase string, repoKeyMap map[string]string) (
	map[string]string, // depRepoMap
	map[string]string, // chartUrlMap
	error,
) {
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, nil, err
	}

	depRepoMap := make(map[string]string)
	chartUrlMap := make(map[string]string)

	for chartName, versions := range index.Entries {
		for _, ver := range versions {
			// --- Dependencies ---
			for _, dep := range ver.Dependencies {
				upstream := dep.Repository
				if strings.HasPrefix(upstream, "http") {
					repoKey, ok := repoKeyMap[upstream]
					if !ok {
						repoKey = makeRepoKey(upstream)
						repoKeyMap[upstream] = repoKey
					}
					depRepoMap[repoKey] = upstream
					// rewrite dep.Repository here if you want to emit a new index.yaml
				}
			}

			// --- Chart URLs ---
			for _, u := range ver.URLs {
				if strings.HasPrefix(u, "http") {
					filename := path.Base(u)
					rewritten := proxyBase + "/" + chartName + "/" + filename
					chartUrlMap[rewritten] = u
					// replace ver.URLs entry with rewritten if emitting new index.yaml
				}
			}
		}
	}

	return depRepoMap, chartUrlMap, nil
}

func makeRepoKey(url string) string {
	u := strings.ToLower(url)
	u = stripTrailingSlash(u)
	key := strings.ReplaceAll(u, "://", "-")
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, ".", "-")
	key = collapseRepeatedDashes(key)
	return key
}

func collapseRepeatedDashes(s string) string {
	var result strings.Builder
	prevDash := false
	for _, char := range s {
		if char == '-' {
			if !prevDash {
				result.WriteRune(char)
				prevDash = true
			}
		} else {
			result.WriteRune(char)
			prevDash = false
		}
	}
	return result.String()
}

func stripTrailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}
