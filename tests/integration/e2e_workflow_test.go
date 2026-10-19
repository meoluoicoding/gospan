// Package integration contains end-to-end tests for the Gospan service.
package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snzpet/gospan/internal/app"
	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/storage"
	"github.com/snzpet/gospan/internal/vocabulary"
)

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestE2E_ConfigIncludesAndReload exercises includes, reload, and duplicate
// detection across files.
func TestE2E_ConfigIncludesAndReload(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extra.yaml"),
		[]byte("matches:\n  - trigger: \";dc\"\n    replace: \"được\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	writeConfig(t, cfgPath, "include:\n  - extra.yaml\nmatches:\n  - trigger: \";a\"\n    replace: \"one\"\n")

	svc, err := app.Open(cfgPath, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if svc.TriggerCount() != 2 {
		t.Fatalf("triggers = %d, want 2", svc.TriggerCount())
	}

	// Reload with a valid change.
	writeConfig(t, cfgPath, "include:\n  - extra.yaml\nmatches:\n  - trigger: \";b\"\n    replace: \"two\"\n")
	if err := svc.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if _, _, ok, _ := svc.TestMatch(";b", ""); !ok {
		t.Error("new trigger not active after reload")
	}

	// A broken reload keeps the previous state.
	writeConfig(t, cfgPath, "matches:\n  - trigger: \"\"\n    replace: \"bad\"\n")
	if err := svc.Reload(); err == nil {
		t.Fatal("invalid reload should fail")
	}
	if _, _, ok, _ := svc.TestMatch(";b", ""); !ok {
		t.Error("previous config lost after failed reload")
	}
}

// TestE2E_StorageBackends exercise both backends through the vocabulary service.
func TestE2E_StorageBackends(t *testing.T) {
	for _, backend := range []string{"jsonl", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")
			writeConfig(t, cfgPath, "storage:\n  backend: "+backend+"\n  path: vocab\nmatches: []\n")

			svc, err := app.Open(cfgPath, discardLogger())
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			cfg := svc.Config()

			store, err := storage.Open(cfg.Storage.Backend, resolveStoragePath(cfg))
			if err != nil {
				t.Fatalf("storage.Open: %v", err)
			}
			vs := vocabulary.NewService(store, cfg.Suggestions)
			defer vs.Close()

			if err := vs.Add(";kt", "kiểm tra", "vi"); err != nil {
				t.Fatalf("Add: %v", err)
			}
			if err := vs.Add(";brse", "Bridge Software Engineer", "en"); err != nil {
				t.Fatalf("Add: %v", err)
			}

			entries, err := vs.List()
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 2 {
				t.Fatalf("entries = %d, want 2", len(entries))
			}

			// Reopen and verify persistence.
			store2, err := storage.Open(cfg.Storage.Backend, resolveStoragePath(cfg))
			if err != nil {
				t.Fatalf("reopen: %v", err)
			}
			vs2 := vocabulary.NewService(store2, cfg.Suggestions)
			entries2, err := vs2.List()
			vs2.Close()
			if err != nil {
				t.Fatalf("List after reopen: %v", err)
			}
			if len(entries2) != 2 {
				t.Fatalf("after reopen entries = %d, want 2", len(entries2))
			}
		})
	}
}

func resolveStoragePath(cfg *config.Config) string {
	p := cfg.Storage.Path
	if filepath.IsAbs(p) {
		return p
	}
	dir := filepath.Dir(cfg.App.ConfigPath)
	if dir == "" || dir == "." {
		return p
	}
	return filepath.Join(dir, p)
}
