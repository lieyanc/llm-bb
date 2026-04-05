package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address              string `json:"address"`
	DatabasePath         string `json:"database_path"`
	AdminUser            string `json:"admin_user"`
	AdminPassword        string `json:"admin_password"`
	DefaultLanguage      string `json:"default_language"`
	DefaultTimeoutMS     int    `json:"default_timeout_ms"`
	DefaultSummaryWindow int    `json:"default_summary_window"`
	SeedDemoData         bool   `json:"seed_demo_data"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		Address:              ":8080",
		DatabasePath:         "data/llm-bb.db",
		AdminUser:            "admin",
		DefaultLanguage:      "zh-CN",
		DefaultTimeoutMS:     20000,
		DefaultSummaryWindow: 24,
		SeedDemoData:         true,
	}

	if path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	applyEnvOverrides(&cfg)

	if strings.TrimSpace(cfg.Address) == "" {
		return Config{}, errors.New("address cannot be empty")
	}

	if strings.TrimSpace(cfg.DatabasePath) == "" {
		return Config{}, errors.New("database path cannot be empty")
	}

	if cfg.DefaultTimeoutMS <= 0 {
		cfg.DefaultTimeoutMS = 20000
	}

	if cfg.DefaultSummaryWindow <= 0 {
		cfg.DefaultSummaryWindow = 24
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		return Config{}, fmt.Errorf("create database directory: %w", err)
	}

	return cfg, nil
}

func (c Config) DefaultTimeout() time.Duration {
	return time.Duration(c.DefaultTimeoutMS) * time.Millisecond
}

type partialConfig struct {
	Address              *string `json:"address"`
	DatabasePath         *string `json:"database_path"`
	AdminUser            *string `json:"admin_user"`
	AdminPassword        *string `json:"admin_password"`
	DefaultLanguage      *string `json:"default_language"`
	DefaultTimeoutMS     *int    `json:"default_timeout_ms"`
	DefaultSummaryWindow *int    `json:"default_summary_window"`
	SeedDemoData         *bool   `json:"seed_demo_data"`
}

func loadFile(path string, cfg *Config) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var fileCfg partialConfig
	if err := json.Unmarshal(content, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fileCfg.Address != nil {
		cfg.Address = *fileCfg.Address
	}
	if fileCfg.DatabasePath != nil {
		cfg.DatabasePath = *fileCfg.DatabasePath
	}
	if fileCfg.AdminUser != nil {
		cfg.AdminUser = *fileCfg.AdminUser
	}
	if fileCfg.AdminPassword != nil {
		cfg.AdminPassword = *fileCfg.AdminPassword
	}
	if fileCfg.DefaultLanguage != nil {
		cfg.DefaultLanguage = *fileCfg.DefaultLanguage
	}
	if fileCfg.DefaultTimeoutMS != nil {
		cfg.DefaultTimeoutMS = *fileCfg.DefaultTimeoutMS
	}
	if fileCfg.DefaultSummaryWindow != nil {
		cfg.DefaultSummaryWindow = *fileCfg.DefaultSummaryWindow
	}
	if fileCfg.SeedDemoData != nil {
		cfg.SeedDemoData = *fileCfg.SeedDemoData
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	if value, ok := envLookup("LLM_BB_ADDRESS"); ok {
		cfg.Address = value
	}
	if value, ok := envLookup("LLM_BB_DATABASE_PATH"); ok {
		cfg.DatabasePath = value
	}
	if value, ok := envLookup("LLM_BB_ADMIN_USER"); ok {
		cfg.AdminUser = value
	}
	if value, ok := envLookup("LLM_BB_ADMIN_PASSWORD"); ok {
		cfg.AdminPassword = value
	}
	if value, ok := envLookup("LLM_BB_DEFAULT_LANGUAGE"); ok {
		cfg.DefaultLanguage = value
	}
	if value, ok := envLookupInt("LLM_BB_DEFAULT_TIMEOUT_MS"); ok {
		cfg.DefaultTimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_DEFAULT_SUMMARY_WINDOW"); ok {
		cfg.DefaultSummaryWindow = value
	}
	if value, ok := envLookupBool("LLM_BB_SEED_DEMO"); ok {
		cfg.SeedDemoData = value
	}
}

func envLookup(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func envLookupInt(key string) (int, bool) {
	value, ok := envLookup(key)
	if !ok || value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func envLookupBool(key string) (bool, bool) {
	value, ok := envLookup(key)
	if !ok {
		return false, false
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
