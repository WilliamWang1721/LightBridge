package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	codexOAuthClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	codexOAuthTokenURL     = "https://auth.openai.com/oauth/token"

	codexDeviceUserCodeURL              = "https://auth.openai.com/api/accounts/deviceauth/usercode"
	codexDeviceTokenURL                 = "https://auth.openai.com/api/accounts/deviceauth/token"
	codexDeviceVerificationURL          = "https://auth.openai.com/codex/device"
	codexDeviceTokenExchangeRedirectURI = "https://auth.openai.com/deviceauth/callback"

	codexDeviceTimeout = 15 * time.Minute
)

type credentials struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	LastRefresh  string `json:"last_refresh"`
	Email        string `json:"email"`
	Type         string `json:"type"`
	Expired      string `json:"expired"`
}

func (c *credentials) expiryTime() (time.Time, bool) {
	if c == nil || strings.TrimSpace(c.Expired) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, c.Expired)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (s *server) loadCredentials() error {
	s.credsMu.Lock()
	defer s.credsMu.Unlock()

	b, err := os.ReadFile(s.credsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var c credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("invalid credentials.json: %w", err)
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return fmt.Errorf("credentials.json missing access_token")
	}
	s.creds = &c
	return nil
}

func (s *server) saveCredentials(c *credentials) error {
	if c == nil {
		return errors.New("nil credentials")
	}
	c.Type = "codex"
	if strings.TrimSpace(c.LastRefresh) == "" {
		c.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.credsPath), 0o700); err != nil {
		return err
	}
	// Best-effort atomic write.
	tmp := s.credsPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.credsPath)
}

func (s *server) getAccessToken() (token, accountID string, ok bool) {
	s.credsMu.Lock()
	defer s.credsMu.Unlock()
	if s.creds == nil {
		return "", "", false
	}
	if strings.TrimSpace(s.creds.AccessToken) == "" {
		return "", "", false
	}
	return s.creds.AccessToken, strings.TrimSpace(s.creds.AccountID), true
}

func (s *server) maybeRefreshCredentials(ctx context.Context) error {
	s.credsMu.Lock()
	creds := s.creds
	near := time.Duration(s.cfg.NearExpiryMinutes) * time.Minute
	s.credsMu.Unlock()

	if creds == nil {
		return errNoCredentials
	}
	exp, ok := creds.expiryTime()
	if !ok {
		return nil
	}
	if time.Until(exp) > near {
		return nil
	}
	_, err := s.refreshTokens(ctx)
	return err
}

func (s *server) refreshOnce(ctx context.Context) bool {
	_, err := s.refreshTokens(ctx)
	return err == nil
}

func (s *server) refreshTokens(ctx context.Context) (*credentials, error) {
	s.credsMu.Lock()
	current := s.creds
	s.credsMu.Unlock()
	if current == nil || strings.TrimSpace(current.RefreshToken) == "" {
		return nil, errNoCredentials
	}
	next, err := refreshWithToken(ctx, s.httpc, current.RefreshToken)
	if err != nil {
		return nil, err
	}
	s.credsMu.Lock()
	s.creds = next
	s.credsMu.Unlock()
	if err := s.saveCredentials(next); err != nil {
		return nil, err
	}
	return next, nil
}

