package api

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Gameserver status values the wait helpers care about.
const (
	StatusStarted    = "started"
	StatusStopped    = "stopped"
	StatusRestarting = "restarting"
	StatusStopping   = "stopping"
)

// TimeoutError is returned by WaitForStatus when the deadline passes before
// the server reaches the wanted state.
type TimeoutError struct {
	Waited     time.Duration
	LastStatus string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("gave up after %s — last seen status was %q", e.Waited.Round(time.Second), e.LastStatus)
}

// WaitOptions controls WaitForStatus.
type WaitOptions struct {
	// Want reports whether a status is the one we're waiting for.
	Want func(status string) bool
	// LeaveFirst, if set, requires the server to be observed in some other
	// status before Want is honored when the status equals LeaveFirst. This
	// stops a restart from "completing" instantly because the status still
	// reads "started" right after the request. If the server never leaves
	// LeaveFirst within GracePeriod, the restart is assumed to have finished
	// faster than we could observe and Want is accepted.
	LeaveFirst  string
	GracePeriod time.Duration
	Interval    time.Duration // default 3s
	Timeout     time.Duration // default 5m
	// OnPoll is called with every successful poll (for progress display).
	OnPoll func(*GameServer)
}

// StatusWatch is the pure decision logic behind WaitForStatus, usable from
// event loops (like the TUI) that do their own polling.
type StatusWatch struct {
	opts  WaitOptions
	start time.Time
	left  bool
}

// NewStatusWatch begins watching at time now. Defaults are filled in for
// Timeout (5m); Want is required.
func NewStatusWatch(opts WaitOptions, now time.Time) *StatusWatch {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	return &StatusWatch{opts: opts, start: now, left: opts.LeaveFirst == ""}
}

// Observe records a polled status and reports whether the wait is over.
func (w *StatusWatch) Observe(status string, now time.Time) bool {
	if status != w.opts.LeaveFirst {
		w.left = true
	}
	if !w.left && w.opts.GracePeriod > 0 && now.Sub(w.start) >= w.opts.GracePeriod {
		w.left = true
	}
	return w.left && w.opts.Want(status)
}

// TimedOut reports whether the deadline has passed.
func (w *StatusWatch) TimedOut(now time.Time) bool {
	return now.Sub(w.start) > w.opts.Timeout
}

// Elapsed is how long the watch has been running.
func (w *StatusWatch) Elapsed(now time.Time) time.Duration { return now.Sub(w.start) }

// WaitForStatus polls fetch until opts.Want accepts the status, the timeout
// elapses (*TimeoutError), or ctx is cancelled. Transient fetch errors are
// tolerated; the last one is reported if the deadline passes.
func WaitForStatus(ctx context.Context, fetch func(context.Context) (*GameServer, error), opts WaitOptions) (*GameServer, error) {
	if opts.Interval <= 0 {
		opts.Interval = 3 * time.Second
	}
	if opts.Want == nil {
		return nil, errors.New("WaitForStatus: Want is required")
	}

	watch := NewStatusWatch(opts, time.Now())
	var last *GameServer
	var lastErr error

	for {
		gs, err := fetch(ctx)
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
			last = gs
			if opts.OnPoll != nil {
				opts.OnPoll(gs)
			}
			if watch.Observe(gs.Status, time.Now()) {
				return gs, nil
			}
		}

		if watch.TimedOut(time.Now()) {
			te := &TimeoutError{Waited: watch.Elapsed(time.Now())}
			if last != nil {
				te.LastStatus = last.Status
			}
			if lastErr != nil {
				return last, fmt.Errorf("%w (last error: %v)", te, lastErr)
			}
			return last, te
		}

		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
}
