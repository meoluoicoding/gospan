package config

import (
	"context"
	"os"
	"time"
)

// Watcher polls a configuration file for changes and notifies a channel. It is
// deliberately simple and dependency-free; the notification carries no data —
// consumers should reload the file themselves so that reload remains atomic.
type Watcher struct {
	path     string
	interval time.Duration
	changes  chan struct{}
}

// NewWatcher returns a Watcher for path. A non-positive interval selects a
// one-second default.
func NewWatcher(path string, interval time.Duration) *Watcher {
	if interval <= 0 {
		interval = time.Second
	}
	return &Watcher{
		path:     path,
		interval: interval,
		changes:  make(chan struct{}, 1),
	}
}

// Changes returns the channel that receives a signal after the file changes.
func (w *Watcher) Changes() <-chan struct{} {
	return w.changes
}

// Start polls until ctx is cancelled. Each detected modification sends a single
// (coalesced) notification.
func (w *Watcher) Start(ctx context.Context) {
	var lastMod time.Time
	var lastSize int64

	if fi, err := os.Stat(w.path); err == nil {
		lastMod = fi.ModTime()
		lastSize = fi.Size()
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(w.changes)
			return
		case <-ticker.C:
			fi, err := os.Stat(w.path)
			if err != nil {
				continue
			}
			if !fi.ModTime().Equal(lastMod) || fi.Size() != lastSize {
				lastMod = fi.ModTime()
				lastSize = fi.Size()

				select {
				case w.changes <- struct{}{}:
				default: // a notification is already pending
				}
			}
		}
	}
}
