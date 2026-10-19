//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// (uses discardLogger defined in service_test.go)
func TestStartStopLifecycle(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("matches:\n  - trigger: \";kt\"\n    replace: \"kiểm tra\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := Open(cfgPath, discardLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if svc.Running() {
		t.Fatal("service should not be running before Start")
	}

	if err := svc.Start(); err != nil {
		t.Skipf("keyboard capture unavailable in this environment: %v", err)
	}
	if !svc.Running() {
		t.Fatal("Running() should be true after Start")
	}

	if err := svc.Start(); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second Start = %v, want ErrAlreadyRunning", err)
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if svc.Running() {
		t.Fatal("Running() should be false after Stop")
	}

	// A stopped service can be started again.
	if err := svc.Start(); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if !svc.Running() {
		t.Fatal("Running() should be true after restart")
	}
	if err := svc.Stop(); err != nil {
		t.Fatalf("final Stop: %v", err)
	}
}
