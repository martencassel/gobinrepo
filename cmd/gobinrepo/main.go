package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martencassel/gobinrepo/internal/configstore"
	"github.com/martencassel/gobinrepo/internal/mw"
	"github.com/martencassel/gobinrepo/internal/remote"
	"github.com/martencassel/gobinrepo/internal/util/blobs"
	"github.com/martencassel/gobinrepo/internal/util/config"
	log "github.com/sirupsen/logrus"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
	showVer   = flag.Bool("version", false, "Print version information and exit")
	tlsCert   = flag.String("tls-cert", "", "Path to TLS certificate file (PEM)")
	tlsKey    = flag.String("tls-key", "", "Path to TLS private key file (PEM)")
)

func main() {
	log.SetLevel(log.DebugLevel)
	httpListenAddr := flag.String("http-listen-addr", ":5000",
		"HTTP listen address (e.g. ':5000', '127.0.0.1:8080')")
	configPath := flag.String("config", "config.yaml",
		"Path to configuration file")

	env := flag.String("env", os.Getenv("APP_ENV"), "Environment (development|production)")
	publicURL := flag.String("publicurl", "", "Public URL clients should use (e.g. 'https://repo.example.com')")
	flag.Parse()
	// Handle version flag
	if *showVer {
		// Print to stdout so it can be piped/parsed
		fmt.Printf("gobinrepo %s (commit %s, built %s)\n", version, commit, buildDate)
		os.Exit(0)
	}
	fmt.Println("Public URL is:", *publicURL)
	devMode := (*env == "" || *env == "development")

	// Check if config file is provided as positional argument
	if flag.NArg() > 0 {
		*configPath = flag.Arg(0)
	}

	// Load config first to allow overriding with flags
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(err)
	}

	// Override config with command line flags
	if *publicURL != "" {
		cfg.Server.PublicURL = *publicURL
		log.Infof("Public URL overridden via flag: %s", *publicURL)
	}

	log.Infof("Using public URL: %s", cfg.Server.PublicURL)

	router, err := buildRouterWithConfig(cfg, devMode)
	if err != nil {
		panic(err)
	}
	remoteKeys := make([]string, 0, len(cfg.Remotes))
	for name := range cfg.Remotes {
		remoteKeys = append(remoteKeys, name)
	}
	for name, r := range cfg.Remotes {
		hasCreds := (r.Username != nil && r.Password != nil)
		log.WithFields(log.Fields{
			"remote":     name,
			"remote_url": r.RemoteURL,
			"has_creds":  hasCreds,
		}).Info("Configured remote")
	}

	sort.Strings(remoteKeys)
	log.Infof("gobinrepo %s (commit %s, built %s)", version, commit, buildDate)
	log.Infof("Loaded configuration file %s", *configPath)
	log.Infof("Configured remotes: %s", strings.Join(remoteKeys, ","))

	log.WithFields(log.Fields{
		"listen_addr": *httpListenAddr,
		"config":      *configPath,
		"cache_path":  cfg.Cache.Path,
		"remotes":     len(cfg.Remotes),
		"gin_mode":    gin.Mode(),
		"log_level":   log.GetLevel().String(),
	}).Info("Starting gobinrepo server")

	srv := &http.Server{
		Addr:    *httpListenAddr,
		Handler: router,
	}
	go func() {
		var err error
		if *tlsCert != "" && *tlsKey != "" {
			log.Infof("Starting HTTPS server on %s", *httpListenAddr)
			err = srv.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			log.Infof("Starting HTTP server on %s", *httpListenAddr)
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Wait for signal (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	} else {
		log.Infof("Server exiting")
	}
}

func initRouter(devMode bool) *gin.Engine {
	var r *gin.Engine
	if devMode {
		r = gin.Default() // includes Gin’s banner + logger
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
		// your own logging middleware
	}
	return r
}

func buildRouterWithConfig(cfg *config.Config, devMode bool) (*gin.Engine, error) {
	r := initRouter(devMode)
	r.Use(mw.RequestTracer())
	r.Use(mw.LoggingMiddleware())

	// Build config store from the loaded config
	store := configstore.NewRepoConfigStore()
	for name, r := range cfg.Remotes {
		if r.Username == nil || r.Password == nil {
			store.Add(configstore.RepoConfig{
				RepoKey:   name,
				RemoteURL: r.RemoteURL,
			})
			continue
		}
		store.Add(configstore.RepoConfig{
			RepoKey:   name,
			RemoteURL: r.RemoteURL,
			Username:  *r.Username,
			Password:  *r.Password,
		})
	}

	blobs, err := blobs.NewBlobStoreFS(cfg.Cache.Path)
	if err != nil {
		return nil, err
	}
	mw := mw.NewRepoKeyMiddleware()
	r.Use(mw.Middleware())
	docker := remote.NewDockerRemoteHandler(blobs, store, true)
	docker.RegisterRoutes(r)
	return r, nil
}
