package scheduler

import (
	"fmt"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

type Config struct {
	HeatingRateCPerHour float64
	ReadinessBuffer     time.Duration
	Location            *time.Location
	ReadyByReheatDelta  int
}

type Evaluation struct {
	Desired        pool.DesiredState     `json:"desired"`
	Source         string                `json:"source"`
	Reason         string                `json:"reason"`
	ReadyByControl *ReadyByControlChange `json:"ready_by_control,omitempty"`
}

// ReadyByControlChange describes a persisted ready-by state transition.
type ReadyByControlChange struct {
	Previous *pool.ReadyByControlState `json:"previous,omitempty"`
	Current  pool.ReadyByControlState  `json:"current"`
	Reason   string                    `json:"reason"`
}

type Scheduler struct {
	config Config
}

func New(config Config) *Scheduler {
	if config.HeatingRateCPerHour <= 0 {
		config.HeatingRateCPerHour = 0.75
	}
	if config.ReadinessBuffer <= 0 {
		config.ReadinessBuffer = 30 * time.Minute
	}
	if config.Location == nil {
		config.Location = time.Local
	}
	if config.ReadyByReheatDelta <= 0 {
		config.ReadyByReheatDelta = 2
	}
	return &Scheduler{config: config}
}

func (s *Scheduler) Evaluate(now time.Time, status pool.Status, base pool.DesiredState, plans []pool.Plan) Evaluation {
	return s.EvaluateWithReadyByControl(now, status, base, plans, nil)
}

// EvaluateWithReadyByControl evaluates plans with persisted ready-by hysteresis state.
func (s *Scheduler) EvaluateWithReadyByControl(now time.Time, status pool.Status, base pool.DesiredState, plans []pool.Plan, states map[string]pool.ReadyByControlState) Evaluation {
	now = now.In(s.config.Location)
	base = base.WithHardwareConstraints()

	for _, plan := range plans {
		if !plan.Enabled || plan.Type != pool.PlanReadyBy {
			continue
		}
		desired, active, _, reason, control := s.evaluateReadyBy(now, status, base, plan, states)
		if !active {
			continue
		}
		return Evaluation{
			Desired:        desired.WithHardwareConstraints(),
			Source:         plan.ID,
			Reason:         reason,
			ReadyByControl: control,
		}
	}

	timeWindowDesired, active, source, reason := s.evaluateTimeWindows(now, base, plans)
	if active {
		return Evaluation{
			Desired: timeWindowDesired.WithHardwareConstraints(),
			Source:  source,
			Reason:  reason,
		}
	}

	return Evaluation{Desired: base, Source: "default", Reason: "default desired state"}
}

func (s *Scheduler) NextWake(now time.Time, status pool.Status, plans []pool.Plan) (time.Time, bool) {
	now = now.In(s.config.Location)
	var next time.Time
	ok := false
	add := func(t time.Time) {
		t = t.In(s.config.Location)
		if !t.After(now) {
			return
		}
		if !ok || t.Before(next) {
			next = t
			ok = true
		}
	}

	for _, plan := range plans {
		if !plan.Enabled {
			continue
		}
		switch plan.Type {
		case pool.PlanReadyBy:
			if startAt, readyAt, ok := s.readyByTimes(now, status, plan); ok {
				add(startAt)
				add(readyAt)
			}
		case pool.PlanTimeWindow:
			s.addTimeWindowWakeTimes(now, plan, add)
		}
	}
	return next, ok
}

func (s *Scheduler) evaluateReadyBy(now time.Time, status pool.Status, base pool.DesiredState, plan pool.Plan, states map[string]pool.ReadyByControlState) (pool.DesiredState, bool, bool, string, *ReadyByControlChange) {
	if plan.TargetTemp == nil {
		return base, false, false, "", nil
	}
	startAt, readyAt, ok := s.readyByTimes(now, status, plan)
	if !ok {
		return base, false, false, "", nil
	}
	target := *plan.TargetTemp
	key := ReadyByOccurrenceKey(plan.ID, readyAt, target)
	state, hasState := states[key]
	if !readyByStateValid(state, key, plan.ID, readyAt, target) {
		hasState = false
	}
	beforeStart := now.Before(startAt)
	if beforeStart {
		hasTemp := status.CurrentTemp != nil
		oneShotReached := strings.TrimSpace(plan.Cron) == "" && hasTemp && *status.CurrentTemp >= target
		heaterAlreadyActive := status.Heater && hasTemp
		if !oneShotReached && !heaterAlreadyActive {
			return base, false, false, "", nil
		}
	}

	mode := pool.ReadyBySeekingTarget
	var previous *pool.ReadyByControlState
	if hasState {
		mode = state.Mode
		previousState := state
		previous = &previousState
	} else if status.CurrentTemp != nil && *status.CurrentTemp >= target {
		mode = pool.ReadyBySatisfiedIdle
	}
	mode, transitionReason := transitionReadyByMode(mode, status.CurrentTemp, target, s.config.ReadyByReheatDelta)
	control := pool.ReadyByControlState{
		Key:        key,
		PlanID:     plan.ID,
		ReadyAt:    readyAt,
		TargetTemp: target,
		Mode:       mode,
		UpdatedAt:  now.UTC(),
	}
	var change *ReadyByControlChange
	if !hasState || state.Mode != mode {
		reason := transitionReason
		if reason == "" {
			reason = readyByModeReason(mode)
		}
		change = &ReadyByControlChange{Previous: previous, Current: control, Reason: reason}
	}

	desired := pool.DesiredState{TargetTemp: plan.TargetTemp}.Overlay(base)
	if mode == pool.ReadyBySatisfiedIdle {
		desired.Heater = pool.BoolPtr(false)
		return desired, true, false, "ready-by target satisfied", change
	}

	desired.Power = pool.BoolPtr(true)
	desired.Filter = pool.BoolPtr(true)
	desired.Heater = pool.BoolPtr(true)
	reason := fmt.Sprintf("ready-by heating window started at %s", startAt.Format(time.RFC3339))
	if beforeStart {
		reason = "ready-by heating already active"
	} else if mode == pool.ReadyByReheating {
		reason = fmt.Sprintf("ready-by reheating at or below %d", target-s.config.ReadyByReheatDelta)
	}
	return desired, true, true, reason, change
}

// ReadyByOccurrenceKey identifies one plan, ready-at instant, and target temperature.
func ReadyByOccurrenceKey(planID string, readyAt time.Time, target int) string {
	return fmt.Sprintf("%s|%s|%d", planID, readyAt.UTC().Format(time.RFC3339Nano), target)
}

func readyByStateValid(state pool.ReadyByControlState, key, planID string, readyAt time.Time, target int) bool {
	if state.Key != key || state.PlanID != planID || state.TargetTemp != target || !state.ReadyAt.Equal(readyAt) {
		return false
	}
	switch state.Mode {
	case pool.ReadyBySeekingTarget, pool.ReadyBySatisfiedIdle, pool.ReadyByReheating:
		return true
	default:
		return false
	}
}

func transitionReadyByMode(mode pool.ReadyByControlMode, current *int, target, reheatDelta int) (pool.ReadyByControlMode, string) {
	if current == nil {
		return mode, ""
	}
	switch mode {
	case pool.ReadyBySeekingTarget:
		if *current >= target {
			return pool.ReadyBySatisfiedIdle, "target temperature reached"
		}
	case pool.ReadyBySatisfiedIdle:
		if *current <= target-reheatDelta {
			return pool.ReadyByReheating, "temperature dropped to reheat threshold"
		}
	case pool.ReadyByReheating:
		if *current >= target {
			return pool.ReadyBySatisfiedIdle, "target temperature reached"
		}
	default:
		return transitionReadyByMode(pool.ReadyBySeekingTarget, current, target, reheatDelta)
	}
	return mode, ""
}

func readyByModeReason(mode pool.ReadyByControlMode) string {
	switch mode {
	case pool.ReadyBySatisfiedIdle:
		return "target temperature reached"
	case pool.ReadyByReheating:
		return "temperature dropped to reheat threshold"
	default:
		return "ready-by seeking target"
	}
}

func (s *Scheduler) readyByTimes(now time.Time, status pool.Status, plan pool.Plan) (time.Time, time.Time, bool) {
	if plan.TargetTemp == nil {
		return time.Time{}, time.Time{}, false
	}
	at, ok := s.readyByAt(now, plan)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	current := *plan.TargetTemp
	if status.CurrentTemp != nil {
		current = *status.CurrentTemp
	}
	delta := *plan.TargetTemp - current
	if delta < 0 {
		delta = 0
	}
	required := time.Duration((float64(delta) / s.config.HeatingRateCPerHour) * float64(time.Hour))
	return at.Add(-required).Add(-s.config.ReadinessBuffer), at, true
}

func (s *Scheduler) readyByAt(now time.Time, plan pool.Plan) (time.Time, bool) {
	if cron := strings.TrimSpace(plan.Cron); cron != "" {
		at, ok, err := pool.NextCronTime(cron, now, s.config.Location)
		if err != nil || !ok {
			return time.Time{}, false
		}
		return at, true
	}
	if plan.At == nil {
		return time.Time{}, false
	}
	return plan.At.In(s.config.Location), true
}

func (s *Scheduler) evaluateTimeWindows(now time.Time, base pool.DesiredState, plans []pool.Plan) (pool.DesiredState, bool, string, string) {
	type activePlan struct {
		id         string
		capability string
	}

	var active []activePlan
	for _, plan := range plans {
		if !plan.Enabled || plan.Type != pool.PlanTimeWindow || plan.Capability == "" {
			continue
		}
		if timeWindowActive(now, plan, s.config.Location) {
			active = append(active, activePlan{
				id:         plan.ID,
				capability: normalizeCapability(plan.Capability),
			})
		}
	}

	if len(active) == 0 {
		return base, false, "", ""
	}

	desired := base
	for _, ap := range active {
		setCapability(&desired, ap.capability, true)
	}

	return desired, true, active[0].id, "time window plan active"
}

// A window occurrence belongs to the day it starts on: Days gates the start
// day, and the occurrence stays active for its full duration even past midnight.
func timeWindowActive(now time.Time, plan pool.Plan, loc *time.Location) bool {
	now = now.In(loc)
	start, err := pool.ParseClock(plan.Start)
	if err != nil {
		return false
	}
	if plan.DurationMinutes <= 0 {
		return false
	}
	duration := time.Duration(plan.DurationMinutes) * time.Minute
	today := midnight(now, loc)
	for offset := -1; offset <= 0; offset++ {
		day := today.AddDate(0, 0, offset)
		if !dayAllowed(day, plan.Days) {
			continue
		}
		begin := clockTime(day, start, loc)
		if !now.Before(begin) && now.Before(begin.Add(duration)) {
			return true
		}
	}
	return false
}

func (s *Scheduler) addTimeWindowWakeTimes(now time.Time, plan pool.Plan, add func(time.Time)) {
	start, err := pool.ParseClock(plan.Start)
	if err != nil {
		return
	}
	if plan.DurationMinutes <= 0 {
		return
	}
	duration := time.Duration(plan.DurationMinutes) * time.Minute

	loc := s.config.Location
	today := midnight(now.In(loc), loc)
	for offset := -1; offset <= 8; offset++ {
		day := today.AddDate(0, 0, offset)
		if !dayAllowed(day, plan.Days) {
			continue
		}
		begin := clockTime(day, start, loc)
		add(begin)
		add(begin.Add(duration))
	}
}

func midnight(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func clockTime(day time.Time, clock pool.Clock, loc *time.Location) time.Time {
	day = day.In(loc)
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour, clock.Minute, 0, 0, loc)
}

func dayAllowed(day time.Time, days []string) bool {
	return len(days) == 0 || dayIncluded(day.Weekday(), days)
}

func dayIncluded(weekday time.Weekday, days []string) bool {
	for _, day := range days {
		switch strings.ToLower(strings.TrimSpace(day)) {
		case "sun", "sunday":
			if weekday == time.Sunday {
				return true
			}
		case "mon", "monday":
			if weekday == time.Monday {
				return true
			}
		case "tue", "tues", "tuesday":
			if weekday == time.Tuesday {
				return true
			}
		case "wed", "wednesday":
			if weekday == time.Wednesday {
				return true
			}
		case "thu", "thur", "thurs", "thursday":
			if weekday == time.Thursday {
				return true
			}
		case "fri", "friday":
			if weekday == time.Friday {
				return true
			}
		case "sat", "saturday":
			if weekday == time.Saturday {
				return true
			}
		}
	}
	return false
}

func setCapability(desired *pool.DesiredState, capability string, state bool) {
	switch normalizeCapability(capability) {
	case "power":
		desired.Power = pool.BoolPtr(state)
	case "filter":
		desired.Filter = pool.BoolPtr(state)
		if state {
			desired.Power = pool.BoolPtr(true)
		}
	case "jets":
		desired.Jets = pool.BoolPtr(state)
		if state {
			desired.Power = pool.BoolPtr(true)
		}
	case "bubbles":
		desired.Bubbles = pool.BoolPtr(state)
		if state {
			desired.Power = pool.BoolPtr(true)
		}
	case "sanitizer":
		desired.Sanitizer = pool.BoolPtr(state)
		if state {
			desired.Power = pool.BoolPtr(true)
		}
	}
}

func normalizeCapability(capability string) string {
	switch strings.ToLower(strings.TrimSpace(capability)) {
	case "heating":
		return "heater"
	case "temp", "temperature", "target_temperature", "preset_temp":
		return "target_temp"
	default:
		return strings.ToLower(strings.TrimSpace(capability))
	}
}
