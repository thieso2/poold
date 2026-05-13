package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

func TestAuthRequired(t *testing.T) {
	handler, _ := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestWebUIIsPublic(t *testing.T) {
	handler, _ := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want html", contentType)
	}
	if !strings.Contains(rec.Body.String(), "Pooly Control") {
		t.Fatal("web UI title missing")
	}
	if !strings.Contains(rec.Body.String(), "webui-manual") {
		t.Fatal("manual control plan id missing")
	}
	if !strings.Contains(rec.Body.String(), "Make permanent") {
		t.Fatal("manual permanent action missing")
	}
	if !strings.Contains(rec.Body.String(), eChartsCDN) {
		t.Fatal("ECharts CDN script missing")
	}
	if strings.Contains(rec.Body.String(), "Desired State") || strings.Contains(rec.Body.String(), "Set Temperature") {
		t.Fatal("legacy desired/direct controls should not be visible")
	}
}

func TestHistoryUIIsPublic(t *testing.T) {
	handler, _ := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/history", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want html", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-page="history"`) {
		t.Fatal("history page mode missing")
	}
	if !strings.Contains(body, `href="/"`) || !strings.Contains(body, "Dashboard") {
		t.Fatal("history dashboard back link missing")
	}
	if !strings.Contains(body, eChartsCDN) {
		t.Fatal("ECharts CDN script missing")
	}
}

func TestFaviconIsPublic(t *testing.T) {
	handler, _ := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "image/svg+xml") {
		t.Fatalf("content type = %q, want svg", contentType)
	}
	if !strings.Contains(rec.Body.String(), "<svg") {
		t.Fatal("favicon svg missing")
	}
}

func TestAppleTouchIconIsPublicPNG(t *testing.T) {
	handler, _ := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/apple-touch-icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", contentType)
	}
	body := rec.Body.Bytes()
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("apple touch icon is not a PNG")
	}
}

func TestStatusEndpointRefreshesHardware(t *testing.T) {
	handler, fake := testAPI(t)
	fake.status = pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36, CurrentTemp: pool.IntPtr(32)}

	rec := authed(handler, http.MethodGet, "/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var status pool.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.CurrentTemp == nil || *status.CurrentTemp != 32 {
		t.Fatalf("status = %+v", status)
	}
}

func TestDesiredStateEndpoint(t *testing.T) {
	handler, fake := testAPI(t)
	fake.status = pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, Filter: true, TargetTemp: 36}

	rec := authed(handler, http.MethodPut, "/desired-state", []byte(`{"heater":true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var desired pool.DesiredState
	if err := json.Unmarshal(rec.Body.Bytes(), &desired); err != nil {
		t.Fatal(err)
	}
	if desired.Heater == nil || !*desired.Heater {
		t.Fatalf("desired = %+v", desired)
	}
	if desired.Filter != nil || desired.Power != nil {
		t.Fatalf("desired any values should be preserved in response: %+v", desired)
	}
	if got := fake.callCount(); got != 1 {
		t.Fatalf("set calls = %d, want heater enforcement", got)
	}
}

func TestCommandsEndpoint(t *testing.T) {
	handler, fake := testAPI(t)
	fake.status = pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}

	rec := authed(handler, http.MethodPost, "/commands", []byte(`{"capability":"filter","state":true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var record pool.CommandRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if !record.Success || record.Capability != "filter" {
		t.Fatalf("record = %+v", record)
	}
	if fake.lastCapability() != "filter" {
		t.Fatalf("last capability = %q", fake.lastCapability())
	}

	rec = authed(handler, http.MethodGet, "/commands?latest=1&limit=10", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Commands []pool.CommandRecord `json:"commands"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Commands) != 1 || response.Commands[0].Capability != "filter" || !response.Commands[0].Success {
		t.Fatalf("commands = %+v", response.Commands)
	}
}

func TestInactiveTimeWindowDoesNotUndoDirectFilterCommand(t *testing.T) {
	loc := time.FixedZone("TEST", 2*60*60)
	sched := scheduler.New(scheduler.Config{Location: loc})
	status := pool.Status{Power: true, Filter: true, TargetTemp: 36}
	eval := sched.Evaluate(
		time.Date(2026, 5, 4, 16, 30, 0, 0, loc),
		status,
		pool.DesiredState{TargetTemp: pool.IntPtr(36)},
		[]pool.Plan{{
			ID:         "evening-filter",
			Type:       pool.PlanTimeWindow,
			Enabled:    true,
			Capability: "filter",
			From:       "18:00",
			To:         "21:00",
		}},
	)
	for _, command := range diffCommands(status, eval.Desired) {
		if command.capability == "filter" {
			t.Fatalf("inactive filter window should not undo direct filter command: %+v", command)
		}
	}
}

func TestPlansEndpoint(t *testing.T) {
	handler, _ := testAPI(t)
	body := []byte(`{"plans":[{"id":"filter","type":"time_window","enabled":true,"capability":"filter","from":"02:00","to":"04:00"}]}`)
	rec := authed(handler, http.MethodPut, "/plans", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = authed(handler, http.MethodGet, "/plans", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Plans []pool.Plan `json:"plans"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Plans) != 1 || response.Plans[0].ID != "filter" {
		t.Fatalf("plans = %+v", response.Plans)
	}
}

func TestPlansEndpointAcceptsPermanentManualOverride(t *testing.T) {
	handler, _ := testAPI(t)
	body := []byte(`{"plans":[{"id":"webui-manual","type":"manual_override","enabled":true,"desired_state":{"heater":false,"filter":false}}]}`)
	rec := authed(handler, http.MethodPut, "/plans", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var response struct {
		Plans []pool.Plan `json:"plans"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Plans) != 1 || response.Plans[0].ExpiresAt != nil {
		t.Fatalf("plans = %+v", response.Plans)
	}
}

func TestEventsEndpoint(t *testing.T) {
	handler, _ := testAPI(t)
	rec := authed(handler, http.MethodPut, "/desired-state", []byte(`{"filter":true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = authed(handler, http.MethodGet, "/events?after=0&limit=10", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Events []pool.Event `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) == 0 {
		t.Fatal("expected events")
	}
}

func TestObservationsEndpoint(t *testing.T) {
	handler, fake := testAPI(t)
	fake.status = pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36, CurrentTemp: pool.IntPtr(32)}

	rec := authed(handler, http.MethodGet, "/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = authed(handler, http.MethodGet, "/observations?after=0&limit=10", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Observations []pool.Observation `json:"observations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Observations) != 1 || response.Observations[0].Status.CurrentTemp == nil || *response.Observations[0].Status.CurrentTemp != 32 {
		t.Fatalf("observations = %+v", response.Observations)
	}
}

func TestHeatingSessionsEndpoint(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	at := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	for _, status := range []pool.Status{
		{ObservedAt: at, Connected: true, Power: true, Filter: true, Heater: true, CurrentTemp: pool.IntPtr(30), TargetTemp: 36},
		{ObservedAt: at.Add(10 * time.Minute), Connected: true, Power: true, Filter: true, Heater: true, CurrentTemp: pool.IntPtr(31), TargetTemp: 36},
		{ObservedAt: at.Add(20 * time.Minute), Connected: true, Power: true, Filter: true, Heater: false, CurrentTemp: pool.IntPtr(31), TargetTemp: 36},
	} {
		if _, err := st.SaveObservation(ctx, status); err != nil {
			t.Fatal(err)
		}
	}

	fake := &fakePoolClient{status: pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{})
	handler := New(service, "secret")

	rec := authed(handler, http.MethodGet, "/heating-sessions?limit=10", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Sessions []pool.HeatingSession `json:"heating_sessions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Sessions) != 1 || response.Sessions[0].DurationSeconds != int64((20*time.Minute).Seconds()) || response.Sessions[0].Active {
		t.Fatalf("sessions = %+v", response.Sessions)
	}
}

func TestWeatherSettingsEndpoint(t *testing.T) {
	handler, _, fakeWeather := testAPIWithWeather(t)
	fakeWeather.raw = json.RawMessage(`{"main":{"temp":21.5},"weather":[{"main":"Clouds","description":"broken clouds"}],"clouds":{"all":75},"name":"Berlin"}`)

	rec := authed(handler, http.MethodPut, "/weather/settings", []byte(`{"api_key":"owm-secret","location":"Berlin,DE"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "owm-secret") {
		t.Fatal("weather API key leaked in response")
	}

	rec = authed(handler, http.MethodGet, "/weather", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Settings WeatherSettingsView     `json:"settings"`
		Latest   pool.WeatherObservation `json:"latest"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Settings.APIKeySet || response.Settings.Location.Name != "Berlin" {
		t.Fatalf("settings = %+v", response.Settings)
	}
	if response.Latest.ID == 0 || !strings.Contains(string(response.Latest.Data), "broken clouds") {
		t.Fatalf("latest = %+v", response.Latest)
	}
}

func testAPI(t *testing.T) (http.Handler, *fakePoolClient) {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	fake := &fakePoolClient{status: pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{})
	return New(service, "secret"), fake
}

func testAPIWithWeather(t *testing.T) (http.Handler, *fakePoolClient, *fakeWeatherProvider) {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	fake := &fakePoolClient{status: pool.Status{ObservedAt: time.Now().UTC(), Connected: true, Power: true, TargetTemp: 36}}
	fakeWeather := &fakeWeatherProvider{}
	service := NewService(st, fake, scheduler.New(scheduler.Config{}), ServiceConfig{WeatherProvider: fakeWeather})
	return New(service, "secret"), fake, fakeWeather
}

func authed(handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer secret")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type fakePoolClient struct {
	mu     sync.Mutex
	status pool.Status
	calls  []string
}

func (f *fakePoolClient) Status(context.Context) (pool.Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, nil
}

func (f *fakePoolClient) Set(_ context.Context, capability string, value any) (pool.Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, capability)
	switch capability {
	case "power":
		f.status.Power = boolValue(value)
	case "filter":
		f.status.Filter = boolValue(value)
	case "heater":
		f.status.Heater = boolValue(value)
	case "jets":
		f.status.Jets = boolValue(value)
	case "bubbles":
		f.status.Bubbles = boolValue(value)
	case "sanitizer":
		f.status.Sanitizer = boolValue(value)
	case "target_temp":
		raw, _ := value.([]byte)
		temp, _ := strconv.Atoi(string(raw))
		f.status.TargetTemp = temp
	}
	f.status.ObservedAt = time.Now().UTC()
	f.status.Connected = true
	return f.status, nil
}

func (f *fakePoolClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakePoolClient) lastCapability() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return ""
	}
	return f.calls[len(f.calls)-1]
}

func boolValue(value any) bool {
	v, _ := value.(bool)
	return v
}

type fakeWeatherProvider struct {
	raw json.RawMessage
}

func (f *fakeWeatherProvider) ResolveLocation(context.Context, string, string) (pool.WeatherLocation, error) {
	return pool.WeatherLocation{Query: "Berlin,DE", Name: "Berlin", Country: "DE", Lat: 52.52, Lon: 13.405}, nil
}

func (f *fakeWeatherProvider) CurrentWeather(context.Context, string, pool.WeatherLocation) (json.RawMessage, error) {
	if len(f.raw) == 0 {
		return json.RawMessage(`{"main":{"temp":20},"weather":[{"main":"Clear","description":"clear sky"}]}`), nil
	}
	return f.raw, nil
}
