package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

const (
	serviceName = "LightBridge Mail Service"
	version     = "0.1.0"
)

type Config struct {
	Host                 string
	Port                 string
	APIKey               string
	Driver               string
	DriverBaseURL        string
	DriverAPIKey         string
	DataPath             string
	RequestTimeout       time.Duration
	VerificationCacheTTL time.Duration
}

func LoadConfig() Config {
	return Config{
		Host:                 envOrDefault("LBMS_HOST", "0.0.0.0"),
		Port:                 envOrDefault("LBMS_PORT", "8091"),
		APIKey:               strings.TrimSpace(os.Getenv("LBMS_API_KEY")),
		Driver:               envOrDefault("LBMS_DRIVER", "outlook_email_plus"),
		DriverBaseURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("LBMS_DRIVER_BASE_URL")), "/"),
		DriverAPIKey:         strings.TrimSpace(os.Getenv("LBMS_DRIVER_API_KEY")),
		DataPath:             envOrDefault("LBMS_DATA_PATH", "data/lbms-store.json"),
		RequestTimeout:       envDurationSeconds("LBMS_REQUEST_TIMEOUT_SECONDS", 10),
		VerificationCacheTTL: envDurationSeconds("LBMS_VERIFICATION_CACHE_SECONDS", 30),
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDurationSeconds(key string, fallback int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallback) * time.Second
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

type Mailbox struct {
	ID              string    `json:"id"`
	EmailAddress    string    `json:"email_address"`
	NormalizedEmail string    `json:"normalized_email"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type OAuthBinding struct {
	ID                     string    `json:"id"`
	MailboxID              string    `json:"mailbox_id"`
	LightBridgeAccountID   int64     `json:"lightbridge_account_id"`
	LightBridgePlatform    string    `json:"lightbridge_platform"`
	LightBridgeAccountType string    `json:"lightbridge_account_type"`
	LightBridgeAccountName string    `json:"lightbridge_account_name,omitempty"`
	Status                 string    `json:"status"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type Store struct {
	mu                sync.RWMutex
	dataPath          string
	mailboxesByID     map[string]*Mailbox
	mailboxIDByEmail  map[string]string
	bindingsByAccount map[int64]*OAuthBinding
	bindingsByMailbox map[string]map[int64]*OAuthBinding
	verificationCache map[string]cachedVerification
}

type persistedStore struct {
	Version   int             `json:"version"`
	SavedAt   time.Time       `json:"saved_at"`
	Mailboxes []*Mailbox      `json:"mailboxes"`
	Bindings  []*OAuthBinding `json:"bindings"`
}

type cachedVerification struct {
	Code       string
	ReceivedAt string
	ExpiresAt  time.Time
}

func NewStore(dataPath string) (*Store, error) {
	s := &Store{
		dataPath:          strings.TrimSpace(dataPath),
		mailboxesByID:     map[string]*Mailbox{},
		mailboxIDByEmail:  map[string]string{},
		bindingsByAccount: map[int64]*OAuthBinding{},
		bindingsByMailbox: map[string]map[int64]*OAuthBinding{},
		verificationCache: map[string]cachedVerification{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	if s.dataPath == "" {
		return nil
	}
	file, err := os.Open(s.dataPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open_store: %w", err)
	}
	defer file.Close()

	var snapshot persistedStore
	if err := json.NewDecoder(file).Decode(&snapshot); err != nil {
		return fmt.Errorf("decode_store: %w", err)
	}
	for _, mailbox := range snapshot.Mailboxes {
		if mailbox == nil || mailbox.ID == "" || mailbox.NormalizedEmail == "" {
			continue
		}
		copy := *mailbox
		s.mailboxesByID[copy.ID] = &copy
		s.mailboxIDByEmail[copy.NormalizedEmail] = copy.ID
	}
	for _, binding := range snapshot.Bindings {
		if binding == nil || binding.LightBridgeAccountID <= 0 || binding.MailboxID == "" {
			continue
		}
		if _, ok := s.mailboxesByID[binding.MailboxID]; !ok {
			continue
		}
		copy := *binding
		s.bindingsByAccount[copy.LightBridgeAccountID] = &copy
		if s.bindingsByMailbox[copy.MailboxID] == nil {
			s.bindingsByMailbox[copy.MailboxID] = map[int64]*OAuthBinding{}
		}
		s.bindingsByMailbox[copy.MailboxID][copy.LightBridgeAccountID] = &copy
	}
	return nil
}

func (s *Store) saveLocked() error {
	if s.dataPath == "" {
		return nil
	}
	mailboxes := make([]*Mailbox, 0, len(s.mailboxesByID))
	for _, mailbox := range s.mailboxesByID {
		copy := *mailbox
		mailboxes = append(mailboxes, &copy)
	}
	sort.Slice(mailboxes, func(i, j int) bool { return mailboxes[i].ID < mailboxes[j].ID })

	bindings := make([]*OAuthBinding, 0, len(s.bindingsByAccount))
	for _, binding := range s.bindingsByAccount {
		copy := *binding
		bindings = append(bindings, &copy)
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].ID < bindings[j].ID })

	snapshot := persistedStore{
		Version:   1,
		SavedAt:   time.Now().UTC(),
		Mailboxes: mailboxes,
		Bindings:  bindings,
	}

	if err := os.MkdirAll(filepath.Dir(s.dataPath), 0o700); err != nil {
		return fmt.Errorf("create_store_dir: %w", err)
	}
	tmp := s.dataPath + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open_store_tmp: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode_store: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync_store: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close_store: %w", err)
	}
	if err := os.Rename(tmp, s.dataPath); err != nil {
		return fmt.Errorf("replace_store: %w", err)
	}
	return nil
}

