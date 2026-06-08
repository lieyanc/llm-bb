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
	"strings"
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

func TestAdminBootstrapUsesEmptyArraysWithoutConfig(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:      "127.0.0.1:0",
		DatabasePath: filepath.Join(t.TempDir(), "app.db"),
		AdminUser:    "admin",
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	bootstrap := decodeBootstrap(t, rec.Body.String())
	if bootstrap.Page != "admin" {
		t.Fatalf("page = %q, want admin", bootstrap.Page)
	}

	for _, field := range []string{"rooms", "personas", "factions", "providers", "relationships"} {
		value, ok := bootstrap.Data[field].([]any)
		if !ok {
			t.Fatalf("data.%s = %#v, want empty array", field, bootstrap.Data[field])
		}
		if len(value) != 0 {
			t.Fatalf("len(data.%s) = %d, want 0", field, len(value))
		}
	}

	roomMembers, ok := bootstrap.Data["roomMembers"].(map[string]any)
	if !ok {
		t.Fatalf("data.roomMembers = %#v, want object", bootstrap.Data["roomMembers"])
	}
	if len(roomMembers) != 0 {
		t.Fatalf("len(data.roomMembers) = %d, want 0", len(roomMembers))
	}
}

func TestAdminListAPIsUseEmptyArraysWithoutConfig(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:      "127.0.0.1:0",
		DatabasePath: filepath.Join(t.TempDir(), "app.db"),
		AdminUser:    "admin",
	})
	defer cleanup()

	tests := []struct {
		path  string
		field string
	}{
		{path: "/api/admin/rooms", field: "rooms"},
		{path: "/api/admin/personas", field: "personas"},
		{path: "/api/admin/factions", field: "factions"},
		{path: "/api/admin/providers", field: "providers"},
		{path: "/api/admin/relationships", field: "relationships"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}

			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			value, ok := payload[tt.field].([]any)
			if !ok {
				t.Fatalf("%s = %#v, want empty array", tt.field, payload[tt.field])
			}
			if len(value) != 0 {
				t.Fatalf("len(%s) = %d, want 0", tt.field, len(value))
			}
		})
	}
}

func TestRoomBootstrapUsesEmptyArraysWithoutMembersOrMessages(t *testing.T) {
	server, cleanup := newTestServer(t, config.Config{
		Address:      "127.0.0.1:0",
		DatabasePath: filepath.Join(t.TempDir(), "app.db"),
		AdminUser:    "admin",
	})
	defer cleanup()

	room := &model.Room{Name: "empty room"}
	if err := server.store.CreateRoom(context.Background(), room); err != nil {
		t.Fatalf("create room: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/rooms/1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	bootstrap := decodeBootstrap(t, rec.Body.String())
	if bootstrap.Page != "room" {
		t.Fatalf("page = %q, want room", bootstrap.Page)
	}

	for _, field := range []string{"members", "messages"} {
		value, ok := bootstrap.Data[field].([]any)
		if !ok {
			t.Fatalf("data.%s = %#v, want empty array", field, bootstrap.Data[field])
		}
		if len(value) != 0 {
			t.Fatalf("len(data.%s) = %d, want 0", field, len(value))
		}
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

func decodeBootstrap(t *testing.T, body string) struct {
	Page  string         `json:"page"`
	Title string         `json:"title"`
	Data  map[string]any `json:"data"`
} {
	t.Helper()

	const prefix = "window.__LLM_BB_BOOTSTRAP__ = "
	start := strings.Index(body, prefix)
	if start == -1 {
		t.Fatalf("bootstrap script not found in body: %s", body)
	}
	start += len(prefix)

	end := strings.Index(body[start:], ";</script>")
	if end == -1 {
		t.Fatalf("bootstrap script end not found in body: %s", body)
	}

	var bootstrap struct {
		Page  string         `json:"page"`
		Title string         `json:"title"`
		Data  map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(body[start:start+end]), &bootstrap); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	return bootstrap
}
