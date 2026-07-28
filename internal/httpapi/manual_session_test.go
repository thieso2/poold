package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

func TestPutManualSessionCommitsBeforeDependencyOrderedConvergence(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		TargetTemp: 30,
	}
	spa := newControllableSpa(base)
	spa.blockCommands = make(chan struct{})
	handler, st, observationID := manualSessionTestAPI(t, spa, base)
	revision := controlRevision(t, handler)

	response := putManualSession(t, handler, "create-1", revision, observationID, "30m", pool.ControllableState{
		Power:      true,
		Filter:     true,
		Heater:     true,
		Jets:       true,
		Bubbles:    true,
		TargetTemp: 38,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}

	var applying pool.PoolControlRepresentation
	decodeJSON(t, response, &applying)
	if applying.Control != pool.ManualControl || applying.Session == nil || applying.Session.State != "applying" {
		t.Fatalf("representation = %+v, want applying Manual session", applying)
	}
	if applying.ControlRevision == revision {
		t.Fatal("control revision did not advance")
	}
	if applying.Session.Outcomes["power"].State != "pending" ||
		applying.Session.Outcomes["target_temp"].State != "pending" {
		t.Fatalf("outcomes = %+v", applying.Session.Outcomes)
	}

	persisted, err := st.ManualSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if persisted == nil || persisted.Revision != applying.ControlRevision {
		t.Fatalf("persisted session = %+v", persisted)
	}
	events, err := st.Events(context.Background(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundCreated := false
	for _, event := range events {
		if event.Type == "manual_session.created" {
			foundCreated = true
		}
	}
	if !foundCreated {
		t.Fatalf("events = %+v, want atomic manual_session.created lifecycle event", events)
	}

	close(spa.blockCommands)
	active := waitForManualSessionState(t, handler, "active")
	for field, outcome := range active.Session.Outcomes {
		if outcome.State != "confirmed" {
			t.Fatalf("%s outcome = %+v, want confirmed", field, outcome)
		}
	}
	wantOrder := []string{"power", "filter", "target_temp", "heater", "jets", "bubbles"}
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("commands = %v, want %v", got, wantOrder)
	}
	if got, want := spa.statusCallsCount(), len(wantOrder)+1; got < want {
		t.Fatalf("status calls = %d, want at least initial read plus one refresh per command (%d)", got, want)
	}
}

func TestPutManualSessionAcceptsOnlyDocumentedDurations(t *testing.T) {
	tests := []struct {
		duration string
		expires  time.Duration
	}{
		{duration: "30m", expires: 30 * time.Minute},
		{duration: "60m", expires: time.Hour},
		{duration: "2h", expires: 2 * time.Hour},
		{duration: "until_off"},
	}
	for _, test := range tests {
		t.Run(test.duration, func(t *testing.T) {
			base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}
			spa := newControllableSpa(base)
			handler, _, observationID := manualSessionTestAPI(t, spa, base)
			revision := controlRevision(t, handler)
			response := putManualSession(t, handler, "duration-"+test.duration, revision, observationID, test.duration, controllableState(base))
			if response.Code != http.StatusAccepted {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
			var representation pool.PoolControlRepresentation
			decodeJSON(t, response, &representation)
			session := representation.Session
			if session == nil || session.Duration != test.duration || session.State != "applying" {
				t.Fatalf("session = %+v", session)
			}
			if test.expires == 0 {
				if session.ExpiresAt != nil {
					t.Fatalf("expires_at = %v, want null", session.ExpiresAt)
				}
			} else {
				if session.ExpiresAt == nil {
					t.Fatal("expires_at is null")
				}
				if got := session.ExpiresAt.Sub(session.StartedAt); got != test.expires {
					t.Fatalf("expiry duration = %v, want %v", got, test.expires)
				}
			}
			active := waitForManualSessionState(t, handler, "active")
			if got := spa.commandCapabilities(); len(got) != 0 {
				t.Fatalf("no-op intent issued commands %v", got)
			}
			if active.Session == nil {
				t.Fatal("active session is missing")
			}
		})
	}
}

func TestTimedManualSessionExpiresAtomicallyAndResumesAutomaticReconciliation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 28, 14, 20, 3, 0, time.UTC)
	base := pool.Status{ObservedAt: now, Connected: true, Power: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	handler := New(service, "secret")
	response := putManualSession(t, handler, "timed-expiry", controlRevision(t, handler), observationID, "30m", controllableState(base))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	manual := waitForManualSessionState(t, handler, "active")
	if err := st.SaveDesiredState(ctx, pool.DesiredState{Power: pool.BoolPtr(false), TargetTemp: pool.IntPtr(36)}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(30 * time.Minute)
	automatic := controlRepresentation(t, handler)
	if automatic.Control != pool.AutomaticControl || automatic.Session != nil {
		t.Fatalf("representation = %+v, want Automatic control", automatic)
	}
	if automatic.ControlRevision == manual.ControlRevision {
		t.Fatal("expiry did not advance the control revision")
	}
	assertNoManualSession(t, st)
	events, err := st.Events(ctx, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEventType(events, "manual_session.expired") {
		t.Fatalf("events = %+v, want manual_session.expired", events)
	}
	waitForCommands(t, spa, []string{"power"})
}

func TestManualSessionRecoveryUsesSameSQLiteDatabaseBeforeSchedulesRun(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 28, 14, 20, 3, 0, time.UTC)
	path := t.TempDir() + "/poold.db"
	base := pool.Status{ObservedAt: now, Connected: true, TargetTemp: 36}
	spa := newControllableSpa(base)

	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	handler := New(service, "secret")
	response := putManualSession(t, handler, "restart-session", controlRevision(t, handler), observationID, "until_off", pool.ControllableState{
		Power: true, TargetTemp: 36,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	manual := waitForManualSessionState(t, handler, "active")
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	spa.mu.Lock()
	spa.status.Power = false
	spa.commands = nil
	spa.mu.Unlock()
	now = now.Add(24 * time.Hour)
	restarted, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	recoveredService := NewService(restarted, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	if err := recoveredService.EstablishControl(ctx); err != nil {
		t.Fatal(err)
	}
	recovered := controlRepresentation(t, New(recoveredService, "secret"))
	if recovered.ControlRevision != manual.ControlRevision || recovered.Control != pool.ManualControl ||
		recovered.Session == nil || recovered.Session.State != "active" {
		t.Fatalf("recovered representation = %+v, want restored active session", recovered)
	}
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, []string{"power"}) {
		t.Fatalf("recovery commands = %v, want Manual intent before schedules", got)
	}
	events, err := restarted.Events(ctx, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEventType(events, "manual_session.recovered") {
		t.Fatalf("events = %+v, want manual_session.recovered", events)
	}
}

func TestStartupExpiresElapsedManualSessionBeforeIssuingAutomaticCommand(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 28, 14, 20, 3, 0, time.UTC)
	path := t.TempDir() + "/poold.db"
	base := pool.Status{ObservedAt: now, Connected: true, Power: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	handler := New(service, "secret")
	response := putManualSession(t, handler, "stopped-expiry", controlRevision(t, handler), observationID, "30m", controllableState(base))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	manual := waitForManualSessionState(t, handler, "active")
	if err := st.SaveDesiredState(ctx, pool.DesiredState{Power: pool.BoolPtr(false), TargetTemp: pool.IntPtr(36)}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Minute)
	spa.mu.Lock()
	spa.commands = nil
	spa.mu.Unlock()

	restarted, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	recoveredService := NewService(restarted, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	if err := recoveredService.EstablishControl(ctx); err != nil {
		t.Fatal(err)
	}
	automatic := controlRepresentation(t, New(recoveredService, "secret"))
	if automatic.Control != pool.AutomaticControl || automatic.Session != nil || automatic.ControlRevision == manual.ControlRevision {
		t.Fatalf("representation = %+v, want expired Automatic ownership", automatic)
	}
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, []string{"power"}) {
		t.Fatalf("startup commands = %v, want schedule command only after expiry", got)
	}
}

func TestManualReconciliationDoesNotCommandAfterTimedOwnershipExpires(t *testing.T) {
	ctx := context.Background()
	var clockMu sync.Mutex
	now := time.Date(2026, 7, 28, 14, 20, 3, 0, time.UTC)
	nowFunc := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	base := pool.Status{ObservedAt: now, Connected: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	spa.statusHook = func(call int) {
		if call == 2 {
			clockMu.Lock()
			now = now.Add(31 * time.Minute)
			clockMu.Unlock()
		}
	}
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveDesiredState(ctx, pool.DesiredState{Power: pool.BoolPtr(false), TargetTemp: pool.IntPtr(36)}); err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: nowFunc})
	handler := New(service, "secret")
	response := putManualSession(t, handler, "expires-before-command", controlRevision(t, handler), observationID, "30m", pool.ControllableState{
		Power: true, TargetTemp: 36,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if controlRepresentation(t, handler).Control == pool.AutomaticControl {
			if got := spa.commandCapabilities(); len(got) != 0 {
				t.Fatalf("commands = %v, want none after Manual ownership expired", got)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Manual session did not expire")
}

func TestPutManualSessionDisablesDependentsBeforeFilterAndPower(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     true,
		Heater:     true,
		Jets:       true,
		Bubbles:    true,
		TargetTemp: 38,
	}
	spa := newControllableSpa(base)
	handler, _, observationID := manualSessionTestAPI(t, spa, base)
	response := putManualSession(t, handler, "disable", controlRevision(t, handler), observationID, "30m", pool.ControllableState{
		TargetTemp: 38,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	_ = waitForManualSessionState(t, handler, "active")
	want := []string{"heater", "jets", "bubbles", "filter", "power"}
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
}

func TestPutManualSessionEditsIntentWhileOfflineAndRestartsDuration(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     true,
		TargetTemp: 36,
	}
	spa := newControllableSpa(base)
	handler, _, observationID := manualSessionTestAPI(t, spa, base)
	original := createActiveManualSession(t, handler, observationID, base)

	spa.mu.Lock()
	spa.statusErr = errors.New("offline")
	spa.mu.Unlock()
	intended := controllableState(base)
	intended.TargetTemp = 38
	response := putManualSessionEdit(t, handler, "edit-offline", original.ControlRevision, "60m", intended)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}

	var edited pool.PoolControlRepresentation
	decodeJSON(t, response, &edited)
	if edited.ControlRevision == original.ControlRevision {
		t.Fatal("edit did not advance control revision")
	}
	if edited.Session == nil || edited.Session.Intended != intended || edited.Session.Duration != "60m" {
		t.Fatalf("edited session = %+v", edited.Session)
	}
	if !edited.Session.StartedAt.After(original.Session.StartedAt) {
		t.Fatalf("started_at = %s, want after %s", edited.Session.StartedAt, original.Session.StartedAt)
	}
	if got := edited.Session.ExpiresAt.Sub(edited.Session.StartedAt); got != time.Hour {
		t.Fatalf("expiry duration = %s, want 1h", got)
	}
}

func TestRetryManualSessionResetsEveryFailureWithoutChangingIntentOrTiming(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     true,
		TargetTemp: 36,
	}
	spa := newControllableSpa(base)
	spa.setErr = errors.New("device rejected command")
	handler, _, observationID := manualSessionTestAPI(t, spa, base)
	intended := controllableState(base)
	intended.Heater = true
	response := putManualSession(t, handler, "create-failure", controlRevision(t, handler), observationID, "60m", intended)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	degraded := waitForManualSessionState(t, handler, "degraded")

	spa.mu.Lock()
	spa.setErr = nil
	spa.mu.Unlock()
	retry := retryManualSession(t, handler, "retry-all", degraded.ControlRevision)
	if retry.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body=%s", retry.Code, retry.Body.String())
	}
	var applying pool.PoolControlRepresentation
	decodeJSON(t, retry, &applying)
	if applying.ControlRevision != degraded.ControlRevision ||
		applying.Session.StartedAt != degraded.Session.StartedAt ||
		!equalOptionalTime(applying.Session.ExpiresAt, degraded.Session.ExpiresAt) ||
		applying.Session.Intended != degraded.Session.Intended {
		t.Fatalf("retry changed session identity or intent: before=%+v after=%+v", degraded, applying)
	}
	if outcome := applying.Session.Outcomes["heater"]; outcome.State != "pending" || outcome.Code != "" || outcome.Message != "" {
		t.Fatalf("heater outcome = %+v, want reset pending", outcome)
	}
	_ = waitForManualSessionState(t, handler, "active")
}

func TestRetryManualSessionExpiresElapsedSessionFirst(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 28, 14, 20, 3, 0, time.UTC)
	base := pool.Status{
		ObservedAt: now,
		Connected:  true,
		Power:      true,
		Filter:     true,
		TargetTemp: 36,
	}
	spa := newControllableSpa(base)
	spa.setErr = errors.New("device rejected command")
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	handler := New(service, "secret")
	intended := controllableState(base)
	intended.Heater = true
	response := putManualSession(t, handler, "create-expiring-failure", controlRevision(t, handler), observationID, "30m", intended)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	degraded := waitForManualSessionState(t, handler, "degraded")

	now = now.Add(31 * time.Minute)
	retry := retryManualSession(t, handler, "retry-expired", degraded.ControlRevision)
	assertManualSessionError(t, retry, http.StatusConflict, "control_changed")
	automatic := controlRepresentation(t, handler)
	if automatic.Control != pool.AutomaticControl || automatic.Session != nil {
		t.Fatalf("representation = %+v, want expired Automatic ownership", automatic)
	}
}

func TestPollingAttemptsManualSessionDriftOnceThenWaitsForRetry(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     true,
		TargetTemp: 36,
	}
	spa := newControllableSpa(base)
	handler, _, service, observationID := manualSessionTestServiceAPI(t, spa, base)
	active := createActiveManualSession(t, handler, observationID, base)

	spa.mu.Lock()
	spa.status.TargetTemp = 34
	spa.setErr = errors.New("device rejected drift correction")
	spa.mu.Unlock()
	if _, err := service.RefreshStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
	degraded := waitForManualSessionState(t, handler, "degraded")
	if outcome := degraded.Session.Outcomes["target_temp"]; outcome.State != "failed" || outcome.Code != "command_failed" {
		t.Fatalf("target_temp outcome = %+v", outcome)
	}
	waitForCommands(t, spa, []string{"target_temp"})

	if _, err := service.RefreshStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, []string{"target_temp"}) {
		t.Fatalf("commands after second poll = %v, want one drift attempt", got)
	}

	spa.mu.Lock()
	spa.setErr = nil
	spa.mu.Unlock()
	retry := retryManualSession(t, handler, "retry-drift", active.ControlRevision)
	if retry.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body=%s", retry.Code, retry.Body.String())
	}
	_ = waitForManualSessionState(t, handler, "active")
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, []string{"target_temp", "target_temp"}) {
		t.Fatalf("commands after retry = %v", got)
	}
}

func TestPutManualSessionRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36}
	tests := []struct {
		name       string
		body       string
		key        string
		wantStatus int
		wantCode   string
	}{
		{name: "missing idempotency key", key: "", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","intended":{"power":true,"filter":true,"heater":false,"jets":false,"bubbles":false,"target_temp":36}}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "unknown field", key: "bad-1", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","extra":true,"intended":{"power":true,"filter":true,"heater":false,"jets":false,"bubbles":false,"target_temp":36}}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "incomplete intent", key: "bad-2", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","intended":{"power":true,"target_temp":36}}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "invalid duration", key: "bad-3", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"45m","intended":{"power":true,"filter":true,"heater":false,"jets":false,"bubbles":false,"target_temp":36}}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "temperature out of range", key: "bad-4", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","intended":{"power":true,"filter":true,"heater":false,"jets":false,"bubbles":false,"target_temp":41}}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "power dependency", key: "bad-5", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","intended":{"power":false,"filter":true,"heater":false,"jets":false,"bubbles":false,"target_temp":36}}`, wantStatus: 422, wantCode: "invalid_state"},
		{name: "heater dependency", key: "bad-6", body: `{"expected_control_revision":"REV","base_observation_id":1,"duration":"30m","intended":{"power":true,"filter":false,"heater":true,"jets":false,"bubbles":false,"target_temp":36}}`, wantStatus: 422, wantCode: "invalid_state"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spa := newControllableSpa(base)
			handler, st, observationID := manualSessionTestAPI(t, spa, base)
			revision := controlRevision(t, handler)
			body := bytes.ReplaceAll([]byte(test.body), []byte(`"REV"`), []byte(`"`+revision+`"`))
			body = bytes.ReplaceAll(body, []byte(`:1,`), []byte(`:`+string(jsonNumber(observationID))+`,`))
			response := manualSessionRequest(handler, test.key, body)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
			var envelope manualSessionErrorEnvelope
			decodeJSON(t, response, &envelope)
			if envelope.Error.Code != test.wantCode {
				t.Fatalf("error = %+v, want code %q", envelope.Error, test.wantCode)
			}
			session, err := st.ManualSession(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if session != nil {
				t.Fatalf("invalid request committed session %+v", session)
			}
			if got := controlRevision(t, handler); got != revision {
				t.Fatalf("revision changed from %q to %q", revision, got)
			}
		})
	}
}

