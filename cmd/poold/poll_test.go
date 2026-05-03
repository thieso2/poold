package main

import (
	"errors"
	"testing"
	"time"

	"pooly/services/poold/internal/config"
	"pooly/services/poold/internal/pool"
)

func TestAdaptivePollerStartupUntilFirstSuccess(t *testing.T) {
	poller := newAdaptivePoller(testPollConfig())

	got := poller.Next(pool.Status{}, errors.New("offline"))
	if got != 10*time.Second {
		t.Fatalf("first error interval = %s, want 10s", got)
	}
}

func TestAdaptivePollerStatusIntervals(t *testing.T) {
	poller := newAdaptivePoller(testPollConfig())

	if got := poller.Next(pool.Status{Power: false}, nil); got != 10*time.Minute {
		t.Fatalf("power-off interval = %s, want 10m", got)
	}
	if got := poller.Next(pool.Status{Power: true}, nil); got != 5*time.Minute {
		t.Fatalf("stable interval = %s, want 5m", got)
	}
	if got := poller.Next(pool.Status{Power: true, Heater: true}, nil); got != time.Minute {
		t.Fatalf("active interval = %s, want 1m", got)
	}
}

func TestAdaptivePollerBackoffAfterSuccess(t *testing.T) {
	poller := newAdaptivePoller(testPollConfig())
	_ = poller.Next(pool.Status{Power: true}, nil)

	for _, want := range []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute} {
		got := poller.Next(pool.Status{}, errors.New("timeout"))
		if got != want {
			t.Fatalf("error interval = %s, want %s", got, want)
		}
	}

	if got := poller.Next(pool.Status{Power: true}, nil); got != 5*time.Minute {
		t.Fatalf("recovered interval = %s, want 5m", got)
	}
}

func testPollConfig() config.Config {
	return config.Config{
		PollStartupInterval:  10 * time.Second,
		PollIdleInterval:     10 * time.Minute,
		PollStableInterval:   5 * time.Minute,
		PollActiveInterval:   time.Minute,
		PollErrorMinInterval: 30 * time.Second,
		PollErrorMaxInterval: 5 * time.Minute,
	}
}
