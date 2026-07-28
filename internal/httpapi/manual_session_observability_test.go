package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

func TestManualSessionLifecycleEventsContainRevisionAndCapabilityOutcomes(t *testing.T) {
	base := observableStatus(false, false, false, false, false, 30)
	spa := newControllableSpa(base)
	handler, st, observationID := manualSessionTestAPI(t, spa, base)

	response := putManualSession(t, handler, "observable-create", controlRevision(t, handler), observationID, "30m", pool.ControllableState{
		Power: true, Filter: true, Heater: true, TargetTemp: 38,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("PUT status = %d, body=%s", response.Code, response.Body.String())
	}
	_ = waitForManualSessionState(t, handler, "active")

	events, err := st.Events(context.Background(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]bool{
		"manual_session.created":  false,
		"manual_session.applying": false,
		"manual_session.active":   false,
	}
	for _, event := range events {
		if _, ok := required[event.Type]; !ok {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(event.Data, &data); err != nil {
			t.Fatalf("%s data: %v", event.Type, err)
		}
		if strings.TrimSpace(stringValue(data["control_revision"])) == "" {
			t.Errorf("%s lacks control_revision: %s", event.Type, event.Data)
		}
		if event.Type == "manual_session.active" {
			outcomes, ok := data["outcomes"].(map[string]any)
			if !ok || len(outcomes) != len(pool.ControllableFields) {
				t.Errorf("active outcomes = %#v, want every capability", data["outcomes"])
			}
		}
		required[event.Type] = true
	}
	for eventType, found := range required {
		if !found {
			t.Errorf("missing durable event %s", eventType)
		}
	}
}

func TestManualSessionConflictIsDurableAndStructuredLogsRedactRequestSecrets(t *testing.T) {
	const (
		token = "recognizable-bearer-secret"
		key   = "recognizable-idempotency-secret"
	)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	base := observableStatus(false, false, false, false, false, 30)
	observationID, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(st, newControllableSpa(base), scheduler.New(scheduler.Config{}), ServiceConfig{Logger: logger})
	handler := New(service, token)
	automatic, err := st.PoolControlRepresentation(ctx)
	if err != nil {
		t.Fatal(err)
	}

	winnerBody, err := json.Marshal(map[string]any{
		"expected_control_revision": automatic.ControlRevision,
		"base_observation_id":       observationID,
		"duration":                  "until_off",
		"intended":                  controllableState(base),
	})
	if err != nil {
		t.Fatal(err)
	}
	response := manualSessionRequestWithToken(handler, token, "winner", winnerBody)
	if response.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body=%s", response.Code, response.Body.String())
	}
	stale := manualSessionRequestWithToken(handler, token, key, []byte(`{
		"expected_control_revision":"stale-revision",
		"duration":"until_off",
		"intended":{"power":false,"filter":false,"heater":false,"jets":false,"bubbles":false,"target_temp":30}
	}`))
	if stale.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, body=%s", stale.Code, stale.Body.String())
	}

	events, err := st.Events(ctx, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	var conflict pool.Event
	for _, event := range events {
		if event.Type == "manual_session.conflict" {
			conflict = event
		}
	}
	if conflict.ID == 0 || !bytes.Contains(conflict.Data, []byte(`"code":"control_changed"`)) ||
		!bytes.Contains(conflict.Data, []byte(`"control_revision"`)) {
		t.Fatalf("conflict event = %+v", conflict)
	}
	evidence := logs.String() + string(conflict.Data)
	for _, secret := range []string{token, key, "Authorization", "Idempotency-Key"} {
		if strings.Contains(evidence, secret) {
			t.Errorf("operational evidence leaked %q: %s", secret, evidence)
		}
	}
	if !strings.Contains(logs.String(), `"event":"manual_session.conflict"`) ||
		!strings.Contains(logs.String(), `"control_revision"`) {
		t.Fatalf("structured conflict log missing required fields: %s", logs.String())
	}
}

func TestFailedManualSessionCommandEventHasSafeCapabilityOutcome(t *testing.T) {
	base := observableStatus(false, false, false, false, false, 30)
	spa := newControllableSpa(base)
	spa.setErr = errors.New("device rejected command")
	handler, st, observationID := manualSessionTestAPI(t, spa, base)

	response := putManualSession(t, handler, "failed-command", controlRevision(t, handler), observationID, "30m", pool.ControllableState{
		Power: true, Filter: true, TargetTemp: 30,
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("PUT status = %d, body=%s", response.Code, response.Body.String())
	}
	_ = waitForManualSessionState(t, handler, "degraded")
	events, err := st.Events(context.Background(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "command" && bytes.Contains(event.Data, []byte(`"outcome":"failed"`)) {
			if !bytes.Contains(event.Data, []byte(`"capability":"power"`)) ||
				!bytes.Contains(event.Data, []byte(`"control_revision"`)) ||
				bytes.Contains(event.Data, []byte("device rejected command")) {
				t.Fatalf("unsafe or incomplete failed command event: %s", event.Data)
			}
			return
		}
	}
	t.Fatalf("events = %+v, want failed command capability outcome", events)
}

func manualSessionRequestWithToken(handler http.Handler, token, key string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/manual-session", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func observableStatus(power, filter, heater, jets, bubbles bool, targetTemp int) pool.Status {
	return pool.Status{
		ObservedAt: time.Now().UTC(), Connected: true, Power: power, Filter: filter,
		Heater: heater, Jets: jets, Bubbles: bubbles, TargetTemp: targetTemp,
	}
}