func TestPutManualSessionRequiresFreshUnchangedSpa(t *testing.T) {
	t.Run("unreachable", func(t *testing.T) {
		base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}
		spa := newControllableSpa(base)
		spa.statusErr = errors.New("dial timeout")
		handler, st, observationID := manualSessionTestAPI(t, spa, base)
		revision := controlRevision(t, handler)
		response := putManualSession(t, handler, "offline", revision, observationID, "30m", controllableState(base))
		assertManualSessionError(t, response, http.StatusServiceUnavailable, "pool_unreachable")
		assertNoManualSession(t, st)
	})

	t.Run("changed controllable field", func(t *testing.T) {
		base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36}
		spa := newControllableSpa(base)
		spa.status.Filter = false
		handler, st, observationID := manualSessionTestAPI(t, spa, base)
		revision := controlRevision(t, handler)
		response := putManualSession(t, handler, "changed", revision, observationID, "30m", controllableState(base))
		assertManualSessionError(t, response, http.StatusConflict, "observed_state_changed")
		var envelope manualSessionErrorEnvelope
		decodeJSON(t, response, &envelope)
		if !reflect.DeepEqual(envelope.Error.ChangedFields, []string{"filter"}) {
			t.Fatalf("changed_fields = %v", envelope.Error.ChangedFields)
		}
		assertNoManualSession(t, st)
	})

	t.Run("temperature and observation time do not conflict", func(t *testing.T) {
		base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36, CurrentTemp: pool.IntPtr(30)}
		spa := newControllableSpa(base)
		spa.status.ObservedAt = base.ObservedAt.Add(time.Minute)
		spa.status.CurrentTemp = pool.IntPtr(31)
		handler, _, observationID := manualSessionTestAPI(t, spa, base)
		revision := controlRevision(t, handler)
		response := putManualSession(t, handler, "temperature-only", revision, observationID, "30m", controllableState(base))
		if response.Code != http.StatusAccepted {
			t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
		}
	})
}

