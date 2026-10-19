package config

import (
	"os"
	"path/filepath"
)

// EnvConfig overrides the default configuration path when set.
const EnvConfig = "GOSPAN_CONFIG"

// DefaultDir returns the per-user Gospan data directory:
//
//	%APPDATA%\gospan on Windows, $XDG_CONFIG_HOME/gospan (or ~/.config/gospan)
//	on other platforms.
//
// If the home directory cannot be determined, it falls back to the current
// working directory so the application still starts.
func DefaultDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "gospan")
	}
	return "gospan"
}

// DefaultPath returns the default configuration file path, honouring the
// GOSPAN_CONFIG environment variable when it is set.
func DefaultPath() string {
	if p := os.Getenv(EnvConfig); p != "" {
		return p
	}
	return filepath.Join(DefaultDir(), "config.yaml")
}

// ResolvePath picks the configuration path from, in order: an explicit
// override, the GOSPAN_CONFIG environment variable, then the default location.
func ResolvePath(override string) string {
	if override != "" {
		return override
	}
	return DefaultPath()
}
