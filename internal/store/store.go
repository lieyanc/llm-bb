package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"llm-bb/internal/model"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.applyPragmas(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) applyPragmas(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}

	for _, stmt := range pragmas {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply pragma %q: %w", stmt, err)
		}
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS provider_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			default_model TEXT NOT NULL DEFAULT '',
			timeout_ms INTEGER NOT NULL DEFAULT 20000,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS factions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			shared_values TEXT NOT NULL DEFAULT '',
			shared_style TEXT NOT NULL DEFAULT '',
			default_bias TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS personas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			avatar TEXT NOT NULL DEFAULT '',
			public_identity TEXT NOT NULL DEFAULT '',
			speaking_style TEXT NOT NULL DEFAULT '',
			stance TEXT NOT NULL DEFAULT '',
			goal TEXT NOT NULL DEFAULT '',
			taboo TEXT NOT NULL DEFAULT '',
			aggression INTEGER NOT NULL DEFAULT 0,
			activity_level INTEGER NOT NULL DEFAULT 50,
			faction_id INTEGER NOT NULL DEFAULT 0,
			provider_config_id INTEGER NOT NULL DEFAULT 0,
			model_name TEXT NOT NULL DEFAULT '',
			temperature REAL NOT NULL DEFAULT 0.9,
			max_tokens INTEGER NOT NULL DEFAULT 220,
			cooldown_seconds INTEGER NOT NULL DEFAULT 120,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS relationships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_persona_id INTEGER NOT NULL,
			target_persona_id INTEGER NOT NULL,
			affinity INTEGER NOT NULL DEFAULT 0,
			hostility INTEGER NOT NULL DEFAULT 0,
			respect INTEGER NOT NULL DEFAULT 0,
			focus_weight INTEGER NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			UNIQUE(source_persona_id, target_persona_id)
		);`,
		`CREATE TABLE IF NOT EXISTS rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			topic TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'running',
			heat INTEGER NOT NULL DEFAULT 50,
			conflict_level INTEGER NOT NULL DEFAULT 50,
			tick_min_seconds INTEGER NOT NULL DEFAULT 25,
			tick_max_seconds INTEGER NOT NULL DEFAULT 55,
			daily_token_budget INTEGER NOT NULL DEFAULT 40000,
			summary_trigger_count INTEGER NOT NULL DEFAULT 24,
			message_retention_count INTEGER NOT NULL DEFAULT 120,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS room_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id INTEGER NOT NULL,
			persona_id INTEGER NOT NULL,
			role_weight INTEGER NOT NULL DEFAULT 100,
			can_initiate INTEGER NOT NULL DEFAULT 1,
			can_reply INTEGER NOT NULL DEFAULT 1,
			UNIQUE(room_id, persona_id)
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id INTEGER NOT NULL,
			persona_id INTEGER NOT NULL DEFAULT 0,
			kind TEXT NOT NULL,
			content TEXT NOT NULL,
			reply_to_message_id INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS summaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id INTEGER NOT NULL,
			from_message_id INTEGER NOT NULL,
			to_message_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_room_created_at ON messages(room_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_room_members_room_id ON room_members(room_id);`,
		`CREATE INDEX IF NOT EXISTS idx_relationships_source_target ON relationships(source_persona_id, target_persona_id);`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_room_id ON summaries(room_id, id DESC);`,
	}

	for _, stmt := range schema {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate schema: %w", err)
		}
	}

	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) ListRooms(ctx context.Context) ([]model.RoomOverview, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			r.id,
			r.name,
			r.topic,
			r.description,
			r.status,
			r.heat,
			r.conflict_level,
			r.tick_min_seconds,
			r.tick_max_seconds,
			r.daily_token_budget,
			r.summary_trigger_count,
			r.message_retention_count,
			r.created_at,
			r.updated_at,
			COALESCE((SELECT COUNT(*) FROM messages m WHERE m.room_id = r.id), 0) AS message_count,
			COALESCE((
				SELECT SUM(m.prompt_tokens + m.completion_tokens)
				FROM messages m
				WHERE m.room_id = r.id
					AND date(m.created_at) = date('now')
			), 0) AS tokens_today,
			COALESCE((SELECT COUNT(*) FROM room_members rm WHERE rm.room_id = r.id), 0) AS members_count,
			COALESCE((
				SELECT CAST(strftime('%s', MAX(m.created_at)) AS INTEGER)
				FROM messages m
				WHERE m.room_id = r.id
			), 0) AS last_message_at_unix
		FROM rooms r
		ORDER BY r.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var rooms []model.RoomOverview
	for rows.Next() {
		room, err := scanRoomOverview(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rooms: %w", err)
	}

	return rooms, nil
}

func (s *Store) ListRunnableRooms(ctx context.Context) ([]model.Room, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
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
		FROM rooms
		WHERE status IN (?, ?)
		ORDER BY id ASC
	`, model.RoomStatusRunning, model.RoomStatusDegrade)
	if err != nil {
		return nil, fmt.Errorf("list runnable rooms: %w", err)
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (s *Store) GetRoom(ctx context.Context, roomID int64) (model.Room, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
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
		FROM rooms
		WHERE id = ?
	`, roomID)

	room, err := scanRoom(row)
	if err != nil {
		return model.Room{}, fmt.Errorf("get room %d: %w", roomID, err)
	}
	return room, nil
}

