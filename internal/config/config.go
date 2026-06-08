package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"llm-bb/internal/update"
)

const DefaultConfigPath = "config.json"

type Config struct {
	Address              string               `json:"address"`
	DatabasePath         string               `json:"database_path"`
	AdminUser            string               `json:"admin_user"`
	AdminPassword        string               `json:"admin_password"`
	DefaultLanguage      string               `json:"default_language"`
	DefaultTimeoutMS     int                  `json:"default_timeout_ms"`
	DefaultSummaryWindow int                  `json:"default_summary_window"`
	SeedDemoData         bool                 `json:"seed_demo_data"`
	LogLevel             string               `json:"log_level"`
	HTTP                 HTTPConfig           `json:"http"`
	LLM                  LLMConfig            `json:"llm"`
	Scheduler            SchedulerConfig      `json:"scheduler"`
	RoomDefaults         RoomDefaults         `json:"room_defaults"`
	PersonaDefaults      PersonaDefaults      `json:"persona_defaults"`
	ProviderDefaults     ProviderDefaults     `json:"provider_defaults"`
	RelationshipDefaults RelationshipDefaults `json:"relationship_defaults"`
	PublicInput          PublicInputConfig    `json:"public_input"`
	Update               UpdateConfig         `json:"update"`
	SQLite               SQLiteConfig         `json:"sqlite"`
}

type HTTPConfig struct {
	ReadTimeoutMS  int `json:"read_timeout_ms"`
	WriteTimeoutMS int `json:"write_timeout_ms"`
	IdleTimeoutMS  int `json:"idle_timeout_ms"`
	ShutdownMS     int `json:"shutdown_ms"`
}

type LLMConfig struct {
	DefaultTemperature float64 `json:"default_temperature"`
	DefaultMaxTokens   int     `json:"default_max_tokens"`
	RequestRetries     int     `json:"request_retries"`
	RetryBackoffMS     int     `json:"retry_backoff_ms"`
	MaxResponseBytes   int64   `json:"max_response_bytes"`
}

type SchedulerConfig struct {
	PollIntervalMS    int `json:"poll_interval_ms"`
	ManualNudgeMS     int `json:"manual_nudge_ms"`
	UserInputNudgeMS  int `json:"user_input_nudge_ms"`
	MinDenseGapMS     int `json:"min_dense_gap_ms"`
	DefaultCooldownMS int `json:"default_cooldown_ms"`
	DefaultTickMinSec int `json:"default_tick_min_seconds"`
}

type RoomDefaults struct {
	Status               string `json:"status"`
	Heat                 int    `json:"heat"`
	ConflictLevel        int    `json:"conflict_level"`
	TickMinSeconds       int    `json:"tick_min_seconds"`
	TickMaxSeconds       int    `json:"tick_max_seconds"`
	DailyTokenBudget     int    `json:"daily_token_budget"`
	SummaryTriggerCount  int    `json:"summary_trigger_count"`
	MessageRetention     int    `json:"message_retention_count"`
	RoomPageMessageLimit int    `json:"room_page_message_limit"`
	RoomAPIMessagesLimit int    `json:"room_api_messages_limit"`
	RecentMessageLimit   int    `json:"recent_message_limit"`
	PromptRecentMessages int    `json:"prompt_recent_messages"`
	SummaryExtraMessages int    `json:"summary_extra_messages"`
}

