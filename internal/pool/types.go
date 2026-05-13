package pool

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Status struct {
	ObservedAt  time.Time `json:"observed_at"`
	Connected   bool      `json:"connected"`
	Power       bool      `json:"power"`
	Filter      bool      `json:"filter"`
	Heater      bool      `json:"heater"`
	Jets        bool      `json:"jets"`
	Bubbles     bool      `json:"bubbles"`
	Sanitizer   bool      `json:"sanitizer"`
	Unit        string    `json:"unit"`
	CurrentTemp *int      `json:"current_temp,omitempty"`
	TargetTemp  int       `json:"preset_temp"`
	ErrorCode   string    `json:"error_code,omitempty"`
	RawData     string    `json:"raw_data,omitempty"`
}

type DesiredState struct {
	Power      *bool `json:"power,omitempty"`
	Filter     *bool `json:"filter,omitempty"`
	Heater     *bool `json:"heater,omitempty"`
	Jets       *bool `json:"jets,omitempty"`
	Bubbles    *bool `json:"bubbles,omitempty"`
	Sanitizer  *bool `json:"sanitizer,omitempty"`
	TargetTemp *int  `json:"target_temp,omitempty"`
}

func (d DesiredState) Empty() bool {
	return d.Power == nil &&
		d.Filter == nil &&
		d.Heater == nil &&
		d.Jets == nil &&
		d.Bubbles == nil &&
		d.Sanitizer == nil &&
		d.TargetTemp == nil
}

func (d DesiredState) Overlay(base DesiredState) DesiredState {
	out := base
	if d.Power != nil {
		out.Power = d.Power
	}
	if d.Filter != nil {
		out.Filter = d.Filter
	}
	if d.Heater != nil {
		out.Heater = d.Heater
	}
	if d.Jets != nil {
		out.Jets = d.Jets
	}
	if d.Bubbles != nil {
		out.Bubbles = d.Bubbles
	}
	if d.Sanitizer != nil {
		out.Sanitizer = d.Sanitizer
	}
	if d.TargetTemp != nil {
		out.TargetTemp = d.TargetTemp
	}
	return out
}

func (d DesiredState) WithHardwareConstraints() DesiredState {
	out := d
	if out.Power != nil && !*out.Power {
		out.Filter = BoolPtr(false)
		out.Heater = BoolPtr(false)
		out.Jets = BoolPtr(false)
		out.Bubbles = BoolPtr(false)
		out.Sanitizer = BoolPtr(false)
	}
	if boolOn(out.Filter) || boolOn(out.Heater) || boolOn(out.Jets) || boolOn(out.Bubbles) || boolOn(out.Sanitizer) {
		out.Power = BoolPtr(true)
	}
	if out.Heater != nil && *out.Heater {
		out.Filter = BoolPtr(true)
		out.Power = BoolPtr(true)
	}
	if out.Filter != nil && !*out.Filter && (out.Heater == nil || !*out.Heater) {
		out.Heater = BoolPtr(false)
	}
	return out
}

func boolOn(value *bool) bool {
	return value != nil && *value
}

func DesiredFromStatus(status Status) DesiredState {
	return DesiredState{
		Power:      BoolPtr(status.Power),
		Filter:     BoolPtr(status.Filter),
		Heater:     BoolPtr(status.Heater),
		Jets:       BoolPtr(status.Jets),
		Bubbles:    BoolPtr(status.Bubbles),
		Sanitizer:  BoolPtr(status.Sanitizer),
		TargetTemp: IntPtr(status.TargetTemp),
	}
}

func BoolPtr(v bool) *bool {
	return &v
}

func IntPtr(v int) *int {
	return &v
}

type PlanType string

const (
	PlanTimeWindow     PlanType = "time_window"
	PlanReadyBy        PlanType = "ready_by"
	PlanManualOverride PlanType = "manual_override"
)

type Plan struct {
	ID           string       `json:"id"`
	Type         PlanType     `json:"type"`
	Name         string       `json:"name,omitempty"`
	Enabled      bool         `json:"enabled"`
	Capability   string       `json:"capability,omitempty"`
	From         string       `json:"from,omitempty"`
	To           string       `json:"to,omitempty"`
	Days         []string     `json:"days,omitempty"`
	Cron         string       `json:"cron,omitempty"`
	TargetTemp   *int         `json:"target_temp,omitempty"`
	At           *time.Time   `json:"at,omitempty"`
	DesiredState DesiredState `json:"desired_state,omitempty"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at,omitempty"`
}