func (s *Store) LinkOrCreate(req LinkOrCreateRequest) (*Mailbox, *OAuthBinding, error) {
	normalized := normalizeEmail(req.EmailAddress)
	if normalized == "" || !strings.Contains(normalized, "@") {
		return nil, nil, errors.New("invalid_email_address")
	}
	if req.LightBridgeAccountID <= 0 {
		return nil, nil, errors.New("invalid_lightbridge_account_id")
	}
	if strings.TrimSpace(req.LightBridgePlatform) == "" {
		return nil, nil, errors.New("invalid_lightbridge_platform")
	}
	if strings.TrimSpace(req.LightBridgeAccountType) == "" {
		req.LightBridgeAccountType = "oauth"
	}

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	mailboxID, ok := s.mailboxIDByEmail[normalized]
	var mailbox *Mailbox
	if ok {
		mailbox = s.mailboxesByID[mailboxID]
		mailbox.UpdatedAt = now
	} else {
		mailbox = &Mailbox{
			ID:              newID("mbx"),
			EmailAddress:    strings.TrimSpace(req.EmailAddress),
			NormalizedEmail: normalized,
			Status:          "active",
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		s.mailboxesByID[mailbox.ID] = mailbox
		s.mailboxIDByEmail[normalized] = mailbox.ID
	}

	if existing := s.bindingsByAccount[req.LightBridgeAccountID]; existing != nil {
		if existing.MailboxID != mailbox.ID {
			delete(s.bindingsByMailbox[existing.MailboxID], req.LightBridgeAccountID)
		}
		existing.MailboxID = mailbox.ID
		existing.LightBridgePlatform = strings.TrimSpace(req.LightBridgePlatform)
		existing.LightBridgeAccountType = strings.TrimSpace(req.LightBridgeAccountType)
		existing.LightBridgeAccountName = strings.TrimSpace(req.LightBridgeAccountName)
		existing.Status = "active"
		existing.UpdatedAt = now
		if s.bindingsByMailbox[mailbox.ID] == nil {
			s.bindingsByMailbox[mailbox.ID] = map[int64]*OAuthBinding{}
		}
		s.bindingsByMailbox[mailbox.ID][req.LightBridgeAccountID] = existing
		if err := s.saveLocked(); err != nil {
			return nil, nil, err
		}
		return cloneMailbox(mailbox), cloneBinding(existing), nil
	}

	binding := &OAuthBinding{
		ID:                     newID("bind"),
		MailboxID:              mailbox.ID,
		LightBridgeAccountID:   req.LightBridgeAccountID,
		LightBridgePlatform:    strings.TrimSpace(req.LightBridgePlatform),
		LightBridgeAccountType: strings.TrimSpace(req.LightBridgeAccountType),
		LightBridgeAccountName: strings.TrimSpace(req.LightBridgeAccountName),
		Status:                 "active",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	s.bindingsByAccount[req.LightBridgeAccountID] = binding
	if s.bindingsByMailbox[mailbox.ID] == nil {
		s.bindingsByMailbox[mailbox.ID] = map[int64]*OAuthBinding{}
	}
	s.bindingsByMailbox[mailbox.ID][req.LightBridgeAccountID] = binding
	if err := s.saveLocked(); err != nil {
		return nil, nil, err
	}
	return cloneMailbox(mailbox), cloneBinding(binding), nil
}

func (s *Store) ListMailboxes() []*MailboxSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*MailboxSummary, 0, len(s.mailboxesByID))
	for _, mailbox := range s.mailboxesByID {
		bindingCount := 0
		for _, binding := range s.bindingsByMailbox[mailbox.ID] {
			if binding.Status == "active" {
				bindingCount++
			}
		}
		items = append(items, &MailboxSummary{
			ID:           mailbox.ID,
			EmailAddress: mailbox.EmailAddress,
			Status:       mailbox.Status,
			BindingCount: bindingCount,
			CreatedAt:    mailbox.CreatedAt,
			UpdatedAt:    mailbox.UpdatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items
}

func (s *Store) BindingByAccount(accountID int64) (*OAuthBinding, *Mailbox, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	binding := s.bindingsByAccount[accountID]
	if binding == nil || binding.Status != "active" {
		return nil, nil, false
	}
	mailbox := s.mailboxesByID[binding.MailboxID]
	if mailbox == nil {
		return nil, nil, false
	}
	return cloneBinding(binding), cloneMailbox(mailbox), true
}

func (s *Store) BindingsByMailbox(mailboxID string) (*Mailbox, []*OAuthBinding, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mailbox := s.mailboxesByID[mailboxID]
	if mailbox == nil {
		return nil, nil, false
	}
	bindings := make([]*OAuthBinding, 0, len(s.bindingsByMailbox[mailboxID]))
	for _, binding := range s.bindingsByMailbox[mailboxID] {
		if binding.Status == "active" {
			bindings = append(bindings, cloneBinding(binding))
		}
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].LightBridgeAccountID < bindings[j].LightBridgeAccountID })
	return cloneMailbox(mailbox), bindings, true
}

func (s *Store) UnlinkAccount(accountID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	binding := s.bindingsByAccount[accountID]
	if binding == nil {
		return false
	}
	delete(s.bindingsByAccount, accountID)
	delete(s.bindingsByMailbox[binding.MailboxID], accountID)
	if err := s.saveLocked(); err != nil {
		log.Printf("%s failed to persist unlink: %v", serviceName, err)
	}
	return true
}

func (s *Store) GetCachedVerification(key string) (cachedVerification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cached, ok := s.verificationCache[key]
	if !ok || time.Now().UTC().After(cached.ExpiresAt) {
		return cachedVerification{}, false
	}
	return cached, true
}

func (s *Store) SetCachedVerification(key, code, receivedAt string, ttl time.Duration) {
	if ttl <= 0 || code == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verificationCache[key] = cachedVerification{
		Code:       code,
		ReceivedAt: receivedAt,
		ExpiresAt:  time.Now().UTC().Add(ttl),
	}
}

func cloneMailbox(mailbox *Mailbox) *Mailbox {
	if mailbox == nil {
		return nil
	}
	copy := *mailbox
	return &copy
}

func cloneBinding(binding *OAuthBinding) *OAuthBinding {
	if binding == nil {
		return nil
	}
	copy := *binding
	return &copy
}

type LinkOrCreateRequest struct {
	EmailAddress           string `json:"email_address"`
	LightBridgeAccountID   int64  `json:"lightbridge_account_id"`
	LightBridgePlatform    string `json:"lightbridge_platform"`
	LightBridgeAccountType string `json:"lightbridge_account_type"`
	LightBridgeAccountName string `json:"lightbridge_account_name"`
}

type MailboxSummary struct {
	ID           string    `json:"id"`
	EmailAddress string    `json:"email_address"`
	Status       string    `json:"status"`
	BindingCount int       `json:"binding_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DriverClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewDriverClient(cfg Config) *DriverClient {
	return &DriverClient{
		baseURL: cfg.DriverBaseURL,
		apiKey:  cfg.DriverAPIKey,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

func (d *DriverClient) Configured() bool {
	return d.baseURL != "" && d.apiKey != ""
}

func (d *DriverClient) Health(ctx context.Context) string {
	if !d.Configured() {
		return "not_configured"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL+"/api/external/health", nil)
	if err != nil {
		return "error"
	}
	req.Header.Set("X-API-Key", d.apiKey)
	resp, err := d.client.Do(req)
	if err != nil {
		return "error"
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok"
	}
	return "error"
}

func (d *DriverClient) VerificationCode(ctx context.Context, email string, sinceMinutes, codeLength int) (map[string]any, error) {
	if !d.Configured() {
		return nil, errors.New("mail_driver_not_configured")
	}
	endpoint, err := url.Parse(d.baseURL + "/api/external/verification-code")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("email", email)
	if sinceMinutes > 0 {
		query.Set("since_minutes", strconv.Itoa(sinceMinutes))
	}
	if codeLength > 0 {
		query.Set("code_length", strconv.Itoa(codeLength))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", d.apiKey)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mail_driver_http_%d", resp.StatusCode)
	}
	return payload, nil
}

type Server struct {
	cfg    Config
	store  *Store
	driver *DriverClient
}

func main() {
	cfg := LoadConfig()
	store, err := NewStore(cfg.DataPath)
	if err != nil {
		log.Fatalf("%s failed to load store: %v", serviceName, err)
	}
	server := &Server{
		cfg:    cfg,
		store:  store,
		driver: NewDriverClient(cfg),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mail/v1/health", server.handleHealth)
	mux.HandleFunc("/mail/v1/mailboxes", server.withAuth(server.handleMailboxes))
	mux.HandleFunc("/mail/v1/mailboxes/link-or-create", server.withAuth(server.handleLinkOrCreate))
	mux.HandleFunc("/mail/v1/accounts/", server.withAuth(server.handleAccountRoute))
	mux.HandleFunc("/mail/v1/mailboxes/", server.withAuth(server.handleMailboxRoute))

	addr := cfg.Host + ":" + cfg.Port
	log.Printf("%s starting on %s", serviceName, addr)
	log.Printf("%s store path: %s", serviceName, cfg.DataPath)
	if err := http.ListenAndServe(addr, requestIDMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"service":       serviceName,
			"status":        "ok",
			"driver_status": s.driver.Health(ctx),
			"store": map[string]any{
				"type": "json_file",
				"path": s.cfg.DataPath,
			},
			"version": version,
		},
	})
}

func (s *Server) handleMailboxes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"mailboxes": s.store.ListMailboxes(),
		},
	})
}

func (s *Server) handleLinkOrCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req LinkOrCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	mailbox, binding, err := s.store.LinkOrCreate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "unable to link mailbox")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"mailbox_id":    mailbox.ID,
			"lbms_link":     "lbms://mailbox/" + mailbox.ID,
			"email_address": mailbox.EmailAddress,
			"binding_id":    binding.ID,
		},
	})
}

func (s *Server) handleAccountRoute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/mail/v1/accounts/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}
	accountID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || accountID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_account_id", "invalid account id")
		return
	}

	switch parts[1] {
	case "verification-code":
		s.handleAccountVerificationCode(w, r, accountID)
	case "mailbox-link":
		s.handleAccountMailboxLink(w, r, accountID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	}
}

func (s *Server) handleAccountVerificationCode(w http.ResponseWriter, r *http.Request, accountID int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	_, mailbox, ok := s.store.BindingByAccount(accountID)
	if !ok {
		writeError(w, http.StatusNotFound, "mailbox_binding_not_found", "mailbox binding not found")
		return
	}
	sinceMinutes := intQuery(r, "since_minutes", 10)
	codeLength := intQuery(r, "code_length", 0)
	cacheKey := fmt.Sprintf("account:%d:%d:%d", accountID, sinceMinutes, codeLength)
	if cached, ok := s.store.GetCachedVerification(cacheKey); ok {
		writeVerificationCode(w, mailbox, cached.Code, cached.ReceivedAt, true)
		return
	}

	payload, err := s.driver.VerificationCode(r.Context(), mailbox.EmailAddress, sinceMinutes, codeLength)
	if err != nil {
		writeError(w, http.StatusBadGateway, "mail_service_driver_error", "LightBridge Mail Service cannot fetch verification code")
		return
	}
	code, receivedAt := extractCode(payload)
	if code != "" {
		s.store.SetCachedVerification(cacheKey, code, receivedAt, s.cfg.VerificationCacheTTL)
	}
	writeVerificationCode(w, mailbox, code, receivedAt, false)
}

func (s *Server) handleAccountMailboxLink(w http.ResponseWriter, r *http.Request, accountID int64) {
	switch r.Method {
	case http.MethodGet:
		binding, mailbox, ok := s.store.BindingByAccount(accountID)
		if !ok {
			writeError(w, http.StatusNotFound, "mailbox_binding_not_found", "mailbox binding not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"lbms_link":     "lbms://mailbox/" + mailbox.ID,
				"mailbox_id":    mailbox.ID,
				"email_address": mailbox.EmailAddress,
				"binding":       binding,
			},
		})
	case http.MethodDelete:
		if !s.store.UnlinkAccount(accountID) {
			writeError(w, http.StatusNotFound, "mailbox_binding_not_found", "mailbox binding not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"lightbridge_account_id": accountID,
				"unlinked":               true,
			},
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (s *Server) handleMailboxRoute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/mail/v1/mailboxes/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "bindings" {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	mailbox, bindings, ok := s.store.BindingsByMailbox(parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "mailbox_not_found", "mailbox not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"mailbox_id":    mailbox.ID,
			"email_address": mailbox.EmailAddress,
			"bindings":      bindings,
		},
	})
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.APIKey == "" {
			writeError(w, http.StatusServiceUnavailable, "lbms_api_key_not_configured", "LightBridge Mail Service API key is not configured")
			return
		}
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			token = strings.TrimSpace(r.Header.Get("X-API-Key"))
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid LightBridge Mail Service API key")
			return
		}
		next(w, r)
	}
}

func bearerToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newID("req")
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func writeVerificationCode(w http.ResponseWriter, mailbox *Mailbox, code, receivedAt string, cached bool) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"mailbox_id":    mailbox.ID,
			"email_address": mailbox.EmailAddress,
			"code":          code,
			"received_at":   receivedAt,
			"confidence":    "high",
			"cached":        cached,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func intQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func newID(prefix string) string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}

func extractCode(payload map[string]any) (string, string) {
	for _, key := range []string{"code", "verification_code"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), stringField(payload, "received_at")
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		for _, key := range []string{"code", "verification_code"} {
			if value, ok := data[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value), stringField(data, "received_at")
			}
		}
	}
	return "", ""
}

func stringField(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}
