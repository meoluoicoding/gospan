package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "w.yaml")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewWatcher(path, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(path, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.Changes():
		// detected
	case <-time.After(1 * time.Second):
		t.Fatal("watcher did not detect the file change")
	}
}

func TestWatcherIgnoresUnchangedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "w.yaml")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewWatcher(path, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)

	select {
	case <-w.Changes():
		t.Fatal("watcher signalled a change for an unmodified file")
	case <-time.After(300 * time.Millisecond):
		// expected: no signal
	}
}
