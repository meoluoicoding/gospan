//go:build windows

package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"io"
	"log/slog"
)

func TestAutoReloadOnChange(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("matches:\n  - trigger: \";a\"\n    replace: \"one\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := Open(cfgPath, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := svc.Start(); err != nil {
		t.Skipf("keyboard capture unavailable: %v", err)
	}
	defer svc.Stop()

	// Wait until capture is active.
	deadline := time.Now().Add(2 * time.Second)
	for !svc.Running() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !svc.Running() {
		t.Fatal("service never started")
	}

	// Change the file: a new trigger should appear without a manual reload.
	if err := os.WriteFile(cfgPath, []byte("matches:\n  - trigger: \";b\"\n    replace: \"two\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, ok, _ := svc.TestMatch(";b", ""); ok {
			if _, _, ok2, _ := svc.TestMatch(";a", ""); !ok2 {
				return // success: ;b active, ;a gone
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("config was not auto-reloaded within 4s (triggers=%d)", svc.TriggerCount())
}
