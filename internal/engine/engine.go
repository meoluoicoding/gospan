package engine

import (
	"sync"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/input"
)

// Config parameterises an Engine. It is derived from the configuration's
// engine section.
type Config struct {
	CaseSensitive bool
	WordBoundary  bool
	EscapeChar    string
	BufferSize    int
}

// Result describes a completed expansion. Typed is the exact text that must be
// removed before the cursor (the trigger plus any trailing runes that were
// consumed while waiting for a longer trigger). Trailer holds any runes that
// followed the trigger and must be re-emitted after the replacement.
// Replacement is the concrete replacement text with regex captures substituted
// (identical to Match.Replacement for exact triggers).
type Result struct {
	Match       Match
	Typed       string
	Trailer     string
	Replacement string
}

// pending tracks a deferred trigger that is a prefix of a longer trigger.
type pending struct {
	match      Match
	triggerLen int
	start      int
}

// Engine consumes typed runes and reports completed triggers. It owns a bounded
// input buffer and an immutable trigger index. Reload swaps the index
// atomically; concurrent readers always observe a complete index.
type Engine struct {
	mu      sync.RWMutex
	idx     *Index
	buf     *input.Buffer
	cfg     Config
	escape  rune
	pending *pending
}

// New builds an Engine from the given configuration and match rules.
func New(cfg Config, matches []config.MatchConfig) (*Engine, error) {
	idx, err := BuildIndex(matches, cfg.CaseSensitive, cfg.WordBoundary)
	if err != nil {
		return nil, err
	}
	bufSize := cfg.BufferSize
	if bufSize <= 0 {
		bufSize = 256
	}
	e := &Engine{
		idx: idx,
		buf: input.NewBuffer(bufSize),
		cfg: cfg,
	}
	if cfg.EscapeChar != "" {
		e.escape = []rune(cfg.EscapeChar)[0]
	}
	return e, nil
}

// Config returns the engine's immutable configuration.
func (e *Engine) Config() Config {
	return e.cfg
}

// TriggerCount returns the number of active triggers.
func (e *Engine) TriggerCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.idx.Len()
}

// AllowedByApps reports whether a rule is active for the given process name.
// Exported for use by the CLI and GUI.
func AllowedByApps(processName string, apps config.MatchApps) bool {
	return allowedByApps(processName, apps)
}

func (e *Engine) Feed(r rune) (Result, bool) {
	return e.FeedWithContext(r, MatchContext{})
}

// FeedWithContext is like Feed but carries a MatchContext (e.g. foreground
// process name) used for app-specific filtering.
func (e *Engine) FeedWithContext(r rune, ctx MatchContext) (Result, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.buf.Push(r)
	return e.resolveLocked(ctx)
}

func (e *Engine) resolveLocked(ctx MatchContext) (Result, bool) {
	if e.idx.Len() == 0 {
		return Result{}, false
	}

	buf := e.buf.Runes()
	ent, start, ok := e.idx.matchAt(buf, ctx)

	if ok {
		// Escaped trigger: suppress expansion and drop the escape rune.
		if e.escape != 0 && start > 0 && buf[start-1] == e.escape {
			e.buf.RemoveAt(start - 1)
			e.pending = nil
			return Result{}, false
		}

		if ent.defer_ {
			// Wait for a possible longer trigger.
			e.pending = &pending{
				match:      ent.match,
				triggerLen: len([]rune(ent.match.Trigger)),
				start:      start,
			}
			return Result{}, false
		}

		e.pending = nil
		m := ent.match
		res := Result{Match: m, Typed: string(buf[start:]), Replacement: m.Replacement}
		e.buf.Reset()
		return res, true
	}

	// No exact match: try regex triggers.
	if rule, matched, relLoc, rok := e.idx.matchRegex(buf, ctx); rok {
		start := len(buf) - len([]rune(matched))
		// Escaped trigger: suppress expansion and drop the escape rune.
		if e.escape != 0 && start > 0 && buf[start-1] == e.escape {
			e.buf.RemoveAt(start - 1)
			e.pending = nil
			return Result{}, false
		}

		e.pending = nil
		m := rule.match
		m.IsRegex = true
		m.Regex = rule.pattern
		m.MatchedText = matched
		m.MatchIndices = relLoc
		concrete := m.Replacement
		if len(relLoc) >= 2 {
			concrete = string(rule.pattern.ExpandString(nil, m.Replacement, matched, relLoc))
		}
		res := Result{Match: m, Typed: matched, Replacement: concrete}
		e.buf.Reset()
		return res, true
	}

	// No match. If a deferred trigger is pending, the new input did not extend
	// it, so flush the deferred expansion now.
	if e.pending != nil {
		p := e.pending
		e.pending = nil
		end := p.start + p.triggerLen
		if end > len(buf) {
			e.buf.Reset()
			return Result{}, false
		}
		res := Result{
			Match:       p.match,
			Typed:       string(buf[p.start:]),
			Trailer:     string(buf[end:]),
			Replacement: p.match.Replacement,
		}
		e.buf.Reset()
		return res, true
	}

	return Result{}, false
}

// ExpandRegexReplacement substitutes capture groups ($1, $2, ...) in m's
// Replacement using m's MatchedText and MatchIndices. For non-regex matches it
// returns Replacement unchanged.
func ExpandRegexReplacement(m Match) string {
	if m.Regex == nil || m.MatchedText == "" || len(m.MatchIndices) < 2 {
		return m.Replacement
	}
	return string(m.Regex.ExpandString(nil, m.Replacement, m.MatchedText, m.MatchIndices))
}
func (e *Engine) FeedString(s string) (last Result, count int) {
	for _, r := range s {
		if res, ok := e.Feed(r); ok {
			last = res
			count++
		}
	}
	return last, count
}

// MatchInput reports the longest trigger at the end of s without disturbing the
// engine's buffer state, ignoring prefix deferral. It backs diagnostics such as
// `match test`.
func (e *Engine) MatchInput(s string) (Match, bool) {
	return e.MatchInputWithContext(s, MatchContext{})
}

// MatchInputWithContext is like MatchInput but carries a MatchContext.
func (e *Engine) MatchInputWithContext(s string, ctx MatchContext) (Match, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	buf := []rune(s)
	if ent, _, ok := e.idx.matchAt(buf, ctx); ok {
		return ent.match, true
	}
	if rule, matched, relLoc, ok := e.idx.matchRegex(buf, ctx); ok {
		m := rule.match
		m.IsRegex = true
		m.Regex = rule.pattern
		m.MatchedText = matched
		m.MatchIndices = relLoc
		return m, true
	}
	return Match{}, false
}

// Reset clears the input buffer and any pending deferred match.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.buf.Reset()
	e.pending = nil
}

// Reload atomically replaces the trigger index. A failure leaves the previous
// index in place, so a bad configuration never takes the engine down.
func (e *Engine) Reload(matches []config.MatchConfig) error {
	idx, err := BuildIndex(matches, e.cfg.CaseSensitive, e.cfg.WordBoundary)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.idx = idx
	e.buf.Reset()
	e.pending = nil
	e.mu.Unlock()
	return nil
}
