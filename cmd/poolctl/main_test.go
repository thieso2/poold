package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestWatchAfterIDDefaultsToLatestEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		switch after {
		case "0":
			fmt.Fprint(w, `{"events":[{"id":1},{"id":2}]}`)
		case "2":
			fmt.Fprint(w, `{"events":[]}`)
		default:
			t.Fatalf("unexpected after query %q", after)
		}
	}))
	defer server.Close()

	c := client{baseURL: server.URL, token: "test", http: server.Client()}
	got, err := c.watchAfterID(watchOptions{After: -1})
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("after = %d, want latest event id 2", got)
	}
}

func TestWatchAfterIDExplicitOptions(t *testing.T) {
	c := client{}
	if got, err := c.watchAfterID(watchOptions{After: 42}); err != nil || got != 42 {
		t.Fatalf("explicit after = %d, %v; want 42", got, err)
	}
	if got, err := c.watchAfterID(watchOptions{After: -1, FromStart: true}); err != nil || got != 0 {
		t.Fatalf("from-start after = %d, %v; want 0", got, err)
	}
}
