package store

import (
	"context"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
)

func TestStoreObservationDesiredPlansAndEvents(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	current := 32
	status := pool.Status{
		ObservedAt:  time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
		Connected:   true,
		Power:       true,
		Filter:      true,
		CurrentTemp: &current,
		TargetTemp:  36,
	}
	if _, err := st.SaveObservation(ctx, status); err != nil {
		t.Fatal(err)
	}
	latest, ok, err := st.LatestStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || latest.CurrentTemp == nil || *latest.CurrentTemp != 32 {
		t.Fatalf("latest = %+v, ok=%v", latest, ok)
	}

	desired := pool.DesiredState{Heater: pool.BoolPtr(true)}
	if err := st.SaveDesiredState(ctx, desired); err != nil {
		t.Fatal(err)
	}
	savedDesired, err := st.DesiredState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if savedDesired.Filter == nil || !*savedDesired.Filter || savedDesired.Power == nil || !*savedDesired.Power {
		t.Fatalf("hardware constraints were not applied: %+v", savedDesired)
	}

	event, err := st.AddEvent(ctx, "test", "hello", map[string]string{"ok": "true"})
	if err != nil {
		t.Fatal(err)
	}
	events, err := st.Events(ctx, event.ID-1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "test" {
		t.Fatalf("events = %+v", events)
	}

	plan := pool.Plan{
		ID:         "daily-filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "02:00",
		To:         "04:00",
	}
	if err := st.SavePlans(ctx, []pool.Plan{plan}); err != nil {
		t.Fatal(err)
	}
	plans, err := st.Plans(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].ID != "daily-filter" || plans[0].CreatedAt.IsZero() || plans[0].UpdatedAt.IsZero() {
		t.Fatalf("plans = %+v", plans)
	}
}

func TestStoreCommandAndRetention(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	id, err := st.InsertCommand(ctx, pool.CommandRecord{Capability: "filter", State: pool.BoolPtr(true), Source: "test"})
	if err != nil {
		t.Fatal(err)
	}
	status := pool.Status{ObservedAt: time.Now().UTC(), Connected: true}
	if err := st.FinishCommand(ctx, id, true, &status, nil); err != nil {
		t.Fatal(err)
	}
	lastCommandAt, err := st.LastCommandAt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lastCommandAt == nil {
		t.Fatal("expected last command time")
	}

	old := pool.Status{ObservedAt: time.Now().Add(-48 * time.Hour).UTC()}
	if _, err := st.SaveObservation(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := st.Prune(ctx, time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if latest, ok, err := st.LatestStatus(ctx); err != nil {
		t.Fatal(err)
	} else if ok && latest.ObservedAt.Equal(old.ObservedAt) {
		t.Fatalf("old observation was not pruned: %+v", latest)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
