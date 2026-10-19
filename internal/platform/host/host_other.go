//go:build !windows

package host

import "github.com/snzpet/gospan/internal/platform"

// unsupported is a no-op platform for operating systems without an integration.
type unsupported struct {
	events chan platform.KeyboardEvent
}

func (u unsupported) Start() error { return platform.ErrUnsupported }
func (u unsupported) Stop() error  { return nil }

func (u unsupported) Events() <-chan platform.KeyboardEvent {
	if u.events == nil {
		ch := make(chan platform.KeyboardEvent)
		close(ch)
		return ch
	}
	return u.events
}

func (u unsupported) ReplacePreviousText(_, _ string, _ platform.InsertionStrategy) error {
	return platform.ErrUnsupported
}

func (u unsupported) MoveCursorLeft(_ int) error { return platform.ErrUnsupported }

func (u unsupported) ReadClipboard() (string, error) { return "", platform.ErrUnsupported }
func (u unsupported) WriteClipboard(_ string) error  { return platform.ErrUnsupported }

// New returns a platform integration that reports ErrUnsupported for every
// capture operation.
func New() (platform.Platform, error) {
	return unsupported{}, nil
}
