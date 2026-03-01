package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type moduleConfig struct {
	BaseURL           string            `json:"base_url"`
	APIKey            string            `json:"api_key"`
	Models            []string          `json:"models"`
	DefaultModel      string            `json:"default_model"`
	RequestTimeoutSec int               `json:"request_timeout_sec"`
	ExtraHeaders      map[string]string `json:"extra_headers"`
}

type chatConversation struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	ModelID            string    `json:"model_id"`
	SystemPrompt       string    `json:"system_prompt"`
	LastMessagePreview string    `json:"last_message_preview"`
	MessageCount       int       `json:"message_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type chatMessage struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	ReasoningText  string    `json:"reasoning_text"`
	ProviderID     string    `json:"provider_id"`
	RouteModel     string    `json:"route_model"`
	CreatedAt      time.Time `json:"created_at"`
}

type conversationRecord struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	ModelID      string    `json:"model_id"`
	SystemPrompt string    `json:"system_prompt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type storeSnapshot struct {
	Conversations map[string]*conversationRecord `json:"conversations"`
	Messages      map[string][]*chatMessage      `json:"messages"`
	NextMessageID int64                          `json:"next_message_id"`
}

type persistentStore struct {
	mu   sync.RWMutex
	file string

	state storeSnapshot
}

func newPersistentStore(filePath string) (*persistentStore, error) {
	s := &persistentStore{file: filePath}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *persistentStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = storeSnapshot{
		Conversations: map[string]*conversationRecord{},
		Messages:      map[string][]*chatMessage{},
		NextMessageID: 1,
	}

	b, err := os.ReadFile(s.file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}

	var snap storeSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	if snap.Conversations == nil {
		snap.Conversations = map[string]*conversationRecord{}
	}
	if snap.Messages == nil {
		snap.Messages = map[string][]*chatMessage{}
	}
	if snap.NextMessageID <= 0 {
		var maxID int64
		for _, rows := range snap.Messages {
			for _, row := range rows {
				if row != nil && row.ID > maxID {
					maxID = row.ID
				}
			}
		}
		snap.NextMessageID = maxID + 1
	}
	s.state = snap
	return nil
}

func (s *persistentStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o755); err != nil {
		return err
	}

	tmpFile := s.file + ".tmp"
	buf, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpFile, buf, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.file)
}

