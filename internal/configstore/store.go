package configstore

import "sync"

// RepoConfig represents a mapping from repoKey → remote registry URL.
type RepoConfig struct {
	RepoKey   string `json:"repoKey"`
	RemoteURL string `json:"remoteURL"`
}

// RepoConfigStore is an in-memory store for repo configurations.
type RepoConfigStore struct {
	mu      sync.RWMutex
	configs map[string]RepoConfig
}

// NewRepoConfigStore creates an empty store.
func NewRepoConfigStore() *RepoConfigStore {
	return &RepoConfigStore{
		configs: make(map[string]RepoConfig),
	}
}

// Add inserts or updates a repoKey mapping.
func (s *RepoConfigStore) Add(cfg RepoConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[cfg.RepoKey] = cfg
}

// Get retrieves a repo config by key.
func (s *RepoConfigStore) Get(repoKey string) (RepoConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configs[repoKey]
	return cfg, ok
}

// Delete removes a repo config by key.
func (s *RepoConfigStore) Delete(repoKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, repoKey)
}

// List returns all repo configs.
func (s *RepoConfigStore) List() []RepoConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]RepoConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		out = append(out, cfg)
	}
	return out
}
