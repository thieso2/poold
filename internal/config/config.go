package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr           string
	PoolAddr             string
	DatabasePath         string
	Token                string
	Location             *time.Location
	HeatingRateCPerHour  float64
	ReadinessBuffer      time.Duration
	PollInterval         time.Duration
	ObservationRetention time.Duration
	EventRetention       time.Duration
}

func Load(args []string) (Config, error) {
	cfg := Config{
		ListenAddr:           envString("POOLD_LISTEN_ADDR", "127.0.0.1:8090"),
		PoolAddr:             envString("POOLD_POOL_ADDR", "127.0.0.1:8990"),
		DatabasePath:         envString("POOLD_DB_PATH", defaultDBPath()),
		Token:                envString("POOLD_TOKEN", "dev-token"),
		HeatingRateCPerHour:  envFloat("POOLD_HEATING_RATE_C_PER_HOUR", 0.75),
		ReadinessBuffer:      envDuration("POOLD_READINESS_BUFFER", 30*time.Minute),
		PollInterval:         envDuration("POOLD_POLL_INTERVAL", 30*time.Second),
		ObservationRetention: envDuration("POOLD_OBSERVATION_RETENTION", 14*24*time.Hour),
		EventRetention:       envDuration("POOLD_EVENT_RETENTION", 14*24*time.Hour),
	}

	locationName := envString("POOLD_TIMEZONE", "Europe/Berlin")
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return Config{}, fmt.Errorf("load timezone %q: %w", locationName, err)
	}
	cfg.Location = location

	fs := flag.NewFlagSet("poold", flag.ContinueOnError)
	fs.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "HTTP listen address")
	fs.StringVar(&cfg.PoolAddr, "pool", cfg.PoolAddr, "Intex pool TCP address")
	fs.StringVar(&cfg.DatabasePath, "db", cfg.DatabasePath, "SQLite database path")
	fs.StringVar(&cfg.Token, "token", cfg.Token, "HTTP bearer token")
	fs.Float64Var(&cfg.HeatingRateCPerHour, "heating-rate", cfg.HeatingRateCPerHour, "heating rate in C per hour")
	fs.DurationVar(&cfg.ReadinessBuffer, "readiness-buffer", cfg.ReadinessBuffer, "ready-by safety buffer")
	fs.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "scheduler/status poll interval")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaultDBPath() string {
	if os.Getenv("POOLD_ENV") == "openwrt" || os.Getenv("POOLD_ENV") == "production" {
		return "/var/lib/poold/poold.db"
	}
	if runtime.GOOS == "linux" && os.Getuid() == 0 {
		return "/var/lib/poold/poold.db"
	}
	return "./var/poold.db"
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
