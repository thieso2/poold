package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

type Service struct {
	store                    *store.Store
	client                   intex.PoolClient
	scheduler                *scheduler.Scheduler
	weather                  WeatherProvider
	startedAt                time.Time
	observationRetention     time.Duration
	eventRetention           time.Duration
	eventHeartbeat           time.Duration
	observationFlushInterval time.Duration
	commandConfirmDelay      time.Duration
	manualControlDuration    time.Duration
	heatingRateCPerHour      float64
	coolingRateCPerHour      float64
	pollIdleInterval         time.Duration
	pollStableInterval       time.Duration
	pollActiveInterval       time.Duration
	refreshRequests          chan time.Duration
	commandMu                sync.Mutex
	manualReconcileMu        sync.Mutex
	weatherMu                sync.Mutex
	statusEventMu            sync.Mutex
	lastStatusEventAt        time.Time
	lastStatusError          string
	lastStatusErrorAt        time.Time
	now                      func() time.Time
}

var ErrWeatherNotConfigured = errors.New("weather is not configured")

type manualSessionFailure struct {
	Code          string
	Message       string
	ChangedFields []string
	Violations    []string
	Current       *pool.PoolControlRepresentation
}

func (e *manualSessionFailure) Error() string {
	return e.Message
}

type createManualSessionRequest struct {
	ExpectedRevision  string
	BaseObservationID *int64
	Duration          string
	Intended          pool.ControllableState
	IdempotencyKey    string
	Fingerprint       string
}

type retryManualSessionRequest struct {
	ExpectedRevision string
	IdempotencyKey   string
	Fingerprint      string
}

type clearManualSessionRequest struct {
	ExpectedRevision string
	IdempotencyKey   string
	Fingerprint      string
}

type WeatherProvider interface {
	ResolveLocation(context.Context, string, string) (pool.WeatherLocation, error)
	CurrentWeather(context.Context, string, pool.WeatherLocation) (json.RawMessage, error)
}

type WeatherSettingsView struct {
	APIKeySet bool                 `json:"api_key_set"`
	Location  pool.WeatherLocation `json:"location,omitempty"`
	UpdatedAt *time.Time           `json:"updated_at,omitempty"`
}

type ServiceConfig struct {
	ObservationRetention     time.Duration
	EventRetention           time.Duration
	EventHeartbeat           time.Duration
	ObservationFlushInterval time.Duration
	CommandConfirmDelay      time.Duration
	ManualControlDuration    time.Duration
	HeatingRateCPerHour      float64
	CoolingRateCPerHour      float64
	PollIdleInterval         time.Duration
	PollStableInterval       time.Duration
	PollActiveInterval       time.Duration
	WeatherProvider          WeatherProvider
	Now                      func() time.Time
}

func publicWeatherSettings(settings pool.WeatherSettings) WeatherSettingsView {
	var updatedAt *time.Time
	if !settings.UpdatedAt.IsZero() {
		t := settings.UpdatedAt
		updatedAt = &t
	}
	return WeatherSettingsView{
		APIKeySet: strings.TrimSpace(settings.APIKey) != "",
		Location:  settings.Location,
		UpdatedAt: updatedAt,
	}
}

