// Package integration exercises the application service end to end without a
// running GUI or Windows hook, which is the Phase 1 exit criterion.
package integration

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/snzpet/gospan/internal/app"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestServiceMatchesAndRenders(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	write(t, cfgPath, `matches:
  - trigger: ";kt"
    replace: "kiểm tra"
  - trigger: ";date"
    replace: "{{date:2006-01-02}}"
`)

	svc, err := app.Open(cfgPath, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if svc.TriggerCount() != 2 {
		t.Fatalf("triggers = %d, want 2", svc.TriggerCount())
	}

	m, rendered, ok, err := svc.TestMatch(";kt", "")
	if err != nil || !ok {
		t.Fatalf("TestMatch: ok=%v err=%v", ok, err)
	}
	if m.Trigger != ";kt" || rendered != "kiểm tra" {
		t.Errorf("got %q → %q", m.Trigger, rendered)
	}

	if _, _, ok, _ := svc.TestMatch("nothing", ""); ok {
		t.Error("unexpected match for non-trigger input")
	}
}

func TestServiceReloadIsAtomic(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	write(t, cfgPath, `matches:
  - trigger: ";a"
    replace: "one"
`)

	svc, err := app.Open(cfgPath, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// A broken reload must keep the previous configuration active.
	write(t, cfgPath, `matches:
  - trigger: ""
    replace: "bad"
`)
	if err := svc.Reload(); err == nil {
		t.Fatal("reload of invalid config should fail")
	}
	if _, _, ok, _ := svc.TestMatch(";a", ""); !ok {
		t.Error("previous configuration should remain active after a failed reload")
	}

	// A valid reload swaps the dictionary.
	write(t, cfgPath, `matches:
  - trigger: ";b"
    replace: "two"
`)
	if err := svc.Reload(); err != nil {
		t.Fatalf("valid reload failed: %v", err)
	}
	if _, _, ok, _ := svc.TestMatch(";a", ""); ok {
		t.Error("old trigger still active after reload")
	}
	if _, rendered, ok, _ := svc.TestMatch(";b", ""); !ok || rendered != "two" {
		t.Errorf("new trigger not active after reload: ok=%v rendered=%q", ok, rendered)
	}
}
