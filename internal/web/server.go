package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-bb/internal/config"
	"llm-bb/internal/model"
	"llm-bb/internal/scheduler"
	"llm-bb/internal/store"
	"llm-bb/internal/stream"
)

type Server struct {
	cfg       config.Config
	store     *store.Store
	scheduler *scheduler.Scheduler
	hub       *stream.Hub
	logger    *log.Logger
	templates *template.Template
	staticFS  fs.FS
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
	AdminOpen     bool                             `json:"adminOpen"`
	RoomMembers   map[int64][]model.RoomMemberView `json:"roomMembers"`
	RunningRooms  int                              `json:"runningRooms"`
	TotalMessages int                              `json:"totalMessages"`
	TotalTokens   int                              `json:"totalTokens"`
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

func NewServer(cfg config.Config, store *store.Store, scheduler *scheduler.Scheduler, hub *stream.Hub, logger *log.Logger) (*Server, error) {
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
	}, nil
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

	return s.loggingMiddleware(mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.store.ListRooms(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.renderApp(w, "llm-bb", "home", indexPageData{
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
	messages, err := s.store.ListRoomMessages(r.Context(), roomID, 80)
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

	s.renderApp(w, fmt.Sprintf("%s - llm-bb", room.Name), "room", roomPageData{
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

	messages, err := s.store.ListRoomMessages(r.Context(), roomID, 100)
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
	if len([]rune(content)) > 280 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content too long"})
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
	s.scheduler.Nudge(roomID, 1500*time.Millisecond)
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
	roomMembers := make(map[int64][]model.RoomMemberView, len(rooms))
	for _, room := range rooms {
		members, err := s.store.ListRoomMembers(r.Context(), room.ID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err)
			return
		}
		roomMembers[room.ID] = members
	}

	s.renderApp(w, "导演台 - llm-bb", "admin", adminPageData{
		Rooms:         rooms,
		Personas:      personas,
		Factions:      factions,
		Providers:     providers,
		AdminOpen:     strings.TrimSpace(s.cfg.AdminPassword) == "",
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
		Status:              model.RoomStatus(stringValueFallback(payload, "status", string(model.RoomStatusRunning))),
		Heat:                intValueFallback(payload, "heat", 60),
		ConflictLevel:       intValueFallback(payload, "conflict_level", 55),
		TickMinSeconds:      intValueFallback(payload, "tick_min_seconds", 25),
		TickMaxSeconds:      intValueFallback(payload, "tick_max_seconds", 55),
		DailyTokenBudget:    intValueFallback(payload, "daily_token_budget", 40000),
		SummaryTriggerCount: intValueFallback(payload, "summary_trigger_count", 24),
		MessageRetention:    intValueFallback(payload, "message_retention_count", 120),
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
	s.scheduler.Nudge(roomID, time.Second)
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
		Aggression:       intValueFallback(payload, "aggression", 50),
		ActivityLevel:    intValueFallback(payload, "activity_level", 50),
		FactionID:        int64Value(payload, "faction_id"),
		ProviderConfigID: int64Value(payload, "provider_config_id"),
		ModelName:        stringValue(payload, "model_name"),
		Temperature:      floatValueFallback(payload, "temperature", 0.9),
		MaxTokens:        intValueFallback(payload, "max_tokens", 220),
		CooldownSeconds:  intValueFallback(payload, "cooldown_seconds", 120),
		Enabled:          boolValueFallback(payload, "enabled", true),
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
		TimeoutMS:    intValueFallback(payload, "timeout_ms", 20000),
		Enabled:      boolValueFallback(payload, "enabled", true),
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

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	if strings.TrimSpace(s.cfg.AdminPassword) == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.cfg.AdminUser || pass != s.cfg.AdminPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="llm-bb-admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) renderApp(w http.ResponseWriter, title, page string, data any) {
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

	s.renderTemplate(w, "app", appTemplateData{
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