func TestPutManualSessionRejectsStaleControlRevision(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	handler, st, observationID := manualSessionTestAPI(t, spa, base)
	response := putManualSession(t, handler, "stale", "stale-revision", observationID, "30m", controllableState(base))
	assertManualSessionError(t, response, http.StatusConflict, "control_changed")
	var envelope manualSessionErrorEnvelope
	decodeJSON(t, response, &envelope)
	if envelope.Error.Current == nil || envelope.Error.Current.ControlRevision == "" {
		t.Fatalf("current = %+v, want complete current representation", envelope.Error.Current)
	}
	assertNoManualSession(t, st)
}

func TestPutManualSessionReplaysCommittedResultBeforeFreshSpaRead(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	handler, _, observationID := manualSessionTestAPI(t, spa, base)
	revision := controlRevision(t, handler)

	first := putManualSession(t, handler, "create-replay", revision, observationID, "30m", controllableState(base))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, body=%s", first.Code, first.Body.String())
	}
	_ = waitForManualSessionState(t, handler, "active")
	spa.statusErr = errors.New("dial timeout")

	replay := putManualSession(t, handler, "create-replay", revision, observationID, "30m", controllableState(base))
	if replay.Code != first.Code || replay.Body.String() != first.Body.String() {
		t.Fatalf("replay = (%d, %s), want (%d, %s)", replay.Code, replay.Body.String(), first.Code, first.Body.String())
	}
}