// ReadyByControlMode is the persisted control state for one ready-by occurrence.
type ReadyByControlMode string

const (
	// ReadyBySeekingTarget heats until the target temperature has been observed.
	ReadyBySeekingTarget ReadyByControlMode = "seeking_target"
	// ReadyBySatisfiedIdle keeps heat off after the target has been observed.
	ReadyBySatisfiedIdle ReadyByControlMode = "satisfied_idle"
	// ReadyByReheating heats after a satisfied occurrence falls below the reheat threshold.
	ReadyByReheating ReadyByControlMode = "reheating"
)

// ReadyByControlState persists hysteresis state for one concrete ready-by occurrence.
type ReadyByControlState struct {
	Key        string             `json:"key"`
	PlanID     string             `json:"plan_id"`
	ReadyAt    time.Time          `json:"ready_at"`
	TargetTemp int                `json:"target_temp"`
	Mode       ReadyByControlMode `json:"mode"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

func (p Plan) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("plan id is required")
	}
	switch p.Type {
	case PlanTimeWindow:
		if p.Capability == "" {
			return fmt.Errorf("time_window plan requires capability")
		}
		if p.From == "" || p.To == "" {
			return fmt.Errorf("time_window plan requires from and to")
		}
		if _, err := ParseClock(p.From); err != nil {
			return fmt.Errorf("invalid from time: %w", err)
		}
		if _, err := ParseClock(p.To); err != nil {
			return fmt.Errorf("invalid to time: %w", err)
		}
	case PlanReadyBy:
		if p.TargetTemp == nil {
			return fmt.Errorf("ready_by plan requires target_temp")
		}
		hasAt := p.At != nil
		hasCron := strings.TrimSpace(p.Cron) != ""
		if !hasAt && !hasCron {
			return fmt.Errorf("ready_by plan requires at or cron")
		}
		if hasAt && hasCron {
			return fmt.Errorf("ready_by plan cannot have both at and cron")
		}
		if hasCron {
			if _, err := ParseCronSpec(p.Cron); err != nil {
				return fmt.Errorf("invalid ready_by cron: %w", err)
			}
		}
	case PlanManualOverride:
		if p.DesiredState.Empty() {
			return fmt.Errorf("manual_override plan requires desired_state")
		}
	default:
		return fmt.Errorf("unknown plan type %q", p.Type)
	}
	return nil
}

type Clock struct {
	Hour   int
	Minute int
}

func ParseClock(value string) (Clock, error) {
	var c Clock
	if _, err := fmt.Sscanf(value, "%d:%d", &c.Hour, &c.Minute); err != nil {
		return c, err
	}
	if c.Hour < 0 || c.Hour > 23 || c.Minute < 0 || c.Minute > 59 {
		return c, fmt.Errorf("time must be HH:MM")
	}
	return c, nil
}

func (c Clock) Minutes() int {
	return c.Hour*60 + c.Minute
}

// CronSpec is a parsed five-field cron schedule.
type CronSpec struct {
	minutes cronField
	hours   cronField
	dom     cronField
	months  cronField
	dow     cronField
}

type cronField struct {
	any    bool
	values map[int]bool
}

// ParseCronSpec parses a five-field cron expression.
func ParseCronSpec(expr string) (CronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return CronSpec{}, fmt.Errorf("cron must have 5 fields")
	}

	minutes, err := parseCronField("minute", fields[0], 0, 59, nil, nil)
	if err != nil {
		return CronSpec{}, err
	}
	hours, err := parseCronField("hour", fields[1], 0, 23, nil, nil)
	if err != nil {
		return CronSpec{}, err
	}
	dom, err := parseCronField("day-of-month", fields[2], 1, 31, nil, nil)
	if err != nil {
		return CronSpec{}, err
	}
	months, err := parseCronField("month", fields[3], 1, 12, monthAliases(), nil)
	if err != nil {
		return CronSpec{}, err
	}
	dow, err := parseCronField("day-of-week", fields[4], 0, 7, weekdayAliases(), normalizeCronWeekday)
	if err != nil {
		return CronSpec{}, err
	}

	return CronSpec{
		minutes: minutes,
		hours:   hours,
		dom:     dom,
		months:  months,
		dow:     dow,
	}, nil
}

// NextCronTime returns the next scheduled minute at or after after.
func NextCronTime(expr string, after time.Time, loc *time.Location) (time.Time, bool, error) {
	spec, err := ParseCronSpec(expr)
	if err != nil {
		return time.Time{}, false, err
	}
	if loc == nil {
		loc = time.Local
	}

	cursor := after.In(loc).Truncate(time.Minute)
	deadline := cursor.AddDate(1, 0, 1)
	for !cursor.After(deadline) {
		if spec.Matches(cursor) {
			return cursor, true, nil
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, false, nil
}

// Matches reports whether the time matches the cron schedule.
func (s CronSpec) Matches(t time.Time) bool {
	if !s.minutes.matches(t.Minute()) || !s.hours.matches(t.Hour()) || !s.months.matches(int(t.Month())) {
		return false
	}

	dayOfMonth := s.dom.matches(t.Day())
	dayOfWeek := s.dow.matches(int(t.Weekday()))
	if !s.dom.any && !s.dow.any {
		return dayOfMonth || dayOfWeek
	}
	return dayOfMonth && dayOfWeek
}

func parseCronField(name, spec string, min, max int, aliases map[string]int, normalize func(int) (int, error)) (cronField, error) {
	spec = strings.ToLower(strings.TrimSpace(spec))
	if spec == "" {
		return cronField{}, fmt.Errorf("%s field is empty", name)
	}

	field := cronField{any: spec == "*", values: map[int]bool{}}
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, fmt.Errorf("%s field has an empty list item", name)
		}
		step := 1
		rangeSpec := part
		if strings.Contains(part, "/") {
			pieces := strings.Split(part, "/")
			if len(pieces) != 2 {
				return cronField{}, fmt.Errorf("%s field has invalid step %q", name, part)
			}
			rangeSpec = pieces[0]
			parsedStep, err := strconv.Atoi(pieces[1])
			if err != nil || parsedStep <= 0 {
				return cronField{}, fmt.Errorf("%s field has invalid step %q", name, pieces[1])
			}
			step = parsedStep
		}

		start, end, err := parseCronRange(name, rangeSpec, min, max, aliases)
		if err != nil {
			return cronField{}, err
		}
		for value := start; value <= end; value += step {
			normalized := value
			if normalize != nil {
				var err error
				normalized, err = normalize(value)
				if err != nil {
					return cronField{}, fmt.Errorf("%s field has invalid value %d: %w", name, value, err)
				}
			}
			field.values[normalized] = true
		}
	}
	return field, nil
}

func parseCronRange(name, spec string, min, max int, aliases map[string]int) (int, int, error) {
	if spec == "*" {
		return min, max, nil
	}
	if strings.Contains(spec, "-") {
		pieces := strings.Split(spec, "-")
		if len(pieces) != 2 {
			return 0, 0, fmt.Errorf("%s field has invalid range %q", name, spec)
		}
		start, err := parseCronValue(name, pieces[0], min, max, aliases)
		if err != nil {
			return 0, 0, err
		}
		end, err := parseCronValue(name, pieces[1], min, max, aliases)
		if err != nil {
			return 0, 0, err
		}
		if start > end {
			return 0, 0, fmt.Errorf("%s field range %q runs backwards", name, spec)
		}
		return start, end, nil
	}
	value, err := parseCronValue(name, spec, min, max, aliases)
	if err != nil {
		return 0, 0, err
	}
	return value, value, nil
}

func parseCronValue(name, value string, min, max int, aliases map[string]int) (int, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if aliases != nil {
		if aliased, ok := aliases[value]; ok {
			return aliased, nil
		}
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s field has invalid value %q", name, value)
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("%s field value %d is outside %d-%d", name, parsed, min, max)
	}
	return parsed, nil
}

func (f cronField) matches(value int) bool {
	return f.any || f.values[value]
}

func normalizeCronWeekday(value int) (int, error) {
	if value == 7 {
		return 0, nil
	}
	return value, nil
}

func monthAliases() map[string]int {
	return map[string]int{
		"jan": 1,
		"feb": 2,
		"mar": 3,
		"apr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dec": 12,
	}
}

func weekdayAliases() map[string]int {
	return map[string]int{
		"sun":       0,
		"sunday":    0,
		"mon":       1,
		"monday":    1,
		"tue":       2,
		"tues":      2,
		"tuesday":   2,
		"wed":       3,
		"wednesday": 3,
		"thu":       4,
		"thur":      4,
		"thurs":     4,
		"thursday":  4,
		"fri":       5,
		"friday":    5,
		"sat":       6,
		"saturday":  6,
	}
}

type CommandRequest struct {
	Command    string          `json:"command,omitempty"`
	Capability string          `json:"capability,omitempty"`
	State      *bool           `json:"state,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"`
	Source     string          `json:"source,omitempty"`
}

func (c CommandRequest) NormalizedCapability() string {
	capability := strings.TrimSpace(strings.ToLower(c.Capability))
	if capability != "" {
		return capability
	}
	command := strings.TrimSpace(strings.ToLower(c.Command))
	command = strings.TrimPrefix(command, "set_")
	command = strings.TrimPrefix(command, "target_")
	switch command {
	case "temp", "temperature", "preset_temp", "preset_temperature":
		return "target_temp"
	case "heating":
		return "heater"
	default:
		return command
	}
}

type CommandRecord struct {
	ID          int64           `json:"id"`
	IssuedAt    time.Time       `json:"issued_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Capability  string          `json:"capability"`
	State       *bool           `json:"state,omitempty"`
	Value       json.RawMessage `json:"value,omitempty"`
	Source      string          `json:"source,omitempty"`
	Success     bool            `json:"success"`
	Error       string          `json:"error,omitempty"`
	Status      *Status         `json:"status,omitempty"`
}

