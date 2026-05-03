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
	"pooly/services/poold/internal/protocol/intex"
	"pooly/services/poold/internal/scheduler"
	"pooly/services/poold/internal/store"
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
	})
	service := httpapi.NewService(st, poolClient, sched, httpapi.ServiceConfig{
		ObservationRetention: cfg.ObservationRetention,
		EventRetention:       cfg.EventRetention,
	})

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(service, cfg.Token),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go poll(ctx, cfg.PollInterval, service)

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

func poll(ctx context.Context, interval time.Duration, service *httpapi.Service) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if _, err := service.RefreshStatus(ctx); err != nil {
			log.Printf("status refresh: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
