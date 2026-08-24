package downloadsources

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultsUseCredentialFreeHTTPSChinaMirrors(t *testing.T) {
	config, err := Normalize(Default())
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"node":           config.NodeArchiveURL,
		"npm archive":    config.NPMArchiveURL,
		"python archive": config.PythonArchiveURL,
		"python":         config.PythonIndexURL,
		"npm registry":   config.NPMRegistryURL,
	} {
		if value == "" {
			t.Fatalf("%s default is empty", name)
		}
	}
}

func TestServiceLoadsLegacyDocumentOverNewDefaults(t *testing.T) {
	store := &memoryStore{found: true, value: []byte(`{"hermesArchiveUrl":"https://example.com/hermes.zip","nodeArchiveUrl":"https://example.com/node.zip","npmArchiveUrl":"https://example.com/npm.tgz","pythonIndexUrl":"https://example.com/simple","npmRegistryUrl":"https://example.com/npm"}`)}
	config, err := NewService(store).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if config.ArtifactPreference != PreferenceBundledFirst || config.PythonArchiveURL != Default().PythonArchiveURL {
		t.Fatalf("legacy defaults were not filled: %#v", config)
	}
}

func TestServiceMigratesPreviousBuiltInHermesArchiveURL(t *testing.T) {
	store := &memoryStore{found: true, value: []byte(`{"artifactPreference":"online-first","hermesArchiveUrl":"https://github.com/NousResearch/hermes-agent/archive/df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip","nodeArchiveUrl":"https://example.com/node.zip","npmArchiveUrl":"https://example.com/npm.tgz","pythonArchiveUrl":"https://example.com/python.tar.gz","pythonIndexUrl":"https://example.com/simple","npmRegistryUrl":"https://example.com/npm"}`)}
	config, err := NewService(store).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if config.HermesArchiveURL != Default().HermesArchiveURL {
		t.Fatalf("HermesArchiveURL = %q, want current built-in URL %q", config.HermesArchiveURL, Default().HermesArchiveURL)
	}
}

func TestServicePreservesCustomHermesArchiveURL(t *testing.T) {
	const customURL = "https://mirror.example/hermes.zip"
	store := &memoryStore{found: true, value: []byte(`{"artifactPreference":"online-first","hermesArchiveUrl":"` + customURL + `","nodeArchiveUrl":"https://example.com/node.zip","npmArchiveUrl":"https://example.com/npm.tgz","pythonArchiveUrl":"https://example.com/python.tar.gz","pythonIndexUrl":"https://example.com/simple","npmRegistryUrl":"https://example.com/npm"}`)}
	config, err := NewService(store).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if config.HermesArchiveURL != customURL {
		t.Fatalf("HermesArchiveURL = %q, want custom URL %q", config.HermesArchiveURL, customURL)
	}
}

func TestNormalizeRejectsUnsafeOrAmbiguousURLs(t *testing.T) {
	for _, value := range []string{
		"",
		"http://mirror.example/node.zip",
		"https://user:secret@mirror.example/node.zip",
		"https://mirror.example/node.zip?token=secret",
		"https://mirror.example/node.zip#fragment",
	} {
		config := Default()
		config.NodeArchiveURL = value
		if _, err := Normalize(config); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Normalize(%q) error = %v, want ErrInvalid", value, err)
		}
	}
}

func TestServicePersistsAndResetsOneNamespacedDocument(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	config := Default()
	config.PythonIndexURL = "https://mirror.example/pypi/simple"

	saved, err := service.Save(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if saved.PythonIndexURL != config.PythonIndexURL || store.key != settingKey || len(store.value) == 0 {
		t.Fatalf("saved=%#v key=%q value=%q", saved, store.key, store.value)
	}
	loaded, err := service.Get(context.Background())
	if err != nil || loaded != saved {
		t.Fatalf("loaded=%#v err=%v", loaded, err)
	}
	defaults, err := service.Reset(context.Background())
	if err != nil || defaults != Default() || store.found {
		t.Fatalf("defaults=%#v found=%v err=%v", defaults, store.found, err)
	}
}

func TestServiceFailsClosedOnCorruptStoredDocument(t *testing.T) {
	store := &memoryStore{found: true, value: []byte(`{"pythonIndexUrl":"https://example.com/simple","unknown":true}`)}
	_, err := NewService(store).Get(context.Background())
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("error = %v, want ErrCorrupt", err)
	}
}

type memoryStore struct {
	key   string
	value []byte
	found bool
}

func (s *memoryStore) GetAppSetting(context.Context, string) ([]byte, bool, error) {
	return append([]byte(nil), s.value...), s.found, nil
}

func (s *memoryStore) PutAppSetting(_ context.Context, key string, value []byte, _ time.Time) error {
	s.key = key
	s.value = append([]byte(nil), value...)
	s.found = true
	return nil
}

func (s *memoryStore) DeleteAppSetting(_ context.Context, key string) error {
	s.key = key
	s.value = nil
	s.found = false
	return nil
}
