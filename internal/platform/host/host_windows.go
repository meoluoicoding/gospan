//go:build windows

// Package host selects the concrete platform integration for the current
// operating system. Keeping the factory in its own package avoids an import
// cycle between the platform interface and its Windows implementation.
package host

import (
	"github.com/snzpet/gospan/internal/platform"
	"github.com/snzpet/gospan/internal/platform/windows"
)

// New returns the platform integration for Windows.
func New() (platform.Platform, error) {
	return windows.New(), nil
}
