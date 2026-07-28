package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr               string
	PoolAddr                 string
	DatabasePath             string
	Token                    string
	Location                 *time.Location
	HeatingRateCPerHour      float64
	CoolingRateCPerHour      float64
	ReadinessBuffer          time.Duration
	ReadyByReheatDelta       int
	PollStartupInterval      time.Duration
	PollIdleInterval         time.Duration
	PollStableInterval       time.Duration
	PollActiveInterval       time.Duration
	PollErrorMinInterval     time.Duration
	PollErrorMaxInterval     time.Duration
	WeatherPollInterval      time.Duration
	CommandConfirmDelay      time.Duration
	EventHeartbeat           time.Duration
	ObservationFlushInterval time.Duration
	ObservationRetention     time.Duration
	EventRetention           time.Duration
}

func Load(args []string) (Config, error) {
	cfg := Config{
		ListenAddr:               envString("POOLD_LISTEN_ADDR", "127.0.0.1:8090"),
		PoolAddr:                 envString("POOLD_POOL_ADDR", "127.0.0.1:8990"),
		DatabasePath:             envString("POOLD_DB_PATH", defaultDBPath()),
		Token:                    envString("POOLD_TOKEN", "dev-token"),
		HeatingRateCPerHour:      envFloat("POOLD_HEATING_RATE_C_PER_HOUR", 0.75),
		CoolingRateCPerHour:      envFloat("POOLD_COOLING_RATE_C_PER_HOUR", 0.10),
		ReadinessBuffer:          envDuration("POOLD_READINESS_BUFFER", 30*time.Minute),
		ReadyByReheatDelta:       envInt("POOLD_READY_BY_REHEAT_DELTA", 2),
		PollStartupInterval:      envDuration("POOLD_POLL_STARTUP_INTERVAL", 10*time.Second),
		PollIdleInterval:         envDuration("POOLD_POLL_IDLE_INTERVAL", 10*time.Minute),
		PollStableInterval:       envDuration("POOLD_POLL_STABLE_INTERVAL", envDuration("POOLD_POLL_INTERVAL", 5*time.Minute)),
		PollActiveInterval:       envDuration("POOLD_POLL_ACTIVE_INTERVAL", 1*time.Minute),
		PollErrorMinInterval:     envDuration("POOLD_POLL_ERROR_MIN_INTERVAL", 30*time.Second),
		PollErrorMaxInterval:     envDuration("POOLD_POLL_ERROR_MAX_INTERVAL", 5*time.Minute),
		WeatherPollInterval:      envDuration("POOLD_WEATHER_POLL_INTERVAL", 5*time.Minute),
		CommandConfirmDelay:      envDuration("POOLD_COMMAND_CONFIRM_DELAY", 10*time.Second),
		EventHeartbeat:           envDuration("POOLD_EVENT_HEARTBEAT", 30*time.Minute),
		ObservationFlushInterval: envDuration("POOLD_OBSERVATION_FLUSH_INTERVAL", 15*time.Minute),
		ObservationRetention:     envDuration("POOLD_OBSERVATION_RETENTION", 14*24*time.Hour),
		EventRetention:           envDuration("POOLD_EVENT_RETENTION", 14*24*time.Hour),
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
	fs.Float64Var(&cfg.CoolingRateCPerHour, "cooling-rate", cfg.CoolingRateCPerHour, "fallback cooling rate in C per hour")
	fs.DurationVar(&cfg.ReadinessBuffer, "readiness-buffer", cfg.ReadinessBuffer, "ready-by safety buffer")
	fs.IntVar(&cfg.ReadyByReheatDelta, "ready-by-reheat-delta", cfg.ReadyByReheatDelta, "ready-by reheating threshold below target in degrees")
	fs.DurationVar(&cfg.PollStableInterval, "poll-interval", cfg.PollStableInterval, "stable scheduler/status poll interval")
	fs.DurationVar(&cfg.PollStartupInterval, "poll-startup-interval", cfg.PollStartupInterval, "startup status poll interval before first success")
	fs.DurationVar(&cfg.PollIdleInterval, "poll-idle-interval", cfg.PollIdleInterval, "idle or power-off status poll interval")
	fs.DurationVar(&cfg.PollActiveInterval, "poll-active-interval", cfg.PollActiveInterval, "status poll interval while equipment is active")
	fs.DurationVar(&cfg.PollErrorMinInterval, "poll-error-min-interval", cfg.PollErrorMinInterval, "initial status error backoff interval")
	fs.DurationVar(&cfg.PollErrorMaxInterval, "poll-error-max-interval", cfg.PollErrorMaxInterval, "maximum status error backoff interval")
	fs.DurationVar(&cfg.WeatherPollInterval, "weather-poll-interval", cfg.WeatherPollInterval, "OpenWeatherMap poll interval")
	fs.DurationVar(&cfg.CommandConfirmDelay, "command-confirm-delay", cfg.CommandConfirmDelay, "delayed status confirmation after commands")
	fs.DurationVar(&cfg.EventHeartbeat, "event-heartbeat", cfg.EventHeartbeat, "maximum interval between unchanged observation/error events")
	fs.DurationVar(&cfg.ObservationFlushInterval, "observation-flush-interval", cfg.ObservationFlushInterval, "maximum interval between unchanged observation database writes")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	cfg.ListenAddr = normalizeListenAddr(cfg.ListenAddr)
	return cfg, nil
}

func defaultDBPath() string {
	if os.Getenv("POOLD_ENV") == "openwrt" || os.Getenv("POOLD_ENV") == "production" {
		return "/data/poold.db"
	}
	if runtime.GOOS == "linux" && os.Getuid() == 0 {
		return "/data/poold.db"
	}
	return "./var/poold.db"
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func normalizeListenAddr(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "*:") {
		return "0.0.0.0:" + strings.TrimPrefix(value, "*:")
	}
	return value
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

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
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
