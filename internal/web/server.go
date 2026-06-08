package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/model"
	"llm-bb/internal/scheduler"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
	"llm-bb/internal/update"
	"llm-bb/internal/version"
)

type Server struct {
	cfg          config.Config
	store        *store.Store
	scheduler    *scheduler.Scheduler
	hub          *stream.Hub
	logger       *log.Logger
	templates    *template.Template
	staticFS     fs.FS
	updater      *update.Client
	onRestart    func()
	inputLimiter *rateLimiter
}

type indexPageData struct {
	Rooms         []model.RoomOverview `json:"rooms"`
	TotalRooms    int                  `json:"totalRooms"`
	RunningRooms  int                  `json:"runningRooms"`
	TotalMessages int                  `json:"totalMessages"`
	TotalTokens   int                  `json:"totalTokens"`
}

type roomPageData struct {
	Room          model.Room             `json:"room"`
	Members       []model.RoomMemberView `json:"members"`
	Messages      []model.Message        `json:"messages"`
	LatestSummary *model.Summary         `json:"latestSummary"`
	TokensToday   int                    `json:"tokensToday"`
	MessageCount  int                    `json:"messageCount"`
	MemberCount   int                    `json:"memberCount"`
}

type adminPageData struct {
	Rooms         []model.RoomOverview             `json:"rooms"`
	Personas      []model.Persona                  `json:"personas"`
	Factions      []model.Faction                  `json:"factions"`
	Providers     []model.ProviderConfig           `json:"providers"`
	Relationships []model.Relationship             `json:"relationships"`
	AdminOpen     bool                             `json:"adminOpen"`
	Defaults      adminDefaults                    `json:"defaults"`
	RoomMembers   map[int64][]model.RoomMemberView `json:"roomMembers"`
	RunningRooms  int                              `json:"runningRooms"`
	TotalMessages int                              `json:"totalMessages"`
	TotalTokens   int                              `json:"totalTokens"`
}

type adminDefaults struct {
	Room         config.RoomDefaults         `json:"room"`
	Persona      config.PersonaDefaults      `json:"persona"`
	Provider     config.ProviderDefaults     `json:"provider"`
	Relationship config.RelationshipDefaults `json:"relationship"`
}

type appTemplateData struct {
	Title     string
	Bootstrap template.JS
}

type appBootstrap struct {
	Page  string `json:"page"`
	Title string `json:"title"`
	Data  any    `json:"data"`
}

func NewServer(cfg config.Config, store *store.Store, scheduler *scheduler.Scheduler, hub *stream.Hub, logger *log.Logger, onRestart func()) (*Server, error) {
	cfg = config.WithDefaults(cfg)
	tmpl, err := template.New("pages").ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static fs: %w", err)
	}

	return &Server{
		cfg:       cfg,
		store:     store,
		scheduler: scheduler,
		hub:       hub,
		logger:    logger,
		templates: tmpl,
		staticFS:  staticFS,
		updater:   updateClientFromConfig(cfg),
		onRestart: onRestart,
		inputLimiter: newRateLimiter(rateLimiterConfig{
			Limit:   cfg.PublicInput.RateLimit,
			Window:  cfg.PublicInputWindow(),
			MaxKeys: cfg.PublicInput.MaxKeys,
		}),
	}, nil
}