type PersonaDefaults struct {
	Aggression      int     `json:"aggression"`
	ActivityLevel   int     `json:"activity_level"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	CooldownSeconds int     `json:"cooldown_seconds"`
	Enabled         bool    `json:"enabled"`
}

type ProviderDefaults struct {
	TimeoutMS int  `json:"timeout_ms"`
	Enabled   bool `json:"enabled"`
}

type RelationshipDefaults struct {
	Affinity    int `json:"affinity"`
	Hostility   int `json:"hostility"`
	Respect     int `json:"respect"`
	FocusWeight int `json:"focus_weight"`
}

type PublicInputConfig struct {
	MaxRunes  int `json:"max_runes"`
	RateLimit int `json:"rate_limit"`
	WindowMS  int `json:"window_ms"`
	MaxKeys   int `json:"max_keys"`
}

type UpdateConfig struct {
	Owner             string `json:"owner"`
	Repo              string `json:"repo"`
	DefaultChannel    string `json:"default_channel"`
	APITimeoutMS      int    `json:"api_timeout_ms"`
	DownloadTimeoutMS int    `json:"download_timeout_ms"`
	MaxDownloadBytes  int64  `json:"max_download_bytes"`
}

type SQLiteConfig struct {
	JournalMode   string `json:"journal_mode"`
	ForeignKeys   bool   `json:"foreign_keys"`
	BusyTimeoutMS int    `json:"busy_timeout_ms"`
	MaxOpenConns  int    `json:"max_open_conns"`
	MaxIdleConns  int    `json:"max_idle_conns"`
}

func Default() Config {
	return Config{
		Address:              "127.0.0.1:8080",
		DatabasePath:         "data/llm-bb.db",
		AdminUser:            "admin",
		DefaultLanguage:      "zh-CN",
		DefaultTimeoutMS:     20000,
		DefaultSummaryWindow: 24,
		SeedDemoData:         true,
		LogLevel:             "info",
		HTTP: HTTPConfig{
			ReadTimeoutMS:  10000,
			WriteTimeoutMS: 30000,
			IdleTimeoutMS:  60000,
			ShutdownMS:     10000,
		},
		LLM: LLMConfig{
			DefaultTemperature: 0.9,
			DefaultMaxTokens:   256,
			RequestRetries:     3,
			RetryBackoffMS:     350,
			MaxResponseBytes:   1 << 20,
		},
		Scheduler: SchedulerConfig{
			PollIntervalMS:    2000,
			ManualNudgeMS:     5000,
			UserInputNudgeMS:  1500,
			MinDenseGapMS:     6000,
			DefaultCooldownMS: 20000,
			DefaultTickMinSec: 20,
		},
		RoomDefaults: RoomDefaults{
			Status:               "running",
			Heat:                 60,
			ConflictLevel:        55,
			TickMinSeconds:       25,
			TickMaxSeconds:       55,
			DailyTokenBudget:     40000,
			SummaryTriggerCount:  24,
			MessageRetention:     120,
			RoomPageMessageLimit: 80,
			RoomAPIMessagesLimit: 100,
			RecentMessageLimit:   18,
			PromptRecentMessages: 10,
			SummaryExtraMessages: 10,
		},
		PersonaDefaults: PersonaDefaults{
			Aggression:      50,
			ActivityLevel:   50,
			Temperature:     0.9,
			MaxTokens:       220,
			CooldownSeconds: 120,
			Enabled:         true,
		},
		ProviderDefaults: ProviderDefaults{
			TimeoutMS: 20000,
			Enabled:   true,
		},
		RelationshipDefaults: RelationshipDefaults{},
		PublicInput: PublicInputConfig{
			MaxRunes:  280,
			RateLimit: 8,
			WindowMS:  10000,
			MaxKeys:   1024,
		},
		Update: UpdateConfig{
			Owner:             update.DefaultOwner,
			Repo:              update.DefaultRepo,
			DefaultChannel:    update.ChannelStable,
			APITimeoutMS:      20000,
			DownloadTimeoutMS: 300000,
			MaxDownloadBytes:  200 << 20,
		},
		SQLite: SQLiteConfig{
			JournalMode:   "WAL",
			ForeignKeys:   true,
			BusyTimeoutMS: 5000,
			MaxOpenConns:  1,
			MaxIdleConns:  1,
		},
	}
}

func WithDefaults(cfg Config) Config {
	defaults := Default()
	mergeConfig(&cfg, defaults)
	normalize(&cfg)
	return cfg
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
	}

	cfg := Default()
	content, exists, err := readFile(path, &cfg)
	if err != nil {
		return Config{}, err
	}

	normalize(&cfg)

	if !exists {
		if err := writeFile(path, cfg); err != nil {
			return Config{}, err
		}
	} else {
		content, changed, err := mergeMissingConfig(content, cfg)
		if err != nil {
			return Config{}, err
		}
		if changed {
			if err := writeRawFile(path, content); err != nil {
				return Config{}, err
			}
		}
	}

	applyEnvOverrides(&cfg)
	normalize(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		return Config{}, fmt.Errorf("create database directory: %w", err)
	}

	return cfg, nil
}

func mergeConfig(cfg *Config, defaults Config) {
	if strings.TrimSpace(cfg.Address) == "" {
		cfg.Address = defaults.Address
	}
	if strings.TrimSpace(cfg.DatabasePath) == "" {
		cfg.DatabasePath = defaults.DatabasePath
	}
	if strings.TrimSpace(cfg.AdminUser) == "" {
		cfg.AdminUser = defaults.AdminUser
	}

	if cfg.HTTP == (HTTPConfig{}) {
		cfg.HTTP = defaults.HTTP
	}
	if cfg.LLM == (LLMConfig{}) {
		cfg.LLM = defaults.LLM
	}
	if cfg.Scheduler == (SchedulerConfig{}) {
		cfg.Scheduler = defaults.Scheduler
	}
	if cfg.RoomDefaults == (RoomDefaults{}) {
		cfg.RoomDefaults = defaults.RoomDefaults
	}
	if cfg.PersonaDefaults == (PersonaDefaults{}) {
		cfg.PersonaDefaults = defaults.PersonaDefaults
	}
	if cfg.ProviderDefaults == (ProviderDefaults{}) {
		cfg.ProviderDefaults = defaults.ProviderDefaults
	}
	if cfg.PublicInput == (PublicInputConfig{}) {
		cfg.PublicInput = defaults.PublicInput
	}
	if cfg.Update == (UpdateConfig{}) {
		cfg.Update = defaults.Update
	}
	if cfg.SQLite == (SQLiteConfig{}) {
		cfg.SQLite = defaults.SQLite
	}
}

func (c Config) DefaultTimeout() time.Duration {
	return time.Duration(c.DefaultTimeoutMS) * time.Millisecond
}

func (c Config) HTTPReadTimeout() time.Duration {
	return time.Duration(c.HTTP.ReadTimeoutMS) * time.Millisecond
}

func (c Config) HTTPWriteTimeout() time.Duration {
	return time.Duration(c.HTTP.WriteTimeoutMS) * time.Millisecond
}

func (c Config) HTTPIdleTimeout() time.Duration {
	return time.Duration(c.HTTP.IdleTimeoutMS) * time.Millisecond
}

func (c Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.HTTP.ShutdownMS) * time.Millisecond
}

func (c Config) SchedulerPollInterval() time.Duration {
	return time.Duration(c.Scheduler.PollIntervalMS) * time.Millisecond
}

func (c Config) ManualNudgeDelay() time.Duration {
	return time.Duration(c.Scheduler.ManualNudgeMS) * time.Millisecond
}

func (c Config) UserInputNudgeDelay() time.Duration {
	return time.Duration(c.Scheduler.UserInputNudgeMS) * time.Millisecond
}

func (c Config) MinDenseGap() time.Duration {
	return time.Duration(c.Scheduler.MinDenseGapMS) * time.Millisecond
}

func (c Config) DefaultCooldown() time.Duration {
	return time.Duration(c.Scheduler.DefaultCooldownMS) * time.Millisecond
}

func (c Config) PublicInputWindow() time.Duration {
	return time.Duration(c.PublicInput.WindowMS) * time.Millisecond
}

func (c Config) UpdateAPITimeout() time.Duration {
	return time.Duration(c.Update.APITimeoutMS) * time.Millisecond
}

func (c Config) UpdateDownloadTimeout() time.Duration {
	return time.Duration(c.Update.DownloadTimeoutMS) * time.Millisecond
}

type partialConfig struct {
	Address              *string                      `json:"address"`
	DatabasePath         *string                      `json:"database_path"`
	AdminUser            *string                      `json:"admin_user"`
	AdminPassword        *string                      `json:"admin_password"`
	DefaultLanguage      *string                      `json:"default_language"`
	DefaultTimeoutMS     *int                         `json:"default_timeout_ms"`
	DefaultSummaryWindow *int                         `json:"default_summary_window"`
	SeedDemoData         *bool                        `json:"seed_demo_data"`
	LogLevel             *string                      `json:"log_level"`
	HTTP                 *partialHTTPConfig           `json:"http"`
	LLM                  *partialLLMConfig            `json:"llm"`
	Scheduler            *partialSchedulerConfig      `json:"scheduler"`
	RoomDefaults         *partialRoomDefaults         `json:"room_defaults"`
	PersonaDefaults      *partialPersonaDefaults      `json:"persona_defaults"`
	ProviderDefaults     *partialProviderDefaults     `json:"provider_defaults"`
	RelationshipDefaults *partialRelationshipDefaults `json:"relationship_defaults"`
	PublicInput          *partialPublicInputConfig    `json:"public_input"`
	Update               *partialUpdateConfig         `json:"update"`
	SQLite               *partialSQLiteConfig         `json:"sqlite"`
}

type partialHTTPConfig struct {
	ReadTimeoutMS  *int `json:"read_timeout_ms"`
	WriteTimeoutMS *int `json:"write_timeout_ms"`
	IdleTimeoutMS  *int `json:"idle_timeout_ms"`
	ShutdownMS     *int `json:"shutdown_ms"`
}

type partialLLMConfig struct {
	DefaultTemperature *float64 `json:"default_temperature"`
	DefaultMaxTokens   *int     `json:"default_max_tokens"`
	RequestRetries     *int     `json:"request_retries"`
	RetryBackoffMS     *int     `json:"retry_backoff_ms"`
	MaxResponseBytes   *int64   `json:"max_response_bytes"`
}

type partialSchedulerConfig struct {
	PollIntervalMS    *int `json:"poll_interval_ms"`
	ManualNudgeMS     *int `json:"manual_nudge_ms"`
	UserInputNudgeMS  *int `json:"user_input_nudge_ms"`
	MinDenseGapMS     *int `json:"min_dense_gap_ms"`
	DefaultCooldownMS *int `json:"default_cooldown_ms"`
	DefaultTickMinSec *int `json:"default_tick_min_seconds"`
}

type partialRoomDefaults struct {
	Status               *string `json:"status"`
	Heat                 *int    `json:"heat"`
	ConflictLevel        *int    `json:"conflict_level"`
	TickMinSeconds       *int    `json:"tick_min_seconds"`
	TickMaxSeconds       *int    `json:"tick_max_seconds"`
	DailyTokenBudget     *int    `json:"daily_token_budget"`
	SummaryTriggerCount  *int    `json:"summary_trigger_count"`
	MessageRetention     *int    `json:"message_retention_count"`
	RoomPageMessageLimit *int    `json:"room_page_message_limit"`
	RoomAPIMessagesLimit *int    `json:"room_api_messages_limit"`
	RecentMessageLimit   *int    `json:"recent_message_limit"`
	PromptRecentMessages *int    `json:"prompt_recent_messages"`
	SummaryExtraMessages *int    `json:"summary_extra_messages"`
}

type partialPersonaDefaults struct {
	Aggression      *int     `json:"aggression"`
	ActivityLevel   *int     `json:"activity_level"`
	Temperature     *float64 `json:"temperature"`
	MaxTokens       *int     `json:"max_tokens"`
	CooldownSeconds *int     `json:"cooldown_seconds"`
	Enabled         *bool    `json:"enabled"`
}

type partialProviderDefaults struct {
	TimeoutMS *int  `json:"timeout_ms"`
	Enabled   *bool `json:"enabled"`
}

type partialRelationshipDefaults struct {
	Affinity    *int `json:"affinity"`
	Hostility   *int `json:"hostility"`
	Respect     *int `json:"respect"`
	FocusWeight *int `json:"focus_weight"`
}

type partialPublicInputConfig struct {
	MaxRunes  *int `json:"max_runes"`
	RateLimit *int `json:"rate_limit"`
	WindowMS  *int `json:"window_ms"`
	MaxKeys   *int `json:"max_keys"`
}

type partialUpdateConfig struct {
	Owner             *string `json:"owner"`
	Repo              *string `json:"repo"`
	DefaultChannel    *string `json:"default_channel"`
	APITimeoutMS      *int    `json:"api_timeout_ms"`
	DownloadTimeoutMS *int    `json:"download_timeout_ms"`
	MaxDownloadBytes  *int64  `json:"max_download_bytes"`
}

type partialSQLiteConfig struct {
	JournalMode   *string `json:"journal_mode"`
	ForeignKeys   *bool   `json:"foreign_keys"`
	BusyTimeoutMS *int    `json:"busy_timeout_ms"`
	MaxOpenConns  *int    `json:"max_open_conns"`
	MaxIdleConns  *int    `json:"max_idle_conns"`
}

func readFile(path string, cfg *Config) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read config file: %w", err)
	}

	var fileCfg partialConfig
	if err := json.Unmarshal(content, &fileCfg); err != nil {
		return nil, false, fmt.Errorf("parse config file: %w", err)
	}
	applyPartial(&fileCfg, cfg)

	return content, true, nil
}

func mergeMissingConfig(content []byte, cfg Config) ([]byte, bool, error) {
	var existing map[string]any
	if err := json.Unmarshal(content, &existing); err != nil {
		return nil, false, fmt.Errorf("parse config file: %w", err)
	}

	rendered, err := marshalConfig(cfg)
	if err != nil {
		return nil, false, err
	}
	var template map[string]any
	if err := json.Unmarshal(rendered, &template); err != nil {
		return nil, false, fmt.Errorf("parse config template: %w", err)
	}

	if !mergeMissingMap(existing, template) {
		return nil, false, nil
	}

	updated, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("marshal merged config: %w", err)
	}
	return append(updated, '\n'), true, nil
}

func mergeMissingMap(dst, template map[string]any) bool {
	changed := false
	for key, templateValue := range template {
		currentValue, ok := dst[key]
		if !ok || currentValue == nil {
			dst[key] = templateValue
			changed = true
			continue
		}

		currentMap, currentOK := currentValue.(map[string]any)
		templateMap, templateOK := templateValue.(map[string]any)
		if currentOK && templateOK && mergeMissingMap(currentMap, templateMap) {
			changed = true
		}
	}
	return changed
}

func applyPartial(fileCfg *partialConfig, cfg *Config) {
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
	if fileCfg.LogLevel != nil {
		cfg.LogLevel = *fileCfg.LogLevel
	}
	if fileCfg.HTTP != nil {
		applyHTTPPartial(fileCfg.HTTP, &cfg.HTTP)
	}
	if fileCfg.LLM != nil {
		applyLLMPartial(fileCfg.LLM, &cfg.LLM)
	}
	if fileCfg.Scheduler != nil {
		applySchedulerPartial(fileCfg.Scheduler, &cfg.Scheduler)
	}
	if fileCfg.RoomDefaults != nil {
		applyRoomDefaultsPartial(fileCfg.RoomDefaults, &cfg.RoomDefaults)
	}
	if fileCfg.PersonaDefaults != nil {
		applyPersonaDefaultsPartial(fileCfg.PersonaDefaults, &cfg.PersonaDefaults)
	}
	if fileCfg.ProviderDefaults != nil {
		applyProviderDefaultsPartial(fileCfg.ProviderDefaults, &cfg.ProviderDefaults)
	}
	if fileCfg.RelationshipDefaults != nil {
		applyRelationshipDefaultsPartial(fileCfg.RelationshipDefaults, &cfg.RelationshipDefaults)
	}
	if fileCfg.PublicInput != nil {
		applyPublicInputPartial(fileCfg.PublicInput, &cfg.PublicInput)
	}
	if fileCfg.Update != nil {
		applyUpdatePartial(fileCfg.Update, &cfg.Update)
	}
	if fileCfg.SQLite != nil {
		applySQLitePartial(fileCfg.SQLite, &cfg.SQLite)
	}
}

func applyHTTPPartial(src *partialHTTPConfig, dst *HTTPConfig) {
	if src.ReadTimeoutMS != nil {
		dst.ReadTimeoutMS = *src.ReadTimeoutMS
	}
	if src.WriteTimeoutMS != nil {
		dst.WriteTimeoutMS = *src.WriteTimeoutMS
	}
	if src.IdleTimeoutMS != nil {
		dst.IdleTimeoutMS = *src.IdleTimeoutMS
	}
	if src.ShutdownMS != nil {
		dst.ShutdownMS = *src.ShutdownMS
	}
}

func applyLLMPartial(src *partialLLMConfig, dst *LLMConfig) {
	if src.DefaultTemperature != nil {
		dst.DefaultTemperature = *src.DefaultTemperature
	}
	if src.DefaultMaxTokens != nil {
		dst.DefaultMaxTokens = *src.DefaultMaxTokens
	}
	if src.RequestRetries != nil {
		dst.RequestRetries = *src.RequestRetries
	}
	if src.RetryBackoffMS != nil {
		dst.RetryBackoffMS = *src.RetryBackoffMS
	}
	if src.MaxResponseBytes != nil {
		dst.MaxResponseBytes = *src.MaxResponseBytes
	}
}

func applySchedulerPartial(src *partialSchedulerConfig, dst *SchedulerConfig) {
	if src.PollIntervalMS != nil {
		dst.PollIntervalMS = *src.PollIntervalMS
	}
	if src.ManualNudgeMS != nil {
		dst.ManualNudgeMS = *src.ManualNudgeMS
	}
	if src.UserInputNudgeMS != nil {
		dst.UserInputNudgeMS = *src.UserInputNudgeMS
	}
	if src.MinDenseGapMS != nil {
		dst.MinDenseGapMS = *src.MinDenseGapMS
	}
	if src.DefaultCooldownMS != nil {
		dst.DefaultCooldownMS = *src.DefaultCooldownMS
	}
	if src.DefaultTickMinSec != nil {
		dst.DefaultTickMinSec = *src.DefaultTickMinSec
	}
}

func applyRoomDefaultsPartial(src *partialRoomDefaults, dst *RoomDefaults) {
	if src.Status != nil {
		dst.Status = *src.Status
	}
	if src.Heat != nil {
		dst.Heat = *src.Heat
	}
	if src.ConflictLevel != nil {
		dst.ConflictLevel = *src.ConflictLevel
	}
	if src.TickMinSeconds != nil {
		dst.TickMinSeconds = *src.TickMinSeconds
	}
	if src.TickMaxSeconds != nil {
		dst.TickMaxSeconds = *src.TickMaxSeconds
	}
	if src.DailyTokenBudget != nil {
		dst.DailyTokenBudget = *src.DailyTokenBudget
	}
	if src.SummaryTriggerCount != nil {
		dst.SummaryTriggerCount = *src.SummaryTriggerCount
	}
	if src.MessageRetention != nil {
		dst.MessageRetention = *src.MessageRetention
	}
	if src.RoomPageMessageLimit != nil {
		dst.RoomPageMessageLimit = *src.RoomPageMessageLimit
	}
	if src.RoomAPIMessagesLimit != nil {
		dst.RoomAPIMessagesLimit = *src.RoomAPIMessagesLimit
	}
	if src.RecentMessageLimit != nil {
		dst.RecentMessageLimit = *src.RecentMessageLimit
	}
	if src.PromptRecentMessages != nil {
		dst.PromptRecentMessages = *src.PromptRecentMessages
	}
	if src.SummaryExtraMessages != nil {
		dst.SummaryExtraMessages = *src.SummaryExtraMessages
	}
}

func applyPersonaDefaultsPartial(src *partialPersonaDefaults, dst *PersonaDefaults) {
	if src.Aggression != nil {
		dst.Aggression = *src.Aggression
	}
	if src.ActivityLevel != nil {
		dst.ActivityLevel = *src.ActivityLevel
	}
	if src.Temperature != nil {
		dst.Temperature = *src.Temperature
	}
	if src.MaxTokens != nil {
		dst.MaxTokens = *src.MaxTokens
	}
	if src.CooldownSeconds != nil {
		dst.CooldownSeconds = *src.CooldownSeconds
	}
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
}

func applyProviderDefaultsPartial(src *partialProviderDefaults, dst *ProviderDefaults) {
	if src.TimeoutMS != nil {
		dst.TimeoutMS = *src.TimeoutMS
	}
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
}

func applyRelationshipDefaultsPartial(src *partialRelationshipDefaults, dst *RelationshipDefaults) {
	if src.Affinity != nil {
		dst.Affinity = *src.Affinity
	}
	if src.Hostility != nil {
		dst.Hostility = *src.Hostility
	}
	if src.Respect != nil {
		dst.Respect = *src.Respect
	}
	if src.FocusWeight != nil {
		dst.FocusWeight = *src.FocusWeight
	}
}

func applyPublicInputPartial(src *partialPublicInputConfig, dst *PublicInputConfig) {
	if src.MaxRunes != nil {
		dst.MaxRunes = *src.MaxRunes
	}
	if src.RateLimit != nil {
		dst.RateLimit = *src.RateLimit
	}
	if src.WindowMS != nil {
		dst.WindowMS = *src.WindowMS
	}
	if src.MaxKeys != nil {
		dst.MaxKeys = *src.MaxKeys
	}
}

func applyUpdatePartial(src *partialUpdateConfig, dst *UpdateConfig) {
	if src.Owner != nil {
		dst.Owner = *src.Owner
	}
	if src.Repo != nil {
		dst.Repo = *src.Repo
	}
	if src.DefaultChannel != nil {
		dst.DefaultChannel = *src.DefaultChannel
	}
	if src.APITimeoutMS != nil {
		dst.APITimeoutMS = *src.APITimeoutMS
	}
	if src.DownloadTimeoutMS != nil {
		dst.DownloadTimeoutMS = *src.DownloadTimeoutMS
	}
	if src.MaxDownloadBytes != nil {
		dst.MaxDownloadBytes = *src.MaxDownloadBytes
	}
}

func applySQLitePartial(src *partialSQLiteConfig, dst *SQLiteConfig) {
	if src.JournalMode != nil {
		dst.JournalMode = *src.JournalMode
	}
	if src.ForeignKeys != nil {
		dst.ForeignKeys = *src.ForeignKeys
	}
	if src.BusyTimeoutMS != nil {
		dst.BusyTimeoutMS = *src.BusyTimeoutMS
	}
	if src.MaxOpenConns != nil {
		dst.MaxOpenConns = *src.MaxOpenConns
	}
	if src.MaxIdleConns != nil {
		dst.MaxIdleConns = *src.MaxIdleConns
	}
}

func writeFile(path string, cfg Config) error {
	content, err := marshalConfig(cfg)
	if err != nil {
		return err
	}
	return writeRawFile(path, content)
}

func writeRawFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func marshalConfig(cfg Config) ([]byte, error) {
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config template: %w", err)
	}
	return append(content, '\n'), nil
}

func normalize(cfg *Config) {
	defaults := Default()

	if cfg.DefaultTimeoutMS <= 0 {
		cfg.DefaultTimeoutMS = defaults.DefaultTimeoutMS
	}
	if cfg.DefaultSummaryWindow <= 0 {
		cfg.DefaultSummaryWindow = defaults.DefaultSummaryWindow
	}
	if strings.TrimSpace(cfg.DefaultLanguage) == "" {
		cfg.DefaultLanguage = defaults.DefaultLanguage
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		cfg.LogLevel = defaults.LogLevel
	}

	if cfg.HTTP.ReadTimeoutMS <= 0 {
		cfg.HTTP.ReadTimeoutMS = defaults.HTTP.ReadTimeoutMS
	}
	if cfg.HTTP.WriteTimeoutMS <= 0 {
		cfg.HTTP.WriteTimeoutMS = defaults.HTTP.WriteTimeoutMS
	}
	if cfg.HTTP.IdleTimeoutMS <= 0 {
		cfg.HTTP.IdleTimeoutMS = defaults.HTTP.IdleTimeoutMS
	}
	if cfg.HTTP.ShutdownMS <= 0 {
		cfg.HTTP.ShutdownMS = defaults.HTTP.ShutdownMS
	}

	if cfg.LLM.DefaultTemperature <= 0 {
		cfg.LLM.DefaultTemperature = defaults.LLM.DefaultTemperature
	}
	if cfg.LLM.DefaultMaxTokens <= 0 {
		cfg.LLM.DefaultMaxTokens = defaults.LLM.DefaultMaxTokens
	}
	if cfg.LLM.RequestRetries <= 0 {
		cfg.LLM.RequestRetries = defaults.LLM.RequestRetries
	}
	if cfg.LLM.RetryBackoffMS <= 0 {
		cfg.LLM.RetryBackoffMS = defaults.LLM.RetryBackoffMS
	}
	if cfg.LLM.MaxResponseBytes <= 0 {
		cfg.LLM.MaxResponseBytes = defaults.LLM.MaxResponseBytes
	}

	if cfg.Scheduler.PollIntervalMS <= 0 {
		cfg.Scheduler.PollIntervalMS = defaults.Scheduler.PollIntervalMS
	}
	if cfg.Scheduler.ManualNudgeMS < 0 {
		cfg.Scheduler.ManualNudgeMS = defaults.Scheduler.ManualNudgeMS
	}
	if cfg.Scheduler.UserInputNudgeMS < 0 {
		cfg.Scheduler.UserInputNudgeMS = defaults.Scheduler.UserInputNudgeMS
	}
	if cfg.Scheduler.MinDenseGapMS <= 0 {
		cfg.Scheduler.MinDenseGapMS = defaults.Scheduler.MinDenseGapMS
	}
	if cfg.Scheduler.DefaultCooldownMS <= 0 {
		cfg.Scheduler.DefaultCooldownMS = defaults.Scheduler.DefaultCooldownMS
	}
	if cfg.Scheduler.DefaultTickMinSec <= 0 {
		cfg.Scheduler.DefaultTickMinSec = defaults.Scheduler.DefaultTickMinSec
	}

	if strings.TrimSpace(cfg.RoomDefaults.Status) == "" {
		cfg.RoomDefaults.Status = defaults.RoomDefaults.Status
	}
	if cfg.RoomDefaults.Heat <= 0 {
		cfg.RoomDefaults.Heat = defaults.RoomDefaults.Heat
	}
	if cfg.RoomDefaults.ConflictLevel <= 0 {
		cfg.RoomDefaults.ConflictLevel = defaults.RoomDefaults.ConflictLevel
	}
	if cfg.RoomDefaults.TickMinSeconds <= 0 {
		cfg.RoomDefaults.TickMinSeconds = defaults.RoomDefaults.TickMinSeconds
	}
	if cfg.RoomDefaults.TickMaxSeconds <= 0 {
		cfg.RoomDefaults.TickMaxSeconds = defaults.RoomDefaults.TickMaxSeconds
	}
	if cfg.RoomDefaults.DailyTokenBudget <= 0 {
		cfg.RoomDefaults.DailyTokenBudget = defaults.RoomDefaults.DailyTokenBudget
	}
	if cfg.RoomDefaults.SummaryTriggerCount <= 0 {
		cfg.RoomDefaults.SummaryTriggerCount = defaults.RoomDefaults.SummaryTriggerCount
	}
	if cfg.RoomDefaults.MessageRetention <= 0 {
		cfg.RoomDefaults.MessageRetention = defaults.RoomDefaults.MessageRetention
	}
	if cfg.RoomDefaults.RoomPageMessageLimit <= 0 {
		cfg.RoomDefaults.RoomPageMessageLimit = defaults.RoomDefaults.RoomPageMessageLimit
	}
	if cfg.RoomDefaults.RoomAPIMessagesLimit <= 0 {
		cfg.RoomDefaults.RoomAPIMessagesLimit = defaults.RoomDefaults.RoomAPIMessagesLimit
	}
	if cfg.RoomDefaults.RecentMessageLimit <= 0 {
		cfg.RoomDefaults.RecentMessageLimit = defaults.RoomDefaults.RecentMessageLimit
	}
	if cfg.RoomDefaults.PromptRecentMessages <= 0 {
		cfg.RoomDefaults.PromptRecentMessages = defaults.RoomDefaults.PromptRecentMessages
	}
	if cfg.RoomDefaults.SummaryExtraMessages < 0 {
		cfg.RoomDefaults.SummaryExtraMessages = defaults.RoomDefaults.SummaryExtraMessages
	}
	if cfg.RoomDefaults.TickMaxSeconds < cfg.RoomDefaults.TickMinSeconds {
		cfg.RoomDefaults.TickMaxSeconds = cfg.RoomDefaults.TickMinSeconds
	}

	if cfg.PersonaDefaults.Aggression <= 0 {
		cfg.PersonaDefaults.Aggression = defaults.PersonaDefaults.Aggression
	}
	if cfg.PersonaDefaults.ActivityLevel <= 0 {
		cfg.PersonaDefaults.ActivityLevel = defaults.PersonaDefaults.ActivityLevel
	}
	if cfg.PersonaDefaults.Temperature <= 0 {
		cfg.PersonaDefaults.Temperature = defaults.PersonaDefaults.Temperature
	}
	if cfg.PersonaDefaults.MaxTokens <= 0 {
		cfg.PersonaDefaults.MaxTokens = defaults.PersonaDefaults.MaxTokens
	}
	if cfg.PersonaDefaults.CooldownSeconds <= 0 {
		cfg.PersonaDefaults.CooldownSeconds = defaults.PersonaDefaults.CooldownSeconds
	}

	if cfg.ProviderDefaults.TimeoutMS <= 0 {
		cfg.ProviderDefaults.TimeoutMS = defaults.ProviderDefaults.TimeoutMS
	}

	if cfg.PublicInput.MaxRunes <= 0 {
		cfg.PublicInput.MaxRunes = defaults.PublicInput.MaxRunes
	}
	if cfg.PublicInput.RateLimit <= 0 {
		cfg.PublicInput.RateLimit = defaults.PublicInput.RateLimit
	}
	if cfg.PublicInput.WindowMS <= 0 {
		cfg.PublicInput.WindowMS = defaults.PublicInput.WindowMS
	}
	if cfg.PublicInput.MaxKeys <= 0 {
		cfg.PublicInput.MaxKeys = defaults.PublicInput.MaxKeys
	}

	if strings.TrimSpace(cfg.Update.Owner) == "" {
		cfg.Update.Owner = defaults.Update.Owner
	}
	if strings.TrimSpace(cfg.Update.Repo) == "" {
		cfg.Update.Repo = defaults.Update.Repo
	}
	if strings.TrimSpace(cfg.Update.DefaultChannel) == "" {
		cfg.Update.DefaultChannel = defaults.Update.DefaultChannel
	}
	if cfg.Update.APITimeoutMS <= 0 {
		cfg.Update.APITimeoutMS = defaults.Update.APITimeoutMS
	}
	if cfg.Update.DownloadTimeoutMS <= 0 {
		cfg.Update.DownloadTimeoutMS = defaults.Update.DownloadTimeoutMS
	}
	if cfg.Update.MaxDownloadBytes <= 0 {
		cfg.Update.MaxDownloadBytes = defaults.Update.MaxDownloadBytes
	}

	if strings.TrimSpace(cfg.SQLite.JournalMode) == "" {
		cfg.SQLite.JournalMode = defaults.SQLite.JournalMode
	}
	if cfg.SQLite.BusyTimeoutMS <= 0 {
		cfg.SQLite.BusyTimeoutMS = defaults.SQLite.BusyTimeoutMS
	}
	if cfg.SQLite.MaxOpenConns <= 0 {
		cfg.SQLite.MaxOpenConns = defaults.SQLite.MaxOpenConns
	}
	if cfg.SQLite.MaxIdleConns < 0 {
		cfg.SQLite.MaxIdleConns = defaults.SQLite.MaxIdleConns
	}
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.Address) == "" {
		return errors.New("address cannot be empty")
	}
	if strings.TrimSpace(cfg.DatabasePath) == "" {
		return errors.New("database path cannot be empty")
	}
	if strings.TrimSpace(cfg.AdminPassword) == "" && !isLoopbackAddress(cfg.Address) {
		return errors.New("admin password is required when listening on a non-loopback address")
	}
	if strings.TrimSpace(cfg.AdminPassword) != "" && strings.TrimSpace(cfg.AdminUser) == "" {
		return errors.New("admin user cannot be empty when admin password is set")
	}
	return nil
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
	if value, ok := envLookup("LLM_BB_LOG_LEVEL"); ok {
		cfg.LogLevel = value
	}

	if value, ok := envLookupInt("LLM_BB_HTTP_READ_TIMEOUT_MS"); ok {
		cfg.HTTP.ReadTimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_HTTP_WRITE_TIMEOUT_MS"); ok {
		cfg.HTTP.WriteTimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_HTTP_IDLE_TIMEOUT_MS"); ok {
		cfg.HTTP.IdleTimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_HTTP_SHUTDOWN_MS"); ok {
		cfg.HTTP.ShutdownMS = value
	}

	if value, ok := envLookupFloat("LLM_BB_LLM_DEFAULT_TEMPERATURE"); ok {
		cfg.LLM.DefaultTemperature = value
	}
	if value, ok := envLookupInt("LLM_BB_LLM_DEFAULT_MAX_TOKENS"); ok {
		cfg.LLM.DefaultMaxTokens = value
	}
	if value, ok := envLookupInt("LLM_BB_LLM_REQUEST_RETRIES"); ok {
		cfg.LLM.RequestRetries = value
	}
	if value, ok := envLookupInt("LLM_BB_LLM_RETRY_BACKOFF_MS"); ok {
		cfg.LLM.RetryBackoffMS = value
	}
	if value, ok := envLookupInt64("LLM_BB_LLM_MAX_RESPONSE_BYTES"); ok {
		cfg.LLM.MaxResponseBytes = value
	}

	if value, ok := envLookupInt("LLM_BB_SCHEDULER_POLL_INTERVAL_MS"); ok {
		cfg.Scheduler.PollIntervalMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SCHEDULER_MANUAL_NUDGE_MS"); ok {
		cfg.Scheduler.ManualNudgeMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SCHEDULER_USER_INPUT_NUDGE_MS"); ok {
		cfg.Scheduler.UserInputNudgeMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SCHEDULER_MIN_DENSE_GAP_MS"); ok {
		cfg.Scheduler.MinDenseGapMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SCHEDULER_DEFAULT_COOLDOWN_MS"); ok {
		cfg.Scheduler.DefaultCooldownMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SCHEDULER_DEFAULT_TICK_MIN_SECONDS"); ok {
		cfg.Scheduler.DefaultTickMinSec = value
	}

	if value, ok := envLookup("LLM_BB_ROOM_DEFAULT_STATUS"); ok {
		cfg.RoomDefaults.Status = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_HEAT"); ok {
		cfg.RoomDefaults.Heat = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_CONFLICT_LEVEL"); ok {
		cfg.RoomDefaults.ConflictLevel = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_TICK_MIN_SECONDS"); ok {
		cfg.RoomDefaults.TickMinSeconds = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_TICK_MAX_SECONDS"); ok {
		cfg.RoomDefaults.TickMaxSeconds = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_DAILY_TOKEN_BUDGET"); ok {
		cfg.RoomDefaults.DailyTokenBudget = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_SUMMARY_TRIGGER_COUNT"); ok {
		cfg.RoomDefaults.SummaryTriggerCount = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_DEFAULT_MESSAGE_RETENTION_COUNT"); ok {
		cfg.RoomDefaults.MessageRetention = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_PAGE_MESSAGE_LIMIT"); ok {
		cfg.RoomDefaults.RoomPageMessageLimit = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_API_MESSAGES_LIMIT"); ok {
		cfg.RoomDefaults.RoomAPIMessagesLimit = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_RECENT_MESSAGE_LIMIT"); ok {
		cfg.RoomDefaults.RecentMessageLimit = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_PROMPT_RECENT_MESSAGES"); ok {
		cfg.RoomDefaults.PromptRecentMessages = value
	}
	if value, ok := envLookupInt("LLM_BB_ROOM_SUMMARY_EXTRA_MESSAGES"); ok {
		cfg.RoomDefaults.SummaryExtraMessages = value
	}

	if value, ok := envLookupInt("LLM_BB_PERSONA_DEFAULT_AGGRESSION"); ok {
		cfg.PersonaDefaults.Aggression = value
	}
	if value, ok := envLookupInt("LLM_BB_PERSONA_DEFAULT_ACTIVITY_LEVEL"); ok {
		cfg.PersonaDefaults.ActivityLevel = value
	}
	if value, ok := envLookupFloat("LLM_BB_PERSONA_DEFAULT_TEMPERATURE"); ok {
		cfg.PersonaDefaults.Temperature = value
	}
	if value, ok := envLookupInt("LLM_BB_PERSONA_DEFAULT_MAX_TOKENS"); ok {
		cfg.PersonaDefaults.MaxTokens = value
	}
	if value, ok := envLookupInt("LLM_BB_PERSONA_DEFAULT_COOLDOWN_SECONDS"); ok {
		cfg.PersonaDefaults.CooldownSeconds = value
	}
	if value, ok := envLookupBool("LLM_BB_PERSONA_DEFAULT_ENABLED"); ok {
		cfg.PersonaDefaults.Enabled = value
	}

	if value, ok := envLookupInt("LLM_BB_PROVIDER_DEFAULT_TIMEOUT_MS"); ok {
		cfg.ProviderDefaults.TimeoutMS = value
	}
	if value, ok := envLookupBool("LLM_BB_PROVIDER_DEFAULT_ENABLED"); ok {
		cfg.ProviderDefaults.Enabled = value
	}

	if value, ok := envLookupInt("LLM_BB_RELATIONSHIP_DEFAULT_AFFINITY"); ok {
		cfg.RelationshipDefaults.Affinity = value
	}
	if value, ok := envLookupInt("LLM_BB_RELATIONSHIP_DEFAULT_HOSTILITY"); ok {
		cfg.RelationshipDefaults.Hostility = value
	}
	if value, ok := envLookupInt("LLM_BB_RELATIONSHIP_DEFAULT_RESPECT"); ok {
		cfg.RelationshipDefaults.Respect = value
	}
	if value, ok := envLookupInt("LLM_BB_RELATIONSHIP_DEFAULT_FOCUS_WEIGHT"); ok {
		cfg.RelationshipDefaults.FocusWeight = value
	}

	if value, ok := envLookupInt("LLM_BB_PUBLIC_INPUT_MAX_RUNES"); ok {
		cfg.PublicInput.MaxRunes = value
	}
	if value, ok := envLookupInt("LLM_BB_PUBLIC_INPUT_RATE_LIMIT"); ok {
		cfg.PublicInput.RateLimit = value
	}
	if value, ok := envLookupInt("LLM_BB_PUBLIC_INPUT_WINDOW_MS"); ok {
		cfg.PublicInput.WindowMS = value
	}
	if value, ok := envLookupInt("LLM_BB_PUBLIC_INPUT_MAX_KEYS"); ok {
		cfg.PublicInput.MaxKeys = value
	}

	if value, ok := envLookup("LLM_BB_UPDATE_OWNER"); ok {
		cfg.Update.Owner = value
	}
	if value, ok := envLookup("LLM_BB_UPDATE_REPO"); ok {
		cfg.Update.Repo = value
	}
	if value, ok := envLookup("LLM_BB_UPDATE_DEFAULT_CHANNEL"); ok {
		cfg.Update.DefaultChannel = value
	}
	if value, ok := envLookupInt("LLM_BB_UPDATE_API_TIMEOUT_MS"); ok {
		cfg.Update.APITimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_UPDATE_DOWNLOAD_TIMEOUT_MS"); ok {
		cfg.Update.DownloadTimeoutMS = value
	}
	if value, ok := envLookupInt64("LLM_BB_UPDATE_MAX_DOWNLOAD_BYTES"); ok {
		cfg.Update.MaxDownloadBytes = value
	}

	if value, ok := envLookup("LLM_BB_SQLITE_JOURNAL_MODE"); ok {
		cfg.SQLite.JournalMode = value
	}
	if value, ok := envLookupBool("LLM_BB_SQLITE_FOREIGN_KEYS"); ok {
		cfg.SQLite.ForeignKeys = value
	}
	if value, ok := envLookupInt("LLM_BB_SQLITE_BUSY_TIMEOUT_MS"); ok {
		cfg.SQLite.BusyTimeoutMS = value
	}
	if value, ok := envLookupInt("LLM_BB_SQLITE_MAX_OPEN_CONNS"); ok {
		cfg.SQLite.MaxOpenConns = value
	}
	if value, ok := envLookupInt("LLM_BB_SQLITE_MAX_IDLE_CONNS"); ok {
		cfg.SQLite.MaxIdleConns = value
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

func envLookupInt64(key string) (int64, bool) {
	value, ok := envLookup(key)
	if !ok || value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func envLookupFloat(key string) (float64, bool) {
	value, ok := envLookup(key)
	if !ok || value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
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
