package vocabulary

import (
	"time"

	"github.com/snzpet/gospan/internal/config"
)

// Candidate is a phrase observed often enough to be offered as a new
// dictionary entry. A candidate is never activated automatically; the user must
// accept it explicitly.
type Candidate struct {
	Phrase    string
	Count     int
	LastSeen  time.Time
	Accepted  bool
	Dismissed bool
}

// Eligible reports whether the candidate meets the configured thresholds.
func (c Candidate) Eligible(cfg config.SuggestionsConfig) bool {
	if c.Accepted || c.Dismissed {
		return false
	}
	if c.Count < cfg.MinimumOccurrences {
		return false
	}
	if len([]rune(c.Phrase)) < cfg.MinimumPhraseLength {
		return false
	}
	return true
}
