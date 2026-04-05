package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-bb/internal/model"
)

func (s *Store) ListProviders(ctx context.Context) ([]model.ProviderConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			base_url,
			api_key,
			default_model,
			timeout_ms,
			enabled,
			created_at,
			updated_at
		FROM provider_configs
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	var providers []model.ProviderConfig
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (s *Store) CreateProvider(ctx context.Context, provider *model.ProviderConfig) error {
	if provider.TimeoutMS <= 0 {
		provider.TimeoutMS = 20000
	}
	now := time.Now().UTC()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_configs (
			name,
			base_url,
			api_key,
			default_model,
			timeout_ms,
			enabled,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		provider.Name,
		provider.BaseURL,
		provider.APIKey,
		provider.DefaultModel,
		provider.TimeoutMS,
		boolToInt(provider.Enabled),
		formatTime(provider.CreatedAt),
		formatTime(provider.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	provider.ID, err = result.LastInsertId()
	return err
}

func (s *Store) PatchProvider(ctx context.Context, id int64, fields map[string]any) error {
	return s.patchRecord(ctx, "provider_configs", id, fields, map[string]patchConverter{
		"name":          patchString("name"),
		"base_url":      patchString("base_url"),
		"api_key":       patchString("api_key"),
		"default_model": patchString("default_model"),
		"timeout_ms":    patchInt("timeout_ms"),
		"enabled":       patchBool("enabled"),
	})
}

func (s *Store) ListFactions(ctx context.Context) ([]model.Faction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			description,
			shared_values,
			shared_style,
			default_bias,
			created_at,
			updated_at
		FROM factions
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list factions: %w", err)
	}
	defer rows.Close()

	var factions []model.Faction
	for rows.Next() {
		faction, err := scanFaction(rows)
		if err != nil {
			return nil, err
		}
		factions = append(factions, faction)
	}
	return factions, rows.Err()
}

func (s *Store) CreateFaction(ctx context.Context, faction *model.Faction) error {
	now := time.Now().UTC()
	faction.CreatedAt = now
	faction.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO factions (
			name,
			description,
			shared_values,
			shared_style,
			default_bias,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		faction.Name,
		faction.Description,
		faction.SharedValues,
		faction.SharedStyle,
		faction.DefaultBias,
		formatTime(faction.CreatedAt),
		formatTime(faction.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create faction: %w", err)
	}
	faction.ID, err = result.LastInsertId()
	return err
}

func (s *Store) PatchFaction(ctx context.Context, id int64, fields map[string]any) error {
	return s.patchRecord(ctx, "factions", id, fields, map[string]patchConverter{
		"name":          patchString("name"),
		"description":   patchString("description"),
		"shared_values": patchString("shared_values"),
		"shared_style":  patchString("shared_style"),
		"default_bias":  patchString("default_bias"),
	})
}

func (s *Store) ListPersonas(ctx context.Context) ([]model.Persona, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			avatar,
			public_identity,
			speaking_style,
			stance,
			goal,
			taboo,
			aggression,
			activity_level,
			faction_id,
			provider_config_id,
			model_name,
			temperature,
			max_tokens,
			cooldown_seconds,
			enabled,
			created_at,
			updated_at
		FROM personas
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list personas: %w", err)
	}
	defer rows.Close()

	var personas []model.Persona
	for rows.Next() {
		persona, err := scanPersona(rows)
		if err != nil {
			return nil, err
		}
		personas = append(personas, persona)
	}
	return personas, rows.Err()
}

func (s *Store) CreatePersona(ctx context.Context, persona *model.Persona) error {
	if persona.Temperature == 0 {
		persona.Temperature = 0.9
	}
	if persona.MaxTokens <= 0 {
		persona.MaxTokens = 220
	}
	if persona.CooldownSeconds <= 0 {
		persona.CooldownSeconds = 120
	}
	if persona.ActivityLevel <= 0 {
		persona.ActivityLevel = 50
	}
	now := time.Now().UTC()
	persona.CreatedAt = now
	persona.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO personas (
			name,
			avatar,
			public_identity,
			speaking_style,
			stance,
			goal,
			taboo,
			aggression,
			activity_level,
			faction_id,
			provider_config_id,
			model_name,
			temperature,
			max_tokens,
			cooldown_seconds,
			enabled,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		persona.Name,
		persona.Avatar,
		persona.PublicIdentity,
		persona.SpeakingStyle,
		persona.Stance,
		persona.Goal,
		persona.Taboo,
		persona.Aggression,
		persona.ActivityLevel,
		persona.FactionID,
		persona.ProviderConfigID,
		persona.ModelName,
		persona.Temperature,
		persona.MaxTokens,
		persona.CooldownSeconds,
		boolToInt(persona.Enabled),
		formatTime(persona.CreatedAt),
		formatTime(persona.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create persona: %w", err)
	}
	persona.ID, err = result.LastInsertId()
	return err
}

func (s *Store) PatchPersona(ctx context.Context, id int64, fields map[string]any) error {
	return s.patchRecord(ctx, "personas", id, fields, map[string]patchConverter{
		"name":               patchString("name"),
		"avatar":             patchString("avatar"),
		"public_identity":    patchString("public_identity"),
		"speaking_style":     patchString("speaking_style"),
		"stance":             patchString("stance"),
		"goal":               patchString("goal"),
		"taboo":              patchString("taboo"),
		"aggression":         patchInt("aggression"),
		"activity_level":     patchInt("activity_level"),
		"faction_id":         patchInt64("faction_id"),
		"provider_config_id": patchInt64("provider_config_id"),
		"model_name":         patchString("model_name"),
		"temperature":        patchFloat("temperature"),
		"max_tokens":         patchInt("max_tokens"),
		"cooldown_seconds":   patchInt("cooldown_seconds"),
		"enabled":            patchBool("enabled"),
	})
}

func (s *Store) CreateRoom(ctx context.Context, room *model.Room) error {
	if room.Status == "" {
		room.Status = model.RoomStatusRunning
	}
	if room.TickMinSeconds <= 0 {
		room.TickMinSeconds = 25
	}
	if room.TickMaxSeconds <= 0 {
		room.TickMaxSeconds = 55
	}
	if room.DailyTokenBudget <= 0 {
		room.DailyTokenBudget = 40000
	}
	if room.SummaryTriggerCount <= 0 {
		room.SummaryTriggerCount = 24
	}
	if room.MessageRetention <= 0 {
		room.MessageRetention = 120
	}
	if room.Heat <= 0 {
		room.Heat = 50
	}
	if room.ConflictLevel <= 0 {
		room.ConflictLevel = 50
	}
	now := time.Now().UTC()
	room.CreatedAt = now
	room.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO rooms (
			name,
			topic,
			description,
			status,
			heat,
			conflict_level,
			tick_min_seconds,
			tick_max_seconds,
			daily_token_budget,
			summary_trigger_count,
			message_retention_count,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		room.Name,
		room.Topic,
		room.Description,
		room.Status,
		room.Heat,
		room.ConflictLevel,
		room.TickMinSeconds,
		room.TickMaxSeconds,
		room.DailyTokenBudget,
		room.SummaryTriggerCount,
		room.MessageRetention,
		formatTime(room.CreatedAt),
		formatTime(room.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	room.ID, err = result.LastInsertId()
	return err
}

func (s *Store) PatchRoom(ctx context.Context, id int64, fields map[string]any) error {
	return s.patchRecord(ctx, "rooms", id, fields, map[string]patchConverter{
		"name":                    patchString("name"),
		"topic":                   patchString("topic"),
		"description":             patchString("description"),
		"status":                  patchString("status"),
		"heat":                    patchInt("heat"),
		"conflict_level":          patchInt("conflict_level"),
		"tick_min_seconds":        patchInt("tick_min_seconds"),
		"tick_max_seconds":        patchInt("tick_max_seconds"),
		"daily_token_budget":      patchInt("daily_token_budget"),
		"summary_trigger_count":   patchInt("summary_trigger_count"),
		"message_retention_count": patchInt("message_retention_count"),
	})
}

func (s *Store) SetRoomStatus(ctx context.Context, roomID int64, status model.RoomStatus) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rooms
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, formatTime(time.Now().UTC()), roomID)
	if err != nil {
		return fmt.Errorf("set room status: %w", err)
	}
	return nil
}

func (s *Store) UpsertRelationship(ctx context.Context, relationship *model.Relationship) error {
	relationship.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO relationships (
			source_persona_id,
			target_persona_id,
			affinity,
			hostility,
			respect,
			focus_weight,
			notes,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_persona_id, target_persona_id) DO UPDATE SET
			affinity = excluded.affinity,
			hostility = excluded.hostility,
			respect = excluded.respect,
			focus_weight = excluded.focus_weight,
			notes = excluded.notes,
			updated_at = excluded.updated_at
	`,
		relationship.SourcePersonaID,
		relationship.TargetPersonaID,
		relationship.Affinity,
		relationship.Hostility,
		relationship.Respect,
		relationship.FocusWeight,
		relationship.Notes,
		formatTime(relationship.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert relationship: %w", err)
	}
	return nil
}

func (s *Store) SeedDemoData(ctx context.Context) error {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rooms`)
	var count int
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("count rooms for seed: %w", err)
	}
	if count > 0 {
		return nil
	}

	torch := model.Faction{
		Name:         "火药桶",
		Description:  "喜欢对线、抢节奏、推高热度。",
		SharedValues: "气势、抢话权、赢下嘴仗",
		SharedStyle:  "直接、阴阳、带刺",
		DefaultBias:  "优先维持冲突热度",
	}
	ice := model.Faction{
		Name:         "冷处理联盟",
		Description:  "擅长拆招、打圆场、装理中客。",
		SharedValues: "控制场面、维持表面秩序",
		SharedStyle:  "克制、拐弯、略带优越感",
		DefaultBias:  "避免失控但不放弃阴阳",
	}
	if err := s.CreateFaction(ctx, &torch); err != nil {
		return err
	}
	if err := s.CreateFaction(ctx, &ice); err != nil {
		return err
	}

	personas := []*model.Persona{
		{
			Name:            "暴论哥",
			PublicIdentity:  "爱下定义的热点评论员",
			SpeakingStyle:   "上来先定性，句子短，攻击性强",
			Stance:          "凡事都要快速站队",
			Goal:            "把房间节奏拉到自己熟悉的主场",
			Taboo:           "不承认自己没看全上下文",
			Aggression:      85,
			ActivityLevel:   88,
			FactionID:       torch.ID,
			ModelName:       "demo/local",
			Temperature:     0.9,
			MaxTokens:       220,
			CooldownSeconds: 35,
			Enabled:         true,
		},
		{
			Name:            "反串姐",
			PublicIdentity:  "假装中立的拱火选手",
			SpeakingStyle:   "表面柔和，实则句句递刀子",
			Stance:          "喜欢用反问和类比挑拨",
			Goal:            "让别人继续吵，自己站在旁边看热闹",
			Taboo:           "不愿意给出明确建设性方案",
			Aggression:      74,
			ActivityLevel:   76,
			FactionID:       torch.ID,
			ModelName:       "demo/local",
			Temperature:     1.1,
			MaxTokens:       220,
			CooldownSeconds: 45,
			Enabled:         true,
		},
		{
			Name:            "理中客",
			PublicIdentity:  "习惯端平水的分析师",
			SpeakingStyle:   "条理化、喜欢说两边都有问题",
			Stance:          "试图给争论套上框架",
			Goal:            "控制局面，不让讨论完全失控",
			Taboo:           "不愿正面承认自己也在站队",
			Aggression:      42,
			ActivityLevel:   68,
			FactionID:       ice.ID,
			ModelName:       "demo/local",
			Temperature:     0.7,
			MaxTokens:       240,
			CooldownSeconds: 55,
			Enabled:         true,
		},
		{
			Name:            "阴阳师",
			PublicIdentity:  "喜欢用礼貌语气说刻薄话的人",
			SpeakingStyle:   "措辞精致，但态度明显不客气",
			Stance:          "讨厌空话，擅长精准挖苦",
			Goal:            "拆穿别人话术并留下记忆点",
			Taboo:           "讨厌直给粗口，觉得不高级",
			Aggression:      66,
			ActivityLevel:   72,
			FactionID:       ice.ID,
			ModelName:       "demo/local",
			Temperature:     0.85,
			MaxTokens:       210,
			CooldownSeconds: 50,
			Enabled:         true,
		},
	}

	for _, persona := range personas {
		if err := s.CreatePersona(ctx, persona); err != nil {
			return err
		}
	}

	room := model.Room{
		Name:                "热榜审判庭",
		Topic:               "围绕热点事件进行长期嘴仗和站队表演",
		Description:         "一个持续运转的群聊背景板。角色会结盟、拱火、反驳和拆台，用户可以随时插话。",
		Status:              model.RoomStatusRunning,
		Heat:                78,
		ConflictLevel:       73,
		TickMinSeconds:      18,
		TickMaxSeconds:      42,
		DailyTokenBudget:    30000,
		SummaryTriggerCount: 18,
		MessageRetention:    120,
	}
	if err := s.CreateRoom(ctx, &room); err != nil {
		return err
	}

	personaIDs := make([]int64, 0, len(personas))
	for _, persona := range personas {
		personaIDs = append(personaIDs, persona.ID)
	}
	if err := s.ReplaceRoomMembers(ctx, room.ID, personaIDs); err != nil {
		return err
	}

	relationships := []*model.Relationship{
		{SourcePersonaID: personas[0].ID, TargetPersonaID: personas[2].ID, Hostility: 60, Respect: 20, FocusWeight: 70, Notes: "讨厌对方打圆场"},
		{SourcePersonaID: personas[2].ID, TargetPersonaID: personas[0].ID, Hostility: 45, Respect: 40, FocusWeight: 60, Notes: "觉得对方太武断"},
		{SourcePersonaID: personas[1].ID, TargetPersonaID: personas[0].ID, Affinity: 42, Hostility: 18, FocusWeight: 55, Notes: "喜欢借对方的冲劲拱火"},
		{SourcePersonaID: personas[3].ID, TargetPersonaID: personas[1].ID, Hostility: 30, Respect: 50, FocusWeight: 48, Notes: "欣赏对方的刀法，但不喜欢太明显"},
		{SourcePersonaID: personas[3].ID, TargetPersonaID: personas[0].ID, Hostility: 55, Respect: 35, FocusWeight: 62, Notes: "觉得对方太糙，适合精准拆台"},
		{SourcePersonaID: personas[1].ID, TargetPersonaID: personas[3].ID, Affinity: 36, Hostility: 12, FocusWeight: 52, Notes: "乐意和对方打配合"},
	}
	for _, relationship := range relationships {
		if err := s.UpsertRelationship(ctx, relationship); err != nil {
			return err
		}
	}

	systemMessage := model.Message{
		RoomID:    room.ID,
		Kind:      model.MessageKindSystem,
		Content:   "房间已启动。当前主题：热榜审判庭。观众可以直接插话或 @角色 点名。",
		Source:    model.MessageSourceManual,
		CreatedAt: time.Now().UTC(),
	}
	return s.CreateMessage(ctx, &systemMessage)
}

type patchConverter func(value any) (column string, arg any, err error)

func patchString(column string) patchConverter {
	return func(value any) (string, any, error) {
		return column, strings.TrimSpace(fmt.Sprintf("%v", value)), nil
	}
}

func patchInt(column string) patchConverter {
	return func(value any) (string, any, error) {
		parsed, err := anyToInt(value)
		return column, parsed, err
	}
}

func patchInt64(column string) patchConverter {
	return func(value any) (string, any, error) {
		parsed, err := anyToInt64(value)
		return column, parsed, err
	}
}

func patchFloat(column string) patchConverter {
	return func(value any) (string, any, error) {
		parsed, err := anyToFloat64(value)
		return column, parsed, err
	}
}

func patchBool(column string) patchConverter {
	return func(value any) (string, any, error) {
		parsed, err := anyToBool(value)
		return column, boolToInt(parsed), err
	}
}

func (s *Store) patchRecord(ctx context.Context, table string, id int64, fields map[string]any, allowed map[string]patchConverter) error {
	if len(fields) == 0 {
		return nil
	}

	var sets []string
	var args []any
	for key, value := range fields {
		convert, ok := allowed[key]
		if !ok {
			continue
		}
		column, arg, err := convert(value)
		if err != nil {
			return fmt.Errorf("patch field %s: %w", key, err)
		}
		sets = append(sets, fmt.Sprintf("%s = ?", column))
		args = append(args, arg)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, formatTime(time.Now().UTC()), id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, strings.Join(sets, ", "))
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("patch %s #%d: %w", table, id, err)
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
