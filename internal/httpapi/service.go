package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

type Service struct {
	store                *store.Store
	client               intex.PoolClient
	scheduler            *scheduler.Scheduler
	startedAt            time.Time
	observationRetention time.Duration
	eventRetention       time.Duration
	eventHeartbeat       time.Duration
	commandConfirmDelay  time.Duration
	refreshRequests      chan time.Duration
	commandMu            sync.Mutex
	statusEventMu        sync.Mutex
	lastStatusEventAt    time.Time
	lastStatusError      string
	lastStatusErrorAt    time.Time
}

type ServiceConfig struct {
	ObservationRetention time.Duration
	EventRetention       time.Duration
	EventHeartbeat       time.Duration
	CommandConfirmDelay  time.Duration
}

func NewService(st *store.Store, client intex.PoolClient, sched *scheduler.Scheduler, cfg ServiceConfig) *Service {
	if cfg.EventHeartbeat <= 0 {
		cfg.EventHeartbeat = 30 * time.Minute
	}
	if cfg.CommandConfirmDelay <= 0 {
		cfg.CommandConfirmDelay = 10 * time.Second
	}
	return &Service{
		store:                st,
		client:               client,
		scheduler:            sched,
		startedAt:            time.Now().UTC(),
		observationRetention: cfg.ObservationRetention,
		eventRetention:       cfg.EventRetention,
		eventHeartbeat:       cfg.EventHeartbeat,
		commandConfirmDelay:  cfg.CommandConfirmDelay,
		refreshRequests:      make(chan time.Duration, 16),
	}
}

func (s *Service) RefreshRequests() <-chan time.Duration {
	return s.refreshRequests
}

func (s *Service) Health(ctx context.Context) pool.Health {
	status, ok, _ := s.store.LatestStatus(ctx)
	lastCommandAt, _ := s.store.LastCommandAt(ctx)
	health := pool.Health{
		Process:       "ok",
		PoolTCP:       "unknown",
		PoolProtocol:  "unknown",
		Connected:     false,
		LastCommandAt: lastCommandAt,
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
	}
	if ok {
		observedAt := status.ObservedAt
		health.LastObservedAt = &observedAt
		health.Connected = status.Connected
		if status.Connected {
			health.PoolTCP = "ok"
			health.PoolProtocol = "ok"
		}
	}
	return health
}

func (s *Service) RefreshStatus(ctx context.Context) (pool.Status, error) {
	previous, previousOK, err := s.store.LatestStatus(ctx)
	if err != nil {
		return pool.Status{}, err
	}
	status, err := s.client.Status(ctx)
	if err != nil {
		s.recordStatusError(ctx, err)
		return pool.Status{}, err
	}
	if status.ObservedAt.IsZero() {
		status.ObservedAt = time.Now().UTC()
	}
	status.Connected = true
	if _, err := s.store.SaveObservation(ctx, status); err != nil {
		return pool.Status{}, err
	}
	s.recordObservationEvent(ctx, previous, previousOK, status)
	_ = s.store.Prune(ctx, s.observationRetention, s.eventRetention)
	_ = s.Enforce(ctx, status)
	return status, nil
}

func (s *Service) ExecuteCommand(ctx context.Context, request pool.CommandRequest) (pool.CommandRecord, error) {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()

	capability := request.NormalizedCapability()
	value, err := commandValue(capability, request)
	if err != nil {
		return pool.CommandRecord{}, err
	}
	record := pool.CommandRecord{
		IssuedAt:   time.Now().UTC(),
		Capability: capability,
		State:      request.State,
		Value:      request.Value,
		Source:     request.Source,
	}
	id, err := s.store.InsertCommand(ctx, record)
	if err != nil {
		return pool.CommandRecord{}, err
	}
	record.ID = id

	status, err := s.client.Set(ctx, capability, value)
	if err != nil {
		record.Error = err.Error()
		_ = s.store.FinishCommand(ctx, id, false, nil, err)
		_, _ = s.store.AddEvent(ctx, "command_error", "command failed", map[string]any{
			"id":         id,
			"capability": capability,
			"error":      err.Error(),
		})
		return record, err
	}
	status.Connected = true
	if status.ObservedAt.IsZero() {
		status.ObservedAt = time.Now().UTC()
	}
	record.Success = true
	record.Status = &status
	now := time.Now().UTC()
	record.CompletedAt = &now

	if _, err := s.store.SaveObservation(ctx, status); err != nil {
		return record, err
	}
	if err := s.store.FinishCommand(ctx, id, true, &status, nil); err != nil {
		return record, err
	}
	_, _ = s.store.AddEvent(ctx, "command", "command completed", record)
	s.requestRefreshAfter(s.commandConfirmDelay)
	_ = s.store.Prune(ctx, s.observationRetention, s.eventRetention)
	return record, nil
}

