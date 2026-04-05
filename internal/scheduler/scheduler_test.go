package scheduler

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"testing"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
)

func TestShutdownWaitsForLoopExit(t *testing.T) {
	dbStore, err := store.Open(filepath.Join(t.TempDir(), "scheduler.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer dbStore.Close()

	if err := dbStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	scheduler := New(
		dbStore,
		nil,
		stream.NewHub(),
		config.Config{},
		log.New(io.Discard, "", 0),
	)

	runCtx, cancelRun := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		scheduler.Run(runCtx)
		close(runDone)
	}()

	waitForLoopStart(t, scheduler)
	cancelRun()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()

	if err := scheduler.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown scheduler: %v", err)
	}

	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler loop did not exit")
	}
}

func waitForLoopStart(t *testing.T, scheduler *Scheduler) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		scheduler.mu.Lock()
		loopRun := scheduler.loopRun
		scheduler.mu.Unlock()
		if loopRun {
			return
		}

		select {
		case <-deadline:
			t.Fatal("scheduler loop did not start")
		case <-ticker.C:
		}
	}
}
