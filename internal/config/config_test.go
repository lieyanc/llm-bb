package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEnvOverrides(t *testing.T) {
	t.Setenv("LLM_BB_ADDRESS", ":19090")
	t.Setenv("LLM_BB_DATABASE_PATH", "ignored-by-load.db")
	t.Setenv("LLM_BB_ADMIN_PASSWORD", "ignored")
	t.Setenv("LLM_BB_DEFAULT_TIMEOUT_MS", "999")
	t.Setenv("LLM_BB_SEED_DEMO", "false")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	configJSON := `{
		"address": ":18080",
		"database_path": "` + filepath.Join(dir, "from-config.db") + `",
		"admin_password": "from-config",
		"default_timeout_ms": 12345,
		"seed_demo_data": true
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Address != ":19090" {
		t.Fatalf("address = %q, want %q", cfg.Address, ":19090")
	}
	if cfg.DatabasePath != "ignored-by-load.db" {
		t.Fatalf("database_path = %q, want %q", cfg.DatabasePath, "ignored-by-load.db")
	}
	if cfg.AdminPassword != "ignored" {
		t.Fatalf("admin_password = %q, want %q", cfg.AdminPassword, "ignored")
	}
	if cfg.DefaultTimeoutMS != 999 {
		t.Fatalf("default_timeout_ms = %d, want %d", cfg.DefaultTimeoutMS, 999)
	}
	if cfg.SeedDemoData {
		t.Fatal("seed_demo_data = true, want false")
	}

	var persisted Config
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("decode persisted config: %v", err)
	}
	if persisted.Address != ":18080" {
		t.Fatalf("persisted address = %q, want file value", persisted.Address)
	}
	if persisted.HTTP.ReadTimeoutMS == 0 {
		t.Fatal("persisted config did not get new http defaults")
	}
	if _, ok := mapFromConfig(t, configPath)["http"]; !ok {
		t.Fatal("persisted config did not receive http section")
	}
}

func TestLoadCreatesDefaultConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Address != "127.0.0.1:8080" {
		t.Fatalf("address = %q, want default", cfg.Address)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var persisted Config
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if persisted.RoomDefaults.TickMinSeconds != 25 {
		t.Fatalf("room default tick min = %d, want 25", persisted.RoomDefaults.TickMinSeconds)
	}
	if persisted.Update.Owner == "" {
		t.Fatal("update owner was not written")
	}
}

func TestLoadBackfillsMissingFieldsAndPreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	configJSON := `{
		"address": "127.0.0.1:18080",
		"database_path": "` + filepath.Join(dir, "app.db") + `",
		"admin_password": "",
		"custom": {"keep": true},
		"room_defaults": {
			"heat": 77
		}
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RoomDefaults.Heat != 77 {
		t.Fatalf("room heat = %d, want 77", cfg.RoomDefaults.Heat)
	}
	if cfg.RoomDefaults.TickMinSeconds != 25 {
		t.Fatalf("tick min = %d, want default", cfg.RoomDefaults.TickMinSeconds)
	}

	persisted := mapFromConfig(t, configPath)
	custom, ok := persisted["custom"].(map[string]any)
	if !ok || custom["keep"] != true {
		t.Fatalf("custom field = %#v, want preserved", persisted["custom"])
	}
	roomDefaults, ok := persisted["room_defaults"].(map[string]any)
	if !ok {
		t.Fatalf("room_defaults = %#v, want object", persisted["room_defaults"])
	}
	if roomDefaults["heat"] != float64(77) {
		t.Fatalf("persisted heat = %#v, want 77", roomDefaults["heat"])
	}
	if _, ok := roomDefaults["tick_min_seconds"]; !ok {
		t.Fatal("missing tick_min_seconds backfill")
	}
}

func TestLoadRejectsOpenAdminWithoutPassword(t *testing.T) {
	t.Setenv("LLM_BB_ADDRESS", ":19090")
	t.Setenv("LLM_BB_DATABASE_PATH", filepath.Join(t.TempDir(), "app.db"))
	t.Setenv("LLM_BB_ADMIN_PASSWORD", "")

	_, err := Load(filepath.Join(t.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("Load succeeded, want error")
	}
	if !strings.Contains(err.Error(), "admin password is required") {
		t.Fatalf("error = %q, want admin password requirement", err.Error())
	}
}

func mapFromConfig(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	return out
}

func TestLoadAllowsLoopbackAdminWithoutPassword(t *testing.T) {
	t.Setenv("LLM_BB_ADDRESS", "127.0.0.1:19090")
	t.Setenv("LLM_BB_DATABASE_PATH", filepath.Join(t.TempDir(), "app.db"))
	t.Setenv("LLM_BB_ADMIN_PASSWORD", "")

	cfg, err := Load(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.AdminPassword != "" {
		t.Fatalf("admin_password = %q, want empty", cfg.AdminPassword)
	}
}
