// Package logging configures the structured logger used across Gospan. Logs
// must never contain typed user content by default.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger writing to w at the given level. An unknown or
// empty level selects info. A nil writer selects stderr.
func New(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
