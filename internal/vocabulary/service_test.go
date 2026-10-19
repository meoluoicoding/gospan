package vocabulary

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/storage"
)

func newVocabService(t *testing.T, cfg config.SuggestionsConfig) (*Service, func()) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.OpenJSONL(filepath.Join(dir, "vocab.jsonl"))
	if err != nil {
		t.Fatalf("OpenJSONL: %v", err)
	}
	vs := NewService(store, cfg)
	return vs, func() { _ = vs.Close() }
}

func TestServiceAddAndList(t *testing.T) {
	cfg := config.Defaults().Suggestions
	vs, cleanup := newVocabService(t, cfg)
	defer cleanup()

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
	if entries[0].Trigger != ";kt" || entries[0].Expansion != "kiểm tra" {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[0].Source != "user" {
		t.Errorf("Source = %q, want user", entries[0].Source)
	}
	if entries[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestServiceAddRejectsEmpty(t *testing.T) {
	cfg := config.Defaults().Suggestions
	vs, cleanup := newVocabService(t, cfg)
	defer cleanup()

	if err := vs.Add("", "x", ""); err == nil {
		t.Error("empty trigger should fail")
	}
	if err := vs.Add(";x", "", ""); err == nil {
		t.Error("empty expansion should fail")
	}
	if err := vs.Add("  ", "x", ""); err == nil {
		t.Error("whitespace-only trigger should fail")
	}
}

func TestServiceExport(t *testing.T) {
	cfg := config.Defaults().Suggestions
	vs, cleanup := newVocabService(t, cfg)
	defer cleanup()

	if err := vs.Add(";kt", "kiểm tra", "vi"); err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := vs.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.Contains(buf.String(), ";kt") || !strings.Contains(buf.String(), "kiểm tra") {
		t.Errorf("export = %q", buf.String())
	}
}

func TestServiceLearningEnabled(t *testing.T) {
	cfg := config.Defaults().Suggestions
	cfg.Enabled = false
	cfg.LearnFromTyping = true
	vs, cleanup := newVocabService(t, cfg)
	defer cleanup()
	if vs.LearningEnabled() {
		t.Error("learning should be off when suggestions are disabled")
	}

	cfg2 := config.Defaults().Suggestions
	cfg2.Enabled = true
	cfg2.LearnFromTyping = true
	vs2, cleanup2 := newVocabService(t, cfg2)
	defer cleanup2()
	if !vs2.LearningEnabled() {
		t.Error("learning should be on when both flags are true")
	}
}

func TestServiceClose(t *testing.T) {
	cfg := config.Defaults().Suggestions
	vs, _ := newVocabService(t, cfg)
	if err := vs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// A second close should not panic.
	_ = vs.Close()
}

func TestServicePersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.jsonl")
	cfg := config.Defaults().Suggestions

	s1, err := storage.OpenJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	vs1 := NewService(s1, cfg)
	if err := vs1.Add(";x", "example", ""); err != nil {
		t.Fatal(err)
	}
	vs1.Close()

	s2, err := storage.OpenJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	vs2 := NewService(s2, cfg)
	entries, err := vs2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Trigger != ";x" {
		t.Fatalf("entries = %+v", entries)
	}
	_ = io.Discard
	_ = slog.Default
}
