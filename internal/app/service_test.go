package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestOpenValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \";kt\"\n    replace: \"kiểm tra\"\n")

	svc, err := Open(path, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if svc.TriggerCount() != 1 {
		t.Errorf("TriggerCount = %d, want 1", svc.TriggerCount())
	}
	if svc.Config().App.Name != "gospan" {
		t.Errorf("default app name not applied: %q", svc.Config().App.Name)
	}
	if svc.Config().App.ConfigPath != path {
		t.Errorf("ConfigPath = %q, want %q", svc.Config().App.ConfigPath, path)
	}
}

func TestOpenInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \"\"\n    replace: \"x\"\n")

	if _, err := Open(path, discardLogger()); err == nil {
		t.Fatal("Open should fail for invalid config")
	}
}

func TestOpenMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.yaml")
	if _, err := Open(path, discardLogger()); err == nil {
		t.Fatal("Open should fail for missing config")
	}
}

func TestTestMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \";kt\"\n    replace: \"kiểm tra\"\n  - trigger: \";date\"\n    replace: \"{{date:2006-01-02}}\"\n")

	svc, err := Open(path, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	m, rendered, ok, err := svc.TestMatch(";kt", "")
	if err != nil || !ok {
		t.Fatalf("TestMatch: ok=%v err=%v", ok, err)
	}
	if m.Trigger != ";kt" || rendered != "kiểm tra" {
		t.Errorf("got %q → %q", m.Trigger, rendered)
	}

	// Template variables are rendered.
	_, rendered, ok, _ = svc.TestMatch(";date", "")
	if !ok || rendered != time.Now().Format("2006-01-02") {
		t.Errorf("date render = %q, want %s", rendered, time.Now().Format("2006-01-02"))
	}

	if _, _, ok, _ := svc.TestMatch("nothing", ""); ok {
		t.Error("unexpected match")
	}
}

func TestReloadSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \";a\"\n    replace: \"one\"\n")

	svc, err := Open(path, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	writeConfig(t, path, "matches:\n  - trigger: \";b\"\n    replace: \"two\"\n")
	if err := svc.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if svc.TriggerCount() != 1 {
		t.Errorf("TriggerCount = %d, want 1", svc.TriggerCount())
	}
	if _, _, ok, _ := svc.TestMatch(";a", ""); ok {
		t.Error("old trigger still active after reload")
	}
	if _, rendered, ok, _ := svc.TestMatch(";b", ""); !ok || rendered != "two" {
		t.Errorf("new trigger not active: ok=%v rendered=%q", ok, rendered)
	}
}

func TestReloadFailureKeepsState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \";a\"\n    replace: \"one\"\n")

	svc, err := Open(path, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	writeConfig(t, path, "matches:\n  - trigger: \"\"\n    replace: \"bad\"\n")
	if err := svc.Reload(); err == nil {
		t.Fatal("Reload should fail for invalid config")
	}
	if _, _, ok, _ := svc.TestMatch(";a", ""); !ok {
		t.Error("previous configuration should remain active after a failed reload")
	}
}