func (s *Service) DesiredState(ctx context.Context) (pool.DesiredState, error) {
	return s.store.DesiredState(ctx)
}

func (s *Service) SaveDesiredState(ctx context.Context, desired pool.DesiredState) error {
	if err := s.store.SaveDesiredState(ctx, desired); err != nil {
		return err
	}
	_, _ = s.store.AddEvent(ctx, "desired_state", "desired state updated", desired)
	s.requestRefreshAfter(0)
	return nil
}

func (s *Service) Plans(ctx context.Context) ([]pool.Plan, error) {
	return s.store.Plans(ctx)
}

func (s *Service) SavePlans(ctx context.Context, plans []pool.Plan) error {
	if err := s.store.SavePlans(ctx, plans); err != nil {
		return err
	}
	_, _ = s.store.AddEvent(ctx, "plans", "plans updated", map[string]int{"count": len(plans)})
	s.requestRefreshAfter(0)
	return nil
}

func (s *Service) Events(ctx context.Context, afterID int64, limit int) ([]pool.Event, error) {
	return s.store.Events(ctx, afterID, limit)
}

func (s *Service) LatestEvents(ctx context.Context, limit int) ([]pool.Event, error) {
	return s.store.LatestEvents(ctx, limit)
}

func (s *Service) Observations(ctx context.Context, afterID int64, limit int) ([]pool.Observation, error) {
	return s.store.Observations(ctx, afterID, limit)
}

func (s *Service) LatestObservations(ctx context.Context, limit int) ([]pool.Observation, error) {
	return s.store.LatestObservations(ctx, limit)
}

func (s *Service) EnforceLatest(ctx context.Context) error {
	status, ok, err := s.store.LatestStatus(ctx)
	if err != nil {
		return err
	}
	if !ok {
		status, err = s.client.Status(ctx)
		if err != nil {
			return err
		}
		if _, err := s.store.SaveObservation(ctx, status); err != nil {
			return err
		}
	}
	return s.Enforce(ctx, status)
}

func (s *Service) Enforce(ctx context.Context, status pool.Status) error {
	base, err := s.store.DesiredState(ctx)
	if err != nil {
		return err
	}
	plans, err := s.store.Plans(ctx)
	if err != nil {
		return err
	}
	evaluation := s.scheduler.Evaluate(time.Now(), status, base, plans)
	commands := diffCommands(status, evaluation.Desired)
	for _, command := range commands {
		req := pool.CommandRequest{
			Capability: command.capability,
			State:      command.state,
			Value:      command.value,
			Source:     "scheduler:" + evaluation.Source,
		}
		if _, err := s.ExecuteCommand(ctx, req); err != nil {
			return err
		}
	}
	if len(commands) > 0 {
		_, _ = s.store.AddEvent(ctx, "scheduler", "desired state enforced", evaluation)
	}
	return nil
}

func (s *Service) NextScheduleWake(ctx context.Context, now time.Time, status pool.Status) (time.Time, bool, error) {
	plans, err := s.store.Plans(ctx)
	if err != nil {
		return time.Time{}, false, err
	}
	wake, ok := s.scheduler.NextWake(now, status, plans)
	return wake, ok, nil
}

func (s *Service) requestRefreshAfter(delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	select {
	case s.refreshRequests <- delay:
	default:
	}
}

