package engine

import (
	"strings"
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

// TestMultiLanguageExpansion locks in Unicode handling for Japanese, Chinese,
// Korean, German, and emoji across the matching engine. These are regression
// guards: a change that breaks rune handling or case folding would fail here.
func TestJapaneseExpansion(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";jp", "こんにちは"), rule(";arigato", "ありがとうございます"))
	cases := map[string]string{
		";jp":      "こんにちは",
		";arigato": "ありがとうございます",
	}
	for trigger, want := range cases {
		res, count := e.FeedString(trigger)
		if count != 1 || res.Match.Replacement != want {
			t.Errorf("%s: got (%q, count=%d), want %q", trigger, res.Match.Replacement, count, want)
		}
	}
}

func TestChineseExpansion(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";cn", "你好"), rule(";nihao", "你好，世界"))
	for _, tc := range []struct{ in, want string }{
		{";cn", "你好"},
		{";nihao", "你好，世界"},
	} {
		res, count := e.FeedString(tc.in)
		if count != 1 || res.Match.Replacement != tc.want {
			t.Errorf("%s: got (%q, count=%d), want %q", tc.in, res.Match.Replacement, count, tc.want)
		}
	}
}

func TestKoreanExpansion(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";kr", "안녕하세요"), rule(";annyeong", "안녕하세요, 반갑습니다"))
	for _, tc := range []struct{ in, want string }{
		{";kr", "안녕하세요"},
		{";annyeong", "안녕하세요, 반갑습니다"},
	} {
		res, count := e.FeedString(tc.in)
		if count != 1 || res.Match.Replacement != tc.want {
			t.Errorf("%s: got (%q, count=%d), want %q", tc.in, res.Match.Replacement, count, tc.want)
		}
	}
}

func TestGermanExpansion(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";de", "Grüße"), rule(";gruss", "Guten Tag, wie geht's?"))
	for _, tc := range []struct{ in, want string }{
		{";de", "Grüße"},
		{";gruss", "Guten Tag, wie geht's?"},
	} {
		res, count := e.FeedString(tc.in)
		if count != 1 || res.Match.Replacement != tc.want {
			t.Errorf("%s: got (%q, count=%d), want %q", tc.in, res.Match.Replacement, count, tc.want)
		}
	}
}

func TestCaseInsensitiveGerman(t *testing.T) {
	e := newTestEngine(t, Config{CaseSensitive: false}, rule(";Grüße", " Grüße "))
	// Lowercasing Unicode (including ß/ü) must still match.
	res, count := e.FeedString(";grüße")
	if count != 1 || res.Match.Replacement != " Grüße " {
		t.Fatalf("case-insensitive German failed: got (%q, count=%d)", res.Match.Replacement, count)
	}
}

func TestEmojiExpansion(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";emoji", "🎉 done ✅"))
	res, count := e.FeedString(";emoji")
	if count != 1 || res.Match.Replacement != "🎉 done ✅" {
		t.Fatalf("emoji failed: got (%q, count=%d)", res.Match.Replacement, count)
	}
}

func TestCJKWordBoundary(t *testing.T) {
	// A trigger that starts with a CJK letter requires a word boundary, so it
	// must not expand when glued to preceding word characters.
	e := newTestEngine(t, Config{WordBoundary: true}, rule("你好", "ni hao"))
	if _, count := e.FeedString("x你好"); count != 0 {
		t.Error("CJK trigger glued to a preceding word should not expand")
	}
	e2 := newTestEngine(t, Config{WordBoundary: true}, rule("你好", "ni hao"))
	res, count := e2.FeedString(" 你好")
	if count != 1 || res.Typed != "你好" {
		t.Fatalf("CJK trigger after a boundary should expand, got (%+v, count=%d)", res, count)
	}
}

func TestMultiLanguageBufferIntegrity(t *testing.T) {
	// Feeding a long mixed-language sentence must not corrupt the buffer or
	// panic, and a trailing trigger must still match.
	sentence := "Hello こんにちは 你好 안녕하세요 Grüße 🎉 "
	e := newTestEngine(t, Config{}, rule(";end", "FIN"))
	res, count := e.FeedString(sentence + ";end")
	if count != 1 || res.Match.Replacement != "FIN" {
		t.Fatalf("mixed-language buffer failed: count=%d replacement=%q", count, res.Match.Replacement)
	}
}

func TestMultiLanguageConfigValidation(t *testing.T) {
	doc := `matches:
  - trigger: ";jp"
    replace: "こんにちは"
  - trigger: ";cn"
    replace: "你好"
  - trigger: ";kr"
    replace: "안녕하세요"
  - trigger: ";de"
    replace: "Grüße"
  - trigger: ";emoji"
    replace: "🎉 done ✅"
`
	cfg, err := config.Parse([]byte(doc), "multilang.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(cfg.Matches) != 5 {
		t.Fatalf("matches = %d, want 5", len(cfg.Matches))
	}
	// Rune counts must be computed correctly (not byte counts).
	if got := len([]rune(cfg.Matches[0].Replace)); got != 5 {
		t.Errorf("Japanese replacement rune count = %d, want 5", got)
	}
	if got := len([]rune(cfg.Matches[4].Replace)); got != 8 {
		t.Errorf("emoji replacement rune count = %d, want 8", got)
	}
}

func TestGermanUmlautLengths(t *testing.T) {
	// "Grüße" is 6 runes but 7 bytes (ü=2, ß=2). Length validation must use
	// runes, so a 6-rune trigger fits a 6-rune limit.
	doc := "engine:\n  max_trigger_length: 6\nmatches:\n  - trigger: \"Grüße\"\n    replace: \"x\"\n"
	cfg, err := config.Parse([]byte(doc), "de.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("Validate rejected a 6-rune trigger under a 6-rune limit: %v", err)
	}
	_ = strings.TrimSpace
}