func updateClientFromConfig(cfg config.Config) *update.Client {
	client := update.NewClient()
	client.Owner = cfg.Update.Owner
	client.Repo = cfg.Update.Repo
	client.APITimeout = cfg.UpdateAPITimeout()
	client.DownloadTimeout = cfg.UpdateDownloadTimeout()
	client.MaxDownloadBytes = cfg.Update.MaxDownloadBytes
	return client
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.staticFS))))

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /rooms/{id}", s.handleRoom)

	mux.HandleFunc("GET /api/rooms/{id}/messages", s.handleRoomMessages)
	mux.HandleFunc("GET /api/rooms/{id}/events", s.handleRoomEvents)
	mux.HandleFunc("POST /api/rooms/{id}/input", s.handleRoomInput)

	mux.Handle("GET /admin", s.requireAdmin(http.HandlerFunc(s.handleAdmin)))
	mux.Handle("GET /api/admin/rooms", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomsList)))
	mux.Handle("POST /api/admin/rooms", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomCreate)))
	mux.Handle("PATCH /api/admin/rooms/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomPatch)))
	mux.Handle("POST /api/admin/rooms/{id}/pause", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomPause)))
	mux.Handle("POST /api/admin/rooms/{id}/resume", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomResume)))
	mux.Handle("POST /api/admin/rooms/{id}/tick", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomTick)))

	mux.Handle("GET /api/admin/personas", s.requireAdmin(http.HandlerFunc(s.handleAdminPersonasList)))
	mux.Handle("POST /api/admin/personas", s.requireAdmin(http.HandlerFunc(s.handleAdminPersonaCreate)))
	mux.Handle("PATCH /api/admin/personas/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminPersonaPatch)))

	mux.Handle("GET /api/admin/factions", s.requireAdmin(http.HandlerFunc(s.handleAdminFactionsList)))
	mux.Handle("POST /api/admin/factions", s.requireAdmin(http.HandlerFunc(s.handleAdminFactionCreate)))
	mux.Handle("PATCH /api/admin/factions/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminFactionPatch)))

	mux.Handle("GET /api/admin/providers", s.requireAdmin(http.HandlerFunc(s.handleAdminProvidersList)))
	mux.Handle("POST /api/admin/providers", s.requireAdmin(http.HandlerFunc(s.handleAdminProviderCreate)))
	mux.Handle("PATCH /api/admin/providers/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminProviderPatch)))

	mux.Handle("DELETE /api/admin/rooms/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminRoomDelete)))
	mux.Handle("DELETE /api/admin/personas/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminPersonaDelete)))
	mux.Handle("DELETE /api/admin/factions/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminFactionDelete)))
	mux.Handle("DELETE /api/admin/providers/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminProviderDelete)))

	mux.Handle("GET /api/admin/relationships", s.requireAdmin(http.HandlerFunc(s.handleAdminRelationshipsList)))
	mux.Handle("POST /api/admin/relationships", s.requireAdmin(http.HandlerFunc(s.handleAdminRelationshipUpsert)))
	mux.Handle("DELETE /api/admin/relationships/{id}", s.requireAdmin(http.HandlerFunc(s.handleAdminRelationshipDelete)))

	mux.Handle("GET /api/admin/version", s.requireAdmin(http.HandlerFunc(s.handleAdminVersion)))
	mux.Handle("GET /api/admin/update/check", s.requireAdmin(http.HandlerFunc(s.handleAdminUpdateCheck)))
	mux.Handle("POST /api/admin/update/apply", s.requireAdmin(http.HandlerFunc(s.handleAdminUpdateApply)))
	mux.Handle("POST /api/admin/update/restart", s.requireAdmin(http.HandlerFunc(s.handleAdminRestart)))

	return s.loggingMiddleware(mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.store.ListRooms(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.renderApp(w, "public", "llm-bb", "home", indexPageData{
		Rooms:         rooms,
		TotalRooms:    len(rooms),
		RunningRooms:  countRunningRooms(rooms),
		TotalMessages: sumMessages(rooms),
		TotalTokens:   sumTokens(rooms),
	})
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err)
		return
	}

	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err)
		return
	}
	members, err := s.store.ListRoomMembers(r.Context(), roomID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	messages, err := s.store.ListRoomMessages(r.Context(), roomID, s.cfg.RoomDefaults.RoomPageMessageLimit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	summary, err := s.store.GetLatestSummary(r.Context(), roomID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	tokensToday, err := s.store.RoomTokenUsageToday(r.Context(), roomID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.renderApp(w, "public", fmt.Sprintf("%s - llm-bb", room.Name), "room", roomPageData{
		Room:          room,
		Members:       members,
		Messages:      messages,
		LatestSummary: summary,
		TokensToday:   tokensToday,
		MessageCount:  len(messages),
		MemberCount:   len(members),
	})
}

func (s *Server) handleRoomMessages(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	messages, err := s.store.ListRoomMessages(r.Context(), roomID, s.cfg.RoomDefaults.RoomAPIMessagesLimit)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

func (s *Server) handleRoomEvents(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	// Disable write deadline for long-lived SSE connections.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	streamCh, cancel := s.hub.Subscribe(roomID)
	defer cancel()

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepAlive.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case message, ok := <-streamCh:
			if !ok {
				return
			}
			raw, _ := json.Marshal(message)
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", raw)
			flusher.Flush()
		}
	}
}

func (s *Server) handleRoomInput(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if !s.inputLimiter.Allow(clientIP(r)) {
		s.writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too many messages, please slow down"})
		return
	}

	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	content := strings.TrimSpace(stringValue(payload, "content"))
	if content == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content is required"})
		return
	}
	if len([]rune(content)) > s.cfg.PublicInput.MaxRunes {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content too long"})
		return
	}

	if _, err := s.store.GetRoom(r.Context(), roomID); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"error": "room not found"})
		return
	}

	message := &model.Message{
		RoomID:    roomID,
		Kind:      model.MessageKindUser,
		Content:   content,
		Source:    model.MessageSourceUser,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateMessage(r.Context(), message); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	s.hub.Publish(*message)
	s.scheduler.Nudge(roomID, s.cfg.UserInputNudgeDelay())
	s.writeJSON(w, http.StatusCreated, map[string]any{"message": message})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.store.ListRooms(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	personas, err := s.store.ListPersonas(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	factions, err := s.store.ListFactions(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	providers, err := s.store.ListProviders(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	relationships, err := s.store.ListAllRelationships(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	roomMembers := make(map[int64][]model.RoomMemberView, len(rooms))
	for _, room := range rooms {
		members, err := s.store.ListRoomMembers(r.Context(), room.ID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err)
			return
		}
		roomMembers[room.ID] = members
	}

	s.renderApp(w, "admin", "导演台 - llm-bb", "admin", adminPageData{
		Rooms:         rooms,
		Personas:      personas,
		Factions:      factions,
		Providers:     providers,
		Relationships: relationships,
		AdminOpen:     strings.TrimSpace(s.cfg.AdminPassword) == "",
		Defaults: adminDefaults{
			Room:         s.cfg.RoomDefaults,
			Persona:      s.cfg.PersonaDefaults,
			Provider:     s.cfg.ProviderDefaults,
			Relationship: s.cfg.RelationshipDefaults,
		},
		RoomMembers:   roomMembers,
		RunningRooms:  countRunningRooms(rooms),
		TotalMessages: sumMessages(rooms),
		TotalTokens:   sumTokens(rooms),
	})
}

func (s *Server) handleAdminRoomsList(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.store.ListRooms(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"rooms": rooms})
}

func (s *Server) handleAdminRoomCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	room := &model.Room{
		Name:                stringValue(payload, "name"),
		Topic:               stringValue(payload, "topic"),
		Description:         stringValue(payload, "description"),
		Status:              model.RoomStatus(stringValueFallback(payload, "status", s.cfg.RoomDefaults.Status)),
		Heat:                intValueFallback(payload, "heat", s.cfg.RoomDefaults.Heat),
		ConflictLevel:       intValueFallback(payload, "conflict_level", s.cfg.RoomDefaults.ConflictLevel),
		TickMinSeconds:      intValueFallback(payload, "tick_min_seconds", s.cfg.RoomDefaults.TickMinSeconds),
		TickMaxSeconds:      intValueFallback(payload, "tick_max_seconds", s.cfg.RoomDefaults.TickMaxSeconds),
		DailyTokenBudget:    intValueFallback(payload, "daily_token_budget", s.cfg.RoomDefaults.DailyTokenBudget),
		SummaryTriggerCount: intValueFallback(payload, "summary_trigger_count", s.cfg.RoomDefaults.SummaryTriggerCount),
		MessageRetention:    intValueFallback(payload, "message_retention_count", s.cfg.RoomDefaults.MessageRetention),
	}
	if strings.TrimSpace(room.Name) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "room name is required"})
		return
	}
	if err := s.store.CreateRoom(r.Context(), room); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	personaIDs := int64SliceValue(payload, "persona_ids")
	if len(personaIDs) > 0 {
		if err := s.store.ReplaceRoomMembers(r.Context(), room.ID, personaIDs); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}

	s.writeJSON(w, http.StatusCreated, map[string]any{"room": room})
}

func (s *Server) handleAdminRoomPatch(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	personaIDs := int64SliceValue(payload, "persona_ids")
	delete(payload, "persona_ids")

	if err := s.store.PatchRoom(r.Context(), roomID, payload); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if len(personaIDs) > 0 {
		if err := s.store.ReplaceRoomMembers(r.Context(), roomID, personaIDs); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminRoomPause(w http.ResponseWriter, r *http.Request) {
	s.handleRoomStatusChange(w, r, model.RoomStatusPaused)
}

func (s *Server) handleAdminRoomResume(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.SetRoomStatus(r.Context(), roomID, model.RoomStatusRunning); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.scheduler.Nudge(roomID, s.cfg.ManualNudgeDelay())
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRoomStatusChange(w http.ResponseWriter, r *http.Request, status model.RoomStatus) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.SetRoomStatus(r.Context(), roomID, status); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminRoomTick(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	message, err := s.scheduler.TriggerRoom(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, scheduler.ErrRoomBusy) {
			s.writeJSON(w, http.StatusConflict, map[string]any{"error": "room tick already running"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"message": message})
}

func (s *Server) handleAdminPersonasList(w http.ResponseWriter, r *http.Request) {
	personas, err := s.store.ListPersonas(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"personas": personas})
}

func (s *Server) handleAdminPersonaCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	persona := &model.Persona{
		Name:             stringValue(payload, "name"),
		Avatar:           stringValue(payload, "avatar"),
		PublicIdentity:   stringValue(payload, "public_identity"),
		SpeakingStyle:    stringValue(payload, "speaking_style"),
		Stance:           stringValue(payload, "stance"),
		Goal:             stringValue(payload, "goal"),
		Taboo:            stringValue(payload, "taboo"),
		Aggression:       intValueFallback(payload, "aggression", s.cfg.PersonaDefaults.Aggression),
		ActivityLevel:    intValueFallback(payload, "activity_level", s.cfg.PersonaDefaults.ActivityLevel),
		FactionID:        int64Value(payload, "faction_id"),
		ProviderConfigID: int64Value(payload, "provider_config_id"),
		ModelName:        stringValue(payload, "model_name"),
		Temperature:      floatValueFallback(payload, "temperature", s.cfg.PersonaDefaults.Temperature),
		MaxTokens:        intValueFallback(payload, "max_tokens", s.cfg.PersonaDefaults.MaxTokens),
		CooldownSeconds:  intValueFallback(payload, "cooldown_seconds", s.cfg.PersonaDefaults.CooldownSeconds),
		Enabled:          boolValueFallback(payload, "enabled", s.cfg.PersonaDefaults.Enabled),
	}
	if strings.TrimSpace(persona.Name) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "persona name is required"})
		return
	}
	if err := s.store.CreatePersona(r.Context(), persona); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"persona": persona})
}