func refreshWithToken(ctx context.Context, httpc *http.Client, refreshToken string) (*credentials, error) {
	form := url.Values{
		"client_id":     {codexOAuthClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {strings.TrimSpace(refreshToken)},
		"scope":         {"openid profile email"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &tokenResp); err != nil {
		return nil, err
	}

	claims, _ := parseJWTClaims(tokenResp.IDToken)
	email := strings.TrimSpace(claims.Email)
	accountID := strings.TrimSpace(claims.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(claims.Sub)
	}

	expiredAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return &credentials{
		IDToken:      tokenResp.IDToken,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		Email:        email,
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
		Type:         "codex",
		Expired:      expiredAt.Format(time.RFC3339),
	}, nil
}

func exchangeAuthCode(ctx context.Context, httpc *http.Client, authCode, redirectURI, codeVerifier string) (*credentials, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {codexOAuthClientID},
		"code":          {strings.TrimSpace(authCode)},
		"redirect_uri":  {strings.TrimSpace(redirectURI)},
		"code_verifier": {strings.TrimSpace(codeVerifier)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code exchange failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &tokenResp); err != nil {
		return nil, err
	}

	claims, _ := parseJWTClaims(tokenResp.IDToken)
	email := strings.TrimSpace(claims.Email)
	accountID := strings.TrimSpace(claims.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(claims.Sub)
	}

	expiredAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return &credentials{
		IDToken:      tokenResp.IDToken,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		Email:        email,
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
		Type:         "codex",
		Expired:      expiredAt.Format(time.RFC3339),
	}, nil
}

const codexOAuthFlowTimeout = 10 * time.Minute

type pkceCodes struct {
	CodeVerifier  string `json:"code_verifier"`
	CodeChallenge string `json:"code_challenge"`
}

func generatePKCECodes() (*pkceCodes, error) {
	// Generate code verifier: 43-128 characters, URL-safe.
	buf := make([]byte, 96)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return &pkceCodes{CodeVerifier: verifier, CodeChallenge: challenge}, nil
}

type oauthFlow struct {
	StartedAt    time.Time `json:"started_at"`
	Status       string    `json:"status"` // pending|authorized|error|timeout
	AuthURL      string    `json:"auth_url,omitempty"`
	State        string    `json:"state"`
	RedirectURI  string    `json:"redirect_uri"`
	CodeVerifier string    `json:"-"`
	Error        string    `json:"error,omitempty"`
}

type oauthStartRequest struct {
	RedirectURI string `json:"redirect_uri"`
}

func (s *server) handleAuthOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	var req oauthStartRequest
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
	_ = r.Body.Close()

	redirectURI := strings.TrimSpace(req.RedirectURI)
	if redirectURI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "redirect_uri is required"})
		return
	}
	if u, err := url.Parse(redirectURI); err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "redirect_uri must be an absolute URL"})
		return
	}

	s.oauthMu.Lock()
	if s.oauth != nil && s.oauth.Status == "pending" && time.Since(s.oauth.StartedAt) < codexOAuthFlowTimeout {
		if strings.TrimSpace(s.oauth.RedirectURI) == redirectURI {
			copy := *s.oauth
			s.oauthMu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "oauth": copy})
			return
		}
	}
	s.oauthMu.Unlock()

	pkce, err := generatePKCECodes()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	state := newUUID()
	params := url.Values{
		"client_id":                  {codexOAuthClientID},
		"response_type":              {"code"},
		"redirect_uri":               {redirectURI},
		"scope":                      {"openid email profile offline_access"},
		"state":                      {state},
		"code_challenge":             {pkce.CodeChallenge},
		"code_challenge_method":      {"S256"},
		"prompt":                     {"login"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
	}
	authURL := codexOAuthAuthorizeURL + "?" + params.Encode()

	flow := &oauthFlow{
		StartedAt:    time.Now().UTC(),
		Status:       "pending",
		AuthURL:      authURL,
		State:        state,
		RedirectURI:  redirectURI,
		CodeVerifier: pkce.CodeVerifier,
	}

	s.oauthMu.Lock()
	s.oauth = flow
	s.oauthMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "oauth": flow})
}

type oauthExchangeRequest struct {
	Code        string `json:"code"`
	State       string `json:"state"`
	CallbackURL string `json:"callback_url"`
}

