package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
)

func TestDashboardTimelineEndpoint(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	base := time.Now().UTC().Add(-90 * time.Minute).Truncate(time.Minute)
	saveWeather(t, st, base, 18)
	saveObservation(t, st, base, 30, 36, false, true)
	saveObservation(t, st, base.Add(30*time.Minute), 30, 36, true, true)
	saveObservation(t, st, base.Add(90*time.Minute), 31, 36, true, true)
	if _, err := st.InsertCommand(ctx, pool.CommandRecord{
		IssuedAt:   base.Add(20 * time.Minute),
		Capability: "heater",
		State:      pool.BoolPtr(true),
		Source:     "test",
		Success:    true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddEvent(ctx, "scheduler", "desired state enforced", map[string]any{
		"source": "ready",
		"reason": "ready-by heating window started",
	}); err != nil {
		t.Fatal(err)
	}

	service := NewService(st, intex.New("127.0.0.1:1"), scheduler.New(scheduler.Config{}), ServiceConfig{})
	handler := New(service, "secret")
	rec := authed(handler, http.MethodGet, "/dashboard/timeline?from="+base.Format(time.RFC3339)+"&to="+base.Add(2*time.Hour).Format(time.RFC3339), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response pool.TimelineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Unit != "C" || len(response.Measured) == 0 || len(response.Predicted) == 0 || len(response.FeatureSpans) == 0 {
		t.Fatalf("timeline response missing chart data: %+v", response)
	}
	if !response.WeatherAvailable {
		t.Fatalf("weather should be available: %+v", response.Measured)
	}
	if len(response.Annotations) < 2 {
		t.Fatalf("annotations = %+v, want command and scheduler", response.Annotations)
	}
}

func TestDashboardTimelineLearnsCoolingRate(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	base := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	saveObservation(t, st, base, 34, 34, false, false)
	saveObservation(t, st, base.Add(2*time.Hour), 33, 34, false, false)
	saveObservation(t, st, base.Add(4*time.Hour), 32, 34, false, false)
	saveObservation(t, st, base.Add(6*time.Hour), 31, 34, false, false)

	service := NewService(st, intex.New("127.0.0.1:1"), scheduler.New(scheduler.Config{}), ServiceConfig{
		CoolingRateCPerHour: 0.1,
	})
	timeline, err := service.DashboardTimeline(ctx, TimelineQuery{From: base, To: base.Add(6 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if timeline.Model.CoolingModel != "learned_cooling_unknown" {
		t.Fatalf("CoolingModel = %q, want learned_cooling_unknown", timeline.Model.CoolingModel)
	}
	if timeline.Model.CoolingSamples != 3 {
		t.Fatalf("CoolingSamples = %d, want 3", timeline.Model.CoolingSamples)
	}
	if timeline.Model.CoolingRateCPerHour != 0.5 {
		t.Fatalf("CoolingRateCPerHour = %v, want 0.5", timeline.Model.CoolingRateCPerHour)
	}
}

func saveObservation(t *testing.T, st *store.Store, observedAt time.Time, current, target int, heater, power bool) {
	t.Helper()
	_, err := st.SaveObservation(context.Background(), pool.Status{
		ObservedAt:  observedAt,
		Connected:   true,
		Power:       power,
		Filter:      heater,
		Heater:      heater,
		CurrentTemp: pool.IntPtr(current),
		TargetTemp:  target,
		Unit:        "C",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func saveWeather(t *testing.T, st *store.Store, observedAt time.Time, outsideTemp float64) {
	t.Helper()
	_, err := st.SaveWeatherObservation(context.Background(), pool.WeatherObservation{
		ObservedAt: observedAt,
		Location:   pool.WeatherLocation{Query: "Berlin,DE", Name: "Berlin"},
		Data:       json.RawMessage(`{"main":{"temp":` + strconvFormatFloat(outsideTemp) + `},"weather":[{"main":"Clear","description":"clear"}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func strconvFormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
