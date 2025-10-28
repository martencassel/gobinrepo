package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/mw"
	"github.com/martencassel/gobinrepo/internal/remote"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
)

func main() {
	httpListenAddr := flag.String("http-listen-addr", ":5000",
		"HTTP listen address (e.g. ':5000', '127.0.0.1:8080')")
	flag.Parse()

	router, err := buildRouter()
	if err != nil {
		panic(err)
	}
	if err := router.Run(*httpListenAddr); err != nil {
		panic(err)
	}
}

func buildRouter() (*gin.Engine, error) {
	r := gin.Default()
	r.Use(mw.LoggingMiddleware())
	blobs, err := blobs.NewBlobStoreFS("/tmp/blobs")
	if err != nil {
		panic(err)
	}
	docker := remote.NewDockerRemoteHandler(blobs)
	docker.RegisterRoutes(r)
	mw := mw.NewRepoKeyMiddleware()
	r.Use(mw.Middleware())
	return r, nil
}