func (s *persistentStore) listConversations() []chatConversation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]chatConversation, 0, len(s.state.Conversations))
	for _, row := range s.state.Conversations {
		if row == nil {
			continue
		}
		item := chatConversation{
			ID:           row.ID,
			Title:        row.Title,
			ModelID:      row.ModelID,
			SystemPrompt: row.SystemPrompt,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
		msgRows := s.state.Messages[row.ID]
		item.MessageCount = len(msgRows)
		if len(msgRows) > 0 {
			last := msgRows[len(msgRows)-1]
			if last != nil {
				item.LastMessagePreview = strings.TrimSpace(last.Content)
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (s *persistentStore) getConversation(id string) (chatConversation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.state.Conversations[id]
	if row == nil {
		return chatConversation{}, false
	}
	item := chatConversation{
		ID:           row.ID,
		Title:        row.Title,
		ModelID:      row.ModelID,
		SystemPrompt: row.SystemPrompt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	msgRows := s.state.Messages[row.ID]
	item.MessageCount = len(msgRows)
	if len(msgRows) > 0 {
		if last := msgRows[len(msgRows)-1]; last != nil {
			item.LastMessagePreview = strings.TrimSpace(last.Content)
		}
	}
	return item, true
}

func (s *persistentStore) listMessages(conversationID string) []chatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := s.state.Messages[conversationID]
	out := make([]chatMessage, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, *row)
	}
	return out
}

func (s *persistentStore) createConversation(title, modelID, systemPrompt string) (chatConversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if strings.TrimSpace(title) == "" {
		title = "新对话"
	}
	rec := &conversationRecord{
		ID:           newConversationID(),
		Title:        strings.TrimSpace(title),
		ModelID:      strings.TrimSpace(modelID),
		SystemPrompt: strings.TrimSpace(systemPrompt),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.state.Conversations[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		delete(s.state.Conversations, rec.ID)
		return chatConversation{}, err
	}
	return chatConversation{
		ID:           rec.ID,
		Title:        rec.Title,
		ModelID:      rec.ModelID,
		SystemPrompt: rec.SystemPrompt,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}, nil
}

func (s *persistentStore) deleteConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.state.Conversations[id]; !ok {
		return nil
	}
	delete(s.state.Conversations, id)
	delete(s.state.Messages, id)
	return s.saveLocked()
}

func (s *persistentStore) appendExchange(conversationID, modelID, title, userContent, assistantContent, reasoningText, providerID, routeModel string) (chatConversation, []chatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv := s.state.Conversations[conversationID]
	if conv == nil {
		return chatConversation{}, nil, fmt.Errorf("conversation not found")
	}
	now := time.Now().UTC()

	if strings.TrimSpace(title) != "" {
		conv.Title = strings.TrimSpace(title)
	}
	if strings.TrimSpace(modelID) != "" {
		conv.ModelID = strings.TrimSpace(modelID)
	}
	conv.UpdatedAt = now

	userMsg := &chatMessage{
		ID:             s.state.NextMessageID,
		ConversationID: conversationID,
		Role:           "user",
		Content:        strings.TrimSpace(userContent),
		CreatedAt:      now,
	}
	s.state.NextMessageID++

	assistantMsg := &chatMessage{
		ID:             s.state.NextMessageID,
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        strings.TrimSpace(assistantContent),
		ReasoningText:  strings.TrimSpace(reasoningText),
		ProviderID:     strings.TrimSpace(providerID),
		RouteModel:     strings.TrimSpace(routeModel),
		CreatedAt:      now,
	}
	s.state.NextMessageID++

	s.state.Messages[conversationID] = append(s.state.Messages[conversationID], userMsg, assistantMsg)
	if err := s.saveLocked(); err != nil {
		return chatConversation{}, nil, err
	}

	msgs := s.state.Messages[conversationID]
	outMsgs := make([]chatMessage, 0, len(msgs))
	for _, row := range msgs {
		if row == nil {
			continue
		}
		outMsgs = append(outMsgs, *row)
	}
	outConv := chatConversation{
		ID:                 conv.ID,
		Title:              conv.Title,
		ModelID:            conv.ModelID,
		SystemPrompt:       conv.SystemPrompt,
		LastMessagePreview: assistantMsg.Content,
		MessageCount:       len(outMsgs),
		CreatedAt:          conv.CreatedAt,
		UpdatedAt:          conv.UpdatedAt,
	}
	return outConv, outMsgs, nil
}

type moduleServer struct {
	cfg   moduleConfig
	store *persistentStore
	http  *http.Client
}

func main() {
	cfgPath := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_CONFIG_PATH"))
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	if cfg.RequestTimeoutSec <= 0 {
		cfg.RequestTimeoutSec = 90
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(cfg.DefaultModel) == "" && len(cfg.Models) > 0 {
		cfg.DefaultModel = strings.TrimSpace(cfg.Models[0])
	}
	if cfg.ExtraHeaders == nil {
		cfg.ExtraHeaders = map[string]string{}
	}

	dataDir := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_DATA_DIR"))
	if dataDir == "" {
		dataDir = "."
	}
	storeFile := filepath.Join(dataDir, "chatbox_store.json")
	pst, err := newPersistentStore(storeFile)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	srv := &moduleServer{
		cfg:   cfg,
		store: pst,
		http: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutSec) * time.Second,
		},
	}

	port := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_HTTP_PORT"))
	if port == "" {
		port = "8787"
	}
	addr := "127.0.0.1:" + port

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/chatbox/conversations", srv.handleConversations)
	mux.HandleFunc("/chatbox/conversations/", srv.handleConversationItem)
	mux.HandleFunc("/v1/chat/completions", srv.handleV1ChatCompletions)
	mux.HandleFunc("/v1/models", srv.handleV1Models)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]any{
				"module": "chatbox-persistent",
				"status": "ok",
			})
			return
		}
		http.NotFound(w, r)
	})

	log.Printf("chatbox-persistent listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func loadConfig(path string) (moduleConfig, error) {
	cfg := moduleConfig{
		BaseURL:           "https://api.openai.com/v1",
		Models:            []string{"gpt-4o-mini"},
		DefaultModel:      "gpt-4o-mini",
		RequestTimeoutSec: 90,
		ExtraHeaders:      map[string]string{},
	}
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (s *moduleServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"module": "chatbox-persistent",
	})
}