func (s *server) handleAuthOAuthExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	var req oauthExchangeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
		return
	}

	if strings.TrimSpace(req.CallbackURL) != "" && strings.TrimSpace(req.Code) == "" {
		u, err := url.Parse(strings.TrimSpace(req.CallbackURL))
		if err != nil || u == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid callback_url"})
			return
		}
		q := u.Query()
		if errStr := strings.TrimSpace(q.Get("error")); errStr != "" {
			desc := strings.TrimSpace(q.Get("error_description"))
			msg := errStr
			if desc != "" {
				msg += ": " + desc
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": msg})
			return
		}
		req.Code = q.Get("code")
		req.State = q.Get("state")
	}

	code := strings.TrimSpace(req.Code)
	state := strings.TrimSpace(req.State)
	if code == "" || state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "code/state is required"})
		return
	}

	s.oauthMu.Lock()
	flow := s.oauth
	s.oauthMu.Unlock()
	if flow == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "no active oauth flow; call /auth/oauth/start first"})
		return
	}
	if strings.TrimSpace(flow.State) == "" || flow.State != state {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "state mismatch"})
		return
	}
	if strings.TrimSpace(flow.RedirectURI) == "" || strings.TrimSpace(flow.CodeVerifier) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "oauth flow missing redirect_uri/code_verifier"})
		return
	}

	ctx := r.Context()
	creds, err := exchangeAuthCode(ctx, s.httpc, code, flow.RedirectURI, flow.CodeVerifier)
	if err != nil {
		s.oauthMu.Lock()
		if s.oauth != nil && s.oauth.State == state {
			s.oauth.Status = "error"
			s.oauth.Error = err.Error()
		}
		s.oauthMu.Unlock()
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	s.credsMu.Lock()
	s.creds = creds
	s.credsMu.Unlock()
	_ = s.saveCredentials(creds)

	s.oauthMu.Lock()
	if s.oauth != nil && s.oauth.State == state {
		s.oauth.Status = "authorized"
		s.oauth.Error = ""
		// Never return verifier in responses.
		s.oauth.CodeVerifier = ""
	}
	flowCopy := *s.oauth
	s.oauthMu.Unlock()

	s.credsMu.Lock()
	cCopy := *s.creds
	cCopy.AccessToken = maskToken(cCopy.AccessToken)
	cCopy.RefreshToken = maskToken(cCopy.RefreshToken)
	cCopy.IDToken = ""
	s.credsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "credentials": cCopy, "oauth": flowCopy})
}

type importRequest struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	AccountID    string `json:"account_id"`
	Email        string `json:"email"`
	Expired      string `json:"expired"`
	LastRefresh  string `json:"last_refresh"`
	Token        string `json:"token"`
}

func (s *server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, 6<<20))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "failed to read body"})
		return
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "body must be valid JSON"})
		return
	}

	m, ok := decoded.(map[string]any)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "body must be a JSON object"})
		return
	}
	if v, ok := m["auth_json"].(map[string]any); ok && v != nil {
		m = v
	}

	pick := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				if s, ok := v.(string); ok {
					if strings.TrimSpace(s) != "" {
						return strings.TrimSpace(s)
					}
				}
			}
		}
		return ""
	}

	req := importRequest{
		RefreshToken: pick("refresh_token", "refreshToken"),
		AccessToken:  pick("access_token", "accessToken"),
		IDToken:      pick("id_token", "idToken"),
		AccountID:    pick("account_id", "accountId"),
		Email:        pick("email"),
		Expired:      pick("expired", "expire", "expires_at", "expiresAt"),
		LastRefresh:  pick("last_refresh", "lastRefresh"),
		Token:        pick("token"),
	}
	if req.RefreshToken == "" && req.Token != "" {
		req.RefreshToken = req.Token
	}
	if req.AccessToken == "" && req.Token != "" && req.RefreshToken == "" {
		req.AccessToken = req.Token
	}

	if req.RefreshToken == "" && req.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing refresh_token or access_token"})
		return
	}

	ctx := r.Context()
	var creds *credentials

	if req.AccessToken == "" && req.RefreshToken != "" {
		// Prefer a refresh-token import: obtain a fresh access token immediately.
		creds, err = refreshWithToken(ctx, s.httpc, req.RefreshToken)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	} else {
		creds = &credentials{
			IDToken:      strings.TrimSpace(req.IDToken),
			AccessToken:  strings.TrimSpace(req.AccessToken),
			RefreshToken: strings.TrimSpace(req.RefreshToken),
			AccountID:    strings.TrimSpace(req.AccountID),
			Email:        strings.TrimSpace(req.Email),
			LastRefresh:  strings.TrimSpace(req.LastRefresh),
			Expired:      strings.TrimSpace(req.Expired),
		}
		if creds.AccountID == "" || creds.Email == "" {
			claims, _ := parseJWTClaims(creds.IDToken)
			if claims != nil {
				if creds.Email == "" {
					creds.Email = strings.TrimSpace(claims.Email)
				}
				if creds.AccountID == "" {
					creds.AccountID = strings.TrimSpace(claims.AccountID)
				}
			}
		}
		if strings.TrimSpace(creds.AccessToken) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "access_token is required"})
			return
		}
	}

	s.credsMu.Lock()
	s.creds = creds
	s.credsMu.Unlock()
	_ = s.saveCredentials(creds)

	// Clear pending flows; importing is an explicit override.
	s.flowMu.Lock()
	if s.flow != nil && s.flow.Status == "pending" {
		s.flow.Status = "timeout"
		s.flow.Error = "credentials imported"
	}
	s.flowMu.Unlock()

	s.oauthMu.Lock()
	if s.oauth != nil && s.oauth.Status == "pending" {
		s.oauth.Status = "timeout"
		s.oauth.Error = "credentials imported"
		s.oauth.CodeVerifier = ""
	}
	s.oauthMu.Unlock()

	masked := *creds
	masked.AccessToken = maskToken(masked.AccessToken)
	masked.RefreshToken = maskToken(masked.RefreshToken)
	masked.IDToken = ""

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "credentials": masked})
}