func NewService(st *store.Store, client intex.PoolClient, sched *scheduler.Scheduler, cfg ServiceConfig) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.EventHeartbeat <= 0 {
		cfg.EventHeartbeat = 30 * time.Minute
	}
	if cfg.CommandConfirmDelay <= 0 {
		cfg.CommandConfirmDelay = 10 * time.Second
	}
	if cfg.ManualControlDuration <= 0 {
		cfg.ManualControlDuration = 2 * time.Hour
	}
	if cfg.HeatingRateCPerHour <= 0 {
		cfg.HeatingRateCPerHour = 0.75
	}
	if cfg.CoolingRateCPerHour <= 0 {
		cfg.CoolingRateCPerHour = 0.10
	}
	if cfg.PollIdleInterval <= 0 {
		cfg.PollIdleInterval = 10 * time.Minute
	}
	if cfg.PollStableInterval <= 0 {
		cfg.PollStableInterval = 5 * time.Minute
	}
	if cfg.PollActiveInterval <= 0 {
		cfg.PollActiveInterval = time.Minute
	}
	return &Service{
		store:                    st,
		client:                   client,
		scheduler:                sched,
		weather:                  cfg.WeatherProvider,
		startedAt:                cfg.Now().UTC(),
		observationRetention:     cfg.ObservationRetention,
		eventRetention:           cfg.EventRetention,
		eventHeartbeat:           cfg.EventHeartbeat,
		observationFlushInterval: cfg.ObservationFlushInterval,
		commandConfirmDelay:      cfg.CommandConfirmDelay,
		manualControlDuration:    cfg.ManualControlDuration,
		heatingRateCPerHour:      cfg.HeatingRateCPerHour,
		coolingRateCPerHour:      cfg.CoolingRateCPerHour,
		pollIdleInterval:         cfg.PollIdleInterval,
		pollStableInterval:       cfg.PollStableInterval,
		pollActiveInterval:       cfg.PollActiveInterval,
		refreshRequests:          make(chan time.Duration, 16),
		now:                      cfg.Now,
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
	if _, err := s.store.SaveObservationThrottled(ctx, status, s.observationFlushInterval); err != nil {
		return pool.Status{}, err
	}
	s.recordObservationEvent(ctx, previous, previousOK, status)
	if session, sessionErr := s.store.ManualSession(ctx); sessionErr == nil && session != nil {
		drifted, driftErr := s.store.MarkManualSessionDrift(ctx, session.Revision, status)
		if driftErr == nil && drifted {
			go s.reconcileManualSession(context.Background(), session.Revision, session.Intended)
		}
	}
	_ = s.store.Prune(ctx, s.observationRetention, s.eventRetention)
	_ = s.Enforce(ctx, status)
	return status, nil
}

func (s *Service) ExecuteCommand(ctx context.Context, request pool.CommandRequest) (pool.CommandRecord, error) {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()
	return s.executeCommandLocked(ctx, request)
}

func (s *Service) executeCommandLocked(ctx context.Context, request pool.CommandRequest) (pool.CommandRecord, error) {
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

// PoolControl returns the current ownership and latest observed pool state.
func (s *Service) PoolControl(ctx context.Context) (pool.PoolControlRepresentation, error) {
	session, err := s.store.ManualSession(ctx)
	if err != nil {
		return pool.PoolControlRepresentation{}, err
	}
	now := s.now().UTC()
	if session != nil && session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		result, err := s.expireManualSession(ctx, session.Revision, now)
		if err != nil {
			return pool.PoolControlRepresentation{}, err
		}
		if result.Expired {
			go func() {
				_ = s.EnforceLatest(context.Background())
			}()
		}
		return result.Representation, nil
	}
	return s.store.PoolControlRepresentation(ctx)
}

func controlObservation(id int64, status pool.Status) *pool.ControlObservation {
	return &pool.ControlObservation{
		ObservationID: id,
		ObservedAt:    status.ObservedAt,
		Connected:     status.Connected,
		State:         pool.ControllableStateFromStatus(status),
	}
}