func TestRetryManualSessionReplaysCommittedResultAfterSessionExpires(t *testing.T) {
	now := time.Now().UTC()
	base := pool.Status{ObservedAt: now, Connected: true, Power: true, Filter: true, Heater: true, TargetTemp: 36}
	spa := newControllableSpa(base)
	databasePath := t.TempDir() + "/poold.db"
	st, err := store.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	observationID, err := st.SaveObservation(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }})
	handler := New(service, "secret")
	create := putManualSession(t, handler, "expiring-retry-create", controlRevision(t, handler), observationID, "30m",
		pool.ControllableState{Power: true, Filter: true, Heater: false, TargetTemp: 36})
	if create.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body=%s", create.Code, create.Body.String())
	}
	degraded := waitForManualSessionState(t, handler, "active")
	first := retryManualSession(t, handler, "expiring-retry", degraded.ControlRevision)
	if first.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body=%s", first.Code, first.Body.String())
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	now = now.Add(31 * time.Minute)
	restarted, err := store.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	handler = New(
		NewService(restarted, spa, scheduler.New(scheduler.Config{}), ServiceConfig{Now: func() time.Time { return now }}),
		"secret",
	)
	replay := retryManualSession(t, handler, "expiring-retry", degraded.ControlRevision)
	if replay.Code != first.Code || replay.Body.String() != first.Body.String() {
		t.Fatalf("replay = (%d, %s), want original (%d, %s)", replay.Code, replay.Body.String(), first.Code, first.Body.String())
	}
}

