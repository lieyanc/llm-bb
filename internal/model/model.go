package model

import "time"

type ProviderConfig struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	BaseURL      string    `json:"base_url"`
	APIKey       string    `json:"api_key"`
	DefaultModel string    `json:"default_model"`
	TimeoutMS    int       `json:"timeout_ms"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Persona struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Avatar           string    `json:"avatar"`
	PublicIdentity   string    `json:"public_identity"`
	SpeakingStyle    string    `json:"speaking_style"`
	Stance           string    `json:"stance"`
	Goal             string    `json:"goal"`
	Taboo            string    `json:"taboo"`
	Aggression       int       `json:"aggression"`
	ActivityLevel    int       `json:"activity_level"`
	FactionID        int64     `json:"faction_id"`
	ProviderConfigID int64     `json:"provider_config_id"`
	ModelName        string    `json:"model_name"`
	Temperature      float64   `json:"temperature"`
	MaxTokens        int       `json:"max_tokens"`
	CooldownSeconds  int       `json:"cooldown_seconds"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Faction struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SharedValues string    `json:"shared_values"`
	SharedStyle  string    `json:"shared_style"`
	DefaultBias  string    `json:"default_bias"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Relationship struct {
	ID              int64     `json:"id"`
	SourcePersonaID int64     `json:"source_persona_id"`
	TargetPersonaID int64     `json:"target_persona_id"`
	Affinity        int       `json:"affinity"`
	Hostility       int       `json:"hostility"`
	Respect         int       `json:"respect"`
	FocusWeight     int       `json:"focus_weight"`
	Notes           string    `json:"notes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RoomStatus string

const (
	RoomStatusRunning RoomStatus = "running"
	RoomStatusPaused  RoomStatus = "paused"
	RoomStatusDegrade RoomStatus = "degraded"
)

type Room struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Topic               string     `json:"topic"`
	Description         string     `json:"description"`
	Status              RoomStatus `json:"status"`
	Heat                int        `json:"heat"`
	ConflictLevel       int        `json:"conflict_level"`
	TickMinSeconds      int        `json:"tick_min_seconds"`
	TickMaxSeconds      int        `json:"tick_max_seconds"`
	DailyTokenBudget    int        `json:"daily_token_budget"`
	SummaryTriggerCount int        `json:"summary_trigger_count"`
	MessageRetention    int        `json:"message_retention_count"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type RoomMember struct {
	ID          int64 `json:"id"`
	RoomID      int64 `json:"room_id"`
	PersonaID   int64 `json:"persona_id"`
	RoleWeight  int   `json:"role_weight"`
	CanInitiate bool  `json:"can_initiate"`
	CanReply    bool  `json:"can_reply"`
}

type RoomMemberView struct {
	RoomMember
	PersonaName        string  `json:"persona_name"`
	Avatar             string  `json:"avatar"`
	PublicIdentity     string  `json:"public_identity"`
	SpeakingStyle      string  `json:"speaking_style"`
	Stance             string  `json:"stance"`
	Goal               string  `json:"goal"`
	Taboo              string  `json:"taboo"`
	Aggression         int     `json:"aggression"`
	ActivityLevel      int     `json:"activity_level"`
	ModelName          string  `json:"model_name"`
	Temperature        float64 `json:"temperature"`
	MaxTokens          int     `json:"max_tokens"`
	CooldownSeconds    int     `json:"cooldown_seconds"`
	PersonaEnabled     bool    `json:"persona_enabled"`
	FactionID          int64   `json:"faction_id"`
	FactionName        string  `json:"faction_name"`
	FactionDescription string  `json:"faction_description"`
	ProviderConfigID   int64   `json:"provider_config_id"`
	ProviderName       string  `json:"provider_name"`
	ProviderBaseURL    string  `json:"provider_base_url"`
	ProviderAPIKey     string  `json:"provider_api_key"`
	ProviderModel      string  `json:"provider_model"`
	ProviderTimeoutMS  int     `json:"provider_timeout_ms"`
	ProviderEnabled    bool    `json:"provider_enabled"`
}

type MessageKind string

const (
	MessageKindChat    MessageKind = "chat"
	MessageKindUser    MessageKind = "user"
	MessageKindSystem  MessageKind = "system"
	MessageKindSummary MessageKind = "summary"
)

type MessageSource string

const (
	MessageSourceScheduler MessageSource = "scheduler"
	MessageSourceUser      MessageSource = "user"
	MessageSourceManual    MessageSource = "manual"
)

type Message struct {
	ID               int64         `json:"id"`
	RoomID           int64         `json:"room_id"`
	PersonaID        int64         `json:"persona_id"`
	PersonaName      string        `json:"persona_name"`
	PersonaAvatar    string        `json:"persona_avatar"`
	Kind             MessageKind   `json:"kind"`
	Content          string        `json:"content"`
	ReplyToMessageID int64         `json:"reply_to_message_id"`
	Source           MessageSource `json:"source"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	CreatedAt        time.Time     `json:"created_at"`
}

type Summary struct {
	ID            int64     `json:"id"`
	RoomID        int64     `json:"room_id"`
	FromMessageID int64     `json:"from_message_id"`
	ToMessageID   int64     `json:"to_message_id"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
}

type RoomOverview struct {
	Room
	MessageCount      int   `json:"message_count"`
	TokensToday       int   `json:"tokens_today"`
	MembersCount      int   `json:"members_count"`
	LastMessageAtUnix int64 `json:"last_message_at_unix"`
}
