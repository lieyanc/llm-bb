package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
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
	logger := log.New(os.Stdout, "[llm-bb] ", log.LstdFlags|log.Lmicroseconds)
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Printf("exit with error: %v", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger, args []string) (runErr error) {
	flags := flag.NewFlagSet("llm-bb", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var configPath string
	flags.StringVar(&configPath, "config", "", "path to config file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer stop()

	dbStore, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			if runErr == nil {
				runErr = fmt.Errorf("close store: %w", err)
				return
			}
			logger.Printf("close store error: %v", err)
		}
	}()

	if err := dbStore.Migrate(rootCtx); err != nil {
		return fmt.Errorf("migrate store: %w", err)
	}

	if cfg.SeedDemoData {
		if err := dbStore.SeedDemoData(rootCtx); err != nil {
			return fmt.Errorf("seed store: %w", err)
		}
	}

	llmClient := llm.NewClient(cfg.DefaultTimeout())
	roomEngine := engine.New(dbStore, llmClient, cfg, logger)
	hub := stream.NewHub()
	defer hub.Close()

	roomScheduler := scheduler.New(dbStore, roomEngine, hub, cfg, logger)
	server, err := web.NewServer(cfg, dbStore, roomScheduler, hub, logger)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Address, err)
	}

	httpServer := &http.Server{
		Addr:         cfg.Address,
		Handler:      server.Handler(),
		BaseContext:  func(net.Listener) context.Context { return rootCtx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go roomScheduler.Run(rootCtx)

	serveDone := make(chan error, 1)
	go func() {
		logger.Printf("listening on %s", listener.Addr().String())
		err := httpServer.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveDone <- fmt.Errorf("serve http: %w", err)
			return
		}
		serveDone <- nil
	}()

	select {
	case <-rootCtx.Done():
	case err := <-serveDone:
		if err != nil {
			runErr = err
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hub.Close()

	if err := shutdownHTTPServer(shutdownCtx, httpServer); err != nil {
		if runErr == nil {
			runErr = err
		} else {
			logger.Printf("http shutdown error: %v", err)
		}
	}

	if err := roomScheduler.Shutdown(shutdownCtx); err != nil {
		if runErr == nil {
			runErr = fmt.Errorf("scheduler shutdown: %w", err)
		} else {
			logger.Printf("scheduler shutdown error: %v", err)
		}
	}

	select {
	case err := <-serveDone:
		if err != nil && runErr == nil {
			runErr = err
		}
	default:
	}

	return runErr
}

func shutdownHTTPServer(ctx context.Context, server *http.Server) error {
	if err := server.Shutdown(ctx); err != nil {
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return fmt.Errorf("shutdown http server: %v (forced close: %w)", err, closeErr)
		}
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}