func TestManualSessionMutationRequiresBearerToken(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}
	handler, _, observationID := manualSessionTestAPI(t, newControllableSpa(base), base)
	body, err := json.Marshal(map[string]any{
		"expected_control_revision": controlRevision(t, handler),
		"base_observation_id":       observationID,
		"duration":                  "30m",
		"intended":                  controllableState(base),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/manual-session", bytes.NewReader(body))
	request.Header.Set("Idempotency-Key", "unauthorized")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertManualSessionError(t, response, http.StatusUnauthorized, "unauthorized")
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestDeleteManualSessionCommitsAutomaticControlBeforeScheduleConvergence(t *testing.T) {
	base := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      true,
		Filter:     true,
		Heater:     true,
		TargetTemp: 38,
	}
	spa := newControllableSpa(base)
	handler, st, observationID := manualSessionTestAPI(t, spa, base)
	if err := st.SaveDesiredState(context.Background(), pool.DesiredState{
		Power:      pool.BoolPtr(true),
		Filter:     pool.BoolPtr(true),
		Heater:     pool.BoolPtr(false),
		TargetTemp: pool.IntPtr(36),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveControlMode(context.Background(), pool.ControlMode{ManualControl: true}); err != nil {
		t.Fatal(err)
	}
	manual := createActiveManualSession(t, handler, observationID, base)
	spa.blockCommands = make(chan struct{})

	response := deleteManualSession(t, handler, "clear-online", manual.ControlRevision)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var automatic pool.PoolControlRepresentation
	decodeJSON(t, response, &automatic)
	if automatic.Control != pool.AutomaticControl || automatic.Session != nil {
		t.Fatalf("representation = %+v, want complete Automatic control state", automatic)
	}
	if automatic.ControlRevision == manual.ControlRevision {
		t.Fatal("control revision did not advance")
	}
	if automatic.Observed == nil || automatic.Observed.State != controllableState(base) {
		t.Fatalf("observed = %+v, want complete latest observation", automatic.Observed)
	}
	assertNoManualSession(t, st)
	events, err := st.Events(context.Background(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundCleared := false
	for _, event := range events {
		if event.Type == "manual_session.cleared" {
			foundCleared = true
		}
	}
	if !foundCleared {
		t.Fatalf("events = %+v, want atomic manual_session.cleared lifecycle event", events)
	}
	mode, err := st.ControlMode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mode.ManualControl {
		t.Fatalf("legacy control mode = %+v, want cleared with Manual session ownership", mode)
	}

	close(spa.blockCommands)
	waitForCommands(t, spa, []string{"target_temp", "heater"})
	latest, ok, err := st.LatestStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ok || latest.Heater || latest.TargetTemp != 36 {
		t.Fatalf("latest status = %+v, want reconciled Automatic control schedule", latest)
	}
}

func TestDeleteManualSessionSucceedsWhileSpaIsOffline(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, Heater: true, TargetTemp: 38}
	spa := newControllableSpa(base)
	handler, st, observationID := manualSessionTestAPI(t, spa, base)
	manual := createActiveManualSession(t, handler, observationID, base)
	spa.statusErr = errors.New("dial timeout")
	spa.setErr = errors.New("dial timeout")

	response := deleteManualSession(t, handler, "clear-offline", manual.ControlRevision)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var automatic pool.PoolControlRepresentation
	decodeJSON(t, response, &automatic)
	if automatic.Control != pool.AutomaticControl || automatic.Session != nil {
		t.Fatalf("representation = %+v, want Automatic", automatic)
	}
	assertNoManualSession(t, st)
}

func TestDeleteManualSessionReplaysOriginalResultWithoutDuplicateCommands(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, Heater: true, TargetTemp: 38}
	spa := newControllableSpa(base)
	handler, st, observationID := manualSessionTestAPI(t, spa, base)
	if err := st.SaveDesiredState(context.Background(), pool.DesiredState{Power: pool.BoolPtr(false), TargetTemp: pool.IntPtr(38)}); err != nil {
		t.Fatal(err)
	}
	manual := createActiveManualSession(t, handler, observationID, base)

	first := deleteManualSession(t, handler, "clear-replay", manual.ControlRevision)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, body=%s", first.Code, first.Body.String())
	}
	waitForCommands(t, spa, []string{"heater", "filter", "power"})
	second := deleteManualSession(t, handler, "clear-replay", manual.ControlRevision)
	if second.Code != first.Code || second.Body.String() != first.Body.String() {
		t.Fatalf("replay = (%d, %s), want (%d, %s)", second.Code, second.Body.String(), first.Code, first.Body.String())
	}
	time.Sleep(50 * time.Millisecond)
	if got := spa.commandCapabilities(); !reflect.DeepEqual(got, []string{"heater", "filter", "power"}) {
		t.Fatalf("commands after replay = %v, want no duplicates", got)
	}
}

func TestDeleteManualSessionRejectsInvalidConcurrencyWithoutMutation(t *testing.T) {
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}
	tests := []struct {
		name       string
		key        string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "missing idempotency key", body: `{"expected_control_revision":"REV"}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "missing revision", key: "clear-missing-revision", body: `{}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "unknown field", key: "clear-unknown", body: `{"expected_control_revision":"REV","extra":true}`, wantStatus: 400, wantCode: "invalid_request"},
		{name: "stale revision", key: "clear-stale", body: `{"expected_control_revision":"stale"}`, wantStatus: 409, wantCode: "control_changed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spa := newControllableSpa(base)
			handler, st, observationID := manualSessionTestAPI(t, spa, base)
			manual := createActiveManualSession(t, handler, observationID, base)
			body := []byte(strings.ReplaceAll(test.body, "REV", manual.ControlRevision))
			response := manualSessionMutationRequest(handler, http.MethodDelete, test.key, body)
			assertManualSessionError(t, response, test.wantStatus, test.wantCode)
			session, err := st.ManualSession(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if session == nil || session.Revision != manual.ControlRevision {
				t.Fatalf("session = %+v, want unchanged revision %q", session, manual.ControlRevision)
			}
		})
	}

	t.Run("reused key", func(t *testing.T) {
		spa := newControllableSpa(base)
		handler, st, observationID := manualSessionTestAPI(t, spa, base)
		manual := createActiveManualSession(t, handler, observationID, base)
		response := deleteManualSession(t, handler, "create-active", manual.ControlRevision)
		assertManualSessionError(t, response, http.StatusConflict, "idempotency_key_reused")
		session, err := st.ManualSession(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if session == nil || session.Revision != manual.ControlRevision {
			t.Fatalf("session = %+v, want unchanged revision %q", session, manual.ControlRevision)
		}
	})
}

func TestDeleteManualSessionPersistenceFailureLeavesOwnershipIntact(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/poold.db"
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	base := pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(NewService(st, newControllableSpa(base), scheduler.New(scheduler.Config{}), ServiceConfig{}), "secret")
	manual := createActiveManualSession(t, handler, observationID, base)
	faultDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer faultDB.Close()
	if _, err := faultDB.ExecContext(ctx, `
		CREATE TRIGGER fail_manual_session_clear
		BEFORE INSERT ON events
		WHEN NEW.type = 'manual_session.cleared'
		BEGIN
			SELECT RAISE(ABORT, 'injected clear persistence failure');
		END
	`); err != nil {
		t.Fatal(err)
	}

	response := deleteManualSession(t, handler, "clear-failure", manual.ControlRevision)
	assertManualSessionError(t, response, http.StatusInternalServerError, "persistence_error")

	session, err := st.ManualSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil || session.Revision != manual.ControlRevision {
		t.Fatalf("session after failed clear = %+v, want unchanged revision %q", session, manual.ControlRevision)
	}
	if revision, err := st.ControlRevision(ctx); err != nil || revision != manual.ControlRevision {
		t.Fatalf("control revision after failed clear = %q, %v; want %q", revision, err, manual.ControlRevision)
	}
	events, err := st.Events(ctx, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "manual_session.cleared" {
			t.Fatalf("failed transaction persisted clear event %+v", event)
		}
	}
	if _, err := faultDB.ExecContext(ctx, `DROP TRIGGER fail_manual_session_clear`); err != nil {
		t.Fatal(err)
	}
	retry := deleteManualSession(t, handler, "clear-failure", manual.ControlRevision)
	if retry.Code != http.StatusOK {
		t.Fatalf("retry status = %d, body=%s", retry.Code, retry.Body.String())
	}
}

type controllableSpa struct {
	mu            sync.Mutex
	status        pool.Status
	statusErr     error
	setErr        error
	statusCalls   int
	commands      []string
	blockCommands chan struct{}
	statusHook    func(int)
}

func newControllableSpa(status pool.Status) *controllableSpa {
	return &controllableSpa{status: status}
}

func (s *controllableSpa) Status(context.Context) (pool.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusCalls++
	if s.statusHook != nil {
		s.statusHook(s.statusCalls)
	}
	if s.statusErr != nil {
		return pool.Status{}, s.statusErr
	}
	status := s.status
	status.ObservedAt = time.Now().UTC()
	return status, nil
}

func (s *controllableSpa) Set(_ context.Context, capability string, value any) (pool.Status, error) {
	if s.blockCommands != nil {
		<-s.blockCommands
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, capability)
	if s.setErr != nil {
		return pool.Status{}, s.setErr
	}
	switch capability {
	case "power":
		s.status.Power = boolValue(value)
	case "filter":
		s.status.Filter = boolValue(value)
	case "heater":
		s.status.Heater = boolValue(value)
	case "jets":
		s.status.Jets = boolValue(value)
	case "bubbles":
		s.status.Bubbles = boolValue(value)
	case "target_temp":
		raw, _ := value.([]byte)
		_ = json.Unmarshal(raw, &s.status.TargetTemp)
	}
	s.status.ObservedAt = time.Now().UTC()
	s.status.Connected = true
	return s.status, nil
}

func (s *controllableSpa) commandCapabilities() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

func (s *controllableSpa) statusCallsCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusCalls
}

