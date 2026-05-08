package config

import (
	"testing"
	"time"
)

func TestLoadUsesDataDatabasePathForOpenWrt(t *testing.T) {
	t.Setenv("POOLD_ENV", "openwrt")
	t.Setenv("POOLD_DB_PATH", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabasePath != "/data/poold.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "/data/poold.db")
	}
}

func TestLoadDatabasePathEnvOverrideWins(t *testing.T) {
	t.Setenv("POOLD_ENV", "openwrt")
	t.Setenv("POOLD_DB_PATH", "/tmp/custom.db")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabasePath != "/tmp/custom.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "/tmp/custom.db")
	}
}

func TestLoadObservationFlushInterval(t *testing.T) {
	t.Setenv("POOLD_OBSERVATION_FLUSH_INTERVAL", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ObservationFlushInterval != 15*time.Minute {
		t.Fatalf("ObservationFlushInterval = %s, want 15m", cfg.ObservationFlushInterval)
	}

	t.Setenv("POOLD_OBSERVATION_FLUSH_INTERVAL", "2m")
	cfg, err = Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ObservationFlushInterval != 2*time.Minute {
		t.Fatalf("ObservationFlushInterval = %s, want 2m", cfg.ObservationFlushInterval)
	}
}