func (s *Service) createManualSession(ctx context.Context, request createManualSessionRequest) (store.CreateManualSessionResult, error) {
	status, representation, found, err := s.store.ManualSessionMutationReplay(
		ctx, request.IdempotencyKey, request.Fingerprint,
	)
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.CreateManualSessionResult{}, &manualSessionFailure{
			Code:    "idempotency_key_reused",
			Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.CreateManualSessionResult{}, err
	}
	if found {
		return store.CreateManualSessionResult{
			Status:         status,
			Representation: representation,
			Replayed:       true,
		}, nil
	}

	current, err := s.PoolControl(ctx)
	if err != nil {
		return store.CreateManualSessionResult{}, err
	}
	if current.ControlRevision != request.ExpectedRevision {
		return store.CreateManualSessionResult{}, &manualSessionFailure{
			Code: "control_changed", Message: "Control changed after this draft was opened.", Current: &current,
		}
	}
	var observed *pool.ControlObservation
	if current.Session == nil {
		if request.BaseObservationID == nil || *request.BaseObservationID <= 0 {
			return store.CreateManualSessionResult{}, &manualSessionFailure{
				Code: "invalid_request", Message: "base_observation_id is required.",
			}
		}
		base, ok, err := s.store.ObservationStatus(ctx, *request.BaseObservationID)
		if err != nil {
			return store.CreateManualSessionResult{}, err
		}
		if !ok {
			return store.CreateManualSessionResult{}, &manualSessionFailure{
				Code: "invalid_request", Message: "base_observation_id does not identify an observation.",
			}
		}
		fresh, err := s.client.Status(ctx)
		if err != nil {
			return store.CreateManualSessionResult{}, &manualSessionFailure{
				Code: "pool_unreachable", Message: "The pool could not be reached for a fresh status read.",
			}
		}
		if fresh.ObservedAt.IsZero() {
			fresh.ObservedAt = time.Now().UTC()
		}
		fresh.Connected = true
		observationID, err := s.store.SaveObservation(ctx, fresh)
		if err != nil {
			return store.CreateManualSessionResult{}, err
		}
		changedFields := changedControllableFields(pool.ControllableStateFromStatus(base), pool.ControllableStateFromStatus(fresh))
		if len(changedFields) != 0 {
			latest, currentErr := s.PoolControl(ctx)
			if currentErr != nil {
				return store.CreateManualSessionResult{}, currentErr
			}
			return store.CreateManualSessionResult{}, &manualSessionFailure{
				Code: "observed_state_changed", Message: "The pool changed after this draft was opened.",
				ChangedFields: changedFields, Current: &latest,
			}
		}
		observed = controlObservation(observationID, fresh)
	} else {
		if request.BaseObservationID != nil {
			return store.CreateManualSessionResult{}, &manualSessionFailure{
				Code: "invalid_request", Message: "base_observation_id must be omitted when editing a Manual session.",
			}
		}
		observed = current.Observed
	}

	startedAt := s.now().UTC()
	var expiresAt *time.Time
	switch request.Duration {
	case "30m":
		value := startedAt.Add(30 * time.Minute)
		expiresAt = &value
	case "60m":
		value := startedAt.Add(time.Hour)
		expiresAt = &value
	case "2h":
		value := startedAt.Add(2 * time.Hour)
		expiresAt = &value
	case "until_off":
	default:
		return store.CreateManualSessionResult{}, &manualSessionFailure{
			Code:    "invalid_request",
			Message: "duration must be 30m, 60m, 2h, or until_off.",
		}
	}
	observedState := pool.ControllableState{}
	if observed != nil {
		observedState = observed.State
	}
	outcomes := initialManualSessionOutcomes(observedState, request.Intended)
	s.commandMu.Lock()
	result, err := s.store.CreateManualSession(ctx, store.CreateManualSessionParams{
		ExpectedRevision: request.ExpectedRevision,
		IdempotencyKey:   request.IdempotencyKey,
		Fingerprint:      request.Fingerprint,
		Duration:         request.Duration,
		StartedAt:        startedAt,
		ExpiresAt:        expiresAt,
		Intended:         request.Intended,
		Outcomes:         outcomes,
		Observed:         observed,
	})
	s.commandMu.Unlock()
	if errors.Is(err, store.ErrControlChanged) {
		current, currentErr := s.PoolControl(ctx)
		if currentErr != nil {
			return store.CreateManualSessionResult{}, currentErr
		}
		return store.CreateManualSessionResult{}, &manualSessionFailure{
			Code:    "control_changed",
			Message: "Control changed after this draft was opened.",
			Current: &current,
		}
	}
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.CreateManualSessionResult{}, &manualSessionFailure{
			Code:    "idempotency_key_reused",
			Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.CreateManualSessionResult{}, err
	}
	if !result.Replayed && result.Representation.Session != nil &&
		result.Representation.Session.State == "applying" {
		revision := result.Representation.ControlRevision
		intended := result.Representation.Session.Intended
		go s.reconcileManualSession(context.Background(), revision, intended)
	}
	return result, nil
}

