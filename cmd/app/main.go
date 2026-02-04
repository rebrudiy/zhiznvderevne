package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"zhiznvderevne/internal/cache"
	"zhiznvderevne/internal/config"
	"zhiznvderevne/internal/db"
	"zhiznvderevne/internal/httpapp"
	"zhiznvderevne/internal/llm"
	"zhiznvderevne/internal/metrics"
	"zhiznvderevne/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	if err := db.Migrate(ctx, pool, "./migrations"); err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	var cacheClient cache.Cache
	if cfg.Redis.Addr != "" {
		if c, err := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.DB); err == nil {
			cacheClient = c
		} else {
			log.Printf("redis disabled: %v", err)
		}
	}

	server, err := httpapp.New(cfg, pool, reg, cacheClient)
	if err != nil {
		log.Fatalf("http error: %v", err)
	}

	var workerClient *llm.Client
	if cfg.Worker.Role != "api" {
		apiKey := os.Getenv(cfg.OpenAI.APIKeyEnv)
		if apiKey == "" {
			log.Fatalf("missing OpenAI API key in env %s", cfg.OpenAI.APIKeyEnv)
		}
		workerClient = llm.New(apiKey, cfg.OpenAI.Model, cfg.OpenAI.Temperature, cfg.OpenAI.MaxTokens)
	}

	if cfg.Worker.Role != "api" {
		w, err := worker.New(cfg, pool, workerClient)
		if err != nil {
			log.Fatalf("worker error: %v", err)
		}
		go w.Run(ctx)
	}

	httpServer := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           server,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("http listening on %s", cfg.HTTP.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	pool.Close()
}
