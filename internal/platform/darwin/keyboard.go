//go:build darwin

// Package darwin implements the macOS platform integration. Keyboard capture and
// text insertion are not yet implemented (they require CGO/CGEventTap); the
// package currently provides best-effort clipboard and foreground-process
// helpers so the core engine and CLI build on macOS.
package darwin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/snzpet/gospan/internal/platform"
)

// Platform is the macOS implementation of platform.Platform.
type Platform struct {
	events chan platform.KeyboardEvent
	mu     sync.Mutex
}

// New returns a macOS platform integration.
func New() *Platform {
	return &Platform{events: make(chan platform.KeyboardEvent)}
}

// Start is not yet implemented on macOS.
func (p *Platform) Start() error { return platform.ErrUnsupported }

// Stop releases resources.
func (p *Platform) Stop() error { return nil }

// Events returns a closed channel; no events are produced yet.
func (p *Platform) Events() <-chan platform.KeyboardEvent {
	ch := make(chan platform.KeyboardEvent)
	close(ch)
	return ch
}

// ReplacePreviousText is not yet implemented on macOS.
func (p *Platform) ReplacePreviousText(_, _ string, _ platform.InsertionStrategy) error {
	return platform.ErrUnsupported
}

// MoveCursorLeft is not yet implemented on macOS.
func (p *Platform) MoveCursorLeft(_ int) error { return platform.ErrUnsupported }

// ReadClipboard returns the current clipboard text via pbpaste.
func (p *Platform) ReadClipboard() (string, error) { return ReadClipboardText() }

// WriteClipboard sets the clipboard text via pbcopy.
func (p *Platform) WriteClipboard(text string) error { return WriteClipboardText(text) }

// ForegroundProcessName returns the frontmost application name.
func (p *Platform) ForegroundProcessName() (string, error) { return ForegroundProcessName() }

// ReadClipboardText reads the clipboard via pbpaste.
func ReadClipboardText() (string, error) {
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return out.String(), nil
}

// WriteClipboardText writes the clipboard via pbcopy.
func WriteClipboardText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewBufferString(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// ForegroundProcessName returns the frontmost application name (lower-cased).
func ForegroundProcessName() (string, error) {
	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to get name of first application process whose frontmost is true`)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(out.String())), nil
}
