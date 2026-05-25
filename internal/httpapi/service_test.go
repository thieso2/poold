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

func TestManualControlModeSkipsEnforcementUntilExpiryWake(t *testing.T) {
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
	manualDuration := time.Hour
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{ManualControlDuration: manualDuration})
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
	mode, err := service.ControlMode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !mode.ManualControl || mode.ExpiresAt == nil {
		t.Fatalf("mode = %+v, want expiring manual control", mode)
	}

	if err := service.Enforce(ctx, fake.status); err != nil {
		t.Fatal(err)
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("set calls = %d, want none while manual control is on", got)
	}
	if wake, ok, err := service.NextScheduleWake(ctx, time.Now(), fake.status); err != nil || !ok || !wake.Equal(*mode.ExpiresAt) {
		t.Fatalf("wake = %v ok=%v err=%v, want manual expiry wake %v", wake, ok, err, mode.ExpiresAt)
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

func TestExpiredManualControlModeResumesEnforcement(t *testing.T) {
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
			Filter:     false,
			Heater:     false,
			TargetTemp: 36,
		},
	}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{})
	if err := st.SaveDesiredState(ctx, pool.DesiredState{Filter: pool.BoolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	expiredAt := time.Now().Add(-time.Minute).UTC()
	if err := st.SaveControlMode(ctx, pool.ControlMode{ManualControl: true, ExpiresAt: &expiredAt}); err != nil {
		t.Fatal(err)
	}

	if err := service.Enforce(ctx, fake.status); err != nil {
		t.Fatal(err)
	}
	if got := fake.callCount(); got != 1 {
		t.Fatalf("set calls = %d, want enforcement after manual control expiry", got)
	}
	mode, err := service.ControlMode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mode.ManualControl || mode.ExpiresAt != nil {
		t.Fatalf("mode = %+v, want automatic control after expiry", mode)
	}
}

func TestManualControlDefaultExpiryDoesNotPassNextScheduleWake(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	loc := time.Local
	startAt := time.Now().In(loc).Add(10 * time.Minute).Truncate(time.Minute)
	if !startAt.After(time.Now().In(loc)) {
		startAt = startAt.Add(time.Minute)
	}
	endAt := startAt.Add(time.Hour)
	status := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     false,
		TargetTemp: 36,
	}
	if _, err := st.SaveObservation(ctx, status); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePlans(ctx, []pool.Plan{{
		ID:         "daily-filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       startAt.Format("15:04"),
		To:         endAt.Format("15:04"),
	}}); err != nil {
		t.Fatal(err)
	}
	service := NewService(st, &fakePoolClient{status: status}, scheduler.New(scheduler.Config{Location: loc}), ServiceConfig{ManualControlDuration: time.Hour})

	if err := service.SaveControlMode(ctx, pool.ControlMode{ManualControl: true}); err != nil {
		t.Fatal(err)
	}
	mode, err := service.ControlMode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mode.ExpiresAt == nil {
		t.Fatalf("mode = %+v, want expiry", mode)
	}
	if delta := mode.ExpiresAt.Sub(startAt); delta < -time.Second || delta > time.Second {
		t.Fatalf("expires_at = %v, want next schedule wake %v", mode.ExpiresAt, startAt)
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
