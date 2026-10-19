package engine

import (
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

func regexMatch(trigger, replace string) config.MatchConfig {
	return config.MatchConfig{Trigger: trigger, TriggerType: "regex", Replace: replace}
}

func TestRegexTriggerCaptureGroup(t *testing.T) {
	// Require at least two digits so the pattern does not fire on ";d4".
	e := newTestEngine(t, Config{}, regexMatch(`^;d(\d{2,})$`, "Section $1"))
	res, count := e.FeedString(";d42")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Replacement != "Section 42" {
		t.Errorf("Replacement = %q, want %q", res.Replacement, "Section 42")
	}
	if res.Typed != ";d42" {
		t.Errorf("Typed = %q, want ;d42", res.Typed)
	}
}

func TestRegexTriggerSingleDigit(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`^;d(\d)$`, "Section $1"))
	res, count := e.FeedString(";d4")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Replacement != "Section 4" {
		t.Errorf("Replacement = %q, want %q", res.Replacement, "Section 4")
	}
}

func TestRegexTriggerMultipleGroups(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`^;(\w+):(\d{2,})$`, "$1 = $2"))
	res, count := e.FeedString(";foo:42")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Replacement != "foo = 42" {
		t.Errorf("Replacement = %q, want %q", res.Replacement, "foo = 42")
	}
}

func TestRegexTriggerNoMatch(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`^;d(\d{2,})$`, "Section $1"))
	if _, count := e.FeedString(";d"); count != 0 {
		t.Error("partial should not match")
	}
	if _, count := e.FeedString(";abc"); count != 0 {
		t.Error("non-matching should not expand")
	}
	if _, count := e.FeedString(";d4"); count != 0 {
		t.Error("single digit should not match a 2+-digit pattern")
	}
}

func TestRegexTriggerSuffixOnly(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`d(\d+)`, "D$1"))
	res, count := e.FeedString("hello d7")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Typed != "d7" {
		t.Errorf("Typed = %q, want d7", res.Typed)
	}
	if res.Replacement != "D7" {
		t.Errorf("Replacement = %q, want D7", res.Replacement)
	}
}

func TestRegexTriggerWordBoundary(t *testing.T) {
	e := newTestEngine(t, Config{WordBoundary: true}, regexMatch(`d(\d+)`, "D$1"))
	if _, count := e.FeedString("xd42"); count != 0 {
		t.Error("regex trigger glued to a word should not expand")
	}
	e2 := newTestEngine(t, Config{WordBoundary: true}, regexMatch(`d(\d+)`, "D$1"))
	if _, count := e2.FeedString(" d42"); count != 1 {
		t.Error("regex trigger after a boundary should expand")
	}
}

func TestRegexTriggerEscape(t *testing.T) {
	e := newTestEngine(t, Config{EscapeChar: `\`}, regexMatch(`^;d(\d{2,})$`, "Section $1"))
	if _, count := e.FeedString(`\;d42`); count != 0 {
		t.Fatal("escaped regex trigger should not expand")
	}
}

func TestRegexTriggerMatchInput(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`^;d(\d{2,})$`, "Section $1"))
	m, ok := e.MatchInput(";d99")
	if !ok {
		t.Fatal("MatchInput should match")
	}
	if m.MatchedText != ";d99" {
		t.Errorf("MatchedText = %q, want ;d99", m.MatchedText)
	}
	got := ExpandRegexReplacement(m)
	if got != "Section 99" {
		t.Errorf("ExpandRegexReplacement = %q, want Section 99", got)
	}
}

func TestRegexTriggerCoexistsWithExact(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";dc", "được"), regexMatch(`^;d(\d{2,})$`, "Section $1"))
	res, count := e.FeedString(";dc")
	if count != 1 || res.Match.Replacement != "được" {
		t.Fatalf(";dc should expand to được, got (%q, count=%d)", res.Match.Replacement, count)
	}
	res, count = e.FeedString(";d42")
	if count != 1 || res.Replacement != "Section 42" {
		t.Fatalf(";d42 should expand to Section 42, got (%q, count=%d)", res.Replacement, count)
	}
}

func TestRegexTriggerCaptureGroupZero(t *testing.T) {
	e := newTestEngine(t, Config{}, regexMatch(`^;(\w{5,})$`, "matched: $0"))
	res, count := e.FeedString(";hello")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Replacement != "matched: ;hello" {
		t.Errorf("Replacement = %q, want %q", res.Replacement, "matched: ;hello")
	}
}

func TestRegexTriggerGreedyFiresEarly(t *testing.T) {
	// Document the greedy behaviour: a pattern that matches a prefix fires as
	// soon as it can, even if more digits follow. The leftover is typed normally.
	e := newTestEngine(t, Config{}, regexMatch(`^;d(\d+)$`, "Section $1"))
	res, count := e.FeedString(";d42")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Typed != ";d4" {
		t.Errorf("Typed = %q, want ;d4 (greedy)", res.Typed)
	}
	if res.Replacement != "Section 4" {
		t.Errorf("Replacement = %q, want Section 4 (greedy)", res.Replacement)
	}
}
