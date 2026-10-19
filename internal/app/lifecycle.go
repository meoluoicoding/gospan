package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/engine"
	"github.com/snzpet/gospan/internal/platform"
	"github.com/snzpet/gospan/internal/platform/host"
)

// foregroundNamer is implemented by platform integrations that can name the
// process owning the foreground window. It is optional.
type foregroundNamer interface {
	ForegroundProcessName() (string, error)
}

// Run starts keyboard capture and the expansion loop, blocking until ctx is
// cancelled or the platform stops. Configuration file changes are picked up
// automatically.
func (s *Service) Run(ctx context.Context) error {
	return s.run(ctx, nil)
}

// run is Run with an optional readiness signal. When ready is non-nil it
// receives nil once capture is established, or the startup error.
func (s *Service) run(ctx context.Context, ready chan<- error) error {
	plat, err := host.New()
	if err != nil {
		signalReady(ready, err)
		return err
	}
	if err := plat.Start(); err != nil {
		wrapped := fmt.Errorf("start keyboard capture: %w", err)
		signalReady(ready, wrapped)
		return wrapped
	}
	defer plat.Stop()

	s.setPlatform(plat)
	defer s.setPlatform(nil)

	signalReady(ready, nil)

	namer, _ := plat.(foregroundNamer)

	// Wire the clipboard reader so {{clipboard}} can be resolved.
	cfg := s.Config()
	s.pipeline.SetClipboardReader(plat, cfg.Clipboard.MaxLength, cfg.Clipboard.Enabled)

	watcher := config.NewWatcher(s.cfgPath, time.Second)
	go watcher.Start(ctx)

	s.logger.Info("gospan started", "config", s.cfgPath, "triggers", s.TriggerCount())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-watcher.Changes():
			if err := s.Reload(); err != nil {
				s.logger.Error("configuration reload failed; keeping previous configuration", "error", err)
			}
		case ev, ok := <-plat.Events():
			if !ok {
				return nil
			}
			s.handleEvent(plat, namer, ev)
		}
	}
}

// signalReady sends exactly one value (nil or err) on ready when it is non-nil.
func signalReady(ready chan<- error, err error) {
	if ready == nil {
		return
	}
	ready <- err
}

func (s *Service) handleEvent(plat platform.Platform, namer foregroundNamer, ev platform.KeyboardEvent) {
	if !ev.IsDown || ev.Rune == 0 {
		return
	}
	if s.isExcluded(namer) {
		return
	}

	s.mu.RLock()
	eng := s.engine
	pipeline := s.pipeline
	s.mu.RUnlock()

	processName := ""
	if namer != nil {
		if n, err := namer.ForegroundProcessName(); err == nil {
			processName = n
		}
	}

	res, ok := eng.FeedWithContext(ev.Rune, engine.MatchContext{ProcessName: processName})
	if !ok {
		return
	}

	snippet, err := pipeline.RenderSnippet(res.Replacement)
	if err != nil {
		s.logger.Error("render replacement failed", "error", err)
		return
	}
	replacement := snippet.Text + res.Trailer

	cfg := s.Config()
	strategy := platform.InsertBackspace
	switch cfg.Insertion.Strategy {
	case "clipboard":
		strategy = platform.InsertClipboard
	case "auto":
		if len([]rune(replacement)) >= cfg.Insertion.ClipboardThreshold || strings.Contains(replacement, "\n") {
			strategy = platform.InsertClipboard
		}
	}

	if err := plat.ReplacePreviousText(res.Typed, replacement, strategy); err != nil {
		s.logger.Error("text replacement failed", "error", err)
		return
	}

	// Place the caret for snippet markers ($|$, $0, $1...).
	if snippet.CursorOffset >= 0 && snippet.CursorOffset < len([]rune(snippet.Text)) {
		_ = plat.MoveCursorLeft(len([]rune(snippet.Text)) - snippet.CursorOffset)
	}

	s.logger.Debug("expanded trigger", "trigger", res.Match.Trigger)
}

// isExcluded reports whether the foreground process is on the exclusion list.
// Application exclusion is best-effort, not a security boundary.
func (s *Service) isExcluded(namer foregroundNamer) bool {
	cfg := s.Config()
	if len(cfg.Input.ExcludedProcesses) == 0 || namer == nil {
		return false
	}
	name, err := namer.ForegroundProcessName()
	if err != nil {
		return false
	}
	for _, ex := range cfg.Input.ExcludedProcesses {
		if strings.EqualFold(ex, name) {
			return true
		}
	}
	return false
}
