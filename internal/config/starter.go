package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// starterConfig is a minimal, self-contained configuration written on first run.
// It deliberately avoids includes so it validates on its own.
const starterConfig = `# Gospan configuration.
# Created automatically on first run. Edit this file directly, or use:
#   gospan match add ";x" "value"
#   gospan config validate

app:
  name: gospan
  log_level: info
  start_with_windows: false

engine:
  enabled: true
  case_sensitive: false
  word_boundary: true
  escape_char: "\\"
  max_trigger_length: 64
  max_expansion_length: 16384

input:
  minimum_interval_ms: 0
  ignore_injected_events: true
  buffer_size: 256
  excluded_processes:
    - keepass.exe

suggestions:
  enabled: true
  learn_from_typing: false
  minimum_occurrences: 5
  minimum_phrase_length: 2
  maximum_candidates: 5

storage:
  # backend: jsonl (default) or sqlite (pure-Go driver, no CGO)
  backend: jsonl
  path: data/vocabulary.jsonl

insertion:
  strategy: auto            # auto | backspace | clipboard
  clipboard_threshold: 256

clipboard:
  enabled: false            # opt-in
  max_length: 4096

matches:
  - trigger: ";dc"
    replace: "được"
  - trigger: ";k"
    replace: "không"
  - trigger: ";kt"
    replace: "kiểm tra"
  - trigger: ";brse"
    replace: "Bridge Software Engineer"
  - trigger: ";date"
    replace: "{{date:2006-01-02}}"
  - trigger: ";sig"
    replace: |
      Trân trọng,
      Khoa
`

// WriteStarter creates a starter configuration at path. It fails if the file
// already exists, so it never overwrites a user's configuration.
func WriteStarter(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("configuration already exists at %s", path)
		}
		return fmt.Errorf("create %s: %w", path, err)
	}

	if _, err := f.WriteString(starterConfig); err != nil {
		f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync %s: %w", path, err)
	}
	return f.Close()
}

// IsNotExist reports whether err indicates a missing file. It lets callers
// distinguish "no configuration yet" from "configuration is invalid".
func IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
