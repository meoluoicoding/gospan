package engine

import (
	"fmt"
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

// buildSyntheticIndex creates n triggers of the form ";k0001" so benchmarks
// exercise the map plus longest-suffix search at realistic scale.
func buildSyntheticIndex(b *testing.B, n int) *Index {
	b.Helper()
	matches := make([]config.MatchConfig, n)
	for i := range matches {
		matches[i] = config.MatchConfig{
			Trigger: fmt.Sprintf(";k%04d", i),
			Replace: "không",
		}
	}
	idx, err := BuildIndex(matches, false, true)
	if err != nil {
		b.Fatalf("BuildIndex: %v", err)
	}
	return idx
}

func BenchmarkBuildIndex10000(b *testing.B) {
	matches := make([]config.MatchConfig, 10000)
	for i := range matches {
		matches[i] = config.MatchConfig{Trigger: fmt.Sprintf(";k%04d", i), Replace: "x"}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := BuildIndex(matches, false, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatchSuffix(b *testing.B) {
	idx := buildSyntheticIndex(b, 10000)
	buf := []rune("hello world ;k9000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := idx.matchAt(buf, MatchContext{}); !ok {
			b.Fatal("expected a match")
		}
	}
}

func BenchmarkFeedString(b *testing.B) {
	matches := []config.MatchConfig{
		{Trigger: ";dc", Replace: "được"},
		{Trigger: ";k", Replace: "không"},
		{Trigger: ";kt", Replace: "kiểm tra"},
	}
	e, err := New(Config{}, matches)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.FeedString("some typed text ;dc")
	}
}
