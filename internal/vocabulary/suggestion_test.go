package vocabulary

import (
	"testing"
	"time"

	"github.com/snzpet/gospan/internal/config"
)

func TestCollectorDisabledByDefault(t *testing.T) {
	c := NewCollector(config.Defaults().Suggestions)
	for i := 0; i < 10; i++ {
		c.Observe("hello world")
	}
	if got := len(c.Candidates()); got != 0 {
		t.Fatalf("candidates = %d, want 0 while learning is disabled", got)
	}
}

func TestCollectorEmitsEligibleCandidate(t *testing.T) {
	cfg := config.Defaults().Suggestions
	cfg.LearnFromTyping = true
	cfg.MinimumOccurrences = 3

	c := NewCollector(cfg)
	for i := 0; i < 3; i++ {
		c.Observe("kiểm tra")
	}

	cands := c.Candidates()
	if len(cands) != 1 {
		t.Fatalf("candidates = %d, want 1", len(cands))
	}
	if cands[0].Phrase != "kiểm tra" || cands[0].Count != 3 {
		t.Errorf("candidate = %+v", cands[0])
	}
}

func TestCollectorReset(t *testing.T) {
	cfg := config.Defaults().Suggestions
	cfg.LearnFromTyping = true
	c := NewCollector(cfg)
	c.Observe("x")
	c.Reset()
	if len(c.Candidates()) != 0 {
		t.Error("Reset did not clear candidates")
	}
}

func TestCandidateEligibility(t *testing.T) {
	cfg := config.Defaults().Suggestions
	cfg.MinimumOccurrences = 5
	cfg.MinimumPhraseLength = 2

	if (Candidate{Phrase: "ab", Count: 5}).Eligible(cfg) != true {
		t.Error("threshold-meeting candidate should be eligible")
	}
	if (Candidate{Phrase: "ab", Count: 4}).Eligible(cfg) {
		t.Error("below-threshold candidate should not be eligible")
	}
	if (Candidate{Phrase: "a", Count: 9}).Eligible(cfg) {
		t.Error("too-short candidate should not be eligible")
	}
	if (Candidate{Phrase: "ab", Count: 9, Accepted: true}).Eligible(cfg) {
		t.Error("accepted candidate should not be offered again")
	}
	_ = time.Now
}
