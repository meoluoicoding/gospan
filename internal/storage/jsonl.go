package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// jsonlStore is an append-only JSON Lines store. It is well suited to the MVP:
// easy to inspect, back up, and export. Updates and deletes require rewriting
// (compaction) and are intentionally out of scope here.
type jsonlStore struct {
	mu   sync.Mutex
	path string
	file *os.File
}

// OpenJSONL opens (creating if necessary) a JSON Lines vocabulary store at path.
func OpenJSONL(path string) (Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage directory: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open vocabulary store %q: %w", path, err)
	}
	return &jsonlStore{path: path, file: f}, nil
}

// Add appends entry as a single JSON line.
func (s *jsonlStore) Add(entry VocabularyEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode vocabulary entry: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write vocabulary entry: %w", err)
	}
	return s.file.Sync()
}

// List reads every entry from the backing file.
func (s *jsonlStore) List() ([]VocabularyEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read vocabulary store %q: %w", s.path, err)
	}
	defer f.Close()

	var entries []VocabularyEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e VocabularyEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode vocabulary entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read vocabulary store %q: %w", s.path, err)
	}
	return entries, nil
}

// Close closes the backing file.
func (s *jsonlStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}
