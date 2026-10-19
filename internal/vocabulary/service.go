// Package vocabulary manages user vocabulary and, when explicitly enabled,
// local suggestion candidates. Learning is disabled by default; suggestions
// must always be reviewed and accepted by the user before they become active.
package vocabulary

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/storage"
)

// Service coordinates vocabulary CRUD and export. It never activates a
// replacement automatically.
type Service struct {
	store storage.Store
	cfg   config.SuggestionsConfig
	now   func() time.Time
}

// NewService wraps store with the suggestion settings from cfg.
func NewService(store storage.Store, cfg config.SuggestionsConfig) *Service {
	return &Service{store: store, cfg: cfg, now: time.Now}
}

// NewServiceWithClock returns a Service with a deterministic clock for tests.
func NewServiceWithClock(store storage.Store, cfg config.SuggestionsConfig, clock func() time.Time) *Service {
	s := NewService(store, cfg)
	if clock != nil {
		s.now = clock
	}
	return s
}

// Add stores a new entry after basic validation.
func (s *Service) Add(trigger, expansion, language string) error {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return fmt.Errorf("trigger must not be empty")
	}
	if expansion == "" {
		return fmt.Errorf("expansion must not be empty")
	}
	return s.store.Add(storage.VocabularyEntry{
		Trigger:   trigger,
		Expansion: expansion,
		Language:  language,
		Source:    "user",
		CreatedAt: s.now(),
	})
}

// List returns every stored entry.
func (s *Service) List() ([]storage.VocabularyEntry, error) {
	return s.store.List()
}

// Export writes the vocabulary as JSON Lines to w.
func (s *Service) Export(w io.Writer) error {
	entries, err := s.store.List()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", e.Trigger, e.Expansion); err != nil {
			return err
		}
	}
	return nil
}

// LearningEnabled reports whether the user has opted into learning.
func (s *Service) LearningEnabled() bool {
	return s.cfg.Enabled && s.cfg.LearnFromTyping
}

// Close releases the underlying store.
func (s *Service) Close() error {
	return s.store.Close()
}
