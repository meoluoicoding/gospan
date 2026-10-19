// Package platform defines the operating-system boundary of Gospan. The core
// engine depends only on this interface, never on a concrete OS integration.
package platform

import "errors"

// ErrUnsupported is returned when a platform capability is unavailable on the
// current operating system.
var ErrUnsupported = errors.New("platform integration is not supported on this operating system")

// KeyboardEvent is a single key transition captured from the OS.
type KeyboardEvent struct {
	Rune      rune
	KeyCode   uint32
	IsDown    bool
	Modifiers uint32
	Timestamp int64
	Injected  bool
}

// Modifier bit flags carried by KeyboardEvent.
const (
	ModShift uint32 = 1 << iota
	ModControl
	ModAlt
	ModWin
)

// InsertionStrategy selects how a replacement is inserted into the target app.
type InsertionStrategy int

const (
	// InsertBackspace removes the trigger with backspaces and types the
	// replacement keystroke by keystroke.
	InsertBackspace InsertionStrategy = iota
	// InsertClipboard uses the clipboard and a paste keystroke.
	InsertClipboard
	// InsertAuto picks backspace for short replacements and clipboard for long
	// or multiline ones.
	InsertAuto
)

// Platform abstracts keyboard capture and text insertion.
type Platform interface {
	// Start begins capturing keyboard events. It returns once the capture
	// mechanism is initialised, or an error if it cannot be.
	Start() error
	// Stop releases capture resources. It is safe to call more than once.
	Stop() error
	// Events returns the channel of captured events. The channel is closed
	// after Stop.
	Events() <-chan KeyboardEvent
	// ReplacePreviousText replaces the last len(typed) characters with
	// replacement using the given strategy.
	ReplacePreviousText(typed, replacement string, strategy InsertionStrategy) error
	// MoveCursorLeft moves the caret left by n positions. Used for snippet
	// cursor placement.
	MoveCursorLeft(n int) error
	// ReadClipboard returns the current clipboard text (capped by the caller).
	ReadClipboard() (string, error)
	// WriteClipboard sets the clipboard text.
	WriteClipboard(text string) error
}
