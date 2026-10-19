// Package expansion renders replacement templates. It supports a small,
// explicit set of variables and must never execute shell commands or evaluate
// arbitrary expressions.
package expansion

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// Default layouts used when a variable is given without an explicit layout.
const (
	DefaultDateLayout     = "2006-01-02"
	DefaultTimeLayout     = "15:04:05"
	DefaultDateTimeLayout = "2006-01-02 15:04:05"
)

// Context carries the ambient values available to template variables.
type Context struct {
	// Now is the reference time used by date and time variables. A zero value
	// is replaced with time.Now by the pipeline.
	Now time.Time
	// ID is a unique identifier used by the {{uuid}} variable. When empty, a
	// random identifier is generated.
	ID string
	// Clipboard holds the (possibly truncated) clipboard text used by the
	// {{clipboard}} variable. It is empty when the variable is unavailable or
	// disabled.
	Clipboard string
}

// resolve evaluates a single variable. It returns an error for unknown names so
// configuration mistakes surface early.
func resolve(name, arg string, ctx Context) (string, error) {
	switch name {
	case "date":
		return formatTime(ctx.Now, arg, DefaultDateLayout), nil
	case "time":
		return formatTime(ctx.Now, arg, DefaultTimeLayout), nil
	case "datetime":
		return formatTime(ctx.Now, arg, DefaultDateTimeLayout), nil
	case "uuid":
		if ctx.ID != "" {
			return ctx.ID, nil
		}
		return NewID()
	case "clipboard":
		return ctx.Clipboard, nil
	default:
		return "", fmt.Errorf("unknown template variable %q", name)
	}
}

func formatTime(t time.Time, layout, fallback string) string {
	if strings.TrimSpace(layout) == "" {
		layout = fallback
	}
	return t.Format(layout)
}

// NewID returns a random RFC 4122 version 4 UUID string.
func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// knownVariables lists the variable names the engine understands.
var knownVariables = map[string]bool{
	"date":      true,
	"time":      true,
	"datetime":  true,
	"uuid":      true,
	"clipboard": true,
}