func (s *Server) handleAdminPersonaPatch(w http.ResponseWriter, r *http.Request) {
	personaID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.PatchPersona(r.Context(), personaID, payload); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminFactionsList(w http.ResponseWriter, r *http.Request) {
	factions, err := s.store.ListFactions(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"factions": factions})
}

func (s *Server) handleAdminFactionCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	faction := &model.Faction{
		Name:         stringValue(payload, "name"),
		Description:  stringValue(payload, "description"),
		SharedValues: stringValue(payload, "shared_values"),
		SharedStyle:  stringValue(payload, "shared_style"),
		DefaultBias:  stringValue(payload, "default_bias"),
	}
	if strings.TrimSpace(faction.Name) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "faction name is required"})
		return
	}
	if err := s.store.CreateFaction(r.Context(), faction); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"faction": faction})
}

func (s *Server) handleAdminFactionPatch(w http.ResponseWriter, r *http.Request) {
	factionID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.PatchFaction(r.Context(), factionID, payload); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminProvidersList(w http.ResponseWriter, r *http.Request) {
	providers, err := s.store.ListProviders(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (s *Server) handleAdminProviderCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	provider := &model.ProviderConfig{
		Name:         stringValue(payload, "name"),
		BaseURL:      stringValue(payload, "base_url"),
		APIKey:       stringValue(payload, "api_key"),
		DefaultModel: stringValue(payload, "default_model"),
		TimeoutMS:    intValueFallback(payload, "timeout_ms", s.cfg.ProviderDefaults.TimeoutMS),
		Enabled:      boolValueFallback(payload, "enabled", s.cfg.ProviderDefaults.Enabled),
	}
	if strings.TrimSpace(provider.Name) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provider name is required"})
		return
	}
	if err := s.store.CreateProvider(r.Context(), provider); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"provider": provider})
}

