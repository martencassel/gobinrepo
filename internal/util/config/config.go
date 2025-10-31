package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/martencassel/gobinrepo/internal/configstore"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Listen    string `yaml:"listen"`
		Trace     bool   `yaml:"trace"`
		PublicURL string `yaml:"public_url"`
	} `yaml:"server"`

	Cache struct {
		Path string `yaml:"path"`
	} `yaml:"cache"`

	Remotes map[string]RemoteConfig `yaml:"remotes"`
}

type RemoteConfig struct {
	PackageType configstore.PackageType `yaml:"package_type"`
	RemoteURL   string                  `yaml:"remote_url"`
	Username    *string                 `yaml:"username,omitempty"`
	Password    *string                 `yaml:"password,omitempty"`
}

// LoadConfig reads a YAML config file, expands env vars, and unmarshals into Config.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Expand ${VAR} placeholders
	expanded := os.Expand(string(raw), func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			// Leave placeholder intact so we can detect it later
			return "${" + key + "}"
		}
		return val
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Normalize env placeholders → nil
	for name, r := range cfg.Remotes {
		r.Username = normalizeEnv(r.Username)
		r.Password = normalizeEnv(r.Password)
		cfg.Remotes[name] = r
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// normalizeEnv turns "${VAR}" or "" into nil, leaves real values intact.
func normalizeEnv(s *string) *string {
	if s == nil {
		return nil
	}
	if *s == "" {
		return nil
	}
	if strings.HasPrefix(*s, "${") && strings.HasSuffix(*s, "}") {
		return nil
	}
	return s
}

func applyDefaults(cfg *Config) {
	// Server defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":5000"
	}
	// Cache defaults
	if cfg.Cache.Path == "" {
		cfg.Cache.Path = "/tmp/gobinrepo/cache"
	}
	if cfg.Server.PublicURL == "" {
		cfg.Server.PublicURL = "http://localhost:5000"
	}
}