func (s *Store) ListRoomMembers(ctx context.Context, roomID int64) ([]model.RoomMemberView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			rm.id,
			rm.room_id,
			rm.persona_id,
			rm.role_weight,
			rm.can_initiate,
			rm.can_reply,
			p.name,
			p.avatar,
			p.public_identity,
			p.speaking_style,
			p.stance,
			p.goal,
			p.taboo,
			p.aggression,
			p.activity_level,
			p.model_name,
			p.temperature,
			p.max_tokens,
			p.cooldown_seconds,
			p.enabled,
			p.faction_id,
			COALESCE(f.name, ''),
			COALESCE(f.description, ''),
			p.provider_config_id,
			COALESCE(pc.name, ''),
			COALESCE(pc.base_url, ''),
			COALESCE(pc.api_key, ''),
			COALESCE(pc.default_model, ''),
			COALESCE(pc.timeout_ms, 0),
			COALESCE(pc.enabled, 0)
		FROM room_members rm
		INNER JOIN personas p ON p.id = rm.persona_id
		LEFT JOIN factions f ON f.id = p.faction_id
		LEFT JOIN provider_configs pc ON pc.id = p.provider_config_id
		WHERE rm.room_id = ?
		ORDER BY rm.role_weight DESC, p.id ASC
	`, roomID)
	if err != nil {
		return nil, fmt.Errorf("list room members: %w", err)
	}
	defer rows.Close()

	var members []model.RoomMemberView
	for rows.Next() {
		member, err := scanRoomMemberView(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) ListRoomMessages(ctx context.Context, roomID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			m.id,
			m.room_id,
			m.persona_id,
			COALESCE(p.name, ''),
			COALESCE(p.avatar, ''),
			m.kind,
			m.content,
			m.reply_to_message_id,
			m.source,
			m.prompt_tokens,
			m.completion_tokens,
			m.created_at
		FROM messages m
		LEFT JOIN personas p ON p.id = m.persona_id
		WHERE m.room_id = ?
		ORDER BY m.id DESC
		LIMIT ?
	`, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("list room messages: %w", err)
	}
	defer rows.Close()

	var reversed []model.Message
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		reversed = append(reversed, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	messages := make([]model.Message, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		messages = append(messages, reversed[i])
	}
	return messages, nil
}