func (s *Service) retryManualSession(ctx context.Context, request retryManualSessionRequest) (store.RetryManualSessionResult, error) {
	status, representation, found, err := s.store.ManualSessionMutationReplay(
		ctx, request.IdempotencyKey, request.Fingerprint,
	)
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.RetryManualSessionResult{}, &manualSessionFailure{
			Code: "idempotency_key_reused", Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.RetryManualSessionResult{}, err
	}
	if found {
		return store.RetryManualSessionResult{
			Status: status, Representation: representation, Replayed: true,
		}, nil
	}

	current, err := s.PoolControl(ctx)
	if err != nil {
		return store.RetryManualSessionResult{}, err
	}
	if current.ControlRevision != request.ExpectedRevision || current.Session == nil {
		return store.RetryManualSessionResult{}, &manualSessionFailure{
			Code: "control_changed", Message: "Control changed after this draft was opened.", Current: &current,
		}
	}
	result, err := s.store.RetryManualSession(ctx, store.RetryManualSessionParams{
		ExpectedRevision: request.ExpectedRevision,
		IdempotencyKey:   request.IdempotencyKey,
		Fingerprint:      request.Fingerprint,
		RetriedAt:        s.now().UTC(),
	})
	var changed *store.ControlChangedError
	if errors.As(err, &changed) {
		return store.RetryManualSessionResult{}, &manualSessionFailure{
			Code: "control_changed", Message: "Control changed after this draft was opened.", Current: &changed.Current,
		}
	}
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.RetryManualSessionResult{}, &manualSessionFailure{
			Code: "idempotency_key_reused", Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.RetryManualSessionResult{}, err
	}
	if !result.Replayed && result.Representation.Session != nil &&
		result.Representation.Session.State == "applying" {
		go s.reconcileManualSession(context.Background(), result.Representation.ControlRevision, result.Representation.Session.Intended)
	}
	return result, nil
}

func (s *Service) clearManualSession(ctx context.Context, request clearManualSessionRequest) (store.ClearManualSessionResult, error) {
	status, representation, found, err := s.store.ManualSessionMutationReplay(
		ctx, request.IdempotencyKey, request.Fingerprint,
	)
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.ClearManualSessionResult{}, &manualSessionFailure{
			Code: "idempotency_key_reused", Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.ClearManualSessionResult{}, err
	}
	if found {
		return store.ClearManualSessionResult{
			Status: status, Representation: representation, Replayed: true,
		}, nil
	}

	current, err := s.PoolControl(ctx)
	if err != nil {
		return store.ClearManualSessionResult{}, err
	}
	s.commandMu.Lock()
	result, err := s.store.ClearManualSession(ctx, store.ClearManualSessionParams{
		ExpectedRevision: request.ExpectedRevision,
		IdempotencyKey:   request.IdempotencyKey,
		Fingerprint:      request.Fingerprint,
		ClearedAt:        s.now().UTC(),
		Observed:         current.Observed,
	})
	s.commandMu.Unlock()
	var changed *store.ControlChangedError
	if errors.As(err, &changed) {
		return store.ClearManualSessionResult{}, &manualSessionFailure{
			Code:    "control_changed",
			Message: "Control changed after this draft was opened.",
			Current: &changed.Current,
		}
	}
	if errors.Is(err, store.ErrControlChanged) {
		current, currentErr := s.PoolControl(ctx)
		if currentErr != nil {
			return store.ClearManualSessionResult{}, currentErr
		}
		return store.ClearManualSessionResult{}, &manualSessionFailure{
			Code:    "control_changed",
			Message: "Control changed after this draft was opened.",
			Current: &current,
		}
	}
	if errors.Is(err, store.ErrIdempotencyKeyReused) {
		return store.ClearManualSessionResult{}, &manualSessionFailure{
			Code:    "idempotency_key_reused",
			Message: "The idempotency key was already used for a different operation.",
		}
	}
	if err != nil {
		return store.ClearManualSessionResult{}, err
	}
	if !result.Replayed {
		go func() {
			_ = s.EnforceLatest(context.Background())
		}()
	}
	return result, nil
}

