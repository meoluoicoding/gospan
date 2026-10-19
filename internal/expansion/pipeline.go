package expansion

import (
	"strings"
	"time"
)

// ClipboardReader is implemented by platform integrations that can read the
// current clipboard text. The expansion package depends on this interface, not
// on any platform package.
type ClipboardReader interface {
	ReadClipboard() (string, error)
}

// Pipeline renders replacement templates with a reliable time source and an
// optional clipboard reader.
type Pipeline struct {
	now     func() time.Time
	clip    ClipboardReader
	clipMax int
	clipOn  bool
}

// NewPipeline returns a Pipeline using time.Now as its clock.
func NewPipeline() *Pipeline {
	return &Pipeline{now: time.Now, clipMax: 4096}
}

// NewPipelineWithClock returns a Pipeline using the supplied clock. It is used
// by tests and callers that need deterministic timestamps.
func NewPipelineWithClock(clock func() time.Time) *Pipeline {
	if clock == nil {
		clock = time.Now
	}
	return &Pipeline{now: clock, clipMax: 4096}
}

// SetClipboardReader installs (or clears) the clipboard reader used by the
// {{clipboard}} variable.
func (p *Pipeline) SetClipboardReader(reader ClipboardReader, max int, enabled bool) {
	p.clip = reader
	p.clipMax = max
	if p.clipMax <= 0 {
		p.clipMax = 4096
	}
	p.clipOn = enabled
}

// Render expands every variable in tmpl.
func (p *Pipeline) Render(tmpl string) (string, error) {
	ctx := Context{Now: p.now()}
	if p.clipOn && p.clip != nil && strings.Contains(tmpl, "clipboard") {
		text, err := p.clip.ReadClipboard()
		if err == nil {
			if len([]rune(text)) > p.clipMax {
				text = string([]rune(text)[:p.clipMax])
			}
			ctx.Clipboard = text
		}
	}
	return Expand(tmpl, ctx)
}