func (s *Store) ListRecentMessagesDescending(ctx context.Context, roomID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			m.id,
			m.room_id,
			m.persona_id,
			COALESCE(p.name, ''),
			COALESCE(p.avatar, ''),
			m.kind,
			m.content,
			m.reply_to_message_id,
			m.source,
			m.prompt_tokens,
			m.completion_tokens,
			m.created_at
		FROM messages m
		LEFT JOIN personas p ON p.id = m.persona_id
		WHERE m.room_id = ?
		ORDER BY m.id DESC
		LIMIT ?
	`, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent messages descending: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *Store) GetLatestSummary(ctx context.Context, roomID int64) (*model.Summary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			room_id,
			from_message_id,
			to_message_id,
			content,
			created_at
		FROM summaries
		WHERE room_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, roomID)

	summary, err := scanSummary(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest summary: %w", err)
	}
	return &summary, nil
}

func (s *Store) ListRelationshipsForPersonas(ctx context.Context, personaIDs []int64) ([]model.Relationship, error) {
	if len(personaIDs) == 0 {
		return nil, nil
	}

	var args []any
	placeholders := make([]string, 0, len(personaIDs))
	for _, id := range personaIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			source_persona_id,
			target_persona_id,
			affinity,
			hostility,
			respect,
			focus_weight,
			notes,
			updated_at
		FROM relationships
		WHERE source_persona_id IN (%s) AND target_persona_id IN (%s)
	`, strings.Join(placeholders, ","), strings.Join(placeholders, ","))

	args = append(args, args...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}
	defer rows.Close()

	var relationships []model.Relationship
	for rows.Next() {
		relationship, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	return relationships, rows.Err()
}

func (s *Store) CreateMessage(ctx context.Context, message *model.Message) error {
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (
			room_id,
			persona_id,
			kind,
			content,
			reply_to_message_id,
			source,
			prompt_tokens,
			completion_tokens,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		message.RoomID,
		message.PersonaID,
		message.Kind,
		message.Content,
		message.ReplyToMessageID,
		message.Source,
		message.PromptTokens,
		message.CompletionTokens,
		formatTime(message.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("message last insert id: %w", err)
	}
	message.ID = id

	if message.PersonaID > 0 && message.PersonaName == "" {
		row := s.db.QueryRowContext(ctx, "SELECT name, avatar FROM personas WHERE id = ?", message.PersonaID)
		_ = row.Scan(&message.PersonaName, &message.PersonaAvatar)
	}

	return nil
}

func (s *Store) CreateSummary(ctx context.Context, summary *model.Summary) error {
	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = time.Now().UTC()
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO summaries (
			room_id,
			from_message_id,
			to_message_id,
			content,
			created_at
		) VALUES (?, ?, ?, ?, ?)
	`, summary.RoomID, summary.FromMessageID, summary.ToMessageID, summary.Content, formatTime(summary.CreatedAt))
	if err != nil {
		return fmt.Errorf("create summary: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("summary last insert id: %w", err)
	}
	summary.ID = id
	return nil
}

func (s *Store) RoomTokenUsageToday(ctx context.Context, roomID int64) (int, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(prompt_tokens + completion_tokens), 0)
		FROM messages
		WHERE room_id = ? AND date(created_at) = date('now')
	`, roomID)

	var used int
	if err := row.Scan(&used); err != nil {
		return 0, fmt.Errorf("room token usage: %w", err)
	}
	return used, nil
}

func (s *Store) CountMessagesSinceSummary(ctx context.Context, roomID int64, summaryID int64) (int, error) {
	if summaryID <= 0 {
		row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE room_id = ? AND kind IN (?, ?, ?)`, roomID, model.MessageKindChat, model.MessageKindUser, model.MessageKindSystem)
		var count int
		if err := row.Scan(&count); err != nil {
			return 0, fmt.Errorf("count messages without summary: %w", err)
		}
		return count, nil
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM messages m
		WHERE m.room_id = ?
			AND m.id > COALESCE((SELECT to_message_id FROM summaries WHERE id = ?), 0)
			AND m.kind IN (?, ?, ?)
	`, roomID, summaryID, model.MessageKindChat, model.MessageKindUser, model.MessageKindSystem)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count messages since summary: %w", err)
	}
	return count, nil
}

