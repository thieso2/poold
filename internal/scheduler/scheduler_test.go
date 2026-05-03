package scheduler

import (
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
)

func TestDailyFilterWindow(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	plan := pool.Plan{
		ID:         "filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "02:00",
		To:         "04:00",
	}

	on := s.Evaluate(at(loc, 2026, 5, 4, 2, 30), pool.Status{}, pool.DesiredState{}, []pool.Plan{plan})
	if on.Desired.Filter == nil || !*on.Desired.Filter {
		t.Fatalf("filter should be on inside window: %+v", on)
	}

	off := s.Evaluate(at(loc, 2026, 5, 4, 4, 0), pool.Status{}, pool.DesiredState{}, []pool.Plan{plan})
	if off.Desired.Filter == nil || *off.Desired.Filter {
		t.Fatalf("filter should be off outside window: %+v", off)
	}
}

func TestReadyByHeatingStartCalculation(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	current := 30
	target := 36
	readyAt := at(loc, 2026, 5, 9, 8, 30)
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: &target,
		At:         &readyAt,
	}
	status := pool.Status{CurrentTemp: &current, TargetTemp: 30}

	before := s.Evaluate(at(loc, 2026, 5, 9, 1, 59), status, pool.DesiredState{}, []pool.Plan{plan})
	if before.Source != "default" {
		t.Fatalf("before start source = %q, want default", before.Source)
	}

	active := s.Evaluate(at(loc, 2026, 5, 9, 2, 0), status, pool.DesiredState{}, []pool.Plan{plan})
	if active.Desired.Heater == nil || !*active.Desired.Heater || active.Desired.Filter == nil || !*active.Desired.Filter {
		t.Fatalf("heater/filter should be on at calculated start: %+v", active)
	}
	if active.Desired.TargetTemp == nil || *active.Desired.TargetTemp != 36 {
		t.Fatalf("target temp = %+v, want 36", active.Desired.TargetTemp)
	}
}

func TestManualOverridePrecedence(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	now := at(loc, 2026, 5, 4, 2, 30)
	target := 36
	readyAt := now.Add(time.Hour)
	plans := []pool.Plan{
		{
			ID:         "ready",
			Type:       pool.PlanReadyBy,
			Enabled:    true,
			TargetTemp: &target,
			At:         &readyAt,
		},
		{
			ID:           "override",
			Type:         pool.PlanManualOverride,
			Enabled:      true,
			DesiredState: pool.DesiredState{Heater: pool.BoolPtr(false), Filter: pool.BoolPtr(false)},
			ExpiresAt:    ptrTime(now.Add(time.Hour)),
		},
	}
	current := 30
	eval := s.Evaluate(now, pool.Status{CurrentTemp: &current}, pool.DesiredState{}, plans)
	if eval.Source != "override" {
		t.Fatalf("source = %q, want override", eval.Source)
	}
	if eval.Desired.Heater == nil || *eval.Desired.Heater {
		t.Fatalf("manual override should keep heater off: %+v", eval)
	}
}

func TestExpiredOverrideFallsThrough(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	now := at(loc, 2026, 5, 4, 2, 30)
	plan := pool.Plan{
		ID:           "override",
		Type:         pool.PlanManualOverride,
		Enabled:      true,
		DesiredState: pool.DesiredState{Heater: pool.BoolPtr(true)},
		ExpiresAt:    ptrTime(now.Add(-time.Minute)),
	}
	eval := s.Evaluate(now, pool.Status{}, pool.DesiredState{Filter: pool.BoolPtr(false)}, []pool.Plan{plan})
	if eval.Source != "default" {
		t.Fatalf("source = %q, want default", eval.Source)
	}
	if eval.Desired.Filter == nil || *eval.Desired.Filter {
		t.Fatalf("default desired state not used: %+v", eval)
	}
}

func fixedZone() *time.Location {
	return time.FixedZone("TEST", 2*60*60)
}

func at(loc *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