func manualSessionTestAPI(t *testing.T, spa intex.PoolClient, initial pool.Status) (http.Handler, *store.Store, int64) {
	t.Helper()
	handler, st, _, observationID := manualSessionTestServiceAPI(t, spa, initial)
	return handler, st, observationID
}

func manualSessionTestServiceAPI(t *testing.T, spa intex.PoolClient, initial pool.Status) (http.Handler, *store.Store, *Service, int64) {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	observationID, err := st.SaveObservation(context.Background(), initial)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, spa, scheduler.New(scheduler.Config{}), ServiceConfig{})
	return New(service, "secret"), st, service, observationID
}

func controlRevision(t *testing.T, handler http.Handler) string {
	t.Helper()
	return controlRepresentation(t, handler).ControlRevision
}

func controlRepresentation(t *testing.T, handler http.Handler) pool.PoolControlRepresentation {
	t.Helper()
	response := authed(handler, http.MethodGet, "/manual-session", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body=%s", response.Code, response.Body.String())
	}
	var representation pool.PoolControlRepresentation
	decodeJSON(t, response, &representation)
	return representation
}

func hasEventType(events []pool.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func putManualSession(t *testing.T, handler http.Handler, key, revision string, observationID int64, duration string, intended pool.ControllableState) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"expected_control_revision": revision,
		"base_observation_id":       observationID,
		"duration":                  duration,
		"intended":                  intended,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manualSessionRequest(handler, key, body)
}

