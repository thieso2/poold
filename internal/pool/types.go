package pool

import (
	"encoding/json"
	"fmt"
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
	if out.Heater != nil && *out.Heater {
		out.Filter = BoolPtr(true)
		out.Power = BoolPtr(true)
	}
	if out.Filter != nil && !*out.Filter && (out.Heater == nil || !*out.Heater) {
		out.Heater = BoolPtr(false)
	}
	return out
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
	TargetTemp   *int         `json:"target_temp,omitempty"`
	At           *time.Time   `json:"at,omitempty"`
	DesiredState DesiredState `json:"desired_state,omitempty"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at,omitempty"`
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
		if p.At == nil {
			return fmt.Errorf("ready_by plan requires at")
		}
	case PlanManualOverride:
		if p.ExpiresAt == nil {
			return fmt.Errorf("manual_override plan requires expires_at")
		}
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

type Event struct {
	ID        int64           `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Type      string          `json:"type"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type Observation struct {
	ID     int64  `json:"id"`
	Status Status `json:"status"`
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
