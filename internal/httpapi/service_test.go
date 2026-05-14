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

func TestManualControlModeSkipsEnforcementAndScheduleWake(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	fake := &fakePoolClient{
		status: pool.Status{
			ObservedAt: time.Now().UTC(),
			Connected:  true,
			Power:      true,
			Filter:     true,
			Heater:     false,
			TargetTemp: 36,
		},
	}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{})
	if err := st.SaveDesiredState(ctx, pool.DesiredState{Heater: pool.BoolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePlans(ctx, []pool.Plan{{
		ID:         "daily-filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "02:00",
		To:         "04:00",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := service.SaveControlMode(ctx, pool.ControlMode{ManualControl: true}); err != nil {
		t.Fatal(err)
	}

	if err := service.Enforce(ctx, fake.status); err != nil {
		t.Fatal(err)
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("set calls = %d, want none while manual control is on", got)
	}
	if wake, ok, err := service.NextScheduleWake(ctx, time.Now(), fake.status); err != nil || ok {
		t.Fatalf("wake = %v ok=%v err=%v, want no schedule wake while manual control is on", wake, ok, err)
	}

	if err := service.SaveControlMode(ctx, pool.ControlMode{ManualControl: false}); err != nil {
		t.Fatal(err)
	}
	if err := service.Enforce(ctx, fake.status); err != nil {
		t.Fatal(err)
	}
	if got := fake.callCount(); got != 1 {
		t.Fatalf("set calls = %d, want enforcement after automatic control resumes", got)
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
