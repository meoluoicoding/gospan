package expansion

import (
	"strings"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Date(2026, 10, 18, 14, 30, 5, 0, time.UTC)
	return func() time.Time { return t }
}

func TestRenderDateVariables(t *testing.T) {
	p := NewPipelineWithClock(fixedClock())
	got, err := p.Render("{{date:2006-01-02}} {{time:15:04}}")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got != "2026-10-18 14:30" {
		t.Errorf("Render = %q, want %q", got, "2026-10-18 14:30")
	}
}

func TestRenderDefaultLayouts(t *testing.T) {
	p := NewPipelineWithClock(fixedClock())
	got, err := p.Render("{{date}}|{{time}}")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got != "2026-10-18|14:30:05" {
		t.Errorf("Render = %q, want %q", got, "2026-10-18|14:30:05")
	}
}

func TestRenderPlainTextUnchanged(t *testing.T) {
	p := NewPipelineWithClock(fixedClock())
	got, err := p.Render("just text")
	if err != nil || got != "just text" {
		t.Fatalf("Render = (%q, %v)", got, err)
	}
}

func TestRenderUnknownVariable(t *testing.T) {
	_, err := Expand("{{nope}}", Context{Now: time.Now()})
	if err == nil || !strings.Contains(err.Error(), "unknown template variable") {
		t.Fatalf("Expand error = %v, want unknown-variable", err)
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := ValidateTemplate("{{date:2006-01-02}} and {{uuid}}"); err != nil {
		t.Errorf("valid template rejected: %v", err)
	}
	if err := ValidateTemplate("{{bogus}}"); err == nil {
		t.Error("invalid template accepted")
	}
}

type clipboardStub struct{ text string }

func (c clipboardStub) ReadClipboard() (string, error) { return c.text, nil }

func TestRenderClipboardVariable(t *testing.T) {
	p := NewPipelineWithClock(fixedClock())
	p.SetClipboardReader(clipboardStub{"hello"}, 10, true)
	got, err := p.Render("{{clipboard}}")
	if err != nil || got != "hello" {
		t.Fatalf("Render = (%q, %v), want hello", got, err)
	}

	p2 := NewPipelineWithClock(fixedClock())
	p2.SetClipboardReader(clipboardStub{"hello"}, 10, false)
	got2, err := p2.Render("{{clipboard}}")
	if err != nil || got2 != "" {
		t.Fatalf("disabled Render = (%q, %v), want empty", got2, err)
	}

	p3 := NewPipelineWithClock(fixedClock())
	p3.SetClipboardReader(clipboardStub{"abcdefghij"}, 4, true)
	got3, err := p3.Render("{{clipboard}}")
	if err != nil || got3 != "abcd" {
		t.Fatalf("truncated Render = (%q, %v), want abcd", got3, err)
	}
}

func TestUUIDShape(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("uuid %q has length %d, want 36", id, len(id))
	}
	if _, err := NewID(); err != nil {
		t.Fatalf("NewID: %v", err)
	}
}
