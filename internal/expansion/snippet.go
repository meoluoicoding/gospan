// Package expansion renders replacement templates, including snippet markers.
package expansion

import (
	"strings"
)

// SnippetResult is the rendered form of a snippet template. Text contains the
// replacement with all markers removed. CursorOffset is the rune index where
// the caret should end up (-1 means "leave at the end"). TabStops lists the
// rune offsets of $1..$9 followed by $0 (the final stop), in navigation order.
type SnippetResult struct {
	Text         string
	CursorOffset int
	TabStops     []int
}

// ParseSnippet converts a (already variable-expanded) template into a
// SnippetResult. Supported markers:
//
//	$$   -> literal '$'
//	$|$  -> cursor position
//	$0   -> final tab stop (also the cursor if no $|$ is present)
//	$1..$9 -> tab stops, in numeric order
//
// Markers are stripped from the output text.
func ParseSnippet(template string) SnippetResult {
	var b strings.Builder
	runes := []rune(template)
	i := 0
	n := len(runes)

	type stop struct {
		num int
		off int
	}
	var stops []stop
	cursorIdx := -1

	finalStop := -1
	runeCount := 0

	for i < n {
		if runes[i] != '$' {
			b.WriteRune(runes[i])
			runeCount++
			i++
			continue
		}
		// $$ -> literal $
		if i+1 < n && runes[i+1] == '$' {
			b.WriteRune('$')
			runeCount++
			i += 2
			continue
		}
		// $|$ -> cursor
		if i+2 < n && runes[i+1] == '|' && runes[i+2] == '$' {
			if cursorIdx < 0 {
				cursorIdx = runeCount
			}
			stops = append(stops, stop{num: 0, off: runeCount})
			i += 3
			continue
		}
		// $digit
		if i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9' {
			num := int(runes[i+1] - '0')
			stops = append(stops, stop{num: num, off: runeCount})
			if num == 0 {
				finalStop = runeCount
			}
			i += 2
			continue
		}
		// lone $ -> literal
		b.WriteRune('$')
		runeCount++
		i++
	}

	text := b.String()
	textRunes := []rune(text)

	// Resolve cursor offset.
	if cursorIdx < 0 {
		// No explicit $| marker: prefer first numbered tab stop, then $0, then end.
		firstTab := -1
		for num := 1; num <= 9; num++ {
			for _, s := range stops {
				if s.num == num {
					if firstTab < 0 || s.off < firstTab {
						firstTab = s.off
					}
				}
			}
			if firstTab >= 0 {
				break
			}
		}
		switch {
		case firstTab >= 0:
			cursorIdx = firstTab
		case finalStop >= 0:
			cursorIdx = finalStop
		default:
			cursorIdx = len(textRunes)
		}
	}
	if cursorIdx > len(textRunes) {
		cursorIdx = len(textRunes)
	}

	// Tab stops in navigation order: $1..$9 (by number), then $0 (final).
	// Deduplicate by offset.
	seen := map[int]bool{}
	var tabStops []int
	for num := 1; num <= 9; num++ {
		for _, s := range stops {
			if s.num == num && !seen[s.off] {
				seen[s.off] = true
				tabStops = append(tabStops, s.off)
			}
		}
	}
	// final stop ($0 / $|$: recorded with num 0)
	for _, s := range stops {
		if s.num == 0 && !seen[s.off] {
			seen[s.off] = true
			tabStops = append(tabStops, s.off)
		}
	}

	if cursorIdx < 0 {
		cursorIdx = len(textRunes)
	}

	return SnippetResult{
		Text:         text,
		CursorOffset: cursorIdx,
		TabStops:     tabStops,
	}
}

// RenderSnippet expands template variables first, then parses snippet markers.
func (p *Pipeline) RenderSnippet(tmpl string) (SnippetResult, error) {
	text, err := Expand(tmpl, Context{Now: p.now(), Clipboard: p.clipboardText()})
	if err != nil {
		return SnippetResult{}, err
	}
	return ParseSnippet(text), nil
}

// clipboardText returns the (possibly truncated) clipboard text, or "".
func (p *Pipeline) clipboardText() string {
	if !p.clipOn || p.clip == nil {
		return ""
	}
	text, err := p.clip.ReadClipboard()
	if err != nil {
		return ""
	}
	if len([]rune(text)) > p.clipMax {
		text = string([]rune(text)[:p.clipMax])
	}
	return text
}

// NextTabStop returns the next tab stop strictly after current, wrapping to the
// first stop. It returns -1 when there are no tab stops.
func (r SnippetResult) NextTabStop(current int) int {
	if len(r.TabStops) == 0 {
		return -1
	}
	for _, off := range r.TabStops {
		if off > current {
			return off
		}
	}
	return r.TabStops[0]
}
