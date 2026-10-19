package engine

import (
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

func rule(trigger, replace string) config.MatchConfig {
	return config.MatchConfig{Trigger: trigger, Replace: replace}
}

func newTestEngine(t *testing.T, cfg Config, matches ...config.MatchConfig) *Engine {
	t.Helper()
	e, err := New(cfg, matches)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func TestFeedExpandsBasicTrigger(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";dc", "được"))
	res, count := e.FeedString(";dc")
	if count != 1 {
		t.Fatalf("count = %d, want 1 (%+v)", count, res)
	}
	if res.Match.Replacement != "được" {
		t.Errorf("replacement = %q, want được", res.Match.Replacement)
	}
	if res.Typed != ";dc" {
		t.Errorf("Typed = %q, want ;dc", res.Typed)
	}
}

func TestPartialTriggerDoesNotExpand(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";dc", "được"))
	if _, count := e.FeedString(";d"); count != 0 {
		t.Fatal("partial trigger should not expand")
	}
}

func TestPrefixConflictPrefersLongerTrigger(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";k", "không"), rule(";kt", "kiểm tra"))

	res, count := e.FeedString(";kt")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Trigger != ";kt" {
		t.Fatalf("expanded %q, want ;kt", res.Match.Trigger)
	}
	if res.Typed != ";kt" {
		t.Errorf("Typed = %q, want ;kt", res.Typed)
	}
}

func TestDeferredPrefixFlushesOnTerminator(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";k", "không"), rule(";kt", "kiểm tra"))

	res, count := e.FeedString(";k ")
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if res.Match.Trigger != ";k" {
		t.Fatalf("expanded %q, want ;k", res.Match.Trigger)
	}
	if res.Typed != ";k " {
		t.Errorf("Typed = %q, want %q", res.Typed, ";k ")
	}
	if res.Trailer != " " {
		t.Errorf("Trailer = %q, want a single space", res.Trailer)
	}
}

func TestCaseInsensitiveMatching(t *testing.T) {
	e := newTestEngine(t, Config{CaseSensitive: false}, rule(";DC", "được"))
	if _, count := e.FeedString(";dc"); count != 1 {
		t.Fatal("case-insensitive engine should match ;dc against ;DC")
	}

	sensitive := newTestEngine(t, Config{CaseSensitive: true}, rule(";DC", "được"))
	if _, count := sensitive.FeedString(";dc"); count != 0 {
		t.Fatal("case-sensitive engine should not match ;dc against ;DC")
	}
}

func TestWordBoundary(t *testing.T) {
	e := newTestEngine(t, Config{WordBoundary: true}, rule("aaa", "expanded"))
	if _, count := e.FeedString("xaaa"); count != 0 {
		t.Error("trigger inside a word should not expand")
	}

	e2 := newTestEngine(t, Config{WordBoundary: true}, rule("aaa", "expanded"))
	res, count := e2.FeedString(" aaa")
	if count != 1 || res.Typed != "aaa" {
		t.Fatalf("trigger after a boundary should expand, got (%+v, count=%d)", res, count)
	}
}

func TestSymbolTriggerNeedsNoBoundary(t *testing.T) {
	e := newTestEngine(t, Config{WordBoundary: true}, rule(";dc", "được"))
	if _, count := e.FeedString("abc;dc"); count != 1 {
		t.Error("symbol trigger after word characters should still expand")
	}
}

func TestEscapeSuppressesExpansion(t *testing.T) {
	e := newTestEngine(t, Config{EscapeChar: `\`}, rule(";dc", "được"))
	if _, count := e.FeedString(`\;dc`); count != 0 {
		t.Fatal("escaped trigger should not expand")
	}
}

func TestDisabledRuleIsIgnored(t *testing.T) {
	disabled := false
	m := config.MatchConfig{Trigger: ";off", Replace: "x", Enabled: &disabled}
	e := newTestEngine(t, Config{}, m)
	if _, count := e.FeedString(";off"); count != 0 {
		t.Fatal("disabled rule should not expand")
	}
}

func TestReloadSwapsTriggers(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";a", "one"))
	if err := e.Reload([]config.MatchConfig{rule(";b", "two")}); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if _, count := e.FeedString(";a"); count != 0 {
		t.Error("old trigger should not match after reload")
	}
	res, count := e.FeedString(";b")
	if count != 1 || res.Match.Replacement != "two" {
		t.Fatalf("new trigger should match after reload, got (%+v, count=%d)", res, count)
	}
}

func TestUnicodeTrigger(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";ưu", "ưu tiên"))
	res, count := e.FeedString(";ưu")
	if count != 1 || res.Match.Replacement != "ưu tiên" {
		t.Fatalf("unicode trigger failed: (%+v, count=%d)", res, count)
	}
}

func TestMatchInputIgnoresDeferral(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";k", "không"), rule(";kt", "kiểm tra"))
	m, ok := e.MatchInput(";kt")
	if !ok || m.Trigger != ";kt" {
		t.Fatalf("MatchInput = (%+v, %v), want ;kt", m, ok)
	}
}
