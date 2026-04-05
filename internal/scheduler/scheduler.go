package scheduler

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/engine"
	"llm-bb/internal/model"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
)

type Scheduler struct {
	store  *store.Store
	engine *engine.Engine
	hub    *stream.Hub
	cfg    config.Config
	logger *log.Logger

	mu       sync.Mutex
	nextRun  map[int64]time.Time
	running  map[int64]bool
	rng      *rand.Rand
	wg       sync.WaitGroup
	closed   bool
	loopDone chan struct{}
	loopRun  bool
}

func New(store *store.Store, engine *engine.Engine, hub *stream.Hub, cfg config.Config, logger *log.Logger) *Scheduler {
	loopDone := make(chan struct{})
	close(loopDone)

	return &Scheduler{
		store:    store,
		engine:   engine,
		hub:      hub,
		cfg:      cfg,
		logger:   logger,
		nextRun:  make(map[int64]time.Time),
		running:  make(map[int64]bool),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		loopDone: loopDone,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()
	if s.closed || s.loopRun {
		s.mu.Unlock()
		return
	}
	s.loopRun = true
	s.loopDone = make(chan struct{})
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.loopRun = false
		close(s.loopDone)
		s.mu.Unlock()
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := s.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Printf("scheduler tick error: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) TriggerRoom(ctx context.Context, roomID int64) (*model.Message, error) {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil, context.Canceled
	}

	result, err := s.engine.GenerateNextMessage(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if result.Message != nil {
		s.hub.Publish(*result.Message)
	}
	s.Nudge(roomID, 5*time.Second)
	return result.Message, nil
}

func (s *Scheduler) Nudge(roomID int64, delay time.Duration) {
	if delay < 0 {
		delay = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	next := time.Now().Add(delay)
	current, ok := s.nextRun[roomID]
	if !ok || next.Before(current) {
		s.nextRun[roomID] = next
	}
}

func (s *Scheduler) tick(ctx context.Context) error {
	rooms, err := s.store.ListRunnableRooms(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, room := range rooms {
		if !s.shouldRun(room.ID, now, room) {
			continue
		}
		s.startRun(ctx, room)
	}

	return nil
}

func (s *Scheduler) shouldRun(roomID int64, now time.Time, room model.Room) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}
	if s.running[roomID] {
		return false
	}

	next, ok := s.nextRun[roomID]
	if !ok {
		s.nextRun[roomID] = now.Add(s.randomDelay(room))
		return false
	}

	return !next.After(now)
}

func (s *Scheduler) startRun(ctx context.Context, room model.Room) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.running[room.ID] = true
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, room.ID)
			if !s.closed {
				s.nextRun[room.ID] = time.Now().Add(s.randomDelay(room))
			}
			s.mu.Unlock()
			s.wg.Done()
		}()

		result, err := s.engine.GenerateNextMessage(ctx, room.ID)
		if err != nil {
			s.logger.Printf("room %d tick failed: %v", room.ID, err)
			return
		}
		if result.Message != nil {
			s.hub.Publish(*result.Message)
		}
	}()
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	loopDone := s.loopDone
	s.mu.Unlock()

	select {
	case <-loopDone:
	case <-ctx.Done():
		return ctx.Err()
	}

	waitDone := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) randomDelay(room model.Room) time.Duration {
	minSeconds := room.TickMinSeconds
	maxSeconds := room.TickMaxSeconds
	if minSeconds <= 0 {
		minSeconds = 20
	}
	if maxSeconds < minSeconds {
		maxSeconds = minSeconds
	}
	if minSeconds == maxSeconds {
		return time.Duration(minSeconds) * time.Second
	}
	span := maxSeconds - minSeconds + 1
	return time.Duration(minSeconds+s.rng.Intn(span)) * time.Second
}
