// Package storage persists learned vocabulary and optional statistics. It is
// deliberately separate from configuration: human-maintained rules live in
// YAML, while generated or learned data lives here.
package storage

import "time"

// VocabularyEntry is a single learned or user-authored vocabulary item.
type VocabularyEntry struct {
	Trigger   string    `json:"trigger"`
	Expansion string    `json:"expansion"`
	Language  string    `json:"language,omitempty"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is the persistence interface for vocabulary entries.
type Store interface {
	// Add appends an entry. Implementations must be safe for concurrent use.
	Add(entry VocabularyEntry) error
	// List returns every stored entry in insertion order.
	List() ([]VocabularyEntry, error)
	// Close releases resources.
	Close() error
}
