package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
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

type controllableSpa struct {
	mu            sync.Mutex
	status        pool.Status
	statusErr     error
	statusCalls   int
	commands      []string
	blockCommands chan struct{}
}

func newControllableSpa(status pool.Status) *controllableSpa {
	return &controllableSpa{status: status}
}

func (s *controllableSpa) Status(context.Context) (pool.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusCalls++
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
	return New(service, "secret"), st, observationID
}

func controlRevision(t *testing.T, handler http.Handler) string {
	t.Helper()
	response := authed(handler, http.MethodGet, "/manual-session", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body=%s", response.Code, response.Body.String())
	}
	var representation pool.PoolControlRepresentation
	decodeJSON(t, response, &representation)
	return representation.ControlRevision
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

func manualSessionRequest(handler http.Handler, key string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/manual-session", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
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
