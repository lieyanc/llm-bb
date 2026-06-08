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
	"path/filepath"
	"sync/atomic"
	"syscall"

	"llm-bb/internal/config"
	"llm-bb/internal/engine"
	"llm-bb/internal/llm"
	"llm-bb/internal/scheduler"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
	"llm-bb/internal/update"
	"llm-bb/internal/version"
	"llm-bb/internal/web"
)

func main() {
	logger := log.New(os.Stdout, "[llm-bb] ", log.LstdFlags|log.Lmicroseconds)

	update.CleanupOldBinary()

	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			printVersion()
			return
		case "update":
			if err := runUpdateCmd(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	if err := run(logger, os.Args[1:]); err != nil {
		logger.Printf("exit with error: %v", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("llm-bb - LLM chat backdrop")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  llm-bb [-config <path>]        run the server")
	fmt.Println("  llm-bb version                 print version info")
	fmt.Println("  llm-bb update [options]        update the binary from GitHub releases")
	fmt.Println()
	fmt.Println("Update options:")
	fmt.Println("  --config <path>                path to config file")
	fmt.Println("  --channel <stable|dev>         release channel (default: build channel)")
	fmt.Println("  --check                        only check, don't apply")
}

func printVersion() {
	info := version.Get()
	fmt.Printf("llm-bb %s (%s)\n", info.Version, info.Channel)
	fmt.Printf("  commit:     %s\n", info.Commit)
	fmt.Printf("  build date: %s\n", info.BuildDate)
	fmt.Printf("  platform:   %s/%s\n", info.GOOS, info.GOARCH)
}

func runUpdateCmd(args []string) error {
	flags := flag.NewFlagSet("update", flag.ContinueOnError)
	var configPath string
	flags.StringVar(&configPath, "config", "", "path to config file")
	channel := flags.String("channel", "", "release channel: stable or dev")
	checkOnly := flags.Bool("check", false, "only check, don't apply")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if *channel == "" {
		*channel = defaultChannel(cfg)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := updateClientFromConfig(cfg)
	result, err := client.Check(ctx, *channel, version.Commit, version.Version)
	if err != nil {
		return err
	}

	fmt.Printf("Current: %s (%s, %s)\n", version.Version, version.ShortCommit(), version.Channel)
	fmt.Printf("Latest:  %s\n", result.LatestVersion)
	fmt.Printf("Tag:     %s\n", result.LatestTag)
	fmt.Printf("Asset:   %s (%s)\n", result.AssetName, formatBytes(result.AssetSize))

	if !result.UpdateAvailable {
		fmt.Println("Already up to date.")
		return nil
	}

	if *checkOnly {
		fmt.Println("Update available. Run without --check to apply.")
		return nil
	}

	fmt.Println("Downloading and verifying...")
	if err := client.Apply(ctx, *channel); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}
	fmt.Println("Update applied. Restart llm-bb to use the new version.")
	return nil
}

func defaultChannel(cfg config.Config) string {
	if version.Channel == update.ChannelDev || version.Channel == update.ChannelStable {
		return version.Channel
	}
	if cfg.Update.DefaultChannel == update.ChannelDev || cfg.Update.DefaultChannel == update.ChannelStable {
		return cfg.Update.DefaultChannel
	}
	return update.ChannelStable
}

func updateClientFromConfig(cfg config.Config) *update.Client {
	client := update.NewClient()
	client.Owner = cfg.Update.Owner
	client.Repo = cfg.Update.Repo
	client.APITimeout = cfg.UpdateAPITimeout()
	client.DownloadTimeout = cfg.UpdateDownloadTimeout()
	client.MaxDownloadBytes = cfg.Update.MaxDownloadBytes
	return client
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
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

	logger.Printf("llm-bb %s (%s, commit %s)", version.Version, version.Channel, version.ShortCommit())

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer stop()

	dbStore, err := store.OpenWithConfig(cfg.DatabasePath, cfg)
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

	llmClient := llm.NewClientWithConfig(cfg.DefaultTimeout(), cfg.LLM)
	roomEngine := engine.New(dbStore, llmClient, cfg, logger)
	hub := stream.NewHub()
	defer hub.Close()

	var shouldReExec atomic.Bool
	onRestart := func() {
		shouldReExec.Store(true)
		stop()
	}

	roomScheduler := scheduler.New(dbStore, roomEngine, hub, cfg, logger)
	server, err := web.NewServer(cfg, dbStore, roomScheduler, hub, logger, onRestart)
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
		ReadTimeout:  cfg.HTTPReadTimeout(),
		WriteTimeout: cfg.HTTPWriteTimeout(),
		IdleTimeout:  cfg.HTTPIdleTimeout(),
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
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

	if shouldReExec.Load() && runErr == nil {
		if err := reExecSelf(logger); err != nil {
			runErr = fmt.Errorf("re-exec: %w", err)
		}
	}

	return runErr
}

func reExecSelf(logger *log.Logger) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	logger.Printf("re-executing %s", exe)
	return update.ReExec(exe, os.Args, os.Environ())
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