type HeatingSession struct {
	ID                 int64      `json:"id"`
	FirstObservationID int64      `json:"first_observation_id"`
	LastObservationID  int64      `json:"last_observation_id"`
	StartedAt          time.Time  `json:"started_at"`
	LastObservedAt     time.Time  `json:"last_observed_at"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
	DurationSeconds    int64      `json:"duration_seconds"`
	ObservationCount   int        `json:"observation_count"`
	SpanCount          int        `json:"span_count"`
	StartTemp          *int       `json:"start_temp,omitempty"`
	EndTemp            *int       `json:"end_temp,omitempty"`
	TargetTemp         int        `json:"target_temp"`
	Active             bool       `json:"active"`
}

type Event struct {
	ID        int64           `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Type      string          `json:"type"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type Observation struct {
	ID               int64            `json:"id"`
	FirstObservedAt  time.Time        `json:"first_observed_at"`
	LastObservedAt   time.Time        `json:"last_observed_at"`
	ObservationCount int              `json:"observation_count"`
	Status           Status           `json:"status"`
	Weather          *WeatherSnapshot `json:"weather,omitempty"`
}

// TimelineResponse is the chart-ready dashboard history payload.
type TimelineResponse struct {
	From             time.Time             `json:"from"`
	To               time.Time             `json:"to"`
	Range            string                `json:"range"`
	GeneratedAt      time.Time             `json:"generated_at"`
	BucketSeconds    int64                 `json:"bucket_seconds"`
	Unit             string                `json:"unit"`
	RawUnit          string                `json:"raw_unit,omitempty"`
	WeatherAvailable bool                  `json:"weather_available"`
	Measured         []TimelinePoint       `json:"measured"`
	Predicted        []TimelinePoint       `json:"predicted"`
	Target           []TimelineTargetPoint `json:"target"`
	FeatureSpans     []TimelineFeatureSpan `json:"feature_spans"`
	Annotations      []TimelineAnnotation  `json:"annotations"`
	Model            TimelineModel         `json:"model"`
	Warnings         []TimelineWarning     `json:"warnings,omitempty"`
}

// TimelinePoint is a measured or predicted temperature point.
type TimelinePoint struct {
	T                    time.Time `json:"t"`
	PoolTemp             *float64  `json:"pool_temp,omitempty"`
	OutsideTempC         *float64  `json:"outside_temp_c,omitempty"`
	WeatherAgeSeconds    *int64    `json:"weather_age_seconds,omitempty"`
	Confidence           float64   `json:"confidence"`
	Kind                 string    `json:"kind,omitempty"`
	Model                string    `json:"model,omitempty"`
	SourceObservationIDs []int64   `json:"source_observation_ids,omitempty"`
}

// TimelineTargetPoint is a target-temperature step point.
type TimelineTargetPoint struct {
	T                    time.Time `json:"t"`
	TargetTemp           *float64  `json:"target_temp,omitempty"`
	SourceObservationIDs []int64   `json:"source_observation_ids,omitempty"`
}

// TimelineFeatureSpan is a merged span of binary feature state.
type TimelineFeatureSpan struct {
	From                 time.Time `json:"from"`
	To                   time.Time `json:"to"`
	Power                bool      `json:"power"`
	Filter               bool      `json:"filter"`
	Heater               bool      `json:"heater"`
	Jets                 bool      `json:"jets"`
	Bubbles              bool      `json:"bubbles"`
	Sanitizer            bool      `json:"sanitizer"`
	Connected            bool      `json:"connected"`
	SourceObservationIDs []int64   `json:"source_observation_ids,omitempty"`
}

// TimelineAnnotation is a compact command or plan marker for chart tooltips.
type TimelineAnnotation struct {
	T        time.Time `json:"t"`
	Type     string    `json:"type"`
	Label    string    `json:"label"`
	Detail   string    `json:"detail,omitempty"`
	SourceID int64     `json:"source_id"`
}

// TimelineModel describes the rates used for prediction.
type TimelineModel struct {
	HeatingRateCPerHour    float64                 `json:"heating_rate_c_per_hour"`
	HeatingModel           string                  `json:"heating_model"`
	HeatingSamples         int                     `json:"heating_samples"`
	HeatingDurationSeconds int64                   `json:"heating_duration_seconds"`
	CoolingRateCPerHour    float64                 `json:"cooling_rate_c_per_hour"`
	CoolingModel           string                  `json:"cooling_model"`
	CoolingSamples         int                     `json:"cooling_samples"`
	CoolingDurationSeconds int64                   `json:"cooling_duration_seconds"`
	CoolingBuckets         map[string]TimelineRate `json:"cooling_buckets,omitempty"`
}

// TimelineRate reports evidence for a learned or fallback rate.
type TimelineRate struct {
	RateCPerHour    float64 `json:"rate_c_per_hour"`
	Model           string  `json:"model"`
	Samples         int     `json:"samples"`
	DurationSeconds int64   `json:"duration_seconds"`
}

// TimelineWarning reports non-fatal prediction or data-quality issues.
type TimelineWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Health struct {
	Process        string     `json:"process"`
	PoolTCP        string     `json:"pool_tcp"`
	PoolProtocol   string     `json:"pool_protocol"`
	Connected      bool       `json:"connected"`
	LastObservedAt *time.Time `json:"last_observed_at,omitempty"`
	LastCommandAt  *time.Time `json:"last_command_at,omitempty"`
	UptimeSeconds  int64      `json:"uptime_seconds"`
}

type WeatherLocation struct {
	Query   string  `json:"query,omitempty"`
	Name    string  `json:"name,omitempty"`
	Country string  `json:"country,omitempty"`
	State   string  `json:"state,omitempty"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

type WeatherSettings struct {
	APIKey    string          `json:"api_key,omitempty"`
	Location  WeatherLocation `json:"location,omitempty"`
	UpdatedAt time.Time       `json:"updated_at,omitempty"`
}

func (s WeatherSettings) Configured() bool {
	return strings.TrimSpace(s.APIKey) != "" &&
		strings.TrimSpace(s.Location.Query) != "" &&
		(s.Location.Lat != 0 || s.Location.Lon != 0)
}

type WeatherObservation struct {
	ID         int64           `json:"id"`
	ObservedAt time.Time       `json:"observed_at"`
	Location   WeatherLocation `json:"location"`
	Data       json.RawMessage `json:"data"`
}

type WeatherSnapshot struct {
	ObservationID     int64    `json:"observation_id"`
	OutsideTempC      *float64 `json:"outside_temp_c,omitempty"`
	CloudsPercent     *int     `json:"clouds_percent,omitempty"`
	Main              string   `json:"main,omitempty"`
	Description       string   `json:"description,omitempty"`
	WindSpeed         *float64 `json:"wind_speed,omitempty"`
	WeatherAgeSeconds int64    `json:"weather_age_seconds"`
}
