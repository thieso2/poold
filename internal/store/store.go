package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pooly/services/poold/internal/pool"

	_ "github.com/ncruces/go-sqlite3/driver"
)

type Store struct {
	db                 *sql.DB
	observationMu      sync.Mutex
	pendingObservation *pendingObservation
}

const weatherSnapshotMaxAge = 15 * time.Minute
const readyByControlKeyPrefix = "ready_by_control:"

type pendingObservation struct {
	id          int64
	lastFlushAt time.Time
	countDelta  int
	body        []byte
	status      pool.Status
	weather     *pool.WeatherSnapshot
}

type latestObservation struct {
	id             int64
	lastObservedAt time.Time
	body           []byte
	status         pool.Status
}

type compactObservationRow struct {
	id     int64
	first  string
	last   string
	count  int
	body   []byte
	status pool.Status
	dirty  bool
}

type EventQuery struct {
	AfterID     int64
	Limit       int
	Offset      int
	Latest      bool
	Type        string
	ChangesOnly bool
}

type heatingObservationRow struct {
	id          int64
	first       time.Time
	last        time.Time
	count       int
	heater      bool
	currentTemp *int
	targetTemp  int
}

func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	_ = s.FlushObservations(context.Background())
	return s.db.Close()
}

func (s *Store) SaveObservation(ctx context.Context, status pool.Status) (int64, error) {
	return s.saveObservation(ctx, status, 0)
}

func (s *Store) SaveObservationThrottled(ctx context.Context, status pool.Status, flushInterval time.Duration) (int64, error) {
	return s.saveObservation(ctx, status, flushInterval)
}

func (s *Store) FlushObservations(ctx context.Context) error {
	s.observationMu.Lock()
	defer s.observationMu.Unlock()

	return s.flushPendingObservationLocked(ctx)
}

func (s *Store) saveObservation(ctx context.Context, status pool.Status, flushInterval time.Duration) (int64, error) {
	if status.ObservedAt.IsZero() {
		status.ObservedAt = time.Now().UTC()
	}
	body, err := json.Marshal(status)
	if err != nil {
		return 0, err
	}
	weather := s.weatherSnapshot(ctx, status.ObservedAt)

	s.observationMu.Lock()
	defer s.observationMu.Unlock()

	if pending := s.pendingObservation; pending != nil {
		if flushInterval > 0 && sameObservationState(pending.status, status) {
			pending.countDelta++
			pending.body = body
			pending.status = status
			pending.weather = weather
			if observationFlushDue(pending.lastFlushAt, status.ObservedAt, flushInterval) {
				if err := s.flushPendingObservationLocked(ctx); err != nil {
					return 0, err
				}
			}
			return pending.id, nil
		}
		if err := s.flushPendingObservationLocked(ctx); err != nil {
			return 0, err
		}
	}

	latest, ok, err := s.latestObservationLocked(ctx)
	if err != nil {
		return 0, err
	}
	if ok && sameObservationState(latest.status, status) {
		if flushInterval > 0 && !observationFlushDue(latest.lastObservedAt, status.ObservedAt, flushInterval) {
			s.pendingObservation = &pendingObservation{
				id:          latest.id,
				lastFlushAt: latest.lastObservedAt,
				countDelta:  1,
				body:        body,
				status:      status,
				weather:     weather,
			}
			return latest.id, nil
		}
		if err := s.updateObservationSpanLocked(ctx, latest.id, status, body, weather, 1); err != nil {
			return 0, err
		}
		return latest.id, nil
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO observations (
			observed_at,
			last_observed_at,
			observation_count,
			status_json,
			connected,
			power,
			filter,
			heater,
			jets,
			bubbles,
			sanitizer,
			current_temp,
			target_temp,
			unit,
			error_code,
			raw_data,
			weather_observation_id,
			outside_temp_c,
			clouds_percent,
			weather_main,
			weather_description,
			wind_speed,
			weather_age_seconds
		) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, observationValues(status, body, weather, 0, true)...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) latestObservationLocked(ctx context.Context) (latestObservation, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(last_observed_at, observed_at), status_json
		FROM observations
		ORDER BY id DESC
		LIMIT 1
	`)
	var latest latestObservation
	var observedAt string
	if err := row.Scan(&latest.id, &observedAt, &latest.body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return latestObservation{}, false, nil
		}
		return latestObservation{}, false, err
	}
	if err := json.Unmarshal(latest.body, &latest.status); err != nil {
		return latestObservation{}, false, err
	}
	latest.lastObservedAt = decodeTime(observedAt)
	latest.status.ObservedAt = latest.lastObservedAt
	return latest, true, nil
}

func (s *Store) updateObservationSpanLocked(ctx context.Context, id int64, status pool.Status, body []byte, weather *pool.WeatherSnapshot, countDelta int) error {
	values := observationUpdateValues(status, body, weather, id, countDelta)
	_, err := s.db.ExecContext(ctx, `
		UPDATE observations
		SET last_observed_at = ?,
			observation_count = COALESCE(observation_count, 1) + ?,
			status_json = ?,
			connected = ?,
			power = ?,
			filter = ?,
			heater = ?,
			jets = ?,
			bubbles = ?,
			sanitizer = ?,
			current_temp = ?,
			target_temp = ?,
			unit = ?,
			error_code = ?,
			raw_data = ?,
			weather_observation_id = ?,
			outside_temp_c = ?,
			clouds_percent = ?,
			weather_main = ?,
			weather_description = ?,
			wind_speed = ?,
			weather_age_seconds = ?
		WHERE id = ?
	`, values...)
	return err
}

func (s *Store) flushPendingObservationLocked(ctx context.Context) error {
	pending := s.pendingObservation
	if pending == nil {
		return nil
	}
	if err := s.updateObservationSpanLocked(ctx, pending.id, pending.status, pending.body, pending.weather, pending.countDelta); err != nil {
		return err
	}
	s.pendingObservation = nil
	return nil
}

func observationFlushDue(lastFlushAt, observedAt time.Time, flushInterval time.Duration) bool {
	if flushInterval <= 0 || lastFlushAt.IsZero() || observedAt.Before(lastFlushAt) {
		return true
	}
	return !observedAt.Before(lastFlushAt.Add(flushInterval))
}

func (s *Store) LatestStatus(ctx context.Context) (pool.Status, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(last_observed_at, observed_at), status_json
		FROM observations
		ORDER BY id DESC
		LIMIT 1
	`)
	var id int64
	var observedAt string
	var body []byte
	if err := row.Scan(&id, &observedAt, &body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pool.Status{}, false, nil
		}
		return pool.Status{}, false, err
	}
	var status pool.Status
	if err := json.Unmarshal(body, &status); err != nil {
		return pool.Status{}, false, err
	}
	status.ObservedAt = decodeTime(observedAt)
	s.observationMu.Lock()
	if pending := s.pendingObservation; pending != nil && pending.id == id {
		status = pending.status
	}
	s.observationMu.Unlock()
	return status, true, nil
}

