package api

import (
	"context"
	"errors"
	"testing"
	"time"
)

func seqFetch(statuses ...string) func(context.Context) (*GameServer, error) {
	i := 0
	return func(context.Context) (*GameServer, error) {
		s := statuses[len(statuses)-1]
		if i < len(statuses) {
			s = statuses[i]
		}
		i++
		return &GameServer{Status: s}, nil
	}
}

func TestWaitForStatus_RestartWaitsForLeaveThenReturn(t *testing.T) {
	polls := 0
	gs, err := WaitForStatus(context.Background(), seqFetch("started", "restarting", "restarting", "started"), WaitOptions{
		Want:       func(s string) bool { return s == StatusStarted },
		LeaveFirst: StatusStarted,
		Interval:   time.Millisecond,
		OnPoll:     func(*GameServer) { polls++ },
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if gs.Status != StatusStarted || polls != 4 {
		t.Errorf("status=%q polls=%d, want started after 4 polls", gs.Status, polls)
	}
}

func TestWaitForStatus_NeverLeavesAcceptedAfterGrace(t *testing.T) {
	gs, err := WaitForStatus(context.Background(), seqFetch("started"), WaitOptions{
		Want:        func(s string) bool { return s == StatusStarted },
		LeaveFirst:  StatusStarted,
		GracePeriod: 5 * time.Millisecond,
		Interval:    time.Millisecond,
	})
	if err != nil || gs.Status != StatusStarted {
		t.Fatalf("gs=%v err=%v, want started accepted after grace", gs, err)
	}
}

func TestWaitForStatus_Stop(t *testing.T) {
	gs, err := WaitForStatus(context.Background(), seqFetch("started", "stopping", "stopped"), WaitOptions{
		Want:     func(s string) bool { return s == StatusStopped },
		Interval: time.Millisecond,
	})
	if err != nil || gs.Status != StatusStopped {
		t.Fatalf("gs=%v err=%v", gs, err)
	}
}

func TestWaitForStatus_Timeout(t *testing.T) {
	_, err := WaitForStatus(context.Background(), seqFetch("restarting"), WaitOptions{
		Want:     func(s string) bool { return s == StatusStarted },
		Interval: time.Millisecond,
		Timeout:  10 * time.Millisecond,
	})
	var te *TimeoutError
	if !errors.As(err, &te) || te.LastStatus != "restarting" {
		t.Fatalf("err = %v, want TimeoutError with last status restarting", err)
	}
}

func TestWaitForStatus_ToleratesTransientErrors(t *testing.T) {
	calls := 0
	fetch := func(context.Context) (*GameServer, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("blip")
		}
		return &GameServer{Status: StatusStopped}, nil
	}
	gs, err := WaitForStatus(context.Background(), fetch, WaitOptions{
		Want: func(s string) bool { return s == StatusStopped }, Interval: time.Millisecond,
	})
	if err != nil || gs.Status != StatusStopped {
		t.Fatalf("gs=%v err=%v", gs, err)
	}
}

func TestWaitForStatus_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := WaitForStatus(ctx, seqFetch("restarting"), WaitOptions{
		Want: func(string) bool { return false }, Interval: time.Hour,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
