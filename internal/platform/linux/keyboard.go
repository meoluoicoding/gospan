//go:build linux

// Package linux implements the Linux platform integration. Keyboard capture and
// text insertion are not yet implemented (they require X11/Wayland bindings);
// the package currently provides best-effort clipboard and foreground-process
// helpers so the core engine and CLI build on Linux.
package linux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/snzpet/gospan/internal/platform"
)

// Platform is the Linux implementation of platform.Platform.
type Platform struct {
	events chan platform.KeyboardEvent
	mu     sync.Mutex
}

// New returns a Linux platform integration.
func New() *Platform {
	return &Platform{events: make(chan platform.KeyboardEvent)}
}

// Start is not yet implemented on Linux.
func (p *Platform) Start() error { return platform.ErrUnsupported }

// Stop releases resources.
func (p *Platform) Stop() error { return nil }

// Events returns a closed channel; no events are produced yet.
func (p *Platform) Events() <-chan platform.KeyboardEvent {
	ch := make(chan platform.KeyboardEvent)
	close(ch)
	return ch
}

// ReplacePreviousText is not yet implemented on Linux.
func (p *Platform) ReplacePreviousText(_, _ string, _ platform.InsertionStrategy) error {
	return platform.ErrUnsupported
}

// MoveCursorLeft is not yet implemented on Linux.
func (p *Platform) MoveCursorLeft(_ int) error { return platform.ErrUnsupported }

// ReadClipboard returns the current clipboard text via xclip/xsel.
func (p *Platform) ReadClipboard() (string, error) { return ReadClipboardText() }

// WriteClipboard sets the clipboard text via xclip/xsel.
func (p *Platform) WriteClipboard(text string) error { return WriteClipboardText(text) }

// ForegroundProcessName returns the active window's process name.
func (p *Platform) ForegroundProcessName() (string, error) { return ForegroundProcessName() }

// ReadClipboardText tries xclip then xsel.
func ReadClipboardText() (string, error) {
	cmd := exec.Command("xclip", "-o", "-selection", "clipboard")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return out.String(), nil
	}
	cmd = exec.Command("xsel", "--clipboard", "--output")
	out.Reset()
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return out.String(), nil
	}
	return "", fmt.Errorf("no clipboard tool available (install xclip or xsel)")
}

// WriteClipboardText tries xclip then xsel.
func WriteClipboardText(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = bytes.NewBufferString(text)
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmd = exec.Command("xsel", "--clipboard", "--input")
	cmd.Stdin = bytes.NewBufferString(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no clipboard tool available (install xclip or xsel): %w", err)
	}
	return nil
}

// ForegroundProcessName uses xdotool + /proc when available.
func ForegroundProcessName() (string, error) {
	cmd := exec.Command("xdotool", "getactivewindow", "getwindowpid")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("xdotool: %w", err)
	}
	pid := strings.TrimSpace(out.String())
	data, err := os.ReadFile("/proc/" + pid + "/comm")
	if err != nil {
		return "", fmt.Errorf("read /proc/%s/comm: %w", pid, err)
	}
	return strings.ToLower(strings.TrimSpace(string(data))), nil
}
