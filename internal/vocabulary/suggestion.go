package vocabulary

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/snzpet/gospan/internal/config"
)

// Collector accumulates phrase occurrence counts from typing. It is inert
// unless learning is explicitly enabled in configuration, and it never stores
// full keystroke histories or complete sentences.
type Collector struct {
	cfg   config.SuggestionsConfig
	mu    sync.Mutex
	seen  map[string]*Candidate
	clock func() time.Time
}

// NewCollector returns a Collector for the given suggestion settings.
func NewCollector(cfg config.SuggestionsConfig) *Collector {
	return &Collector{
		cfg:   cfg,
		seen:  make(map[string]*Candidate),
		clock: time.Now,
	}
}

// enabled reports whether learning is active.
func (c *Collector) enabled() bool {
	return c.cfg.Enabled && c.cfg.LearnFromTyping
}

// Observe records a phrase occurrence. It is a no-op when learning is disabled.
func (c *Collector) Observe(phrase string) {
	if !c.enabled() || phrase == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	cand, ok := c.seen[phrase]
	if !ok {
		if len(c.seen) >= c.cfg.MaximumCandidates*100 {
			return // bound memory growth
		}
		cand = &Candidate{Phrase: phrase}
		c.seen[phrase] = cand
	}
	cand.Count++
	cand.LastSeen = c.clock()
}

// Candidates returns the eligible candidates ordered by count, descending.
func (c *Collector) Candidates() []Candidate {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out []Candidate
	for _, cand := range c.seen {
		if cand.Eligible(c.cfg) {
			out = append(out, *cand)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Phrase < out[j].Phrase
	})
	if len(out) > c.cfg.MaximumCandidates {
		out = out[:c.cfg.MaximumCandidates]
	}
	return out
}

// Reset discards all collected statistics. It backs the "disable learning and
// delete learned statistics" privacy control.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = make(map[string]*Candidate)
}

// String implements fmt.Stringer for diagnostics.
func (c *Collector) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("collector{phrases:%d enabled:%t}", len(c.seen), c.enabled())
}