func (s *Store) Observations(ctx context.Context, afterID int64, limit int) ([]pool.Observation, error) {
	limit = normalizedListLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,
			observed_at,
			COALESCE(last_observed_at, observed_at),
			COALESCE(observation_count, 1),
			status_json,
			weather_observation_id,
			outside_temp_c,
			clouds_percent,
			weather_main,
			weather_description,
			wind_speed,
			weather_age_seconds
		FROM observations
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanObservations(rows)
}

func (s *Store) LatestObservations(ctx context.Context, limit int) ([]pool.Observation, error) {
	return s.LatestObservationsPage(ctx, limit, 0)
}

func (s *Store) LatestObservationsPage(ctx context.Context, limit, offset int) ([]pool.Observation, error) {
	limit = normalizedListLimit(limit)
	offset = normalizedListOffset(offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,
			observed_at,
			COALESCE(last_observed_at, observed_at),
			COALESCE(observation_count, 1),
			status_json,
			weather_observation_id,
			outside_temp_c,
			clouds_percent,
			weather_main,
			weather_description,
			wind_speed,
			weather_age_seconds
		FROM observations
		ORDER BY id DESC
		LIMIT ?
		OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanObservations(rows)
}

func (s *Store) ObservationsRange(ctx context.Context, from, to time.Time) ([]pool.Observation, error) {
	var observations []pool.Observation

	priorRows, err := s.db.QueryContext(ctx, `
		SELECT id,
			observed_at,
			COALESCE(last_observed_at, observed_at),
			COALESCE(observation_count, 1),
			status_json,
			weather_observation_id,
			outside_temp_c,
			clouds_percent,
			weather_main,
			weather_description,
			wind_speed,
			weather_age_seconds
		FROM observations
		WHERE COALESCE(last_observed_at, observed_at) < ?
		ORDER BY COALESCE(last_observed_at, observed_at) DESC
		LIMIT 1
	`, encodeTime(from))
	if err != nil {
		return nil, err
	}
	prior, err := scanObservations(priorRows)
	priorRows.Close()
	if err != nil {
		return nil, err
	}
	observations = append(observations, prior...)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id,
			observed_at,
			COALESCE(last_observed_at, observed_at),
			COALESCE(observation_count, 1),
			status_json,
			weather_observation_id,
			outside_temp_c,
			clouds_percent,
			weather_main,
			weather_description,
			wind_speed,
			weather_age_seconds
		FROM observations
		WHERE COALESCE(last_observed_at, observed_at) >= ?
			AND observed_at <= ?
		ORDER BY observed_at ASC
	`, encodeTime(from), encodeTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inRange, err := scanObservations(rows)
	if err != nil {
		return nil, err
	}
	observations = append(observations, inRange...)
	s.applyPendingObservation(observations)
	return observations, nil
}

func (s *Store) applyPendingObservation(observations []pool.Observation) {
	s.observationMu.Lock()
	defer s.observationMu.Unlock()

	pending := s.pendingObservation
	if pending == nil {
		return
	}
	for i := range observations {
		if observations[i].ID != pending.id {
			continue
		}
		observations[i].LastObservedAt = pending.status.ObservedAt
		observations[i].Status = pending.status
		observations[i].Weather = pending.weather
		observations[i].ObservationCount += pending.countDelta
		return
	}
}

func scanObservations(rows *sql.Rows) ([]pool.Observation, error) {
	var observations []pool.Observation
	for rows.Next() {
		var observation pool.Observation
		var firstObservedAt, lastObservedAt string
		var body []byte
		var weatherID sql.NullInt64
		var outsideTemp sql.NullFloat64
		var clouds sql.NullInt64
		var weatherMain sql.NullString
		var weatherDescription sql.NullString
		var windSpeed sql.NullFloat64
		var weatherAgeSeconds sql.NullInt64
		if err := rows.Scan(
			&observation.ID,
			&firstObservedAt,
			&lastObservedAt,
			&observation.ObservationCount,
			&body,
			&weatherID,
			&outsideTemp,
			&clouds,
			&weatherMain,
			&weatherDescription,
			&windSpeed,
			&weatherAgeSeconds,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &observation.Status); err != nil {
			return nil, err
		}
		observation.FirstObservedAt = decodeTime(firstObservedAt)
		observation.LastObservedAt = decodeTime(lastObservedAt)
		observation.Status.ObservedAt = observation.LastObservedAt
		if observation.ObservationCount <= 0 {
			observation.ObservationCount = 1
		}
		if weatherID.Valid {
			observation.Weather = &pool.WeatherSnapshot{
				ObservationID:     weatherID.Int64,
				Main:              weatherMain.String,
				Description:       weatherDescription.String,
				WeatherAgeSeconds: weatherAgeSeconds.Int64,
			}
			if outsideTemp.Valid {
				v := outsideTemp.Float64
				observation.Weather.OutsideTempC = &v
			}
			if clouds.Valid {
				v := int(clouds.Int64)
				observation.Weather.CloudsPercent = &v
			}
			if windSpeed.Valid {
				v := windSpeed.Float64
				observation.Weather.WindSpeed = &v
			}
		}
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}

func observationValues(status pool.Status, body []byte, weather *pool.WeatherSnapshot, id int64, includeFirstObservedAt bool) []any {
	values := []any{}
	if includeFirstObservedAt {
		values = append(values, encodeTime(status.ObservedAt))
	}
	values = append(values,
		encodeTime(status.ObservedAt),
		body,
		boolInt(status.Connected),
		boolInt(status.Power),
		boolInt(status.Filter),
		boolInt(status.Heater),
		boolInt(status.Jets),
		boolInt(status.Bubbles),
		boolInt(status.Sanitizer),
		intPtrValue(status.CurrentTemp),
		status.TargetTemp,
		nullString(status.Unit),
		nullString(status.ErrorCode),
		nullString(status.RawData),
	)
	if weather == nil {
		values = append(values, nil, nil, nil, nil, nil, nil, nil)
	} else {
		values = append(values,
			weather.ObservationID,
			floatPtrValue(weather.OutsideTempC),
			intPtrValueFromInt(weather.CloudsPercent),
			nullString(weather.Main),
			nullString(weather.Description),
			floatPtrValue(weather.WindSpeed),
			weather.WeatherAgeSeconds,
		)
	}
	if id != 0 {
		values = append(values, id)
	}
	return values
}

func observationUpdateValues(status pool.Status, body []byte, weather *pool.WeatherSnapshot, id int64, countDelta int) []any {
	values := []any{encodeTime(status.ObservedAt), countDelta}
	values = append(values,
		body,
		boolInt(status.Connected),
		boolInt(status.Power),
		boolInt(status.Filter),
		boolInt(status.Heater),
		boolInt(status.Jets),
		boolInt(status.Bubbles),
		boolInt(status.Sanitizer),
		intPtrValue(status.CurrentTemp),
		status.TargetTemp,
		nullString(status.Unit),
		nullString(status.ErrorCode),
		nullString(status.RawData),
	)
	if weather == nil {
		values = append(values, nil, nil, nil, nil, nil, nil, nil)
	} else {
		values = append(values,
			weather.ObservationID,
			floatPtrValue(weather.OutsideTempC),
			intPtrValueFromInt(weather.CloudsPercent),
			nullString(weather.Main),
			nullString(weather.Description),
			floatPtrValue(weather.WindSpeed),
			weather.WeatherAgeSeconds,
		)
	}
	values = append(values, id)
	return values
}

func (s *Store) AddEvent(ctx context.Context, typ, message string, data any) (pool.Event, error) {
	createdAt := time.Now().UTC()
	raw, err := marshalRaw(data)
	if err != nil {
		return pool.Event{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO events (created_at, type, message, data_json)
		VALUES (?, ?, ?, ?)
	`, encodeTime(createdAt), typ, message, raw)
	if err != nil {
		return pool.Event{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return pool.Event{}, err
	}
	return pool.Event{ID: id, CreatedAt: createdAt, Type: typ, Message: message, Data: raw}, nil
}

func (s *Store) Events(ctx context.Context, afterID int64, limit int) ([]pool.Event, error) {
	return s.EventsPage(ctx, EventQuery{AfterID: afterID, Limit: limit})
}

func (s *Store) LatestEvents(ctx context.Context, limit int) ([]pool.Event, error) {
	return s.EventsPage(ctx, EventQuery{Limit: limit, Latest: true})
}

func (s *Store) EventsPage(ctx context.Context, query EventQuery) ([]pool.Event, error) {
	limit := normalizedListLimit(query.Limit)
	offset := normalizedListOffset(query.Offset)
	where := "WHERE 1 = 1"
	args := []any{}
	if !query.Latest {
		where += " AND id > ?"
		args = append(args, query.AfterID)
	}
	if query.Type != "" {
		where += " AND type = ?"
		args = append(args, query.Type)
	}
	if query.ChangesOnly {
		where += " AND NOT (type = 'observation' AND message = 'status heartbeat')"
	}
	order := "ASC"
	if query.Latest {
		order = "DESC"
	}
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, created_at, type, message, data_json
		FROM events
		%s
		ORDER BY id %s
		LIMIT ?
		OFFSET ?
	`, where, order), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *Store) EventsRange(ctx context.Context, from, to time.Time, types []string, limit int) ([]pool.Event, error) {
	limit = normalizedListLimit(limit)
	where := "WHERE created_at >= ? AND created_at <= ?"
	args := []any{encodeTime(from), encodeTime(to)}
	if len(types) > 0 {
		placeholders := make([]string, 0, len(types))
		for _, typ := range types {
			placeholders = append(placeholders, "?")
			args = append(args, typ)
		}
		where += " AND type IN (" + strings.Join(placeholders, ",") + ")"
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, created_at, type, message, data_json
		FROM events
		%s
		ORDER BY created_at ASC, id ASC
		LIMIT ?
	`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func scanEvents(rows *sql.Rows) ([]pool.Event, error) {
	var events []pool.Event
	for rows.Next() {
		var event pool.Event
		var createdAt string
		if err := rows.Scan(&event.ID, &createdAt, &event.Type, &event.Message, &event.Data); err != nil {
			return nil, err
		}
		event.CreatedAt = decodeTime(createdAt)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) Commands(ctx context.Context, afterID int64, limit int) ([]pool.CommandRecord, error) {
	limit = normalizedListLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issued_at, completed_at, capability, state_json, value_json, source, success, error, status_json
		FROM commands
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommands(rows)
}

func (s *Store) LatestCommands(ctx context.Context, limit, offset int) ([]pool.CommandRecord, error) {
	limit = normalizedListLimit(limit)
	offset = normalizedListOffset(offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issued_at, completed_at, capability, state_json, value_json, source, success, error, status_json
		FROM commands
		ORDER BY id DESC
		LIMIT ?
		OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommands(rows)
}

func (s *Store) CommandsRange(ctx context.Context, from, to time.Time, limit int) ([]pool.CommandRecord, error) {
	limit = normalizedListLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issued_at, completed_at, capability, state_json, value_json, source, success, error, status_json
		FROM commands
		WHERE issued_at >= ?
			AND issued_at <= ?
		ORDER BY issued_at ASC
		LIMIT ?
	`, encodeTime(from), encodeTime(to), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommands(rows)
}

func scanCommands(rows *sql.Rows) ([]pool.CommandRecord, error) {
	var commands []pool.CommandRecord
	for rows.Next() {
		var command pool.CommandRecord
		var issuedAt string
		var completedAt sql.NullString
		var stateJSON []byte
		var valueJSON []byte
		var source sql.NullString
		var success int
		var errorText sql.NullString
		var statusJSON []byte
		if err := rows.Scan(
			&command.ID,
			&issuedAt,
			&completedAt,
			&command.Capability,
			&stateJSON,
			&valueJSON,
			&source,
			&success,
			&errorText,
			&statusJSON,
		); err != nil {
			return nil, err
		}
		command.IssuedAt = decodeTime(issuedAt)
		if completedAt.Valid {
			t := decodeTime(completedAt.String)
			command.CompletedAt = &t
		}
		if len(stateJSON) > 0 {
			var state bool
			if err := json.Unmarshal(stateJSON, &state); err != nil {
				return nil, err
			}
			command.State = &state
		}
		if len(valueJSON) > 0 {
			command.Value = append(json.RawMessage(nil), valueJSON...)
		}
		if source.Valid {
			command.Source = source.String
		}
		command.Success = success != 0
		if errorText.Valid {
			command.Error = errorText.String
		}
		if len(statusJSON) > 0 {
			var status pool.Status
			if err := json.Unmarshal(statusJSON, &status); err != nil {
				return nil, err
			}
			command.Status = &status
		}
		commands = append(commands, command)
	}
	return commands, rows.Err()
}

func (s *Store) DesiredState(ctx context.Context) (pool.DesiredState, error) {
	body, ok, err := s.getKV(ctx, "desired_state")
	if err != nil || !ok {
		return pool.DesiredState{}, err
	}
	var desired pool.DesiredState
	if err := json.Unmarshal(body, &desired); err != nil {
		return pool.DesiredState{}, err
	}
	return desired, nil
}

func (s *Store) SaveDesiredState(ctx context.Context, desired pool.DesiredState) error {
	body, err := json.Marshal(desired)
	if err != nil {
		return err
	}
	return s.setKV(ctx, "desired_state", body)
}

func (s *Store) WeatherSettings(ctx context.Context) (pool.WeatherSettings, error) {
	body, ok, err := s.getKV(ctx, "weather_settings")
	if err != nil || !ok {
		return pool.WeatherSettings{}, err
	}
	var settings pool.WeatherSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return pool.WeatherSettings{}, err
	}
	return settings, nil
}

func (s *Store) SaveWeatherSettings(ctx context.Context, settings pool.WeatherSettings) error {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = time.Now().UTC()
	}
	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return s.setKV(ctx, "weather_settings", body)
}

func (s *Store) SaveWeatherObservation(ctx context.Context, observation pool.WeatherObservation) (int64, error) {
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now().UTC()
	}
	if len(observation.Data) == 0 {
		return 0, fmt.Errorf("weather observation data is required")
	}
	locationJSON, err := json.Marshal(observation.Location)
	if err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO weather_observations (observed_at, location_json, weather_json)
		VALUES (?, ?, ?)
	`, encodeTime(observation.ObservedAt), locationJSON, observation.Data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) LatestWeatherObservation(ctx context.Context) (pool.WeatherObservation, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, observed_at, location_json, weather_json
		FROM weather_observations
		ORDER BY id DESC
		LIMIT 1
	`)
	var observation pool.WeatherObservation
	var observedAt string
	var locationJSON []byte
	if err := row.Scan(&observation.ID, &observedAt, &locationJSON, &observation.Data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pool.WeatherObservation{}, false, nil
		}
		return pool.WeatherObservation{}, false, err
	}
	observation.ObservedAt = decodeTime(observedAt)
	if len(locationJSON) > 0 {
		if err := json.Unmarshal(locationJSON, &observation.Location); err != nil {
			return pool.WeatherObservation{}, false, err
		}
	}
	return observation, true, nil
}

func (s *Store) weatherSnapshot(ctx context.Context, observedAt time.Time) *pool.WeatherSnapshot {
	weather, ok, err := s.LatestWeatherObservation(ctx)
	if err != nil || !ok {
		return nil
	}
	age := observedAt.Sub(weather.ObservedAt)
	if age < 0 {
		age = -age
	}
	if age > weatherSnapshotMaxAge {
		return nil
	}
	var body struct {
		Main struct {
			Temp *float64 `json:"temp"`
		} `json:"main"`
		Weather []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
		Clouds struct {
			All *int `json:"all"`
		} `json:"clouds"`
		Wind struct {
			Speed *float64 `json:"speed"`
		} `json:"wind"`
	}
	if err := json.Unmarshal(weather.Data, &body); err != nil {
		return nil
	}
	snapshot := &pool.WeatherSnapshot{
		ObservationID:     weather.ID,
		OutsideTempC:      body.Main.Temp,
		CloudsPercent:     body.Clouds.All,
		WindSpeed:         body.Wind.Speed,
		WeatherAgeSeconds: int64(age.Seconds()),
	}
	if len(body.Weather) > 0 {
		snapshot.Main = body.Weather[0].Main
		snapshot.Description = body.Weather[0].Description
	}
	return snapshot
}

func (s *Store) Plans(ctx context.Context) ([]pool.Plan, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT plan_json
		FROM plans
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []pool.Plan
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var plan pool.Plan
		if err := json.Unmarshal(body, &plan); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (s *Store) SavePlans(ctx context.Context, plans []pool.Plan) error {
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM plans`); err != nil {
		return err
	}
	for i := range plans {
		if plans[i].CreatedAt.IsZero() {
			plans[i].CreatedAt = now
		}
		plans[i].UpdatedAt = now
		if err := plans[i].Validate(); err != nil {
			return err
		}
		body, err := json.Marshal(plans[i])
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO plans (id, updated_at, plan_json)
			VALUES (?, ?, ?)
		`, plans[i].ID, encodeTime(now), body); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReadyByControlStates loads persisted ready-by hysteresis states keyed by occurrence.
func (s *Store) ReadyByControlStates(ctx context.Context) (map[string]pool.ReadyByControlState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, value, updated_at
		FROM kv
		WHERE substr(key, 1, ?) = ?
	`, len(readyByControlKeyPrefix), readyByControlKeyPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := map[string]pool.ReadyByControlState{}
	for rows.Next() {
		var key string
		var body []byte
		var updatedAt string
		if err := rows.Scan(&key, &body, &updatedAt); err != nil {
			return nil, err
		}
		var state pool.ReadyByControlState
		if err := json.Unmarshal(body, &state); err != nil {
			return nil, err
		}
		if state.Key == "" {
			state.Key = strings.TrimPrefix(key, readyByControlKeyPrefix)
		}
		if state.UpdatedAt.IsZero() {
			state.UpdatedAt = decodeTime(updatedAt)
		}
		states[state.Key] = state
	}
	return states, rows.Err()
}

// SaveReadyByControlState persists one ready-by hysteresis state.
func (s *Store) SaveReadyByControlState(ctx context.Context, state pool.ReadyByControlState) error {
	if state.Key == "" {
		return fmt.Errorf("ready-by control state key is required")
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO kv (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, readyByControlKeyPrefix+state.Key, body, encodeTime(state.UpdatedAt))
	return err
}

func (s *Store) InsertCommand(ctx context.Context, record pool.CommandRecord) (int64, error) {
	if record.IssuedAt.IsZero() {
		record.IssuedAt = time.Now().UTC()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO commands (issued_at, capability, state_json, value_json, source, success, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, encodeTime(record.IssuedAt), record.Capability, jsonBool(record.State), nullRaw(record.Value), record.Source, record.Success, record.Error)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) FinishCommand(ctx context.Context, id int64, success bool, status *pool.Status, commandErr error) error {
	completedAt := time.Now().UTC()
	var statusJSON []byte
	var errText string
	if status != nil {
		var err error
		statusJSON, err = json.Marshal(status)
		if err != nil {
			return err
		}
	}
	if commandErr != nil {
		errText = commandErr.Error()
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE commands
		SET completed_at = ?, success = ?, error = ?, status_json = ?
		WHERE id = ?
	`, encodeTime(completedAt), success, errText, nullRaw(statusJSON), id)
	return err
}

func (s *Store) LastCommandAt(ctx context.Context) (*time.Time, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT issued_at
		FROM commands
		ORDER BY id DESC
		LIMIT 1
	`)
	var issuedAt string
	if err := row.Scan(&issuedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t := decodeTime(issuedAt)
	return &t, nil
}

func (s *Store) LatestHeatingSessions(ctx context.Context, limit, offset int) ([]pool.HeatingSession, error) {
	limit = normalizedListLimit(limit)
	offset = normalizedListOffset(offset)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id,
			observed_at,
			COALESCE(last_observed_at, observed_at),
			COALESCE(observation_count, 1),
			COALESCE(heater, 0),
			current_temp,
			target_temp
		FROM observations
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []pool.HeatingSession
	var active *pool.HeatingSession
	for rows.Next() {
		row := heatingObservationRow{}
		var firstObservedAt, lastObservedAt string
		var currentTemp sql.NullInt64
		var targetTemp sql.NullInt64
		var heater int
		if err := rows.Scan(&row.id, &firstObservedAt, &lastObservedAt, &row.count, &heater, &currentTemp, &targetTemp); err != nil {
			return nil, err
		}
		row.first = decodeTime(firstObservedAt)
		row.last = decodeTime(lastObservedAt)
		row.heater = heater != 0
		if row.count <= 0 {
			row.count = 1
		}
		if currentTemp.Valid {
			v := int(currentTemp.Int64)
			row.currentTemp = &v
		}
		if targetTemp.Valid {
			row.targetTemp = int(targetTemp.Int64)
		}

		if !row.heater {
			if active != nil {
				endedAt := row.first
				active.EndedAt = &endedAt
				active.DurationSeconds = durationSeconds(active.StartedAt, endedAt)
				active.Active = false
				sessions = append(sessions, *active)
				active = nil
			}
			continue
		}
		if active == nil {
			active = &pool.HeatingSession{
				ID:                 row.id,
				FirstObservationID: row.id,
				StartedAt:          row.first,
				StartTemp:          cloneInt(row.currentTemp),
				TargetTemp:         row.targetTemp,
				Active:             true,
			}
		}
		active.LastObservationID = row.id
		active.LastObservedAt = row.last
		active.ObservationCount += row.count
		active.SpanCount++
		active.EndTemp = cloneInt(row.currentTemp)
		active.TargetTemp = row.targetTemp
		active.DurationSeconds = durationSeconds(active.StartedAt, row.last)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if active != nil {
		sessions = append(sessions, *active)
	}

	for i, j := 0, len(sessions)-1; i < j; i, j = i+1, j-1 {
		sessions[i], sessions[j] = sessions[j], sessions[i]
	}
	if offset >= len(sessions) {
		return nil, nil
	}
	end := offset + limit
	if end > len(sessions) {
		end = len(sessions)
	}
	return sessions[offset:end], nil
}

func (s *Store) Prune(ctx context.Context, observationRetention, eventRetention time.Duration) error {
	now := time.Now().UTC()
	if observationRetention > 0 {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM observations WHERE COALESCE(last_observed_at, observed_at) < ?`, encodeTime(now.Add(-observationRetention))); err != nil {
			return err
		}
	}
	if eventRetention > 0 {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE created_at < ?`, encodeTime(now.Add(-eventRetention))); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA journal_size_limit = 1048576;

		CREATE TABLE IF NOT EXISTS observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			observed_at TEXT NOT NULL,
			last_observed_at TEXT,
			observation_count INTEGER NOT NULL DEFAULT 1,
			status_json BLOB NOT NULL,
			connected INTEGER,
			power INTEGER,
			filter INTEGER,
			heater INTEGER,
			jets INTEGER,
			bubbles INTEGER,
			sanitizer INTEGER,
			current_temp INTEGER,
			target_temp INTEGER,
			unit TEXT,
			error_code TEXT,
			raw_data TEXT,
			weather_observation_id INTEGER,
			outside_temp_c REAL,
			clouds_percent INTEGER,
			weather_main TEXT,
			weather_description TEXT,
			wind_speed REAL,
			weather_age_seconds INTEGER
		);
		CREATE INDEX IF NOT EXISTS observations_observed_at_idx ON observations(observed_at);

		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			type TEXT NOT NULL,
			message TEXT NOT NULL,
			data_json BLOB
		);
		CREATE INDEX IF NOT EXISTS events_created_at_idx ON events(created_at);

		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issued_at TEXT NOT NULL,
			completed_at TEXT,
			capability TEXT NOT NULL,
			state_json BLOB,
			value_json BLOB,
			source TEXT,
			success INTEGER NOT NULL DEFAULT 0,
			error TEXT,
			status_json BLOB
		);
		CREATE INDEX IF NOT EXISTS commands_issued_at_idx ON commands(issued_at);

		CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY,
			updated_at TEXT NOT NULL,
			plan_json BLOB NOT NULL
		);

		CREATE TABLE IF NOT EXISTS weather_observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			observed_at TEXT NOT NULL,
			location_json BLOB NOT NULL,
			weather_json BLOB NOT NULL
		);
		CREATE INDEX IF NOT EXISTS weather_observations_observed_at_idx ON weather_observations(observed_at);
	`)
	if err != nil {
		return err
	}
	if err := s.migrateObservationSpans(ctx); err != nil {
		return err
	}
	return s.compactObservationSpans(ctx)
}

func (s *Store) migrateObservationSpans(ctx context.Context) error {
	columns := map[string]string{
		"last_observed_at":       "TEXT",
		"observation_count":      "INTEGER NOT NULL DEFAULT 1",
		"connected":              "INTEGER",
		"power":                  "INTEGER",
		"filter":                 "INTEGER",
		"heater":                 "INTEGER",
		"jets":                   "INTEGER",
		"bubbles":                "INTEGER",
		"sanitizer":              "INTEGER",
		"current_temp":           "INTEGER",
		"target_temp":            "INTEGER",
		"unit":                   "TEXT",
		"error_code":             "TEXT",
		"raw_data":               "TEXT",
		"weather_observation_id": "INTEGER",
		"outside_temp_c":         "REAL",
		"clouds_percent":         "INTEGER",
		"weather_main":           "TEXT",
		"weather_description":    "TEXT",
		"wind_speed":             "REAL",
		"weather_age_seconds":    "INTEGER",
	}
	for name, definition := range columns {
		ok, err := s.columnExists(ctx, "observations", name)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE observations ADD COLUMN %s %s`, name, definition)); err != nil {
			return err
		}
	}
	_, err := s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS observations_last_observed_at_idx ON observations(last_observed_at);
		UPDATE observations
		SET last_observed_at = COALESCE(last_observed_at, observed_at),
			observation_count = COALESCE(observation_count, 1),
			connected = COALESCE(connected, json_extract(status_json, '$.connected')),
			power = COALESCE(power, json_extract(status_json, '$.power')),
			filter = COALESCE(filter, json_extract(status_json, '$.filter')),
			heater = COALESCE(heater, json_extract(status_json, '$.heater')),
			jets = COALESCE(jets, json_extract(status_json, '$.jets')),
			bubbles = COALESCE(bubbles, json_extract(status_json, '$.bubbles')),
			sanitizer = COALESCE(sanitizer, json_extract(status_json, '$.sanitizer')),
			current_temp = COALESCE(current_temp, json_extract(status_json, '$.current_temp')),
			target_temp = COALESCE(target_temp, json_extract(status_json, '$.preset_temp')),
			unit = COALESCE(unit, json_extract(status_json, '$.unit')),
			error_code = COALESCE(error_code, json_extract(status_json, '$.error_code')),
			raw_data = COALESCE(raw_data, json_extract(status_json, '$.raw_data'));
	`)
	return err
}

func (s *Store) compactObservationSpans(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, observed_at, COALESCE(last_observed_at, observed_at), COALESCE(observation_count, 1), status_json
		FROM observations
		ORDER BY id ASC
	`)
	if err != nil {
		return err
	}

	var allRows []compactObservationRow
	for rows.Next() {
		current := compactObservationRow{}
		if err := rows.Scan(&current.id, &current.first, &current.last, &current.count, &current.body); err != nil {
			_ = rows.Close()
			return err
		}
		if err := json.Unmarshal(current.body, &current.status); err != nil {
			_ = rows.Close()
			return err
		}
		if current.count <= 0 {
			current.count = 1
			current.dirty = true
		}
		allRows = append(allRows, current)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var previous *compactObservationRow
	var updates []compactObservationRow
	var deleteIDs []int64
	for _, current := range allRows {
		if previous != nil && sameObservationState(previous.status, current.status) {
			previous.last = current.last
			previous.count += current.count
			previous.body = current.body
			previous.status = current.status
			previous.dirty = true
			deleteIDs = append(deleteIDs, current.id)
			continue
		}
		if previous != nil && previous.dirty {
			updates = append(updates, *previous)
		}
		currentCopy := current
		previous = &currentCopy
	}
	if previous != nil && previous.dirty {
		updates = append(updates, *previous)
	}
	if len(deleteIDs) == 0 && len(updates) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, update := range updates {
		if err := updateCompactedObservationTx(ctx, tx, update); err != nil {
			return err
		}
	}
	for _, id := range deleteIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM observations WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func updateCompactedObservationTx(ctx context.Context, tx *sql.Tx, row compactObservationRow) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE observations
		SET last_observed_at = ?,
			observation_count = ?,
			status_json = ?,
			connected = ?,
			power = ?,
			filter = ?,
			heater = ?,
			jets = ?,
			bubbles = ?,
			sanitizer = ?,
			current_temp = ?,
			target_temp = ?,
			unit = ?,
			error_code = ?,
			raw_data = ?
		WHERE id = ?
	`, row.last, row.count, row.body, boolInt(row.status.Connected), boolInt(row.status.Power), boolInt(row.status.Filter), boolInt(row.status.Heater), boolInt(row.status.Jets), boolInt(row.status.Bubbles), boolInt(row.status.Sanitizer), intPtrValue(row.status.CurrentTemp), row.status.TargetTemp, nullString(row.status.Unit), nullString(row.status.ErrorCode), nullString(row.status.RawData), row.id)
	return err
}

func (s *Store) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) getKV(ctx context.Context, key string) ([]byte, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT value FROM kv WHERE key = ?`, key)
	var body []byte
	if err := row.Scan(&body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return body, true, nil
}

func (s *Store) setKV(ctx context.Context, key string, value []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO kv (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, encodeTime(time.Now().UTC()))
	return err
}

func marshalRaw(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	if raw, ok := value.(json.RawMessage); ok {
		return raw, nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func nullRaw(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func jsonBool(value *bool) any {
	if value == nil {
		return nil
	}
	body, _ := json.Marshal(value)
	return body
}

func sameObservationState(a, b pool.Status) bool {
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
		return false
	}
	if a.CurrentTemp == nil || b.CurrentTemp == nil {
		return a.CurrentTemp == nil && b.CurrentTemp == nil
	}
	return *a.CurrentTemp == *b.CurrentTemp
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func intPtrValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func intPtrValueFromInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func floatPtrValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func encodeTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func decodeTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func normalizedListLimit(limit int) int {
	if limit <= 0 || limit > 500 {
		return 100
	}
	return limit
}

func normalizedListOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func durationSeconds(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