type deviceFlow struct {
	StartedAt       time.Time `json:"started_at"`
	Status          string    `json:"status"` // pending|authorized|error|timeout
	VerificationURL string    `json:"verification_url"`
	DeviceAuthID    string    `json:"device_auth_id"`
	UserCode        string    `json:"user_code"`
	IntervalSeconds int       `json:"interval_seconds"`
	Error           string    `json:"error,omitempty"`
}

func (s *server) handleAuthDeviceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	s.flowMu.Lock()
	if s.flow != nil && s.flow.Status == "pending" && time.Since(s.flow.StartedAt) < codexDeviceTimeout {
		flowCopy := *s.flow
		s.flowMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "flow": flowCopy})
		return
	}
	s.flowMu.Unlock()

	ctx := r.Context()
	flow, err := requestDeviceFlow(ctx, s.httpc)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	s.flowMu.Lock()
	s.flow = flow
	s.flowMu.Unlock()

	go s.runDeviceFlow(flow)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "flow": flow})
}

func (s *server) runDeviceFlow(flow *deviceFlow) {
	ctx, cancel := context.WithTimeout(context.Background(), codexDeviceTimeout)
	defer cancel()

	authCode, verifier, err := pollDeviceToken(ctx, s.httpc, flow.DeviceAuthID, flow.UserCode, time.Duration(flow.IntervalSeconds)*time.Second)
	if err != nil {
		s.flowMu.Lock()
		if s.flow != nil && s.flow.DeviceAuthID == flow.DeviceAuthID {
			s.flow.Status = "error"
			s.flow.Error = err.Error()
		}
		s.flowMu.Unlock()
		return
	}

	creds, err := exchangeAuthCode(ctx, s.httpc, authCode, codexDeviceTokenExchangeRedirectURI, verifier)
	if err != nil {
		s.flowMu.Lock()
		if s.flow != nil && s.flow.DeviceAuthID == flow.DeviceAuthID {
			s.flow.Status = "error"
			s.flow.Error = err.Error()
		}
		s.flowMu.Unlock()
		return
	}

	s.credsMu.Lock()
	s.creds = creds
	s.credsMu.Unlock()
	_ = s.saveCredentials(creds)

	s.flowMu.Lock()
	if s.flow != nil && s.flow.DeviceAuthID == flow.DeviceAuthID {
		s.flow.Status = "authorized"
		s.flow.Error = ""
	}
	s.flowMu.Unlock()
}

func (s *server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	s.credsMu.Lock()
	var credsCopy *credentials
	if s.creds != nil {
		c := *s.creds
		c.AccessToken = maskToken(c.AccessToken)
		c.RefreshToken = maskToken(c.RefreshToken)
		c.IDToken = ""
		credsCopy = &c
	}
	s.credsMu.Unlock()

	s.flowMu.Lock()
	var flowCopy *deviceFlow
	if s.flow != nil {
		f := *s.flow
		flowCopy = &f
	}
	s.flowMu.Unlock()

	s.oauthMu.Lock()
	var oauthCopy *oauthFlow
	if s.oauth != nil {
		f := *s.oauth
		oauthCopy = &f
	}
	s.oauthMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"credentials": credsCopy,
		"flow":        flowCopy,
		"oauth":       oauthCopy,
	})
}