func changedControllableFields(base, fresh pool.ControllableState) []string {
	var changed []string
	for _, field := range pool.ControllableFields {
		if !base.FieldEqual(field, fresh) {
			changed = append(changed, field)
		}
	}
	return changed
}

func initialManualSessionOutcomes(observed, intended pool.ControllableState) map[string]pool.CommandOutcome {
	outcomes := make(map[string]pool.CommandOutcome, len(pool.ControllableFields))
	pending := false
	for _, field := range pool.ControllableFields {
		state := "pending"
		if observed.FieldEqual(field, intended) {
			state = "confirmed"
		} else {
			pending = true
		}
		outcomes[field] = pool.CommandOutcome{State: state}
	}
	// Ownership always becomes visibly applying before the asynchronous worker
	// confirms a no-op intent from its own fresh reconciliation read.
	if !pending {
		outcomes["power"] = pool.CommandOutcome{State: "pending"}
	}
	return outcomes
}

func (s *Service) SaveDesiredState(ctx context.Context, desired pool.DesiredState) error {
	if err := s.store.SaveDesiredState(ctx, desired); err != nil {
		return err
	}
	_, _ = s.store.AddEvent(ctx, "desired_state", "desired state updated", desired)
	s.requestRefreshAfter(0)
	return nil
}

func (s *Service) ControlMode(ctx context.Context) (pool.ControlMode, error) {
	return s.controlMode(ctx, time.Now())
}

func (s *Service) SaveControlMode(ctx context.Context, mode pool.ControlMode) error {
	now := time.Now().UTC()
	if mode.ManualControl {
		if mode.ExpiresAt == nil {
			expiresAt := s.defaultManualControlExpiresAt(ctx, now)
			mode.ExpiresAt = &expiresAt
		} else if !mode.ExpiresAt.After(now) {
			mode.ManualControl = false
			mode.ExpiresAt = nil
		}
	} else {
		mode.ExpiresAt = nil
	}
	if err := s.store.SaveControlMode(ctx, mode); err != nil {
		return err
	}
	_, _ = s.store.AddEvent(ctx, "control_mode", "control mode updated", mode)
	s.requestRefreshAfter(0)
	return nil
}

func (s *Service) defaultManualControlExpiresAt(ctx context.Context, now time.Time) time.Time {
	expiresAt := now.Add(s.manualControlDuration)
	status, ok, err := s.store.LatestStatus(ctx)
	if err != nil || !ok {
		return expiresAt
	}
	plans, err := s.store.Plans(ctx)
	if err != nil {
		return expiresAt
	}
	if wake, ok := s.scheduler.NextWake(now, status, plans); ok && wake.After(now) && wake.Before(expiresAt) {
		return wake.UTC()
	}
	return expiresAt
}

func (s *Service) controlMode(ctx context.Context, now time.Time) (pool.ControlMode, error) {
	mode, err := s.store.ControlMode(ctx)
	if err != nil {
		return pool.ControlMode{}, err
	}
	if mode.ManualControl && mode.ExpiresAt != nil && !mode.ExpiresAt.After(now) {
		cleared := pool.ControlMode{ManualControl: false}
		if err := s.store.SaveControlMode(ctx, cleared); err != nil {
			return pool.ControlMode{}, err
		}
		_, _ = s.store.AddEvent(ctx, "control_mode", "manual control expired", cleared)
		return cleared, nil
	}
	return mode, nil
}

func (s *Service) WeatherSettings(ctx context.Context) (pool.WeatherSettings, error) {
	return s.store.WeatherSettings(ctx)
}