func (s *moduleServer) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := s.store.listConversations()
		writeJSON(w, http.StatusOK, map[string]any{"conversations": items})
	case http.MethodPost:
		var req struct {
			Title        string `json:"title"`
			Model        string `json:"model"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := decodeJSON(r.Body, 1<<20, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		model := strings.TrimSpace(req.Model)
		if model == "" {
			model = strings.TrimSpace(s.cfg.DefaultModel)
		}
		if model == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model is required"})
			return
		}
		item, err := s.store.createConversation(req.Title, model, req.SystemPrompt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"conversation": item,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *moduleServer) handleConversationItem(w http.ResponseWriter, r *http.Request) {
	conversationID, action, ok := parseConversationPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "":
		switch r.Method {
		case http.MethodGet:
			conv, ok := s.store.getConversation(conversationID)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "conversation not found"})
				return
			}
			msgs := s.store.listMessages(conversationID)
			writeJSON(w, http.StatusOK, map[string]any{
				"conversation": conv,
				"messages":     msgs,
			})
		case http.MethodDelete:
			if err := s.store.deleteConversation(conversationID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
	case "messages":
		s.handleConversationMessages(w, r, conversationID)
	default:
		http.NotFound(w, r)
	}
}

func (s *moduleServer) handleConversationMessages(w http.ResponseWriter, r *http.Request, conversationID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req struct {
		Content string         `json:"content"`
		Model   string         `json:"model"`
		Params  map[string]any `json:"params"`
	}
	if err := decodeJSON(r.Body, 4<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	userContent := strings.TrimSpace(req.Content)
	if userContent == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content is required"})
		return
	}

	conv, ok := s.store.getConversation(conversationID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "conversation not found"})
		return
	}

	modelID := strings.TrimSpace(req.Model)
	if modelID == "" {
		modelID = strings.TrimSpace(conv.ModelID)
	}
	if modelID == "" {
		modelID = strings.TrimSpace(s.cfg.DefaultModel)
	}
	if modelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model is required"})
		return
	}

	history := s.store.listMessages(conversationID)
	requestMessages := make([]map[string]any, 0, len(history)+2)
	if sp := strings.TrimSpace(conv.SystemPrompt); sp != "" {
		requestMessages = append(requestMessages, map[string]any{"role": "system", "content": sp})
	}
	for _, row := range history {
		role := strings.TrimSpace(row.Role)
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		requestMessages = append(requestMessages, map[string]any{"role": role, "content": content})
	}
	requestMessages = append(requestMessages, map[string]any{"role": "user", "content": userContent})

	status, respObj, assistantText, reasoningText, err := s.chatCompletion(r.Context(), modelID, requestMessages, req.Params)
	if err != nil {
		writeJSON(w, statusOr(status, http.StatusBadGateway), map[string]any{"error": err.Error()})
		return
	}
	if status >= 400 {
		writeJSON(w, status, map[string]any{
			"error":    nonEmpty(extractErrorText(respObj), "upstream request failed"),
			"response": respObj,
		})
		return
	}
	if strings.TrimSpace(assistantText) == "" {
		assistantText = "[返回成功，但未提取到文本，请查看原始响应]"
	}

	title := strings.TrimSpace(conv.Title)
	if len(history) == 0 || title == "" || title == "新对话" {
		title = titleFromInput(userContent)
	}

	updatedConv, msgs, err := s.store.appendExchange(
		conversationID,
		modelID,
		title,
		userContent,
		assistantText,
		reasoningText,
		"chatbox-persistent",
		modelID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"assistant_text":  assistantText,
		"reasoning_text":  reasoningText,
		"response":        respObj,
		"conversation":    updatedConv,
		"conversation_id": conversationID,
		"messages":        msgs,
	})
}

func (s *moduleServer) handleV1ChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	status, contentType, payload, err := s.callUpstream(r.Context(), http.MethodPost, "/chat/completions", body)
	if err != nil {
		writeJSON(w, statusOr(status, http.StatusBadGateway), map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	w.Header().Set("content-type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func (s *moduleServer) handleV1Models(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if len(s.cfg.Models) > 0 {
		now := time.Now().Unix()
		data := make([]map[string]any, 0, len(s.cfg.Models))
		for _, id := range s.cfg.Models {
			mid := strings.TrimSpace(id)
			if mid == "" {
				continue
			}
			data = append(data, map[string]any{
				"id":       mid,
				"object":   "model",
				"created":  now,
				"owned_by": "chatbox-persistent",
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   data,
		})
		return
	}

	status, contentType, payload, err := s.callUpstream(r.Context(), http.MethodGet, "/models", nil)
	if err != nil {
		writeJSON(w, statusOr(status, http.StatusBadGateway), map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	w.Header().Set("content-type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func (s *moduleServer) chatCompletion(ctx context.Context, model string, messages []map[string]any, params map[string]any) (int, any, string, string, error) {
	payload := map[string]any{
		"model":    strings.TrimSpace(model),
		"messages": messages,
		"stream":   false,
	}
	for k, v := range params {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		payload[key] = v
	}
	payload["stream"] = false
	payload["model"] = strings.TrimSpace(model)
	payload["messages"] = messages

	body, err := json.Marshal(payload)
	if err != nil {
		return http.StatusBadRequest, nil, "", "", fmt.Errorf("invalid payload")
	}
	status, _, respBytes, err := s.callUpstream(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return status, nil, "", "", err
	}

	respObj := decodeJSONAny(respBytes)
	assistant := extractAssistantText(respObj)
	reasoning := extractReasoningText(respObj)
	return status, respObj, assistant, reasoning, nil
}

func (s *moduleServer) callUpstream(ctx context.Context, method, reqPath string, body []byte) (status int, contentType string, payload []byte, err error) {
	endpoint, err := joinUpstreamURL(strings.TrimSpace(s.cfg.BaseURL), reqPath)
	if err != nil {
		return http.StatusBadGateway, "", nil, err
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return http.StatusBadGateway, "", nil, err
	}
	if len(body) > 0 {
		req.Header.Set("content-type", "application/json")
	}
	req.Header.Set("accept", "application/json")
	if strings.TrimSpace(s.cfg.APIKey) != "" {
		req.Header.Set("authorization", "Bearer "+strings.TrimSpace(s.cfg.APIKey))
	}
	for k, v := range s.cfg.ExtraHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return http.StatusBadGateway, "", nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return http.StatusBadGateway, "", nil, err
	}
	return resp.StatusCode, resp.Header.Get("content-type"), respBytes, nil
}

func parseConversationPath(rawPath string) (conversationID, action string, ok bool) {
	const prefix = "/chatbox/conversations/"
	if !strings.HasPrefix(rawPath, prefix) {
		return "", "", false
	}
	rest := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) > 2 {
		return "", "", false
	}
	id, err := url.PathUnescape(parts[0])
	if err != nil {
		id = parts[0]
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", false
	}
	if len(parts) == 2 {
		action = strings.TrimSpace(parts[1])
	}
	return id, action, true
}

func joinUpstreamURL(baseURL, reqPath string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("base_url is empty")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("base_url missing scheme")
	}

	p := strings.TrimSpace(reqPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	basePath := strings.TrimRight(u.Path, "/")
	if basePath == "" || basePath == "/" {
		u.Path = p
		return u.String(), nil
	}
	if p == basePath || strings.HasPrefix(p, basePath+"/") {
		u.Path = p
		return u.String(), nil
	}
	u.Path = basePath + p
	return u.String(), nil
}

func decodeJSON(r io.Reader, limit int64, out any) error {
	if limit <= 0 {
		limit = 1 << 20
	}
	dec := json.NewDecoder(io.LimitReader(r, limit))
	return dec.Decode(out)
}

func decodeJSONAny(raw []byte) any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(trimmed, &out); err == nil {
		return out
	}
	return string(trimmed)
}

func extractAssistantText(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok {
		return ""
	}
	if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
		if c0, ok := choices[0].(map[string]any); ok {
			if msg, ok := c0["message"].(map[string]any); ok {
				if txt := extractMessageContent(msg["content"]); txt != "" {
					return txt
				}
			}
			if delta, ok := c0["delta"].(map[string]any); ok {
				if txt := extractMessageContent(delta["content"]); txt != "" {
					return txt
				}
			}
		}
	}
	if txt, ok := root["output_text"].(string); ok {
		return strings.TrimSpace(txt)
	}
	return ""
}

func extractMessageContent(v any) string {
	switch vv := v.(type) {
	case string:
		return strings.TrimSpace(vv)
	case []any:
		parts := make([]string, 0, len(vv))
		for _, item := range vv {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if txt, ok := obj["text"].(string); ok && strings.TrimSpace(txt) != "" {
				parts = append(parts, strings.TrimSpace(txt))
				continue
			}
			if txt, ok := obj["content"].(string); ok && strings.TrimSpace(txt) != "" {
				parts = append(parts, strings.TrimSpace(txt))
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func extractReasoningText(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, 4)
	if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
		if c0, ok := choices[0].(map[string]any); ok {
			if msg, ok := c0["message"].(map[string]any); ok {
				appendNonEmpty(&parts, extractReasoningValue(msg["reasoning"]))
				appendNonEmpty(&parts, extractReasoningValue(msg["reasoning_content"]))
			}
			appendNonEmpty(&parts, extractReasoningValue(c0["reasoning"]))
			appendNonEmpty(&parts, extractReasoningValue(c0["reasoning_content"]))
		}
	}
	appendNonEmpty(&parts, extractReasoningValue(root["reasoning"]))
	appendNonEmpty(&parts, extractReasoningValue(root["reasoning_content"]))
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func extractReasoningValue(v any) string {
	switch vv := v.(type) {
	case string:
		return strings.TrimSpace(vv)
	case []any:
		parts := make([]string, 0, len(vv))
		for _, item := range vv {
			appendNonEmpty(&parts, extractReasoningValue(item))
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	case map[string]any:
		parts := make([]string, 0, 4)
		appendNonEmpty(&parts, extractReasoningValue(vv["text"]))
		appendNonEmpty(&parts, extractReasoningValue(vv["content"]))
		appendNonEmpty(&parts, extractReasoningValue(vv["summary"]))
		appendNonEmpty(&parts, extractReasoningValue(vv["reasoning"]))
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func extractErrorText(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok {
		if txt, ok := resp.(string); ok {
			return strings.TrimSpace(txt)
		}
		return ""
	}
	if errObj, ok := root["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
	}
	if msg, ok := root["error"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if msg, ok := root["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return ""
}

func titleFromInput(content string) string {
	flat := strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(content, "\n", " ")), " "))
	if flat == "" {
		return "新对话"
	}
	runes := []rune(flat)
	if len(runes) > 28 {
		return string(runes[:28]) + "..."
	}
	return flat
}

func newConversationID() string {
	base := strconv.FormatInt(time.Now().UnixNano(), 36)
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "chat_" + base
	}
	return fmt.Sprintf("chat_%s_%x", base, buf)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		_, _ = w.Write([]byte(`{"error":"encode failed"}`))
	}
}

func appendNonEmpty(parts *[]string, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	*parts = append(*parts, text)
}

func nonEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	return fallback
}

func statusOr(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
