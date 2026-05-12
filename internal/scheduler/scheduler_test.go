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

	outside := s.Evaluate(at(loc, 2026, 5, 4, 4, 0), pool.Status{}, pool.DesiredState{}, []pool.Plan{plan})
	if outside.Desired.Filter != nil {
		t.Fatalf("filter should be unmanaged outside window: %+v", outside)
	}
}

func TestEveningFilterWindowOverridesOffBaseDesiredState(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	plan := pool.Plan{
		ID:         "evening",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "18:00",
		To:         "21:00",
		Days:       []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
	}
	base := pool.DesiredState{
		Power:      pool.BoolPtr(false),
		Filter:     pool.BoolPtr(false),
		Heater:     pool.BoolPtr(false),
		Jets:       pool.BoolPtr(false),
		Bubbles:    pool.BoolPtr(false),
		Sanitizer:  pool.BoolPtr(false),
		TargetTemp: pool.IntPtr(36),
	}

	eval := s.Evaluate(at(loc, 2026, 5, 4, 18, 10), pool.Status{}, base, []pool.Plan{plan})
	if eval.Source != "time_window" || eval.Desired.Power == nil || !*eval.Desired.Power || eval.Desired.Filter == nil || !*eval.Desired.Filter {
		t.Fatalf("evening filter window should turn power/filter on: %+v", eval)
	}
}

func TestDesiredHeaterWinsOverInactiveFilterWindow(t *testing.T) {
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

	eval := s.Evaluate(
		at(loc, 2026, 5, 4, 4, 0),
		pool.Status{},
		pool.DesiredState{Heater: pool.BoolPtr(true)},
		[]pool.Plan{plan},
	)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.Desired.Filter == nil || !*eval.Desired.Filter || eval.Desired.Power == nil || !*eval.Desired.Power {
		t.Fatalf("heater desired on should imply filter/power even outside filter window: %+v", eval)
	}
}

