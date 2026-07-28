package httpapi

import (
	"context"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

func TestRefreshStatusDedupesObservationEvents(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	current := 30
	fake := &fakePoolClient{
		status: pool.Status{
			ObservedAt:  time.Now().UTC(),
			Connected:   true,
			Power:       true,
			CurrentTemp: &current,
			TargetTemp:  36,
			Unit:        "C",
		},
	}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{EventHeartbeat: time.Hour})

	if _, err := service.RefreshStatus(ctx); err != nil {
		t.Fatal(err)
	}
	if got := eventCount(t, st); got != 1 {
		t.Fatalf("events = %d, want initial observation", got)
	}

	if _, err := service.RefreshStatus(ctx); err != nil {
		t.Fatal(err)
	}
	if got := eventCount(t, st); got != 1 {
		t.Fatalf("events = %d, want unchanged status suppressed", got)
	}

	current = 31
	if _, err := service.RefreshStatus(ctx); err != nil {
		t.Fatal(err)
	}
	if got := eventCount(t, st); got != 2 {
		t.Fatalf("events = %d, want changed temperature event", got)
	}

	service.statusEventMu.Lock()
	service.lastStatusEventAt = time.Now().Add(-2 * time.Hour)
	service.statusEventMu.Unlock()
	if _, err := service.RefreshStatus(ctx); err != nil {
		t.Fatal(err)
	}
	events, err := st.Events(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[2].Message != "status heartbeat" {
		t.Fatalf("events = %+v, want heartbeat event", events)
	}
}

func eventCount(t *testing.T, st *store.Store) int {
	t.Helper()
	events, err := st.Events(context.Background(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	return len(events)
}