func (s *Server) handleAdminProviderPatch(w http.ResponseWriter, r *http.Request) {
	providerID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.PatchProvider(r.Context(), providerID, payload); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminRoomDelete(w http.ResponseWriter, r *http.Request) {
	roomID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.DeleteRoom(r.Context(), roomID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminPersonaDelete(w http.ResponseWriter, r *http.Request) {
	personaID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.DeletePersona(r.Context(), personaID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminFactionDelete(w http.ResponseWriter, r *http.Request) {
	factionID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.DeleteFaction(r.Context(), factionID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminProviderDelete(w http.ResponseWriter, r *http.Request) {
	providerID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.DeleteProvider(r.Context(), providerID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminRelationshipsList(w http.ResponseWriter, r *http.Request) {
	relationships, err := s.store.ListAllRelationships(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"relationships": relationships})
}

func (s *Server) handleAdminRelationshipUpsert(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	rel := &model.Relationship{
		SourcePersonaID: int64Value(payload, "source_persona_id"),
		TargetPersonaID: int64Value(payload, "target_persona_id"),
		Affinity:        intValueFallback(payload, "affinity", s.cfg.RelationshipDefaults.Affinity),
		Hostility:       intValueFallback(payload, "hostility", s.cfg.RelationshipDefaults.Hostility),
		Respect:         intValueFallback(payload, "respect", s.cfg.RelationshipDefaults.Respect),
		FocusWeight:     intValueFallback(payload, "focus_weight", s.cfg.RelationshipDefaults.FocusWeight),
		Notes:           stringValue(payload, "notes"),
	}
	if rel.SourcePersonaID <= 0 || rel.TargetPersonaID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source and target persona IDs are required"})
		return
	}
	if rel.SourcePersonaID == rel.TargetPersonaID {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source and target must be different"})
		return
	}
	if err := s.store.UpsertRelationship(r.Context(), rel); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"relationship": rel})
}

func (s *Server) handleAdminRelationshipDelete(w http.ResponseWriter, r *http.Request) {
	relID, err := parsePathInt64(r, "id")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.DeleteRelationship(r.Context(), relID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"version": version.Get()})
}

func (s *Server) handleAdminUpdateCheck(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		channel = s.defaultUpdateChannel()
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, err := s.updater.Check(ctx, channel, version.Commit, version.Version)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"current": version.Get(),
		"result":  result,
	})
}

func (s *Server) handleAdminUpdateApply(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	channel := strings.TrimSpace(stringValue(payload, "channel"))
	if channel == "" {
		channel = s.defaultUpdateChannel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := s.updater.Apply(ctx, channel); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"channel":         channel,
		"requiresRestart": true,
	})
}

func (s *Server) defaultUpdateChannel() string {
	if version.Channel == update.ChannelDev || version.Channel == update.ChannelStable {
		return version.Channel
	}
	if s.cfg.Update.DefaultChannel == update.ChannelDev || s.cfg.Update.DefaultChannel == update.ChannelStable {
		return s.cfg.Update.DefaultChannel
	}
	return update.ChannelStable
}

func (s *Server) handleAdminRestart(w http.ResponseWriter, r *http.Request) {
	if s.onRestart == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "restart not supported in this process"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "restarting": true})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.onRestart()
	}()
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	if strings.TrimSpace(s.cfg.AdminPassword) == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !constantTimeEqual(user, s.cfg.AdminUser) || !constantTimeEqual(pass, s.cfg.AdminPassword) {
			w.Header().Set("WWW-Authenticate", `Basic realm="llm-bb-admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func constantTimeEqual(got, want string) bool {
	if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return false
	}
	return len(got) == len(want)
}

func (s *Server) renderApp(w http.ResponseWriter, tmplName, title, page string, data any) {
	bootstrap, err := json.Marshal(appBootstrap{
		Page:  page,
		Title: title,
		Data:  data,
	})
	if err != nil {
		s.logger.Printf("marshal bootstrap: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderTemplate(w, tmplName, appTemplateData{
		Title:     title,
		Bootstrap: template.JS(bootstrap),
	})
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Printf("render template %s: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Printf("write json: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, err error) {
	type errorPageData struct {
		Status int
		Error  string
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.templates.ExecuteTemplate(w, "error", errorPageData{Status: status, Error: err.Error()})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	value := strings.TrimSpace(r.PathValue(key))
	if value == "" {
		return 0, errors.New("missing route parameter")
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", value)
	}
	return id, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

type rateLimiterConfig struct {
	Limit   int
	Window  time.Duration
	MaxKeys int
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	maxKeys int
	keys    map[string]rateBucket
}

type rateBucket struct {
	start time.Time
	count int
}

func newRateLimiter(cfg rateLimiterConfig) *rateLimiter {
	if cfg.Limit <= 0 {
		cfg.Limit = 1
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Second
	}
	if cfg.MaxKeys <= 0 {
		cfg.MaxKeys = 1024
	}
	return &rateLimiter{
		limit:   cfg.Limit,
		window:  cfg.Window,
		maxKeys: cfg.MaxKeys,
		keys:    make(map[string]rateBucket),
	}
}

func (l *rateLimiter) Allow(key string) bool {
	if key == "" {
		key = "unknown"
	}

	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.keys) >= l.maxKeys {
		for existing, bucket := range l.keys {
			if now.Sub(bucket.start) >= l.window {
				delete(l.keys, existing)
			}
		}
	}

	bucket := l.keys[key]
	if bucket.start.IsZero() || now.Sub(bucket.start) >= l.window {
		l.keys[key] = rateBucket{start: now, count: 1}
		return true
	}

	if bucket.count >= l.limit {
		return false
	}

	bucket.count++
	l.keys[key] = bucket
	return true
}

func readPayload(r *http.Request) (map[string]any, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&payload); err != nil {
			return nil, err
		}
		return payload, nil
	}

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	payload := make(map[string]any, len(r.Form))
	for key, values := range r.Form {
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		clone := make([]string, len(values))
		copy(clone, values)
		payload[key] = clone
	}
	return payload, nil
}

func stringValue(payload map[string]any, key string) string {
	return strings.TrimSpace(fmt.Sprintf("%v", payload[key]))
}

func stringValueFallback(payload map[string]any, key, fallback string) string {
	value := stringValue(payload, key)
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func intValueFallback(payload map[string]any, key string, fallback int) int {
	value := payload[key]
	switch v := value.(type) {
	case nil:
		return fallback
	case float64:
		return int(v)
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func int64Value(payload map[string]any, key string) int64 {
	value := payload[key]
	switch v := value.(type) {
	case float64:
		return int64(v)
	case string:
		if strings.TrimSpace(v) == "" {
			return 0
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func floatValueFallback(payload map[string]any, key string, fallback float64) float64 {
	value := payload[key]
	switch v := value.(type) {
	case nil:
		return fallback
	case float64:
		return v
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func boolValueFallback(payload map[string]any, key string, fallback bool) bool {
	value := payload[key]
	switch v := value.(type) {
	case nil:
		return fallback
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		case "":
			return fallback
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func int64SliceValue(payload map[string]any, key string) []int64 {
	value, ok := payload[key]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			switch value := item.(type) {
			case float64:
				out = append(out, int64(value))
			case string:
				if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
					out = append(out, parsed)
				}
			}
		}
		return out
	case []string:
		return parseCSVInt64(strings.Join(v, ","))
	case string:
		return parseCSVInt64(v)
	default:
		return nil
	}
}

func parseCSVInt64(raw string) []int64 {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err == nil && value > 0 {
			out = append(out, value)
		}
	}
	return out
}

func countRunningRooms(rooms []model.RoomOverview) int {
	count := 0
	for _, room := range rooms {
		if room.Status == model.RoomStatusRunning {
			count++
		}
	}
	return count
}

func sumMessages(rooms []model.RoomOverview) int {
	total := 0
	for _, room := range rooms {
		total += room.MessageCount
	}
	return total
}

func sumTokens(rooms []model.RoomOverview) int {
	total := 0
	for _, room := range rooms {
		total += room.TokensToday
	}
	return total
}
