package expansion

import (
	"strings"
	"testing"
)

func TestParseSnippetCursor(t *testing.T) {
	r := ParseSnippet("hello$|$world")
	if r.Text != "helloworld" {
		t.Errorf("Text = %q, want helloworld", r.Text)
	}
	if r.CursorOffset != 5 {
		t.Errorf("CursorOffset = %d, want 5", r.CursorOffset)
	}
}

func TestParseSnippetTabStops(t *testing.T) {
	r := ParseSnippet("func $1() {\n\t$0\n}")
	if r.Text != "func () {\n\t\n}" {
		t.Errorf("Text = %q", r.Text)
	}
	if len(r.TabStops) != 2 {
		t.Fatalf("TabStops = %v, want 2", r.TabStops)
	}
	// $1 at offset 5, $0 at offset 11
	if r.TabStops[0] != 5 {
		t.Errorf("first tab stop = %d, want 5", r.TabStops[0])
	}
	if r.TabStops[1] != 11 {
		t.Errorf("second tab stop = %d, want 11", r.TabStops[1])
	}
	// cursor defaults to first tab stop when no $|
	if r.CursorOffset != 5 {
		t.Errorf("CursorOffset = %d, want 5 (first tab stop)", r.CursorOffset)
	}
}

func TestParseSnippetCursorOverridesTabStop(t *testing.T) {
	r := ParseSnippet("$1 hello $|$")
	if r.CursorOffset != 7 {
		t.Errorf("CursorOffset = %d, want 7", r.CursorOffset)
	}
}

func TestParseSnippetEscapeDollar(t *testing.T) {
	r := ParseSnippet("price: $$5")
	if r.Text != "price: $5" {
		t.Errorf("Text = %q, want price: $5", r.Text)
	}
}

func TestParseSnippetLoneDollar(t *testing.T) {
	r := ParseSnippet("cost $x")
	if r.Text != "cost $x" {
		t.Errorf("Text = %q, want cost $x", r.Text)
	}
}

func TestParseSnippetNoMarkers(t *testing.T) {
	r := ParseSnippet("plain text")
	if r.Text != "plain text" {
		t.Errorf("Text = %q", r.Text)
	}
	if r.CursorOffset != len([]rune("plain text")) {
		t.Errorf("CursorOffset = %d, want end", r.CursorOffset)
	}
	if len(r.TabStops) != 0 {
		t.Errorf("TabStops = %v, want none", r.TabStops)
	}
}

func TestParseSnippetUnicode(t *testing.T) {
	r := ParseSnippet("xin chào $|$")
	if r.Text != "xin chào " {
		t.Errorf("Text = %q", r.Text)
	}
	if r.CursorOffset != 9 {
		t.Errorf("CursorOffset = %d, want 9 (rune count)", r.CursorOffset)
	}
}

func TestNextTabStop(t *testing.T) {
	r := ParseSnippet("$1 a $2 b $0")
	if r.NextTabStop(-1) != r.TabStops[0] {
		t.Errorf("NextTabStop(-1) = %d, want %d", r.NextTabStop(-1), r.TabStops[0])
	}
	last := r.TabStops[len(r.TabStops)-1]
	if r.NextTabStop(last) != r.TabStops[0] {
		t.Errorf("wrap failed: got %d, want %d", r.NextTabStop(last), r.TabStops[0])
	}
	empty := ParseSnippet("no stops")
	if empty.NextTabStop(0) != -1 {
		t.Errorf("empty NextTabStop = %d, want -1", empty.NextTabStop(0))
	}
}

func TestRenderSnippetBasic(t *testing.T) {
	clock := func() interface{ Format(string) string } { return nil }
	_ = clock
	p := NewPipeline()
	r, err := p.RenderSnippet("Hello {{date:2006}} $|$")
	if err != nil {
		t.Fatalf("RenderSnippet: %v", err)
	}
	if !strings.HasPrefix(r.Text, "Hello ") {
		t.Errorf("Text = %q", r.Text)
	}
	if r.CursorOffset < 0 {
		t.Errorf("CursorOffset = %d", r.CursorOffset)
	}
}