func (s *Service) recordObservationEvent(ctx context.Context, previous pool.Status, previousOK bool, status pool.Status) {
	now := time.Now().UTC()
	s.statusEventMu.Lock()
	hadError := s.lastStatusError != ""
	heartbeatDue := s.lastStatusEventAt.IsZero() || now.Sub(s.lastStatusEventAt) >= s.eventHeartbeat
	changed := !previousOK || statusMeaningfullyChanged(previous, status)
	shouldRecord := changed || hadError || heartbeatDue
	if shouldRecord {
		s.lastStatusEventAt = now
	}
	s.lastStatusError = ""
	s.lastStatusErrorAt = time.Time{}
	s.statusEventMu.Unlock()

	if !shouldRecord {
		return
	}
	message := "status refreshed"
	if !changed && !hadError {
		message = "status heartbeat"
	}
	_, _ = s.store.AddEvent(ctx, "observation", message, status)
}

func (s *Service) recordStatusError(ctx context.Context, err error) {
	now := time.Now().UTC()
	errText := err.Error()

	s.statusEventMu.Lock()
	heartbeatDue := s.lastStatusErrorAt.IsZero() || now.Sub(s.lastStatusErrorAt) >= s.eventHeartbeat
	shouldRecord := s.lastStatusError != errText || heartbeatDue
	if shouldRecord {
		s.lastStatusError = errText
		s.lastStatusErrorAt = now
	}
	s.statusEventMu.Unlock()

	if shouldRecord {
		_, _ = s.store.AddEvent(ctx, "status_error", "status refresh failed", map[string]string{"error": errText})
	}
}

func statusMeaningfullyChanged(a, b pool.Status) bool {
	if a.Connected != b.Connected ||
		a.Power != b.Power ||
		a.Filter != b.Filter ||
		a.Heater != b.Heater ||
		a.Jets != b.Jets ||
		a.Bubbles != b.Bubbles ||
		a.Sanitizer != b.Sanitizer ||
		a.TargetTemp != b.TargetTemp ||
		a.Unit != b.Unit ||
		a.ErrorCode != b.ErrorCode {
		return true
	}
	return intPointerValue(a.CurrentTemp) != intPointerValue(b.CurrentTemp)
}

func intPointerValue(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

type commandDiff struct {
	capability string
	state      *bool
	value      json.RawMessage
}

func diffCommands(status pool.Status, desired pool.DesiredState) []commandDiff {
	desired = desired.WithHardwareConstraints()
	var commands []commandDiff
	appendBool := func(capability string, current bool, wanted *bool) {
		if wanted != nil && current != *wanted {
			commands = append(commands, commandDiff{capability: capability, state: wanted})
		}
	}

	if desired.Power != nil && *desired.Power && !status.Power {
		appendBool("power", status.Power, desired.Power)
	}
	if desired.Filter != nil && *desired.Filter && !status.Filter {
		appendBool("filter", status.Filter, desired.Filter)
	}
	if desired.TargetTemp != nil && status.TargetTemp != *desired.TargetTemp {
		commands = append(commands, commandDiff{
			capability: "target_temp",
			value:      json.RawMessage(fmt.Sprintf("%d", *desired.TargetTemp)),
		})
	}
	appendBool("heater", status.Heater, desired.Heater)
	appendBool("jets", status.Jets, desired.Jets)
	appendBool("bubbles", status.Bubbles, desired.Bubbles)
	appendBool("sanitizer", status.Sanitizer, desired.Sanitizer)
	if desired.Filter != nil && !*desired.Filter && status.Filter {
		appendBool("filter", status.Filter, desired.Filter)
	}
	if desired.Power != nil && !*desired.Power && status.Power {
		appendBool("power", status.Power, desired.Power)
	}
	return commands
}

func commandValue(capability string, request pool.CommandRequest) (any, error) {
	if capability == "" {
		return nil, fmt.Errorf("command capability is required")
	}
	if capability == "target_temp" {
		if len(request.Value) == 0 {
			return nil, fmt.Errorf("target_temp command requires value")
		}
		return []byte(request.Value), nil
	}
	if request.State == nil {
		return nil, fmt.Errorf("%s command requires state", capability)
	}
	return *request.State, nil
}
