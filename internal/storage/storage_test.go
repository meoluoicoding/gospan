package storage

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// runStoreSuite exercises a Store implementation with the same expectations, so
// every backend behaves identically.
func runStoreSuite(t *testing.T, open func(string) (Store, error), ext string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vocab"+ext)

	s, err := open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	entries := []VocabularyEntry{
		{Trigger: ";kt", Expansion: "kiểm tra", Language: "vi", Source: "test", CreatedAt: now},
		{Trigger: ";brse", Expansion: "Bridge Software Engineer", Language: "en", Source: "test", CreatedAt: now.Add(time.Second)},
	}
	for _, e := range entries {
		if err := s.Add(e); err != nil {
			t.Fatalf("Add(%q): %v", e.Trigger, err)
		}
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Trigger != ";kt" || got[0].Expansion != "kiểm tra" {
		t.Errorf("first entry = %+v", got[0])
	}
	if got[1].Trigger != ";brse" {
		t.Errorf("second entry = %+v", got[1])
	}
	if !got[0].CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got[0].CreatedAt, now)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopening must preserve the data.
	s2, err := open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	got2, err := s2.List()
	if err != nil {
		t.Fatalf("List after reopen: %v", err)
	}
	if len(got2) != 2 {
		t.Fatalf("after reopen len = %d, want 2", len(got2))
	}
}

func TestJSONLStore(t *testing.T)  { runStoreSuite(t, OpenJSONL, ".jsonl") }
func TestSQLiteStore(t *testing.T) { runStoreSuite(t, OpenSQLite, ".db") }

func TestOpenSelectsBackend(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		backend string
		ext     string
	}{
		{"jsonl", ".jsonl"},
		{"sqlite", ".db"},
		{"", ".jsonl"},
		{" SQLite ", ".db"},
	}
	for i, tc := range cases {
		path := filepath.Join(dir, fmt.Sprintf("v%d%s", i, tc.ext))
		s, err := Open(tc.backend, path)
		if err != nil {
			t.Fatalf("Open(%q): %v", tc.backend, err)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("Close(%q): %v", tc.backend, err)
		}
	}

	if _, err := Open("bogus", filepath.Join(dir, "x")); err == nil {
		t.Error("expected an error for an unknown backend")
	}
}
