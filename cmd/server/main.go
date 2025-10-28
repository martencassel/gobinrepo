package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
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

func buildConfigStore() (*configstore.RepoConfigStore, error) {
	// For simplicity, using an in-memory config store with a single repo config
	store := configstore.NewRepoConfigStore()
	store.Add(configstore.RepoConfig{
		RepoKey:   "dockerhub",
		RemoteURL: "https://registry-1.docker.io",
	})
	store.Add(configstore.RepoConfig{
		RepoKey:   "quayio",
		RemoteURL: "https://quay.io",
	})
	return store, nil
}

func buildRouter() (*gin.Engine, error) {
	r := gin.Default()
	r.Use(mw.LoggingMiddleware())
	blobs, err := blobs.NewBlobStoreFS("/tmp/blobs")
	if err != nil {
		panic(err)
	}
	configStore, err := buildConfigStore()
	if err != nil {
		return nil, err
	}
	mw := mw.NewRepoKeyMiddleware()
	r.Use(mw.Middleware())
	docker := remote.NewDockerRemoteHandler(blobs, configStore, true)
	docker.RegisterRoutes(r)
	return r, nil
}