func (s *Store) ReplaceRoomMembers(ctx context.Context, roomID int64, personaIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace room members: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM room_members WHERE room_id = ?`, roomID); err != nil {
		return fmt.Errorf("clear room members: %w", err)
	}

	for _, personaID := range personaIDs {
		if personaID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO room_members (
				room_id,
				persona_id,
				role_weight,
				can_initiate,
				can_reply
			) VALUES (?, ?, 100, 1, 1)
		`, roomID, personaID); err != nil {
			return fmt.Errorf("insert room member %d: %w", personaID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE rooms SET updated_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), roomID); err != nil {
		return fmt.Errorf("touch room after members update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace room members: %w", err)
	}
	return nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err == nil {
		return parsed, nil
	}
	parsed, err = time.Parse(time.RFC3339, raw)
	if err == nil {
		return parsed, nil
	}
	return time.Time{}, err
}

func anyToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, nil
		}
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported int64 type %T", value)
	}
}

func anyToInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, nil
		}
		return strconv.Atoi(strings.TrimSpace(v))
	default:
		return 0, fmt.Errorf("unsupported int type %T", value)
	}
}

func anyToFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, nil
		}
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return 0, fmt.Errorf("unsupported float type %T", value)
	}
}

func anyToBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "", "0", "false", "off", "no":
			return false, nil
		case "1", "true", "on", "yes":
			return true, nil
		default:
			return false, fmt.Errorf("unsupported bool string %q", v)
		}
	default:
		return false, fmt.Errorf("unsupported bool type %T", value)
	}
}

func scanRoom(scanner interface{ Scan(dest ...any) error }) (model.Room, error) {
	var room model.Room
	var status string
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&room.ID,
		&room.Name,
		&room.Topic,
		&room.Description,
		&status,
		&room.Heat,
		&room.ConflictLevel,
		&room.TickMinSeconds,
		&room.TickMaxSeconds,
		&room.DailyTokenBudget,
		&room.SummaryTriggerCount,
		&room.MessageRetention,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.Room{}, err
	}

	room.Status = model.RoomStatus(status)
	room.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Room{}, fmt.Errorf("parse room created_at: %w", err)
	}
	room.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Room{}, fmt.Errorf("parse room updated_at: %w", err)
	}
	return room, nil
}

func scanRoomOverview(scanner interface{ Scan(dest ...any) error }) (model.RoomOverview, error) {
	var overview model.RoomOverview
	var status string
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&overview.ID,
		&overview.Name,
		&overview.Topic,
		&overview.Description,
		&status,
		&overview.Heat,
		&overview.ConflictLevel,
		&overview.TickMinSeconds,
		&overview.TickMaxSeconds,
		&overview.DailyTokenBudget,
		&overview.SummaryTriggerCount,
		&overview.MessageRetention,
		&createdAt,
		&updatedAt,
		&overview.MessageCount,
		&overview.TokensToday,
		&overview.MembersCount,
		&overview.LastMessageAtUnix,
	)
	if err != nil {
		return model.RoomOverview{}, fmt.Errorf("scan room overview: %w", err)
	}

	overview.Status = model.RoomStatus(status)
	overview.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.RoomOverview{}, err
	}
	overview.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.RoomOverview{}, err
	}
	return overview, nil
}

func scanRoomMemberView(scanner interface{ Scan(dest ...any) error }) (model.RoomMemberView, error) {
	var member model.RoomMemberView
	var canInitiate int
	var canReply int
	var enabled int
	var providerEnabled int

	err := scanner.Scan(
		&member.ID,
		&member.RoomID,
		&member.PersonaID,
		&member.RoleWeight,
		&canInitiate,
		&canReply,
		&member.PersonaName,
		&member.Avatar,
		&member.PublicIdentity,
		&member.SpeakingStyle,
		&member.Stance,
		&member.Goal,
		&member.Taboo,
		&member.Aggression,
		&member.ActivityLevel,
		&member.ModelName,
		&member.Temperature,
		&member.MaxTokens,
		&member.CooldownSeconds,
		&enabled,
		&member.FactionID,
		&member.FactionName,
		&member.FactionDescription,
		&member.ProviderConfigID,
		&member.ProviderName,
		&member.ProviderBaseURL,
		&member.ProviderAPIKey,
		&member.ProviderModel,
		&member.ProviderTimeoutMS,
		&providerEnabled,
	)
	if err != nil {
		return model.RoomMemberView{}, fmt.Errorf("scan room member: %w", err)
	}

	member.CanInitiate = canInitiate != 0
	member.CanReply = canReply != 0
	member.PersonaEnabled = enabled != 0
	member.ProviderEnabled = providerEnabled != 0
	return member, nil
}

