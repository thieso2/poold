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
}

type Evaluation struct {
	Desired pool.DesiredState `json:"desired"`
	Source  string            `json:"source"`
	Reason  string            `json:"reason"`
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
	return &Scheduler{config: config}
}

func (s *Scheduler) Evaluate(now time.Time, status pool.Status, base pool.DesiredState, plans []pool.Plan) Evaluation {
	now = now.In(s.config.Location)
	base = base.WithHardwareConstraints()

	for _, plan := range plans {
		if !plan.Enabled || plan.Type != pool.PlanManualOverride {
			continue
		}
		if plan.ExpiresAt == nil || !plan.ExpiresAt.After(now) {
			continue
		}
		return Evaluation{
			Desired: plan.DesiredState.Overlay(base).WithHardwareConstraints(),
			Source:  plan.ID,
			Reason:  "manual override active",
		}
	}

	for _, plan := range plans {
		if !plan.Enabled || plan.Type != pool.PlanReadyBy {
			continue
		}
		desired, active, reason := s.evaluateReadyBy(now, status, base, plan)
		if active {
			return Evaluation{Desired: desired.WithHardwareConstraints(), Source: plan.ID, Reason: reason}
		}
	}

	timeWindowDesired, active, reason := s.evaluateTimeWindows(now, base, plans)
	if active {
		return Evaluation{Desired: timeWindowDesired.WithHardwareConstraints(), Source: "time_window", Reason: reason}
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
		case pool.PlanManualOverride:
			if plan.ExpiresAt != nil {
				add(*plan.ExpiresAt)
			}
		case pool.PlanReadyBy:
			if startAt, readyAt, ok := s.readyByTimes(status, plan); ok {
				add(startAt)
				add(readyAt)
			}
		case pool.PlanTimeWindow:
			s.addTimeWindowWakeTimes(now, plan, add)
		}
	}
	return next, ok
}

func (s *Scheduler) evaluateReadyBy(now time.Time, status pool.Status, base pool.DesiredState, plan pool.Plan) (pool.DesiredState, bool, string) {
	if plan.TargetTemp == nil || plan.At == nil {
		return base, false, ""
	}
	current := 0
	if status.CurrentTemp != nil {
		current = *status.CurrentTemp
	} else {
		current = *plan.TargetTemp
	}
	desired := pool.DesiredState{TargetTemp: plan.TargetTemp}.Overlay(base)
	if current >= *plan.TargetTemp {
		desired.Heater = pool.BoolPtr(false)
		return desired, true, "target temperature reached"
	}

	startAt, _, _ := s.readyByTimes(status, plan)
	if now.Before(startAt) {
		return base, false, ""
	}
	desired.Power = pool.BoolPtr(true)
	desired.Filter = pool.BoolPtr(true)
	desired.Heater = pool.BoolPtr(true)
	return desired, true, fmt.Sprintf("ready-by heating window started at %s", startAt.Format(time.RFC3339))
}

func (s *Scheduler) readyByTimes(status pool.Status, plan pool.Plan) (time.Time, time.Time, bool) {
	if plan.TargetTemp == nil || plan.At == nil {
		return time.Time{}, time.Time{}, false
	}
	at := plan.At.In(s.config.Location)
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

func (s *Scheduler) evaluateTimeWindows(now time.Time, base pool.DesiredState, plans []pool.Plan) (pool.DesiredState, bool, string) {
	desired := base
	seen := map[string]bool{}
	activeCaps := map[string]bool{}

	for _, plan := range plans {
		if !plan.Enabled || plan.Type != pool.PlanTimeWindow || plan.Capability == "" {
			continue
		}
		capability := normalizeCapability(plan.Capability)
		seen[capability] = true
		if timeWindowActive(now, plan, s.config.Location) {
			activeCaps[capability] = true
		}
	}

	if len(seen) == 0 {
		return base, false, ""
	}
	for capability := range seen {
		setCapability(&desired, capability, activeCaps[capability])
	}
	return desired, true, "time window plan evaluated"
}

func timeWindowActive(now time.Time, plan pool.Plan, loc *time.Location) bool {
	now = now.In(loc)
	from, err := pool.ParseClock(plan.From)
	if err != nil {
		return false
	}
	to, err := pool.ParseClock(plan.To)
	if err != nil {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	start := from.Minutes()
	end := to.Minutes()
	if start == end {
		return dayAllowed(now, plan.Days)
	}
	if start < end {
		return dayAllowed(now, plan.Days) && current >= start && current < end
	}
	if current >= start {
		return dayAllowed(now, plan.Days)
	}
	return current < end && dayAllowed(now.AddDate(0, 0, -1), plan.Days)
}

func (s *Scheduler) addTimeWindowWakeTimes(now time.Time, plan pool.Plan, add func(time.Time)) {
	from, err := pool.ParseClock(plan.From)
	if err != nil {
		return
	}
	to, err := pool.ParseClock(plan.To)
	if err != nil {
		return
	}
	if from.Minutes() == to.Minutes() {
		return
	}

	loc := s.config.Location
	today := midnight(now.In(loc), loc)
	for offset := 0; offset <= 8; offset++ {
		day := today.AddDate(0, 0, offset)
		if dayAllowed(day, plan.Days) {
			start := clockTime(day, from, loc)
			endDay := day
			if to.Minutes() <= from.Minutes() {
				endDay = endDay.AddDate(0, 0, 1)
			}
			add(start)
			add(clockTime(endDay, to, loc))
		}
	}

	if to.Minutes() < from.Minutes() {
		previousDay := today.AddDate(0, 0, -1)
		if dayAllowed(previousDay, plan.Days) {
			add(clockTime(today, to, loc))
		}
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
	case "heater":
		desired.Heater = pool.BoolPtr(state)
	case "jets":
		desired.Jets = pool.BoolPtr(state)
	case "bubbles":
		desired.Bubbles = pool.BoolPtr(state)
	case "sanitizer":
		desired.Sanitizer = pool.BoolPtr(state)
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
