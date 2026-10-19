package storage

import (
	"fmt"
	"strings"
)

// Open returns a Store for the configured backend. It is the single entry point
// callers should use so that the configured backend is actually honoured.
func Open(backend, path string) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "jsonl":
		return OpenJSONL(path)
	case "sqlite":
		return OpenSQLite(path)
	default:
		return nil, fmt.Errorf("unknown storage backend %q (want jsonl or sqlite)", backend)
	}
}
