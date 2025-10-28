package mw

import (
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type RepoKeyMiddleware struct {
}

func NewRepoKeyMiddleware() *RepoKeyMiddleware {
	return &RepoKeyMiddleware{}
}

func (m *RepoKeyMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		repoKey := c.Param("repoKey")
		subPath := strings.TrimPrefix(c.Param("path"), "/")

		if repoKey != "" {
			c.Set("RepoKey", repoKey)
			c.Set("SubPath", subPath)
			log.Infof("RepoKeyMiddleware: repoKey=%s, subPath=%s", repoKey, subPath)
		}
		c.Next()
	}
}
