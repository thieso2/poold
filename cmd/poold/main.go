package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pooly/services/poold/internal/config"
	"pooly/services/poold/internal/httpapi"
	"pooly/services/poold/internal/pool"
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
	"pooly/services/poold/internal/weather"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	poolClient := intex.New(cfg.PoolAddr)
	sched := scheduler.New(scheduler.Config{
		HeatingRateCPerHour: cfg.HeatingRateCPerHour,
		ReadinessBuffer:     cfg.ReadinessBuffer,
		Location:            cfg.Location,
		ReadyByReheatDelta:  cfg.ReadyByReheatDelta,
	})
	service := httpapi.NewService(st, poolClient, sched, httpapi.ServiceConfig{
		ObservationRetention:     cfg.ObservationRetention,
		EventRetention:           cfg.EventRetention,
		EventHeartbeat:           cfg.EventHeartbeat,
		ObservationFlushInterval: cfg.ObservationFlushInterval,
		CommandConfirmDelay:      cfg.CommandConfirmDelay,
		HeatingRateCPerHour:      cfg.HeatingRateCPerHour,
		CoolingRateCPerHour:      cfg.CoolingRateCPerHour,
		PollIdleInterval:         cfg.PollIdleInterval,
		PollStableInterval:       cfg.PollStableInterval,
		PollActiveInterval:       cfg.PollActiveInterval,
		WeatherProvider:          weather.New(),
	})

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(service, cfg.Token),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go poll(ctx, cfg, service)
	go pollWeather(ctx, cfg, service)

	go func() {
		log.Printf("poold listening on %s, pool=%s, db=%s", cfg.ListenAddr, cfg.PoolAddr, cfg.DatabasePath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}

func pollWeather(ctx context.Context, cfg config.Config, service *httpapi.Service) {
	interval := durationDefault(cfg.WeatherPollInterval, 5*time.Minute)
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if _, err := service.RefreshWeather(ctx); err != nil && !errors.Is(err, httpapi.ErrWeatherNotConfigured) {
				log.Printf("weather refresh: %v", err)
			}
			resetTimer(timer, interval)
		}
	}
}

func poll(ctx context.Context, cfg config.Config, service *httpapi.Service) {
	poller := newAdaptivePoller(cfg)
	timer := time.NewTimer(0)
	defer timer.Stop()
	nextDue := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case delay := <-service.RefreshRequests():
			if delay < 0 {
				delay = 0
			}
			due := time.Now().Add(delay)
			if !due.Before(nextDue) && delay != 0 {
				continue
			}
			resetTimer(timer, delay)
			nextDue = due
		case <-timer.C:
			status, err := service.RefreshStatus(ctx)
			if err != nil {
				log.Printf("status refresh: %v", err)
			}
			interval := poller.Next(status, err)
			if err == nil {
				now := time.Now()
				wake, ok, wakeErr := service.NextScheduleWake(ctx, now, status)
				if wakeErr != nil {
					log.Printf("next schedule wake: %v", wakeErr)
				} else if ok {
					untilWake := time.Until(wake)
					if untilWake < 0 {
						untilWake = 0
					}
					if untilWake < interval {
						interval = untilWake
					}
				}
			}
			nextDue = time.Now().Add(interval)
			resetTimer(timer, interval)
		}
	}
}

type adaptivePoller struct {
	startupInterval  time.Duration
	idleInterval     time.Duration
	stableInterval   time.Duration
	activeInterval   time.Duration
	errorMinInterval time.Duration
	errorMaxInterval time.Duration
	hadSuccess       bool
	errorCount       int
}

func newAdaptivePoller(cfg config.Config) *adaptivePoller {
	return &adaptivePoller{
		startupInterval:  durationDefault(cfg.PollStartupInterval, 10*time.Second),
		idleInterval:     durationDefault(cfg.PollIdleInterval, 10*time.Minute),
		stableInterval:   durationDefault(cfg.PollStableInterval, 5*time.Minute),
		activeInterval:   durationDefault(cfg.PollActiveInterval, time.Minute),
		errorMinInterval: durationDefault(cfg.PollErrorMinInterval, 30*time.Second),
		errorMaxInterval: durationDefault(cfg.PollErrorMaxInterval, 5*time.Minute),
	}
}

func (p *adaptivePoller) Next(status pool.Status, err error) time.Duration {
	if err != nil {
		if !p.hadSuccess {
			return p.startupInterval
		}
		p.errorCount++
		return cappedBackoff(p.errorMinInterval, p.errorMaxInterval, p.errorCount)
	}
	p.hadSuccess = true
	p.errorCount = 0
	if !status.Power {
		return p.idleInterval
	}
	if status.Filter || status.Heater || status.Jets || status.Bubbles || status.Sanitizer {
		return p.activeInterval
	}
	return p.stableInterval
}

func cappedBackoff(minInterval, maxInterval time.Duration, failures int) time.Duration {
	if failures <= 1 {
		return minInterval
	}
	interval := minInterval
	for i := 1; i < failures; i++ {
		interval *= 2
		if interval >= maxInterval {
			return maxInterval
		}
	}
	return interval
}

func durationDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(delay)
}
