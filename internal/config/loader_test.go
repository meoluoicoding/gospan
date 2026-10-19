package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadResolvesIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "vietnamese.yaml", "matches:\n  - trigger: \";dc\"\n    replace: \"được\"\n")
	writeFile(t, dir, "english.yaml", "matches:\n  - trigger: \";ty\"\n    replace: \"Thank you\"\n")

	base := writeFile(t, dir, "config.yaml", `app:
  name: gospan
include:
  - vietnamese.yaml
  - english.yaml
matches:
  - trigger: ";k"
    replace: "không"
`)

	cfg, err := Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Matches) != 3 {
		t.Fatalf("matches = %d, want 3", len(cfg.Matches))
	}
	if cfg.Matches[0].Trigger != ";k" {
		t.Errorf("inline match should come first, got %q", cfg.Matches[0].Trigger)
	}
	if cfg.Matches[1].Trigger != ";dc" {
		t.Errorf("included match order wrong: %q", cfg.Matches[1].Trigger)
	}
	if cfg.Matches[1].Source == "" {
		t.Error("included match should record its source file")
	}
}

func TestLoadDetectsIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	writeFile(t, dir, "b.yaml", "include:\n  - a.yaml\nmatches: []\n")
	writeFile(t, dir, "a.yaml", "include:\n  - b.yaml\nmatches: []\n")

	if _, err := Load(a); err == nil {
		t.Fatal("expected an include-cycle error")
	}
}

func TestLoadValidatesIncludeDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "extra.yaml", "matches:\n  - trigger: \";dc\"\n    replace: \"other\"\n")
	base := writeFile(t, dir, "config.yaml", `include:
  - extra.yaml
matches:
  - trigger: ";dc"
    replace: "được"
`)
	_, err := Load(base)
	if err == nil {
		t.Fatal("expected duplicate-trigger error across files")
	}
}