func (s *Service) SaveWeatherSettings(ctx context.Context, settings pool.WeatherSettings) (pool.WeatherSettings, error) {
	settings.APIKey = strings.TrimSpace(settings.APIKey)
	settings.Location.Query = strings.TrimSpace(settings.Location.Query)
	if settings.APIKey != "" && settings.Location.Query != "" {
		if s.weather == nil {
			return pool.WeatherSettings{}, fmt.Errorf("weather provider is not configured")
		}
		location, err := s.weather.ResolveLocation(ctx, settings.APIKey, settings.Location.Query)
		if err != nil {
			return pool.WeatherSettings{}, err
		}
		settings.Location = location
	}
	settings.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveWeatherSettings(ctx, settings); err != nil {
		return pool.WeatherSettings{}, err
	}
	_, _ = s.store.AddEvent(ctx, "weather_settings", "weather settings updated", publicWeatherSettings(settings))
	return settings, nil
}

func (s *Service) LatestWeatherObservation(ctx context.Context) (pool.WeatherObservation, bool, error) {
	return s.store.LatestWeatherObservation(ctx)
}

func (s *Service) RefreshWeather(ctx context.Context) (pool.WeatherObservation, error) {
	if s.weather == nil {
		return pool.WeatherObservation{}, ErrWeatherNotConfigured
	}
	settings, err := s.store.WeatherSettings(ctx)
	if err != nil {
		return pool.WeatherObservation{}, err
	}
	if !settings.Configured() {
		return pool.WeatherObservation{}, ErrWeatherNotConfigured
	}

	s.weatherMu.Lock()
	defer s.weatherMu.Unlock()

	raw, err := s.weather.CurrentWeather(ctx, settings.APIKey, settings.Location)
	if err != nil {
		return pool.WeatherObservation{}, err
	}
	observation := pool.WeatherObservation{
		ObservedAt: time.Now().UTC(),
		Location:   settings.Location,
		Data:       raw,
	}
	id, err := s.store.SaveWeatherObservation(ctx, observation)
	if err != nil {
		return pool.WeatherObservation{}, err
	}
	observation.ID = id
	return observation, nil
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

func (s *Service) EventsPage(ctx context.Context, query store.EventQuery) ([]pool.Event, error) {
	return s.store.EventsPage(ctx, query)
}

func (s *Service) Observations(ctx context.Context, afterID int64, limit int) ([]pool.Observation, error) {
	return s.store.Observations(ctx, afterID, limit)
}

func (s *Service) LatestObservations(ctx context.Context, limit int) ([]pool.Observation, error) {
	return s.store.LatestObservations(ctx, limit)
}

func (s *Service) LatestObservationsPage(ctx context.Context, limit, offset int) ([]pool.Observation, error) {
	return s.store.LatestObservationsPage(ctx, limit, offset)
}

func (s *Service) Commands(ctx context.Context, afterID int64, limit int) ([]pool.CommandRecord, error) {
	return s.store.Commands(ctx, afterID, limit)
}

func (s *Service) LatestCommands(ctx context.Context, limit, offset int) ([]pool.CommandRecord, error) {
	return s.store.LatestCommands(ctx, limit, offset)
}

func (s *Service) LatestHeatingSessions(ctx context.Context, limit, offset int) ([]pool.HeatingSession, error) {
	return s.store.LatestHeatingSessions(ctx, limit, offset)
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

// EstablishControl restores durable ownership before the polling loop can run schedules.
func (s *Service) EstablishControl(ctx context.Context) error {
	session, err := s.store.ManualSession(ctx)
	if err != nil || session == nil {
		return err
	}
	now := s.now().UTC()
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		result, err := s.expireManualSession(ctx, session.Revision, now)
		if err != nil {
			return err
		}
		if result.Expired {
			return s.EnforceLatest(ctx)
		}
		return nil
	}
	recovered, err := s.store.RecoverManualSession(ctx, session.Revision, now)
	if err != nil || !recovered {
		return err
	}
	return s.reconcileManualSession(ctx, session.Revision, session.Intended)
}

func (s *Service) expireManualSession(ctx context.Context, revision string, now time.Time) (store.ExpireManualSessionResult, error) {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()
	return s.store.ExpireManualSession(ctx, revision, now)
}

func (s *Service) Enforce(ctx context.Context, status pool.Status) error {
	session, err := s.store.ManualSession(ctx)
	if err != nil {
		return err
	}
	if session != nil {
		now := s.now().UTC()
		if session.ExpiresAt == nil || session.ExpiresAt.After(now) {
			return nil
		}
		result, err := s.expireManualSession(ctx, session.Revision, now)
		if err != nil {
			return err
		}
		if !result.Expired {
			return nil
		}
	}
	now := s.now()
	mode, err := s.controlMode(ctx, now)
	if err != nil {
		return err
	}
	if mode.Active(now) {
		return nil
	}
	automatic, err := s.store.PoolControlRepresentation(ctx)
	if err != nil {
		return err
	}
	if automatic.Control != pool.AutomaticControl {
		return nil
	}
	automaticRevision := automatic.ControlRevision

	base, err := s.store.DesiredState(ctx)
	if err != nil {
		return err
	}
	plans, err := s.store.Plans(ctx)
	if err != nil {
		return err
	}
	states, err := s.store.ReadyByControlStates(ctx)
	if err != nil {
		return err
	}
	evaluation := s.scheduler.EvaluateWithReadyByControl(now, status, base, plans, states)
	if evaluation.ReadyByControl != nil {
		if err := s.store.SaveReadyByControlState(ctx, evaluation.ReadyByControl.Current); err != nil {
			return err
		}
		_, _ = s.store.AddEvent(ctx, "scheduler", "ready-by control state changed", evaluation.ReadyByControl)
	}
	commands := diffCommands(status, evaluation.Desired)
	for _, command := range commands {
		req := pool.CommandRequest{
			Capability: command.capability,
			State:      command.state,
			Value:      command.value,
			Source:     "scheduler:" + evaluation.Source,
		}
		executed, err := s.executeAutomaticCommand(ctx, automaticRevision, req)
		if err != nil {
			return err
		}
		if !executed {
			return nil
		}
	}
	if len(commands) > 0 {
		_, _ = s.store.AddEvent(ctx, "scheduler", "desired state enforced", evaluation)
	}
	return nil
}

func (s *Service) executeAutomaticCommand(ctx context.Context, revision string, request pool.CommandRequest) (bool, error) {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()
	owns, err := s.store.AutomaticRevisionCurrent(ctx, revision)
	if err != nil || !owns {
		return false, err
	}
	_, err = s.executeCommandLocked(ctx, request)
	return true, err
}

func (s *Service) NextScheduleWake(ctx context.Context, now time.Time, status pool.Status) (time.Time, bool, error) {
	session, err := s.store.ManualSession(ctx)
	if err != nil {
		return time.Time{}, false, err
	}
	if session != nil {
		if session.ExpiresAt != nil {
			return *session.ExpiresAt, true, nil
		}
		return time.Time{}, false, nil
	}
	mode, err := s.controlMode(ctx, now)
	if err != nil {
		return time.Time{}, false, err
	}
	if mode.Active(now) {
		if mode.ExpiresAt != nil {
			return *mode.ExpiresAt, true, nil
		}
		return time.Time{}, false, nil
	}

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

func (s *Service) reconcileManualSession(ctx context.Context, revision string, intended pool.ControllableState) error {
	s.manualReconcileMu.Lock()
	defer s.manualReconcileMu.Unlock()
	for _, field := range manualReconciliationOrder(intended) {
		current, err := s.store.ManualSession(ctx)
		if err != nil || current == nil || current.Revision != revision {
			return err
		}
		if current.Outcomes[field].State != "pending" {
			continue
		}

		status, err := s.client.Status(ctx)
		if err != nil {
			_ = s.store.FailManualSessionOutcome(ctx, revision, field, "pool_unreachable", "The pool could not be refreshed before this command.")
			continue
		}
		status.Connected = true
		if status.ObservedAt.IsZero() {
			status.ObservedAt = time.Now().UTC()
		}
		if _, err := s.store.SaveObservation(ctx, status); err != nil {
			_ = s.store.FailManualSessionOutcome(ctx, revision, field, "persistence_error", "The refreshed pool status could not be stored.")
			continue
		}
		if err := s.store.ConfirmManualSessionStatus(ctx, revision, status); err != nil {
			return err
		}
		current, err = s.store.ManualSession(ctx)
		if err != nil || current == nil || current.Revision != revision {
			return err
		}
		if current.Outcomes[field].State != "pending" {
			continue
		}
		if dependency := unmetManualDependency(field, intended, status); dependency != "" {
			_ = s.store.FailManualSessionOutcome(ctx, revision, field, "dependency_not_met", dependency)
			continue
		}
		request := manualSessionCommand(field, intended)
		request.Source = "manual_session:" + revision
		executed, expired, err := s.executeManualCommand(ctx, revision, request)
		if err != nil {
			_ = s.store.FailManualSessionOutcome(ctx, revision, field, "command_failed", err.Error())
			continue
		}
		if expired {
			result, expireErr := s.expireManualSession(ctx, revision, s.now().UTC())
			if expireErr != nil {
				return expireErr
			}
			if result.Expired {
				go func() {
					_ = s.EnforceLatest(context.Background())
				}()
			}
			return nil
		}
		if !executed {
			return nil
		}
		resultStatus, ok, err := s.store.LatestStatus(ctx)
		if err != nil {
			return err
		}
		if ok {
			_ = s.store.ConfirmManualSessionStatus(ctx, revision, resultStatus)
		}
	}
	return nil
}

func (s *Service) executeManualCommand(ctx context.Context, revision string, request pool.CommandRequest) (executed, expired bool, err error) {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()
	current, err := s.store.ManualSession(ctx)
	if err != nil || current == nil || current.Revision != revision {
		return false, false, err
	}
	if current.ExpiresAt != nil && !current.ExpiresAt.After(s.now().UTC()) {
		return false, true, nil
	}
	if _, err := s.executeCommandLocked(ctx, request); err != nil {
		return true, false, err
	}
	return true, false, nil
}

func manualReconciliationOrder(intended pool.ControllableState) []string {
	var fields []string
	appendIf := func(condition bool, field string) {
		if condition {
			fields = append(fields, field)
		}
	}
	appendIf(!intended.Heater, "heater")
	appendIf(!intended.Jets, "jets")
	appendIf(!intended.Bubbles, "bubbles")
	appendIf(intended.Power, "power")
	appendIf(intended.Filter, "filter")
	fields = append(fields, "target_temp")
	appendIf(intended.Heater, "heater")
	appendIf(intended.Jets, "jets")
	appendIf(intended.Bubbles, "bubbles")
	appendIf(!intended.Filter, "filter")
	appendIf(!intended.Power, "power")
	return fields
}

func manualSessionCommand(field string, intended pool.ControllableState) pool.CommandRequest {
	request := pool.CommandRequest{Capability: field}
	switch field {
	case "power":
		request.State = pool.BoolPtr(intended.Power)
	case "filter":
		request.State = pool.BoolPtr(intended.Filter)
	case "heater":
		request.State = pool.BoolPtr(intended.Heater)
	case "jets":
		request.State = pool.BoolPtr(intended.Jets)
	case "bubbles":
		request.State = pool.BoolPtr(intended.Bubbles)
	case "target_temp":
		request.Value = json.RawMessage(fmt.Sprintf("%d", intended.TargetTemp))
	}
	return request
}

func unmetManualDependency(field string, intended pool.ControllableState, status pool.Status) string {
	enabling := false
	switch field {
	case "power":
		enabling = intended.Power
	case "filter":
		enabling = intended.Filter
	case "heater":
		enabling = intended.Heater
	case "jets":
		enabling = intended.Jets
	case "bubbles":
		enabling = intended.Bubbles
	}
	if enabling && field != "power" && !status.Power {
		return "Power must be on before enabling " + field + "."
	}
	if field == "heater" && intended.Heater && !status.Filter {
		return "Filter must be on before enabling heater."
	}
	return ""
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
