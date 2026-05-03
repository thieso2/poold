package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pooly/services/poold/internal/pool"

	_ "github.com/ncruces/go-sqlite3/driver"
)

type Store struct {
	db *sql.DB
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
	return s.db.Close()
}

func (s *Store) SaveObservation(ctx context.Context, status pool.Status) (int64, error) {
	if status.ObservedAt.IsZero() {
		status.ObservedAt = time.Now().UTC()
	}
	body, err := json.Marshal(status)
	if err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO observations (observed_at, status_json)
		VALUES (?, ?)
	`, encodeTime(status.ObservedAt), body)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) LatestStatus(ctx context.Context) (pool.Status, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT observed_at, status_json
		FROM observations
		ORDER BY id DESC
		LIMIT 1
	`)
	var observedAt string
	var body []byte
	if err := row.Scan(&observedAt, &body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pool.Status{}, false, nil
		}
		return pool.Status{}, false, err
	}
	var status pool.Status
	if err := json.Unmarshal(body, &status); err != nil {
		return pool.Status{}, false, err
	}
	if status.ObservedAt.IsZero() {
		status.ObservedAt = decodeTime(observedAt)
	}
	return status, true, nil
}

func (s *Store) Observations(ctx context.Context, afterID int64, limit int) ([]pool.Observation, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, observed_at, status_json
		FROM observations
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []pool.Observation
	for rows.Next() {
		var observation pool.Observation
		var observedAt string
		var body []byte
		if err := rows.Scan(&observation.ID, &observedAt, &body); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &observation.Status); err != nil {
			return nil, err
		}
		if observation.Status.ObservedAt.IsZero() {
			observation.Status.ObservedAt = decodeTime(observedAt)
		}
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}

func (s *Store) LatestObservations(ctx context.Context, limit int) ([]pool.Observation, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, observed_at, status_json
		FROM observations
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []pool.Observation
	for rows.Next() {
		var observation pool.Observation
		var observedAt string
		var body []byte
		if err := rows.Scan(&observation.ID, &observedAt, &body); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &observation.Status); err != nil {
			return nil, err
		}
		if observation.Status.ObservedAt.IsZero() {
			observation.Status.ObservedAt = decodeTime(observedAt)
		}
		observations = append(observations, observation)
	}
	return observations, rows.Err()
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
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, type, message, data_json
		FROM events
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (s *Store) LatestEvents(ctx context.Context, limit int) ([]pool.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, type, message, data_json
		FROM events
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (s *Store) Prune(ctx context.Context, observationRetention, eventRetention time.Duration) error {
	now := time.Now().UTC()
	if observationRetention > 0 {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM observations WHERE observed_at < ?`, encodeTime(now.Add(-observationRetention))); err != nil {
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
		PRAGMA busy_timeout = 5000;

		CREATE TABLE IF NOT EXISTS observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			observed_at TEXT NOT NULL,
			status_json BLOB NOT NULL
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
	`)
	return err
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
