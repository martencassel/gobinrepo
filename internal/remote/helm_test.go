package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// fake handler that records which branch was called
type fakeHelmHandler struct {
	called string
}

func (f *fakeHelmHandler) handleRedirectedChartFile(c *gin.Context) { f.called = "redirect" }
func (f *fakeHelmHandler) handleIndex(c *gin.Context)               { f.called = "index" }
func (f *fakeHelmHandler) handleChartFile(c *gin.Context)           { f.called = "chart" }

func TestHandleHelmRequest_Dispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		path     string
		expected string
		status   int
	}{
		{"external redirect", "/helm/myrepo/external/https/github.com/chart.tgz", "redirect", http.StatusOK},
		{"index.yaml root", "/helm/myrepo/index.yaml", "index", http.StatusOK},
		{"index.yaml subdir", "/helm/myrepo/subdir/index.yaml", "index", http.StatusOK},
		{"chart tgz", "/helm/myrepo/foo-1.0.0.tgz", "chart", http.StatusOK},
		{"chart tar.gz", "/helm/myrepo/foo-1.0.0.tar.gz", "chart", http.StatusOK},
		{"not found", "/helm/myrepo/unknown.txt", "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeHelmHandler{}
			h := &HelmRepoHandler{} // real type
			// monkey‑patch methods for testing
			h.onRedirect = f.handleRedirectedChartFile
			h.onIndex = f.handleIndex
			h.onChart = f.handleChartFile

			// build router
			r := gin.New()
			r.GET("/helm/:repoKey/*path", h.handleHelmRequest)

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, f.called)
		})
	}
}
