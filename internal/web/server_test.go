package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/model"
	"llm-bb/internal/scheduler"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
)

func TestAdminRequiresBasicAuthWhenPasswordConfigured(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:       "127.0.0.1:0",
		DatabasePath:  filepath.Join(t.TempDir(), "app.db"),
		AdminUser:     "admin",
		AdminPassword: "secret",
	})
	defer cleanup()

	handler := server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/rooms", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/rooms", nil)
	req.SetBasicAuth("admin", "wrong")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/rooms", nil)
	req.SetBasicAuth("admin", "secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRoomInputRejectsMissingRoom(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:      "127.0.0.1:0",
		DatabasePath: filepath.Join(t.TempDir(), "app.db"),
		AdminUser:    "admin",
	})
	defer cleanup()

	req := jsonRequest(http.MethodPost, "/api/rooms/999/input", map[string]any{
		"content": "hello",
	})
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestRoomInputRateLimit(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:      "127.0.0.1:0",
		DatabasePath: filepath.Join(t.TempDir(), "app.db"),
		AdminUser:    "admin",
	})
	defer cleanup()

	room := &model.Room{Name: "test room"}
	if err := server.store.CreateRoom(context.Background(), room); err != nil {
		t.Fatalf("create room: %v", err)
	}
	server.inputLimiter = newRateLimiter(rateLimiterConfig{
		Limit:   2,
		Window:  time.Hour,
		MaxKeys: 16,
	})

	handler := server.Handler()
	for i := 0; i < 2; i++ {
		req := jsonRequest(http.MethodPost, "/api/rooms/1/input", map[string]any{
			"content": "hello",
		})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d; body=%s", i+1, rec.Code, http.StatusCreated, rec.Body.String())
		}
	}

	req := jsonRequest(http.MethodPost, "/api/rooms/1/input", map[string]any{
		"content": "hello",
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
}

func newTestServer(t *testing.T, cfg config.Config) (*Server, func()) {
	t.Helper()

	dbStore, err := store.Open(cfg.DatabasePath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := dbStore.Migrate(context.Background()); err != nil {
		_ = dbStore.Close()
		t.Fatalf("migrate store: %v", err)
	}

	hub := stream.NewHub()
	roomScheduler := scheduler.New(
		dbStore,
		nil,
		hub,
		cfg,
		log.New(io.Discard, "", 0),
	)

	server, err := NewServer(
		cfg,
		dbStore,
		roomScheduler,
		hub,
		log.New(io.Discard, "", 0),
		nil,
	)
	if err != nil {
		_ = dbStore.Close()
		t.Fatalf("new server: %v", err)
	}

	return server, func() {
		hub.Close()
		_ = dbStore.Close()
	}
}

func jsonRequest(method, target string, payload any) *http.Request {
	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(payload)
	req := httptest.NewRequest(method, target, &body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:1234"
	return req
}
