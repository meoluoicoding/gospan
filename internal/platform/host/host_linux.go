//go:build linux

package host

import (
	"github.com/snzpet/gospan/internal/platform"
	"github.com/snzpet/gospan/internal/platform/linux"
)

// New returns the platform integration for Linux.
func New() (platform.Platform, error) {
	return linux.New(), nil
}
