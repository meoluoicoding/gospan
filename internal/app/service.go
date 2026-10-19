// Package app wires configuration, the matching engine, expansion, and the
// platform integration into an application service. GUI and CLI call this
// service; they never touch engine internals directly.
package app

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/engine"
	"github.com/snzpet/gospan/internal/expansion"
	"github.com/snzpet/gospan/internal/logging"
	"github.com/snzpet/gospan/internal/platform"
)

// Service is the application service façade.
type Service struct {
	cfgPath  string
	logger   *slog.Logger
	pipeline *expansion.Pipeline

	mu            sync.RWMutex
	cfg           *config.Config
	engine        *engine.Engine
	lastReloadErr error

	// Platform integration, set while capture is running.
	platformMu sync.Mutex
	platform   platform.Platform

	// Lifecycle state for background capture. Guarded by lifecycleMu.
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	running     bool
}

// Open loads the configuration at cfgPath and builds a ready Service. A nil
// logger is created from the configured log level.
func Open(cfgPath string, logger *slog.Logger) (*Service, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	eng, err := buildEngine(cfg)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = logging.New(cfg.App.LogLevel, os.Stderr)
	}

	return &Service{
		cfgPath:  cfg.App.ConfigPath,
		logger:   logger,
		pipeline: expansion.NewPipeline(),
		cfg:      cfg,
		engine:   eng,
	}, nil
}

func buildEngine(cfg *config.Config) (*engine.Engine, error) {
	return engine.New(engine.Config{
		CaseSensitive: cfg.Engine.CaseSensitive,
		WordBoundary:  cfg.Engine.WordBoundary,
		EscapeChar:    cfg.Engine.EscapeChar,
		BufferSize:    cfg.Input.BufferSize,
	}, cfg.Matches)
}

// Config returns the active configuration. Callers must treat it as read-only.
func (s *Service) Config() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Platform returns the platform integration while capture is running.
func (s *Service) Platform() (platform.Platform, bool) {
	s.platformMu.Lock()
	defer s.platformMu.Unlock()
	if s.platform == nil {
		return nil, false
	}
	return s.platform, true
}

func (s *Service) setPlatform(p platform.Platform) {
	s.platformMu.Lock()
	s.platform = p
	s.platformMu.Unlock()
}

// RenderTemplate renders a template string (expanding variables).
func (s *Service) RenderTemplate(tmpl string) (string, error) {
	return s.pipeline.Render(tmpl)
}

// TestMatch reports whether input ends with a trigger and, if so, returns the
// match with its replacement rendered (plain text).
func (s *Service) TestMatch(input, processName string) (engine.Match, string, bool, error) {
	m, snippet, matched, err := s.TestSnippet(input, processName)
	if err != nil {
		return m, "", matched, err
	}
	if !matched {
		return m, "", false, nil
	}
	return m, snippet.Text, true, nil
}

// TestSnippet is like TestMatch but also returns snippet metadata (cursor
// offset and tab stops).
func (s *Service) TestSnippet(input, processName string) (engine.Match, expansion.SnippetResult, bool, error) {
	s.mu.RLock()
	eng := s.engine
	s.mu.RUnlock()

	ctx := engine.MatchContext{ProcessName: processName}
	m, ok := eng.MatchInputWithContext(input, ctx)
	if !ok {
		return engine.Match{}, expansion.SnippetResult{}, false, nil
	}
	template := engine.ExpandRegexReplacement(m)
	snippet, err := s.pipeline.RenderSnippet(template)
	if err != nil {
		return m, expansion.SnippetResult{}, true, err
	}
	return m, snippet, true, nil
}

// Reload re-reads the configuration and swaps it in atomically. On failure the
// previously active configuration is preserved and the error is remembered so
// callers (CLI, GUI) can report it.
func (s *Service) Reload() error {
	cfg, err := config.Load(s.cfgPath)
	if err != nil {
		s.mu.Lock()
		s.lastReloadErr = err
		s.mu.Unlock()
		return err
	}
	eng, err := buildEngine(cfg)
	if err != nil {
		s.mu.Lock()
		s.lastReloadErr = err
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.cfg = cfg
	s.engine = eng
	s.lastReloadErr = nil
	s.mu.Unlock()

	s.logger.Info("configuration reloaded", "triggers", eng.TriggerCount(), "path", s.cfgPath)
	return nil
}

// ConfigLoaded reports whether an active configuration is present.
func (s *Service) ConfigLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg != nil
}

// LastReloadError returns the error from the most recent reload attempt, or nil
// if the last attempt succeeded (or no reload has been attempted).
func (s *Service) LastReloadError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastReloadErr
}
