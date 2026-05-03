package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pooly/services/poold/internal/pool"
)

func TestFormatWatchEventObservation(t *testing.T) {
	raw := `{"id":82,"created_at":"2000-01-02T03:04:05Z","type":"observation","message":"status refreshed","data":{"observed_at":"2000-01-02T03:04:05Z","connected":true,"power":true,"filter":true,"heater":true,"jets":false,"bubbles":false,"sanitizer":false,"unit":"\u00b0C","current_temp":30,"preset_temp":36,"raw_data":"FFFF"}}`

	got := formatWatchEvent(raw, time.UTC)
	want := "2000-01-02 03:04:05  #82  STATUS  30\u00b0C -> 36\u00b0C  power filter heater"
	if got != want {
		t.Fatalf("formatWatchEvent() = %q, want %q", got, want)
	}
}

func TestFormatWatchEventStatusError(t *testing.T) {
	raw := `{"id":85,"created_at":"2000-01-02T03:04:05Z","type":"status_error","message":"status refresh failed","data":{"error":"invalid status response: result=\"timeout\" type=2"}}`

	got := formatWatchEvent(raw, time.UTC)
	if !strings.Contains(got, "#85  ERROR") {
		t.Fatalf("formatted error missing id/type: %q", got)
	}
	if !strings.Contains(got, `invalid status response: result="timeout" type=2`) {
		t.Fatalf("formatted error missing message: %q", got)
	}
}

func TestReplayWatchHistoryDefaultsToLastHour(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		switch after {
		case "0":
			fmt.Fprint(w, `{"events":[{"id":1,"created_at":"2026-05-03T10:00:00Z"},{"id":2,"created_at":"2026-05-03T11:30:00Z"}]}`)
		case "2":
			fmt.Fprint(w, `{"events":[]}`)
		default:
			t.Fatalf("unexpected after query %q", after)
		}
	}))
	defer server.Close()

	c := client{baseURL: server.URL, token: "test", http: server.Client()}
	var emitted []int64
	got, err := c.replayWatchHistory(watchOptions{After: -1, History: time.Hour}, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC), func(event pool.Event) error {
		emitted = append(emitted, event.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("after = %d, want latest event id 2", got)
	}
	if len(emitted) != 1 || emitted[0] != 2 {
		t.Fatalf("emitted = %+v, want only last-hour event 2", emitted)
	}
}

func TestReplayWatchHistoryExplicitAfterReplaysFromAfterID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		switch after {
		case "42":
			fmt.Fprint(w, `{"events":[{"id":43,"created_at":"2026-05-03T10:00:00Z"}]}`)
		case "43":
			fmt.Fprint(w, `{"events":[]}`)
		default:
			t.Fatalf("unexpected after query %q", after)
		}
	}))
	defer server.Close()

	c := client{baseURL: server.URL, token: "test", http: server.Client()}
	var emitted []int64
	got, err := c.replayWatchHistory(watchOptions{After: 42}, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC), func(event pool.Event) error {
		emitted = append(emitted, event.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 43 {
		t.Fatalf("after = %d, want 43", got)
	}
	if len(emitted) != 1 || emitted[0] != 43 {
		t.Fatalf("emitted = %+v, want event 43", emitted)
	}
}
