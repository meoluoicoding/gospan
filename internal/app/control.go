package app

import (
	"context"
	"errors"
	"time"
)

// ErrAlreadyRunning is returned by Start when capture is already active.
var ErrAlreadyRunning = errors.New("gospan is already running")

// startupTimeout bounds how long Start waits for the capture mechanism to
// report readiness before returning control to the caller.
const startupTimeout = 5 * time.Second

// stopTimeout bounds how long Stop waits for the capture loop to exit.
const stopTimeout = 3 * time.Second

// Start begins keyboard capture and expansion in the background. It returns
// once capture is established (or has failed), so a GUI can toggle it without
// blocking on the long-running loop.
func (s *Service) Start() error {
	s.lifecycleMu.Lock()
	if s.running {
		s.lifecycleMu.Unlock()
		return ErrAlreadyRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan error, 1)
	done := make(chan struct{})

	s.cancel = cancel
	s.done = done
	s.running = true
	s.lifecycleMu.Unlock()

	go func() {
		_ = s.run(ctx, ready)
		close(done)
	}()

	select {
	case err := <-ready:
		if err != nil {
			s.markStopped()
			return err
		}
		return nil
	case <-time.After(startupTimeout):
		// Still starting; assume it will report asynchronously.
		return nil
	}
}

// Stop cancels capture and waits for the loop to exit.
func (s *Service) Stop() error {
	s.lifecycleMu.Lock()
	running := s.running
	cancel := s.cancel
	done := s.done
	s.lifecycleMu.Unlock()

	if !running {
		return nil
	}
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(stopTimeout):
		}
	}
	s.markStopped()
	return nil
}

// Running reports whether capture is active.
func (s *Service) Running() bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return s.running
}

func (s *Service) markStopped() {
	s.lifecycleMu.Lock()
	s.running = false
	s.cancel = nil
	s.done = nil
	s.lifecycleMu.Unlock()
}