func TestOvernightWindowUsesStartDay(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	plan := pool.Plan{
		ID:         "filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "23:00",
		To:         "01:00",
		Days:       []string{"mon"},
	}

	on := s.Evaluate(at(loc, 2026, 5, 5, 0, 30), pool.Status{}, pool.DesiredState{}, []pool.Plan{plan})
	if on.Desired.Filter == nil || !*on.Desired.Filter {
		t.Fatalf("filter should stay on after midnight for Monday window: %+v", on)
	}

	outside := s.Evaluate(at(loc, 2026, 5, 6, 0, 30), pool.Status{}, pool.DesiredState{}, []pool.Plan{plan})
	if outside.Desired.Filter != nil {
		t.Fatalf("filter should be unmanaged when previous day was not included: %+v", outside)
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

func TestRecurringReadyByCronHeatingStartCalculation(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	current := 30
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		Cron:       "30 8 * * 6",
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

	after := s.Evaluate(at(loc, 2026, 5, 9, 8, 31), status, pool.DesiredState{}, []pool.Plan{plan})
	if after.Source != "default" {
		t.Fatalf("after ready minute source = %q, want default", after.Source)
	}
}

func TestRecurringReadyByCronKeepsHeatingAfterStartTimeRecalculates(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		Cron:       "30 8 * * 6",
	}

	current := 30
	active := s.Evaluate(at(loc, 2026, 5, 9, 2, 0), pool.Status{CurrentTemp: &current}, pool.DesiredState{}, []pool.Plan{plan})
	if active.Desired.Heater == nil || !*active.Desired.Heater {
		t.Fatalf("heater should be on at initial calculated start: %+v", active)
	}

	current = 31
	sticky := s.Evaluate(at(loc, 2026, 5, 9, 2, 5), pool.Status{CurrentTemp: &current, Heater: true}, pool.DesiredState{}, []pool.Plan{plan})
	if sticky.Desired.Heater == nil || !*sticky.Desired.Heater || sticky.Source != "ready" {
		t.Fatalf("ready-by plan should keep heating after recalculated start moves later: %+v", sticky)
	}
}

func TestReadyByControlHysteresisBreaksTargetBoundaryLoop(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute, ReadyByReheatDelta: 2})
	readyAt := at(loc, 2026, 5, 9, 8, 30)
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		At:         &readyAt,
	}
	now := at(loc, 2026, 5, 9, 7, 0)
	states := map[string]pool.ReadyByControlState{}

	current := 35
	eval := s.EvaluateWithReadyByControl(now, pool.Status{CurrentTemp: &current, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyBySeekingTarget {
		t.Fatalf("first below-target observation should seek target: %+v", eval)
	}
	states[eval.ReadyByControl.Current.Key] = eval.ReadyByControl.Current

	current = 36
	eval = s.EvaluateWithReadyByControl(now.Add(20*time.Minute), pool.Status{CurrentTemp: &current, Heater: true, Filter: true, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || *eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyBySatisfiedIdle {
		t.Fatalf("target reached should stop heating and enter satisfied idle: %+v", eval)
	}
	states[eval.ReadyByControl.Current.Key] = eval.ReadyByControl.Current

	current = 35
	eval = s.EvaluateWithReadyByControl(now.Add(21*time.Minute), pool.Status{CurrentTemp: &current, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || *eval.Desired.Heater || eval.ReadyByControl != nil {
		t.Fatalf("target-1 after satisfaction should remain idle without a state transition: %+v", eval)
	}

	current = 34
	eval = s.EvaluateWithReadyByControl(now.Add(22*time.Minute), pool.Status{CurrentTemp: &current, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyByReheating {
		t.Fatalf("target-2 should enter reheating: %+v", eval)
	}
	states[eval.ReadyByControl.Current.Key] = eval.ReadyByControl.Current

	current = 35
	eval = s.EvaluateWithReadyByControl(now.Add(23*time.Minute), pool.Status{CurrentTemp: &current, Heater: true, Filter: true, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.ReadyByControl != nil {
		t.Fatalf("reheating should continue through target-1: %+v", eval)
	}

	current = 36
	eval = s.EvaluateWithReadyByControl(now.Add(24*time.Minute), pool.Status{CurrentTemp: &current, Heater: true, Filter: true, TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, states)
	if eval.Desired.Heater == nil || *eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyBySatisfiedIdle {
		t.Fatalf("reheating should stop at target: %+v", eval)
	}
}

func TestReadyBySatisfiedIdleStillAllowsTimeWindowDemand(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	readyAt := at(loc, 2026, 5, 9, 8, 30)
	ready := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		At:         &readyAt,
	}
	window := pool.Plan{
		ID:         "filter-window",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "07:00",
		To:         "08:00",
	}
	key := ReadyByOccurrenceKey("ready", readyAt, 36)
	states := map[string]pool.ReadyByControlState{
		key: {
			Key:        key,
			PlanID:     "ready",
			ReadyAt:    readyAt,
			TargetTemp: 36,
			Mode:       pool.ReadyBySatisfiedIdle,
		},
	}
	current := 35
	eval := s.EvaluateWithReadyByControl(
		at(loc, 2026, 5, 9, 7, 30),
		pool.Status{CurrentTemp: &current, TargetTemp: 36},
		pool.DesiredState{Filter: pool.BoolPtr(false), Heater: pool.BoolPtr(false)},
		[]pool.Plan{ready, window},
		states,
	)
	if eval.Source != "time_window" || eval.Desired.Filter == nil || !*eval.Desired.Filter || eval.Desired.Heater == nil || *eval.Desired.Heater || eval.Desired.TargetTemp == nil || *eval.Desired.TargetTemp != 36 {
		t.Fatalf("satisfied ready-by target should combine with active time window: %+v", eval)
	}
}

func TestReadyByNilTemperatureDoesNotCreateSatisfaction(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	readyAt := at(loc, 2026, 5, 9, 8, 30)
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		At:         &readyAt,
	}

	eval := s.EvaluateWithReadyByControl(at(loc, 2026, 5, 9, 8, 0), pool.Status{TargetTemp: 36}, pool.DesiredState{}, []pool.Plan{plan}, nil)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyBySeekingTarget {
		t.Fatalf("nil temperature should seek target, not become satisfied: %+v", eval)
	}
}

func TestReadyByControlStateIsPerOccurrence(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		Cron:       "30 8 * * *",
	}
	previousReadyAt := at(loc, 2026, 5, 9, 8, 30)
	previousKey := ReadyByOccurrenceKey("ready", previousReadyAt, 36)
	states := map[string]pool.ReadyByControlState{
		previousKey: {
			Key:        previousKey,
			PlanID:     "ready",
			ReadyAt:    previousReadyAt,
			TargetTemp: 36,
			Mode:       pool.ReadyBySatisfiedIdle,
		},
	}
	current := 35
	eval := s.EvaluateWithReadyByControl(
		at(loc, 2026, 5, 10, 7, 0),
		pool.Status{CurrentTemp: &current, TargetTemp: 36},
		pool.DesiredState{},
		[]pool.Plan{plan},
		states,
	)
	if eval.Desired.Heater == nil || !*eval.Desired.Heater || eval.ReadyByControl == nil || eval.ReadyByControl.Current.Mode != pool.ReadyBySeekingTarget {
		t.Fatalf("previous occurrence state should not satisfy a new occurrence: %+v", eval)
	}
	if eval.ReadyByControl.Current.Key == previousKey {
		t.Fatalf("new occurrence reused previous key %q", previousKey)
	}
}

func TestRecurringReadyByCronDoesNotBlockBeforeStartWhenAlreadyWarm(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	current := 37
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		Cron:       "30 8 * * *",
	}

	eval := s.Evaluate(at(loc, 2026, 5, 8, 10, 0), pool.Status{CurrentTemp: &current}, pool.DesiredState{}, []pool.Plan{plan})
	if eval.Source != "default" {
		t.Fatalf("warm recurring plan before start source = %q, want default", eval.Source)
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

func TestManualPowerOffOverridesBaseHeater(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	now := at(loc, 2026, 5, 4, 2, 30)
	plan := pool.Plan{
		ID:           "manual",
		Type:         pool.PlanManualOverride,
		Enabled:      true,
		DesiredState: pool.DesiredState{Power: pool.BoolPtr(false)},
		ExpiresAt:    ptrTime(now.Add(time.Hour)),
	}

	eval := s.Evaluate(now, pool.Status{}, pool.DesiredState{Heater: pool.BoolPtr(true)}, []pool.Plan{plan})
	if eval.Desired.Power == nil || *eval.Desired.Power || eval.Desired.Heater == nil || *eval.Desired.Heater {
		t.Fatalf("power-off manual override should stop heater: %+v", eval)
	}
}

func TestManualFilterOffOverridesBaseHeater(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	now := at(loc, 2026, 5, 4, 2, 30)
	plan := pool.Plan{
		ID:           "manual",
		Type:         pool.PlanManualOverride,
		Enabled:      true,
		DesiredState: pool.DesiredState{Filter: pool.BoolPtr(false)},
		ExpiresAt:    ptrTime(now.Add(time.Hour)),
	}

	eval := s.Evaluate(now, pool.Status{}, pool.DesiredState{Heater: pool.BoolPtr(true)}, []pool.Plan{plan})
	if eval.Desired.Filter == nil || *eval.Desired.Filter || eval.Desired.Heater == nil || *eval.Desired.Heater {
		t.Fatalf("filter-off manual override should stop heater: %+v", eval)
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

func TestNextWakeTimeWindow(t *testing.T) {
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

	wake, ok := s.NextWake(at(loc, 2026, 5, 4, 1, 0), pool.Status{}, []pool.Plan{plan})
	if !ok || !wake.Equal(at(loc, 2026, 5, 4, 2, 0)) {
		t.Fatalf("wake = %s ok=%v, want 02:00", wake, ok)
	}

	wake, ok = s.NextWake(at(loc, 2026, 5, 4, 2, 30), pool.Status{}, []pool.Plan{plan})
	if !ok || !wake.Equal(at(loc, 2026, 5, 4, 4, 0)) {
		t.Fatalf("wake = %s ok=%v, want 04:00", wake, ok)
	}
}

func TestNextWakeReadyBy(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 0.75, ReadinessBuffer: 30 * time.Minute})
	current := 30
	readyAt := at(loc, 2026, 5, 9, 8, 30)
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		At:         &readyAt,
	}

	wake, ok := s.NextWake(at(loc, 2026, 5, 8, 23, 0), pool.Status{CurrentTemp: &current}, []pool.Plan{plan})
	if !ok || !wake.Equal(at(loc, 2026, 5, 9, 0, 0)) {
		t.Fatalf("wake = %s ok=%v, want heating start", wake, ok)
	}
}

func TestNextWakeRecurringReadyBy(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc, HeatingRateCPerHour: 1, ReadinessBuffer: 30 * time.Minute})
	current := 30
	plan := pool.Plan{
		ID:         "ready",
		Type:       pool.PlanReadyBy,
		Enabled:    true,
		TargetTemp: pool.IntPtr(36),
		Cron:       "30 8 * * 6",
	}

	wake, ok := s.NextWake(at(loc, 2026, 5, 8, 23, 0), pool.Status{CurrentTemp: &current}, []pool.Plan{plan})
	if !ok || !wake.Equal(at(loc, 2026, 5, 9, 2, 0)) {
		t.Fatalf("wake = %s ok=%v, want recurring heating start", wake, ok)
	}
}

func TestNextWakeManualOverrideExpiry(t *testing.T) {
	loc := fixedZone()
	s := New(Config{Location: loc})
	expiresAt := at(loc, 2026, 5, 3, 12, 30)
	plan := pool.Plan{
		ID:        "override",
		Type:      pool.PlanManualOverride,
		Enabled:   true,
		ExpiresAt: &expiresAt,
	}

	wake, ok := s.NextWake(at(loc, 2026, 5, 3, 12, 0), pool.Status{}, []pool.Plan{plan})
	if !ok || !wake.Equal(expiresAt) {
		t.Fatalf("wake = %s ok=%v, want override expiry", wake, ok)
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
