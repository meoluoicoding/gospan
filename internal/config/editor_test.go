package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestEditorAddMatchPreservesSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	orig := "app:\n  name: gospan\n  log_level: info\n\nengine:\n  case_sensitive: false\n\n# my rules\nmatches:\n  - trigger: \";a\"\n    replace: \"one\"\n"
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEditor(path, path)
	if err := e.AddMatch(";b", "two", "a test rule"); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(cfg.Matches))
	}
	if cfg.Matches[1].Trigger != ";b" || cfg.Matches[1].Replace != "two" {
		t.Errorf("new match = %+v", cfg.Matches[1])
	}
	if cfg.Matches[1].Description != "a test rule" {
		t.Errorf("description = %q", cfg.Matches[1].Description)
	}
	if cfg.App.Name != "gospan" || cfg.App.LogLevel != "info" {
		t.Error("unrelated app section was not preserved")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Error("expected a .bak backup")
	}
}

func TestEditorAddMultilineReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("matches: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEditor(path, path)
	if err := e.AddMatch(";sig", "Trân trọng,\nKhoa\n", ""); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.Matches) != 1 || cfg.Matches[0].Replace != "Trân trọng,\nKhoa\n" {
		t.Fatalf("multiline replacement = %q", cfg.Matches[0].Replace)
	}
}

func TestEditorRejectsDuplicateAndKeepsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("matches:\n  - trigger: \";a\"\n    replace: \"one\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)

	err := NewEditor(path, path).AddMatch(";a", "duplicate", "")
	if err == nil {
		t.Fatal("expected duplicate-trigger error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error = %v, want duplicate", err)
	}
	if readFile(t, path) != before {
		t.Error("file changed despite rejected add")
	}
}

func TestEditorRemoveMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	doc := "matches:\n  - trigger: \";a\"\n    replace: \"one\"\n  - trigger: \";b\"\n    replace: \"two\"\n"
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := NewEditor(path, path).RemoveMatch(";a", false)
	if err != nil {
		t.Fatalf("RemoveMatch: %v", err)
	}
	if !removed {
		t.Fatal("expected ;a to be removed")
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.Matches) != 1 || cfg.Matches[0].Trigger != ";b" {
		t.Fatalf("remaining matches = %+v", cfg.Matches)
	}
}

func TestEditorRemoveMissingReportsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("matches:\n  - trigger: \";a\"\n    replace: \"one\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := NewEditor(path, path).RemoveMatch(";zzz", false)
	if err != nil {
		t.Fatalf("RemoveMatch: %v", err)
	}
	if removed {
		t.Fatal("nothing should have been removed")
	}
}

func TestEditorCrossFileErrorHidesTempPath(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(extra, []byte("matches:\n  - trigger: \";dc\"\n    replace: \"a\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("include:\n  - extra.yaml\nmatches:\n  - trigger: \";dc\"\n    replace: \"b\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := NewEditor(path, path).AddMatch(";a", "x", "")
	if err == nil {
		t.Fatal("expected a cross-file duplicate error")
	}
	if strings.Contains(err.Error(), ".gospan-") {
		t.Errorf("error leaks the temp path: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should mention the real path %s: %v", path, err)
	}
	var verrs ValidationErrors
	if !errors.As(err, &verrs) {
		t.Errorf("error type = %T, want ValidationErrors", err)
	}
}

func TestEditorRoundTripsAwkwardTriggers(t *testing.T) {
	awkward := []string{
		";kt", "#hash", "-dash", ":colon", "*star", "!bang", "&amp",
		"123", "true", "null", "yes", "a: b", "@at", "`tick", "%percent",
	}
	for _, trigger := range awkward {
		t.Run(trigger, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte("matches: []\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := NewEditor(path, path).AddMatch(trigger, "value for "+trigger, ""); err != nil {
				t.Fatalf("AddMatch(%q): %v", trigger, err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			if len(cfg.Matches) != 1 {
				t.Fatalf("matches = %d, want 1", len(cfg.Matches))
			}
			if cfg.Matches[0].Trigger != trigger {
				t.Errorf("trigger = %q, want %q", cfg.Matches[0].Trigger, trigger)
			}
			if cfg.Matches[0].Replace != "value for "+trigger {
				t.Errorf("replace = %q", cfg.Matches[0].Replace)
			}
		})
	}
}

func TestEditorCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	e := NewEditor(path, path)
	if err := e.AddMatch(";x", "example", ""); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.Matches) != 1 || cfg.Matches[0].Trigger != ";x" {
		t.Fatalf("matches = %+v", cfg.Matches)
	}
}

func TestEditorRejectsDuplicateAcrossIncludes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extra.yaml"), []byte("matches:\n  - trigger: \";dc\"\n    replace: \"được\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("include:\n  - extra.yaml\nmatches:\n  - trigger: \";a\"\n    replace: \"one\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, path)

	err := NewEditor(path, path).AddMatch(";dc", "conflict", "")
	if err == nil {
		t.Fatal("expected a cross-file duplicate error")
	}
	if readFile(t, path) != before {
		t.Error("file changed despite rejected add")
	}
}
