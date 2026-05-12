package store

import (
	"context"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
)

func TestStoreObservationDesiredPlansAndEvents(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	current := 32
	status := pool.Status{
		ObservedAt:  time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
		Connected:   true,
		Power:       true,
		Filter:      true,
		CurrentTemp: &current,
		TargetTemp:  36,
	}
	observationID, err := st.SaveObservation(ctx, status)
	if err != nil {
		t.Fatal(err)
	}
	repeated := status
	repeated.ObservedAt = status.ObservedAt.Add(time.Minute)
	repeatedID, err := st.SaveObservation(ctx, repeated)
	if err != nil {
		t.Fatal(err)
	}
	if repeatedID != observationID {
		t.Fatalf("repeated observation id = %d, want span id %d", repeatedID, observationID)
	}
	latest, ok, err := st.LatestStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || latest.CurrentTemp == nil || *latest.CurrentTemp != 32 || !latest.ObservedAt.Equal(repeated.ObservedAt) {
		t.Fatalf("latest = %+v, ok=%v", latest, ok)
	}
	observations, err := st.Observations(ctx, observationID-1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 ||
		observations[0].ID != observationID ||
		observations[0].ObservationCount != 2 ||
		!observations[0].FirstObservedAt.Equal(status.ObservedAt) ||
		!observations[0].LastObservedAt.Equal(repeated.ObservedAt) ||
		observations[0].Status.CurrentTemp == nil ||
		*observations[0].Status.CurrentTemp != 32 {
		t.Fatalf("observations = %+v", observations)
	}
	latestObservations, err := st.LatestObservations(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(latestObservations) != 1 || latestObservations[0].ID != observationID {
		t.Fatalf("latest observations = %+v", latestObservations)
	}

	desired := pool.DesiredState{Heater: pool.BoolPtr(true)}
	if err := st.SaveDesiredState(ctx, desired); err != nil {
		t.Fatal(err)
	}
	savedDesired, err := st.DesiredState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if savedDesired.Heater == nil || !*savedDesired.Heater || savedDesired.Filter != nil || savedDesired.Power != nil {
		t.Fatalf("desired any values were not preserved: %+v", savedDesired)
	}

	event, err := st.AddEvent(ctx, "test", "hello", map[string]string{"ok": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddEvent(ctx, "observation", "status heartbeat", status); err != nil {
		t.Fatal(err)
	}
	changedEvent, err := st.AddEvent(ctx, "observation", "status refreshed", repeated)
	if err != nil {
		t.Fatal(err)
	}
	schedulerEvent, err := st.AddEvent(ctx, "scheduler", "desired state enforced", map[string]string{"source": "daily-filter"})
	if err != nil {
		t.Fatal(err)
	}
	events, err := st.Events(ctx, event.ID-1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "test" {
		t.Fatalf("events = %+v", events)
	}
	latestEvents, err := st.LatestEvents(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(latestEvents) != 1 || latestEvents[0].ID != schedulerEvent.ID {
		t.Fatalf("latest events = %+v", latestEvents)
	}
	changedEvents, err := st.EventsPage(ctx, EventQuery{Limit: 10, Latest: true, ChangesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range changedEvents {
		if event.Message == "status heartbeat" {
			t.Fatalf("heartbeat was not filtered: %+v", changedEvents)
		}
	}
	schedulerEvents, err := st.EventsPage(ctx, EventQuery{Limit: 1, Offset: 0, Latest: true, Type: "scheduler"})
	if err != nil {
		t.Fatal(err)
	}
	if len(schedulerEvents) != 1 || schedulerEvents[0].ID != schedulerEvent.ID {
		t.Fatalf("scheduler events = %+v", schedulerEvents)
	}
	observationEvents, err := st.EventsPage(ctx, EventQuery{Limit: 1, Offset: 0, Latest: true, Type: "observation", ChangesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(observationEvents) != 1 || observationEvents[0].ID != changedEvent.ID {
		t.Fatalf("observation events = %+v", observationEvents)
	}

	plan := pool.Plan{
		ID:         "daily-filter",
		Type:       pool.PlanTimeWindow,
		Enabled:    true,
		Capability: "filter",
		From:       "02:00",
		To:         "04:00",
	}
	if err := st.SavePlans(ctx, []pool.Plan{plan}); err != nil {
		t.Fatal(err)
	}
	plans, err := st.Plans(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].ID != "daily-filter" || plans[0].CreatedAt.IsZero() || plans[0].UpdatedAt.IsZero() {
		t.Fatalf("plans = %+v", plans)
	}

	settings := pool.WeatherSettings{
		APIKey: "secret",
		Location: pool.WeatherLocation{
			Query: "Berlin,DE",
			Name:  "Berlin",
			Lat:   52.52,
			Lon:   13.405,
		},
	}
	if err := st.SaveWeatherSettings(ctx, settings); err != nil {
		t.Fatal(err)
	}
	savedWeather, err := st.WeatherSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if savedWeather.APIKey != "secret" || savedWeather.Location.Name != "Berlin" || savedWeather.UpdatedAt.IsZero() {
		t.Fatalf("weather settings = %+v", savedWeather)
	}
	weatherID, err := st.SaveWeatherObservation(ctx, pool.WeatherObservation{
		ObservedAt: status.ObservedAt,
		Location:   savedWeather.Location,
		Data:       []byte(`{"main":{"temp":21.5},"weather":[{"main":"Clouds","description":"broken clouds"}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	latestWeather, ok, err := st.LatestWeatherObservation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || latestWeather.ID != weatherID || latestWeather.Location.Name != "Berlin" || len(latestWeather.Data) == 0 {
		t.Fatalf("latest weather = %+v ok=%v", latestWeather, ok)
	}
	weatherStatus := status
	weatherStatus.ObservedAt = status.ObservedAt.Add(2 * time.Minute)
	weatherStatus.CurrentTemp = pool.IntPtr(33)
	if _, err := st.SaveObservation(ctx, weatherStatus); err != nil {
		t.Fatal(err)
	}
	latestObservations, err = st.LatestObservations(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(latestObservations) != 1 || latestObservations[0].Weather == nil || latestObservations[0].Weather.ObservationID != weatherID {
		t.Fatalf("weather snapshot not attached to observation: %+v", latestObservations)
	}
}

func TestStoreThrottlesUnchangedObservationFlushes(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	base := pool.Status{
		ObservedAt:  time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
		Connected:   true,
		Power:       true,
		CurrentTemp: pool.IntPtr(30),
		TargetTemp:  36,
	}
	id, err := st.SaveObservation(ctx, base)
	if err != nil {
		t.Fatal(err)
	}

	repeated := base
	repeated.ObservedAt = base.ObservedAt.Add(time.Minute)
	repeatedID, err := st.SaveObservationThrottled(ctx, repeated, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if repeatedID != id {
		t.Fatalf("repeated observation id = %d, want %d", repeatedID, id)
	}
	latest, ok, err := st.LatestStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !latest.ObservedAt.Equal(repeated.ObservedAt) {
		t.Fatalf("latest = %+v, ok=%v", latest, ok)
	}
	observations, err := st.Observations(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 || observations[0].ObservationCount != 1 || !observations[0].LastObservedAt.Equal(base.ObservedAt) {
		t.Fatalf("observations before flush = %+v", observations)
	}

	if err := st.FlushObservations(ctx); err != nil {
		t.Fatal(err)
	}
	observations, err = st.Observations(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 || observations[0].ObservationCount != 2 || !observations[0].LastObservedAt.Equal(repeated.ObservedAt) {
		t.Fatalf("observations after flush = %+v", observations)
	}
}

func TestStoreFlushesPendingObservationBeforeStateChange(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	base := pool.Status{
		ObservedAt:  time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
		Connected:   true,
		Power:       true,
		CurrentTemp: pool.IntPtr(30),
		TargetTemp:  36,
	}
	if _, err := st.SaveObservation(ctx, base); err != nil {
		t.Fatal(err)
	}
	repeated := base
	repeated.ObservedAt = base.ObservedAt.Add(time.Minute)
	if _, err := st.SaveObservationThrottled(ctx, repeated, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	changed := repeated
	changed.ObservedAt = base.ObservedAt.Add(2 * time.Minute)
	changed.CurrentTemp = pool.IntPtr(31)
	if _, err := st.SaveObservationThrottled(ctx, changed, 15*time.Minute); err != nil {
		t.Fatal(err)
	}

	observations, err := st.Observations(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 2 {
		t.Fatalf("observations = %+v", observations)
	}
	if observations[0].ObservationCount != 2 || !observations[0].LastObservedAt.Equal(repeated.ObservedAt) {
		t.Fatalf("first span = %+v", observations[0])
	}
	if observations[1].ObservationCount != 1 || observations[1].Status.CurrentTemp == nil || *observations[1].Status.CurrentTemp != 31 {
		t.Fatalf("changed span = %+v", observations[1])
	}
}

func TestStoreCommandAndRetention(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	id, err := st.InsertCommand(ctx, pool.CommandRecord{Capability: "filter", State: pool.BoolPtr(true), Source: "test"})
	if err != nil {
		t.Fatal(err)
	}
	status := pool.Status{ObservedAt: time.Now().UTC(), Connected: true}
	if err := st.FinishCommand(ctx, id, true, &status, nil); err != nil {
		t.Fatal(err)
	}
	commands, err := st.LatestCommands(ctx, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].ID != id || commands[0].State == nil || !*commands[0].State || !commands[0].Success || commands[0].Status == nil {
		t.Fatalf("commands = %+v", commands)
	}
	commands, err = st.Commands(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].ID != id {
		t.Fatalf("commands after = %+v", commands)
	}
	lastCommandAt, err := st.LastCommandAt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lastCommandAt == nil {
		t.Fatal("expected last command time")
	}

	old := pool.Status{ObservedAt: time.Now().Add(-48 * time.Hour).UTC()}
	if _, err := st.SaveObservation(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := st.Prune(ctx, time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if latest, ok, err := st.LatestStatus(ctx); err != nil {
		t.Fatal(err)
	} else if ok && latest.ObservedAt.Equal(old.ObservedAt) {
		t.Fatalf("old observation was not pruned: %+v", latest)
	}
}

func TestStoreReadyByControlState(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	readyAt := time.Date(2026, 5, 9, 6, 30, 0, 0, time.UTC)
	state := pool.ReadyByControlState{
		Key:        "ready|2026-05-09T06:30:00Z|36",
		PlanID:     "ready",
		ReadyAt:    readyAt,
		TargetTemp: 36,
		Mode:       pool.ReadyBySatisfiedIdle,
		UpdatedAt:  readyAt.Add(time.Minute),
	}
	if err := st.SaveReadyByControlState(ctx, state); err != nil {
		t.Fatal(err)
	}
	states, err := st.ReadyByControlStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	saved, ok := states[state.Key]
	if !ok {
		t.Fatalf("state %q not found in %+v", state.Key, states)
	}
	if saved.PlanID != "ready" || !saved.ReadyAt.Equal(readyAt) || saved.TargetTemp != 36 || saved.Mode != pool.ReadyBySatisfiedIdle || !saved.UpdatedAt.Equal(state.UpdatedAt) {
		t.Fatalf("saved state = %+v", saved)
	}
}

func TestStoreHeatingSessions(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)

	at := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	save := func(offset time.Duration, heater bool, current int) {
		t.Helper()
		if _, err := st.SaveObservation(ctx, pool.Status{
			ObservedAt:  at.Add(offset),
			Connected:   true,
			Power:       true,
			Filter:      heater,
			Heater:      heater,
			CurrentTemp: pool.IntPtr(current),
			TargetTemp:  36,
			Unit:        "°C",
		}); err != nil {
			t.Fatal(err)
		}
	}
	save(0, false, 29)
	save(10*time.Minute, true, 30)
	save(20*time.Minute, true, 30)
	save(30*time.Minute, true, 31)
	save(45*time.Minute, false, 31)
	save(60*time.Minute, true, 32)

	sessions, err := st.LatestHeatingSessions(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %+v", sessions)
	}
	if !sessions[0].Active || sessions[0].StartTemp == nil || *sessions[0].StartTemp != 32 {
		t.Fatalf("active session = %+v", sessions[0])
	}
	if sessions[1].Active || sessions[1].EndedAt == nil || sessions[1].DurationSeconds != int64((35*time.Minute).Seconds()) || sessions[1].ObservationCount != 3 {
		t.Fatalf("ended session = %+v", sessions[1])
	}
	paged, err := st.LatestHeatingSessions(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(paged) != 1 || paged[0].ID != sessions[1].ID {
		t.Fatalf("paged sessions = %+v", paged)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), t.TempDir()+"/poold.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