func putManualSessionEdit(t *testing.T, handler http.Handler, key, revision, duration string, intended pool.ControllableState) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"expected_control_revision": revision,
		"duration":                  duration,
		"intended":                  intended,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manualSessionRequest(handler, key, body)
}

func retryManualSession(t *testing.T, handler http.Handler, key, revision string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"expected_control_revision": revision})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/manual-session/retry", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func equalOptionalTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func createActiveManualSession(t *testing.T, handler http.Handler, observationID int64, base pool.Status) pool.PoolControlRepresentation {
	t.Helper()
	response := putManualSession(t, handler, "create-active", controlRevision(t, handler), observationID, "until_off", controllableState(base))
	if response.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body=%s", response.Code, response.Body.String())
	}
	return waitForManualSessionState(t, handler, "active")
}

func deleteManualSession(t *testing.T, handler http.Handler, key, revision string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"expected_control_revision": revision})
	if err != nil {
		t.Fatal(err)
	}
	return manualSessionMutationRequest(handler, http.MethodDelete, key, body)
}

func manualSessionRequest(handler http.Handler, key string, body []byte) *httptest.ResponseRecorder {
	return manualSessionMutationRequest(handler, http.MethodPut, key, body)
}

func manualSessionMutationRequest(handler http.Handler, method, key string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/manual-session", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func waitForCommands(t *testing.T, spa *controllableSpa, want []string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got := spa.commandCapabilities(); reflect.DeepEqual(got, want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("commands = %v, want %v", spa.commandCapabilities(), want)
}

func waitForManualSessionState(t *testing.T, handler http.Handler, want string) pool.PoolControlRepresentation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response := authed(handler, http.MethodGet, "/manual-session", nil)
		var representation pool.PoolControlRepresentation
		decodeJSON(t, response, &representation)
		if representation.Session != nil && representation.Session.State == want {
			return representation
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Manual session did not reach %q", want)
	return pool.PoolControlRepresentation{}
}

func controllableState(status pool.Status) pool.ControllableState {
	return pool.ControllableState{
		Power:      status.Power,
		Filter:     status.Filter,
		Heater:     status.Heater,
		Jets:       status.Jets,
		Bubbles:    status.Bubbles,
		TargetTemp: status.TargetTemp,
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode %q: %v", response.Body.String(), err)
	}
}

func assertNoManualSession(t *testing.T, st *store.Store) {
	t.Helper()
	session, err := st.ManualSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session != nil {
		t.Fatalf("session = %+v, want nil", session)
	}
}

func assertManualSessionError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var envelope manualSessionErrorEnvelope
	decodeJSON(t, response, &envelope)
	if envelope.Error.Code != code {
		t.Fatalf("error = %+v, want code %q", envelope.Error, code)
	}
}

func jsonNumber(value int64) []byte {
	return []byte(strconv.FormatInt(value, 10))
}