func scanMessage(scanner interface{ Scan(dest ...any) error }) (model.Message, error) {
	var message model.Message
	var kind string
	var source string
	var createdAt string

	err := scanner.Scan(
		&message.ID,
		&message.RoomID,
		&message.PersonaID,
		&message.PersonaName,
		&message.PersonaAvatar,
		&kind,
		&message.Content,
		&message.ReplyToMessageID,
		&source,
		&message.PromptTokens,
		&message.CompletionTokens,
		&createdAt,
	)
	if err != nil {
		return model.Message{}, fmt.Errorf("scan message: %w", err)
	}

	message.Kind = model.MessageKind(kind)
	message.Source = model.MessageSource(source)
	message.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Message{}, fmt.Errorf("parse message created_at: %w", err)
	}
	return message, nil
}

func scanSummary(scanner interface{ Scan(dest ...any) error }) (model.Summary, error) {
	var summary model.Summary
	var createdAt string
	err := scanner.Scan(
		&summary.ID,
		&summary.RoomID,
		&summary.FromMessageID,
		&summary.ToMessageID,
		&summary.Content,
		&createdAt,
	)
	if err != nil {
		return model.Summary{}, err
	}
	summary.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Summary{}, err
	}
	return summary, nil
}

func scanRelationship(scanner interface{ Scan(dest ...any) error }) (model.Relationship, error) {
	var relationship model.Relationship
	var updatedAt string
	err := scanner.Scan(
		&relationship.ID,
		&relationship.SourcePersonaID,
		&relationship.TargetPersonaID,
		&relationship.Affinity,
		&relationship.Hostility,
		&relationship.Respect,
		&relationship.FocusWeight,
		&relationship.Notes,
		&updatedAt,
	)
	if err != nil {
		return model.Relationship{}, fmt.Errorf("scan relationship: %w", err)
	}
	relationship.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Relationship{}, err
	}
	return relationship, nil
}

func scanProvider(scanner interface{ Scan(dest ...any) error }) (model.ProviderConfig, error) {
	var provider model.ProviderConfig
	var enabled int
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&provider.ID,
		&provider.Name,
		&provider.BaseURL,
		&provider.APIKey,
		&provider.DefaultModel,
		&provider.TimeoutMS,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.ProviderConfig{}, fmt.Errorf("scan provider: %w", err)
	}

	provider.Enabled = enabled != 0
	provider.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.ProviderConfig{}, err
	}
	provider.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.ProviderConfig{}, err
	}
	return provider, nil
}

func scanFaction(scanner interface{ Scan(dest ...any) error }) (model.Faction, error) {
	var faction model.Faction
	var createdAt string
	var updatedAt string
	err := scanner.Scan(
		&faction.ID,
		&faction.Name,
		&faction.Description,
		&faction.SharedValues,
		&faction.SharedStyle,
		&faction.DefaultBias,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.Faction{}, fmt.Errorf("scan faction: %w", err)
	}
	faction.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Faction{}, err
	}
	faction.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Faction{}, err
	}
	return faction, nil
}

func scanPersona(scanner interface{ Scan(dest ...any) error }) (model.Persona, error) {
	var persona model.Persona
	var enabled int
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&persona.ID,
		&persona.Name,
		&persona.Avatar,
		&persona.PublicIdentity,
		&persona.SpeakingStyle,
		&persona.Stance,
		&persona.Goal,
		&persona.Taboo,
		&persona.Aggression,
		&persona.ActivityLevel,
		&persona.FactionID,
		&persona.ProviderConfigID,
		&persona.ModelName,
		&persona.Temperature,
		&persona.MaxTokens,
		&persona.CooldownSeconds,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.Persona{}, fmt.Errorf("scan persona: %w", err)
	}
	persona.Enabled = enabled != 0
	persona.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Persona{}, err
	}
	persona.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Persona{}, err
	}
	return persona, nil
}
