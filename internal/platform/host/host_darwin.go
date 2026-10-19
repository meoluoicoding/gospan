//go:build darwin

package host

import (
	"github.com/snzpet/gospan/internal/platform"
	"github.com/snzpet/gospan/internal/platform/darwin"
)

// New returns the platform integration for macOS.
func New() (platform.Platform, error) {
	return darwin.New(), nil
}