type deviceUserCodeResponse struct {
	DeviceAuthID string          `json:"device_auth_id"`
	UserCode     string          `json:"user_code"`
	UserCodeAlt  string          `json:"usercode"`
	Interval     json.RawMessage `json:"interval"`
}

func requestDeviceFlow(ctx context.Context, httpc *http.Client) (*deviceFlow, error) {
	body, _ := json.Marshal(map[string]string{"client_id": codexOAuthClientID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexDeviceUserCodeURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device user code failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var parsed deviceUserCodeResponse
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	deviceAuthID := strings.TrimSpace(parsed.DeviceAuthID)
	userCode := strings.TrimSpace(parsed.UserCode)
	if userCode == "" {
		userCode = strings.TrimSpace(parsed.UserCodeAlt)
	}
	if deviceAuthID == "" || userCode == "" {
		return nil, errors.New("device auth did not return required fields")
	}

	interval := parseDeviceInterval(parsed.Interval)

	return &deviceFlow{
		StartedAt:       time.Now().UTC(),
		Status:          "pending",
		VerificationURL: codexDeviceVerificationURL,
		DeviceAuthID:    deviceAuthID,
		UserCode:        userCode,
		IntervalSeconds: int(interval.Seconds()),
	}, nil
}

type deviceTokenResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
	CodeChallenge     string `json:"code_challenge"`
}

func pollDeviceToken(ctx context.Context, httpc *http.Client, deviceAuthID, userCode string, interval time.Duration) (authCode, codeVerifier string, _ error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(codexDeviceTimeout)

	for {
		if time.Now().After(deadline) {
			return "", "", errors.New("device authentication timed out")
		}
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		default:
		}

		body, _ := json.Marshal(map[string]string{
			"device_auth_id": strings.TrimSpace(deviceAuthID),
			"user_code":      strings.TrimSpace(userCode),
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexDeviceTokenURL, bytes.NewReader(body))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("content-type", "application/json")
		req.Header.Set("accept", "application/json")

		resp, err := httpc.Do(req)
		if err != nil {
			return "", "", err
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var parsed deviceTokenResponse
			if err := json.Unmarshal(b, &parsed); err != nil {
				return "", "", err
			}
			authCode = strings.TrimSpace(parsed.AuthorizationCode)
			codeVerifier = strings.TrimSpace(parsed.CodeVerifier)
			if authCode == "" || codeVerifier == "" {
				return "", "", errors.New("device token response missing authorization_code/code_verifier")
			}
			return authCode, codeVerifier, nil
		case http.StatusForbidden, http.StatusNotFound:
			select {
			case <-ctx.Done():
				return "", "", ctx.Err()
			case <-time.After(interval):
				continue
			}
		default:
			return "", "", fmt.Errorf("device token poll failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}
	}
}

func parseDeviceInterval(raw json.RawMessage) time.Duration {
	if len(raw) == 0 {
		return 5 * time.Second
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if i, convErr := strconv.Atoi(strings.TrimSpace(asString)); convErr == nil && i > 0 {
			return time.Duration(i) * time.Second
		}
	}
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil && asInt > 0 {
		return time.Duration(asInt) * time.Second
	}
	return 5 * time.Second
}

type jwtClaims struct {
	Email     string
	Sub       string
	AccountID string
}

func parseJWTClaims(idToken string) (*jwtClaims, error) {
	parts := strings.Split(strings.TrimSpace(idToken), ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt")
	}
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, err
	}
	out := &jwtClaims{}
	if v, ok := m["email"].(string); ok {
		out.Email = v
	}
	if v, ok := m["sub"].(string); ok {
		out.Sub = v
	}
	if v, ok := m["https://api.openai.com/auth"].(map[string]any); ok {
		if id, ok := v["chatgpt_account_id"].(string); ok {
			out.AccountID = id
		}
	}
	return out, nil
}

func base64URLDecode(data string) ([]byte, error) {
	// Add padding.
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}
	return base64.URLEncoding.DecodeString(data)
}

func maskToken(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(tok))
	h := base64.RawURLEncoding.EncodeToString(sum[:])
	if len(tok) <= 12 {
		return "sha256:" + h[:10]
	}
	return tok[:6] + "…" + tok[len(tok)-4:] + " (sha256:" + h[:10] + ")"
}
