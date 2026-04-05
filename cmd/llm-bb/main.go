package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/engine"
	"llm-bb/internal/llm"
	"llm-bb/internal/scheduler"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
	"llm-bb/internal/web"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := log.New(os.Stdout, "[llm-bb] ", log.LstdFlags|log.Lmicroseconds)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbStore, err := store.Open(cfg.DatabasePath)
	if err != nil {
		logger.Fatalf("open store: %v", err)
	}
	defer dbStore.Close()

	if err := dbStore.Migrate(rootCtx); err != nil {
		logger.Fatalf("migrate store: %v", err)
	}

	if cfg.SeedDemoData {
		if err := dbStore.SeedDemoData(rootCtx); err != nil {
			logger.Fatalf("seed store: %v", err)
		}
	}

	llmClient := llm.NewClient(cfg.DefaultTimeout())
	roomEngine := engine.New(dbStore, llmClient, cfg, logger)
	hub := stream.NewHub()
	roomScheduler := scheduler.New(dbStore, roomEngine, hub, cfg, logger)
	server, err := web.NewServer(cfg, dbStore, roomScheduler, hub, logger)
	if err != nil {
		logger.Fatalf("create server: %v", err)
	}

	go roomScheduler.Run(rootCtx)

	httpServer := &http.Server{
		Addr:         cfg.Address,
		Handler:      server.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Printf("listening on %s", cfg.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("http server stopped with error: %v", err)
			stop()
		}
	}()

	<-rootCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("http shutdown error: %v", err)
	}
}
