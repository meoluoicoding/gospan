package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// modernc.org/sqlite is a pure-Go SQLite driver: no CGO, so the core module
	// keeps building with CGO_ENABLED=0.
	_ "modernc.org/sqlite"
)

// sqliteSchema creates the vocabulary table and its lookup index. It is
// idempotent so it doubles as the migration for the initial version.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS vocabulary (
	trigger    TEXT NOT NULL,
	expansion  TEXT NOT NULL,
	language   TEXT NOT NULL DEFAULT '',
	source     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_vocabulary_trigger ON vocabulary(trigger);
`

// sqliteStore persists vocabulary in a local SQLite database.
type sqliteStore struct {
	db *sql.DB
}

// OpenSQLite opens (creating if necessary) a SQLite vocabulary store at path.
func OpenSQLite(path string) (Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store %q: %w", path, err)
	}

	// A single connection avoids SQLITE_BUSY between concurrent writers; the
	// vocabulary workload is low volume, so this is not a bottleneck.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize sqlite store %q: %w", path, err)
	}
	return &sqliteStore{db: db}, nil
}

// Add inserts an entry.
func (s *sqliteStore) Add(entry VocabularyEntry) error {
	created := entry.CreatedAt
	if created.IsZero() {
		created = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO vocabulary (trigger, expansion, language, source, created_at) VALUES (?, ?, ?, ?, ?)`,
		entry.Trigger,
		entry.Expansion,
		entry.Language,
		entry.Source,
		created.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert vocabulary entry: %w", err)
	}
	return nil
}

// List returns every entry in insertion order.
func (s *sqliteStore) List() ([]VocabularyEntry, error) {
	rows, err := s.db.Query(
		`SELECT trigger, expansion, language, source, created_at FROM vocabulary ORDER BY rowid`,
	)
	if err != nil {
		return nil, fmt.Errorf("query vocabulary: %w", err)
	}
	defer rows.Close()

	var entries []VocabularyEntry
	for rows.Next() {
		var (
			e       VocabularyEntry
			created string
		)
		if err := rows.Scan(&e.Trigger, &e.Expansion, &e.Language, &e.Source, &created); err != nil {
			return nil, fmt.Errorf("scan vocabulary row: %w", err)
		}
		if ts, err := time.Parse(time.RFC3339Nano, created); err == nil {
			e.CreatedAt = ts
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read vocabulary: %w", err)
	}
	return entries, nil
}

// Close releases the database handle.
func (s *sqliteStore) Close() error {
	return s.db.Close()
}
