package mw

import (
	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/util/path"
	log "github.com/sirupsen/logrus"
)

type RepoKeyMiddleware struct {
}

func NewRepoKeyMiddleware() *RepoKeyMiddleware {
	return &RepoKeyMiddleware{}
}

func (m *RepoKeyMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := path.ParsePath(c.Request.URL.Path)
		repoKey, ok := path.RepoKeyFromPath(p)
		if ok {
			c.Set("RepoKey", repoKey)
		}
		log.Infof("RepoKeyMiddleware: RepoKey=%s, Path=%s", repoKey, c.Request.URL.Path)
		c.Next()
	}
}
