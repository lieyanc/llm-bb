package config

import (
	"os"
	"path/filepath"
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
}
