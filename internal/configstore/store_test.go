package configstore

import (
	"reflect"
	"testing"
)

func TestRepoConfigStore(t *testing.T) {
	store := NewRepoConfigStore()

	// Initially empty
	if got := store.List(); len(got) != 0 {
		t.Fatalf("expected empty store, got %v", got)
	}

	// Add a config
	cfg := RepoConfig{RepoKey: "dockerhub", RemoteURL: "https://registry-1.docker.io"}
	store.Add(cfg)

	// Get should return it
	got, ok := store.Get("dockerhub")
	if !ok {
		t.Fatalf("expected to find repoKey dockerhub")
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Errorf("expected %+v, got %+v", cfg, got)
	}

	// List should contain it
	list := store.List()
	if len(list) != 1 || list[0].RepoKey != "dockerhub" {
		t.Errorf("expected list with dockerhub, got %+v", list)
	}

	// Add another config
	cfg2 := RepoConfig{RepoKey: "quay", RemoteURL: "https://quay.io"}
	store.Add(cfg2)

	// Both should be present
	if len(store.List()) != 2 {
		t.Errorf("expected 2 configs, got %d", len(store.List()))
	}

	// Delete one
	store.Delete("dockerhub")
	if _, ok := store.Get("dockerhub"); ok {
		t.Errorf("expected dockerhub to be deleted")
	}
	if len(store.List()) != 1 {
		t.Errorf("expected 1 config after delete, got %d", len(store.List()))
	}
}
