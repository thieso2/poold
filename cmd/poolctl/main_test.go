package main

import (
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
