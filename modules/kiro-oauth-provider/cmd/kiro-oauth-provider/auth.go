package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	authMethodSocial  = "social"
	authMethodBuilder = "builder-id"
	authMethodImport  = "import"
	oauthFlowTimeout  = 10 * time.Minute
	deviceAuthTimeout = 10 * time.Minute
	defaultUserAgent  = "KiroIDE"
)

var kiroScopes = []string{
	"codewhisperer:completions",
	"codewhisperer:analysis",
	"codewhisperer:conversations",
}

type oauthFlow struct {
	Status       string    `json:"status"`
	AuthURL      string    `json:"auth_url,omitempty"`
	State        string    `json:"state,omitempty"`
	RedirectURI  string    `json:"redirect_uri,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	CodeVerifier string    `json:"code_verifier,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	Error        string    `json:"error,omitempty"`
}

type deviceFlow struct {
	Status                  string    `json:"status"`
	Region                  string    `json:"region"`
	ClientID                string    `json:"client_id,omitempty"`
	ClientSecret            string    `json:"client_secret,omitempty"`
	DeviceCode              string    `json:"device_code,omitempty"`
	UserCode                string    `json:"user_code,omitempty"`
	VerificationURI         string    `json:"verification_uri,omitempty"`
	VerificationURIComplete string    `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int       `json:"expires_in,omitempty"`
	Interval                int       `json:"interval,omitempty"`
	StartedAt               time.Time `json:"started_at"`
	Error                   string    `json:"error,omitempty"`
}

type oauthStartRequest struct {
	RedirectURI string `json:"redirect_uri"`
	Provider    string `json:"provider"`
	Region      string `json:"region"`
}

type oauthExchangeRequest struct {
	Code        string `json:"code"`
	State       string `json:"state"`
	CallbackURL string `json:"callback_url"`
	DisplayName string `json:"display_name"`
	GroupName   string `json:"group_name"`
	Region      string `json:"region"`
}

type deviceStartRequest struct {
	Region            string `json:"region"`
	BuilderIDStartURL string `json:"builder_id_start_url"`
	DisplayName       string `json:"display_name"`
	GroupName         string `json:"group_name"`
}

type importRequest struct {
	AccountID      string `json:"account_id"`
	DisplayName    string `json:"display_name"`
	GroupName      string `json:"group_name"`
	Enabled        *bool  `json:"enabled"`
	AuthMethod     string `json:"auth_method"`
	SocialProvider string `json:"social_provider"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	ProfileARN     string `json:"profile_arn"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	Region         string `json:"region"`
	IDCRegion      string `json:"idc_region"`
	ExpiresAt      string `json:"expires_at"`
}

type socialTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ProfileARN   string `json:"profileArn"`
	ExpiresIn    int    `json:"expiresIn"`
}

type idcTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

func normalizeSocialProvider(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "github":
		return "Github"
	default:
		return "Google"
	}
}

func extractString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return strings.TrimSpace(t)
				}
			}
		}
	}
	return ""
}

func boolPtrValue(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func (s *server) handleAuthOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req oauthStartRequest
	_ = decodeJSONBody(r, 1<<20, &req)

	redirectURI := strings.TrimSpace(req.RedirectURI)
	if redirectURI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing redirect_uri"})
		return
	}
	provider := normalizeSocialProvider(req.Provider)
	state := newRandomBase64URL(16)
	verifier := newRandomBase64URL(32)
	challenge := sha256Base64URL(verifier)

	params := url.Values{}
	params.Set("idp", provider)
	params.Set("redirect_uri", redirectURI)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", state)
	params.Set("prompt", "select_account")

	authURL := strings.TrimRight(s.cfg.AuthServiceEndpoint, "/") + "/login?" + params.Encode()
	flow := &oauthFlow{
		Status:       "pending",
		AuthURL:      authURL,
		State:        state,
		RedirectURI:  redirectURI,
		Provider:     provider,
		CodeVerifier: verifier,
		StartedAt:    time.Now().UTC(),
	}

	s.oauthMu.Lock()
	s.oauth = flow
	s.oauthMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "oauth": flow})
}

func normalizeCallbackInput(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if (strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"")) ||
		(strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'")) ||
		(strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`")) {
		text = strings.TrimSpace(text[1 : len(text)-1])
	}
	text = strings.ReplaceAll(text, "&amp;", "&")
	for i := 0; i < 2; i++ {
		decoded, err := url.QueryUnescape(text)
		if err != nil || decoded == text {
			break
		}
		text = strings.TrimSpace(decoded)
	}
	re := regexp.MustCompile(`(?i)(https?://[^\s"'<>]+|(?:localhost|127\.0\.0\.1|\[::1\]):\d+[^\s"'<>]*)`)
	if m := re.FindString(text); strings.TrimSpace(m) != "" {
		text = strings.TrimSpace(m)
	}
	if !strings.HasPrefix(strings.ToLower(text), "http://") && !strings.HasPrefix(strings.ToLower(text), "https://") {
		if strings.HasPrefix(strings.ToLower(text), "localhost:") || strings.HasPrefix(strings.ToLower(text), "127.0.0.1:") || strings.HasPrefix(strings.ToLower(text), "[::1]:") {
			text = "http://" + text
		}
	}
	return text
}

func parseCallbackURL(raw string) (normalized, code, state, errMsg string) {
	normalized = normalizeCallbackInput(raw)
	if normalized == "" {
		return "", "", "", ""
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return normalized, "", "", "invalid callback url"
	}
	errCode := strings.TrimSpace(u.Query().Get("error"))
	if errCode != "" {
		errDesc := strings.TrimSpace(u.Query().Get("error_description"))
		if errDesc != "" {
			return normalized, "", "", errCode + ": " + errDesc
		}
		return normalized, "", "", errCode
	}
	code = strings.TrimSpace(u.Query().Get("code"))
	state = strings.TrimSpace(u.Query().Get("state"))
	if (code == "" || state == "") && strings.TrimSpace(u.Fragment) != "" {
		f := strings.TrimPrefix(strings.TrimSpace(u.Fragment), "#")
		fp, _ := url.ParseQuery(f)
		if code == "" {
			code = strings.TrimSpace(fp.Get("code"))
		}
		if state == "" {
			state = strings.TrimSpace(fp.Get("state"))
		}
	}
	return normalized, code, state, ""
}

func (s *server) handleAuthOAuthExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req oauthExchangeRequest
	if err := decodeJSONBody(r, 2<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid body"})
		return
	}
	code := strings.TrimSpace(req.Code)
	state := strings.TrimSpace(req.State)
	if strings.TrimSpace(req.CallbackURL) != "" {
		normalized, parsedCode, parsedState, parseErr := parseCallbackURL(req.CallbackURL)
		if parseErr != "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": parseErr, "callback_url": normalized})
			return
		}
		if code == "" {
			code = parsedCode
		}
		if state == "" {
			state = parsedState
		}
	}
	if code == "" || state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing code/state"})
		return
	}

	s.oauthMu.Lock()
	flow := s.oauth
	s.oauthMu.Unlock()
	if flow == nil || strings.TrimSpace(flow.State) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "no active oauth flow; call /auth/oauth/start first"})
		return
	}
	if time.Since(flow.StartedAt) > oauthFlowTimeout {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "oauth flow expired"})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(flow.State), state) {
		s.oauthMu.Lock()
		if s.oauth != nil {
			s.oauth.Status = "error"
			s.oauth.Error = "state mismatch"
		}
		s.oauthMu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "state mismatch"})
		return
	}

	payload := map[string]any{
		"code":          code,
		"code_verifier": flow.CodeVerifier,
		"redirect_uri":  flow.RedirectURI,
	}
	b, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(s.cfg.AuthServiceEndpoint, "/") + "/oauth/token"
	httpReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(b))
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("user-agent", defaultUserAgent)

	resp, err := s.httpc.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": summarizeHTTPError(resp.StatusCode, respBody)})
		return
	}

	var token socialTokenResponse
	if err := json.Unmarshal(respBody, &token); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "invalid token response"})
		return
	}
	if strings.TrimSpace(token.AccessToken) == "" && strings.TrimSpace(token.RefreshToken) == "" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "token response missing access/refresh token"})
		return
	}

	expiresAt := ""
	if token.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	acc := &account{
		ID:             newUUID(),
		DisplayName:    strings.TrimSpace(req.DisplayName),
		GroupName:      strings.TrimSpace(req.GroupName),
		Enabled:        true,
		AuthMethod:     authMethodSocial,
		SocialProvider: flow.Provider,
		AccessToken:    strings.TrimSpace(token.AccessToken),
		RefreshToken:   strings.TrimSpace(token.RefreshToken),
		ProfileARN:     strings.TrimSpace(token.ProfileARN),
		Region:         nonEmpty(req.Region, s.cfg.Region),
		IDCRegion:      nonEmpty(req.Region, s.cfg.Region),
		ExpiresAt:      expiresAt,
		LastRefresh:    time.Now().UTC().Format(time.RFC3339),
	}
	if strings.TrimSpace(acc.DisplayName) == "" {
		acc.DisplayName = fmt.Sprintf("%s-%s", strings.ToLower(flow.Provider), acc.ID[:8])
	}
	saved, saveErr := s.store.addOrUpdateAccount(acc, true)
	if saveErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": saveErr.Error()})
		return
	}

	s.oauthMu.Lock()
	if s.oauth != nil {
		s.oauth.Status = "authorized"
		s.oauth.Error = ""
		s.oauth.CodeVerifier = ""
	}
	flowCopy := s.oauth
	s.oauthMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account": saved, "oauth": flowCopy})
}

func (s *server) handleAuthDeviceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req deviceStartRequest
	_ = decodeJSONBody(r, 1<<20, &req)

	region := strings.TrimSpace(req.Region)
	if region == "" {
		region = strings.TrimSpace(s.cfg.Region)
	}
	if region == "" {
		region = "us-east-1"
	}
	startURL := strings.TrimSpace(req.BuilderIDStartURL)
	if startURL == "" {
		startURL = strings.TrimSpace(s.cfg.BuilderIDStartURL)
	}
	if startURL == "" {
		startURL = "https://view.awsapps.com/start"
	}

	oidcBase := strings.TrimRight(renderRegionTemplate(s.cfg.AWSOIDCEndpoint, region), "/")

	registerPayload := map[string]any{
		"clientName": "Kiro IDE",
		"clientType": "public",
		"scopes":     kiroScopes,
	}
	regBody, _ := json.Marshal(registerPayload)
	regReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, oidcBase+"/client/register", bytes.NewReader(regBody))
	regReq.Header.Set("content-type", "application/json")
	regReq.Header.Set("user-agent", defaultUserAgent)
	regResp, err := s.httpc.Do(regReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	defer regResp.Body.Close()
	regRespBody, _ := io.ReadAll(io.LimitReader(regResp.Body, 2<<20))
	if regResp.StatusCode < 200 || regResp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": summarizeHTTPError(regResp.StatusCode, regRespBody)})
		return
	}
	var regData struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := json.Unmarshal(regRespBody, &regData); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "invalid register response"})
		return
	}
	if strings.TrimSpace(regData.ClientID) == "" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "register response missing clientId"})
		return
	}

	devicePayload := map[string]any{
		"clientId":     regData.ClientID,
		"clientSecret": regData.ClientSecret,
		"startUrl":     startURL,
	}
	devBody, _ := json.Marshal(devicePayload)
	devReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, oidcBase+"/device_authorization", bytes.NewReader(devBody))
	devReq.Header.Set("content-type", "application/json")
	devReq.Header.Set("user-agent", defaultUserAgent)
	devResp, err := s.httpc.Do(devReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	defer devResp.Body.Close()
	devRespBody, _ := io.ReadAll(io.LimitReader(devResp.Body, 2<<20))
	if devResp.StatusCode < 200 || devResp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": summarizeHTTPError(devResp.StatusCode, devRespBody)})
		return
	}

	var devData struct {
		DeviceCode              string `json:"deviceCode"`
		UserCode                string `json:"userCode"`
		VerificationURI         string `json:"verificationUri"`
		VerificationURIComplete string `json:"verificationUriComplete"`
		ExpiresIn               int    `json:"expiresIn"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(devRespBody, &devData); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "invalid device authorization response"})
		return
	}
	if strings.TrimSpace(devData.DeviceCode) == "" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "device authorization missing deviceCode"})
		return
	}
	if devData.Interval <= 0 {
		devData.Interval = 5
	}

	flow := &deviceFlow{
		Status:                  "pending",
		Region:                  region,
		ClientID:                regData.ClientID,
		ClientSecret:            regData.ClientSecret,
		DeviceCode:              devData.DeviceCode,
		UserCode:                devData.UserCode,
		VerificationURI:         devData.VerificationURI,
		VerificationURIComplete: devData.VerificationURIComplete,
		ExpiresIn:               devData.ExpiresIn,
		Interval:                devData.Interval,
		StartedAt:               time.Now().UTC(),
	}

	s.deviceMu.Lock()
	s.device = flow
	s.deviceMu.Unlock()

	go s.pollDeviceToken(flow, strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.GroupName))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "flow": flow})
}

func (s *server) pollDeviceToken(flow *deviceFlow, displayName, groupName string) {
	if flow == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), deviceAuthTimeout)
	defer cancel()

	oidcBase := strings.TrimRight(renderRegionTemplate(s.cfg.AWSOIDCEndpoint, flow.Region), "/")
	interval := time.Duration(flow.Interval) * time.Second
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}
	deadline := time.Now().UTC().Add(time.Duration(flow.ExpiresIn) * time.Second)
	if flow.ExpiresIn <= 0 {
		deadline = time.Now().UTC().Add(15 * time.Minute)
	}

	for {
		if time.Now().UTC().After(deadline) {
			s.deviceMu.Lock()
			if s.device != nil {
				s.device.Status = "timeout"
				s.device.Error = "device authorization expired"
			}
			s.deviceMu.Unlock()
			return
		}

		payload := map[string]any{
			"clientId":     flow.ClientID,
			"clientSecret": flow.ClientSecret,
			"deviceCode":   flow.DeviceCode,
			"grantType":    "urn:ietf:params:oauth:grant-type:device_code",
		}
		b, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, oidcBase+"/token", bytes.NewReader(b))
		req.Header.Set("content-type", "application/json")
		req.Header.Set("user-agent", defaultUserAgent)
		resp, err := s.httpc.Do(req)
		if err != nil {
			time.Sleep(interval)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var token idcTokenResponse
			if json.Unmarshal(body, &token) == nil && strings.TrimSpace(token.AccessToken) != "" {
				expiresAt := ""
				if token.ExpiresIn > 0 {
					expiresAt = time.Now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)
				}
				acc := &account{
					ID:           newUUID(),
					DisplayName:  displayName,
					GroupName:    groupName,
					Enabled:      true,
					AuthMethod:   authMethodBuilder,
					AccessToken:  strings.TrimSpace(token.AccessToken),
					RefreshToken: strings.TrimSpace(token.RefreshToken),
					ClientID:     flow.ClientID,
					ClientSecret: flow.ClientSecret,
					Region:       flow.Region,
					IDCRegion:    flow.Region,
					ExpiresAt:    expiresAt,
					LastRefresh:  time.Now().UTC().Format(time.RFC3339),
				}
				if strings.TrimSpace(acc.DisplayName) == "" {
					acc.DisplayName = "builder-id-" + acc.ID[:8]
				}
				_, _ = s.store.addOrUpdateAccount(acc, true)

				s.deviceMu.Lock()
				if s.device != nil {
					s.device.Status = "authorized"
					s.device.Error = ""
				}
				s.deviceMu.Unlock()
				return
			}
		}

		var errObj map[string]any
		_ = json.Unmarshal(body, &errObj)
		errCode := strings.ToLower(strings.TrimSpace(extractString(errObj, "error", "errorCode")))
		switch errCode {
		case "authorization_pending":
			time.Sleep(interval)
			continue
		case "slow_down":
			time.Sleep(interval + 5*time.Second)
			continue
		default:
			s.deviceMu.Lock()
			if s.device != nil {
				s.device.Status = "error"
				if errCode != "" {
					s.device.Error = errCode
				} else {
					s.device.Error = summarizeHTTPError(resp.StatusCode, body)
				}
			}
			s.deviceMu.Unlock()
			return
		}
	}
}

func parseImportRequest(raw map[string]any) importRequest {
	req := importRequest{}
	req.AccountID = extractString(raw, "account_id", "accountId", "id")
	req.DisplayName = extractString(raw, "display_name", "displayName", "name")
	req.GroupName = extractString(raw, "group_name", "groupName")
	req.AuthMethod = extractString(raw, "auth_method", "authMethod")
	req.SocialProvider = extractString(raw, "social_provider", "socialProvider")
	req.AccessToken = extractString(raw, "access_token", "accessToken")
	req.RefreshToken = extractString(raw, "refresh_token", "refreshToken")
	req.ProfileARN = extractString(raw, "profile_arn", "profileArn")
	req.ClientID = extractString(raw, "client_id", "clientId")
	req.ClientSecret = extractString(raw, "client_secret", "clientSecret")
	req.Region = extractString(raw, "region")
	req.IDCRegion = extractString(raw, "idc_region", "idcRegion")
	req.ExpiresAt = extractString(raw, "expires_at", "expiresAt", "expired")
	if v, ok := raw["enabled"].(bool); ok {
		req.Enabled = &v
	}
	if nested, ok := raw["auth_json"].(map[string]any); ok {
		if req.AccessToken == "" {
			req.AccessToken = extractString(nested, "access_token", "accessToken")
		}
		if req.RefreshToken == "" {
			req.RefreshToken = extractString(nested, "refresh_token", "refreshToken")
		}
		if req.ProfileARN == "" {
			req.ProfileARN = extractString(nested, "profile_arn", "profileArn")
		}
		if req.ClientID == "" {
			req.ClientID = extractString(nested, "client_id", "clientId")
		}
		if req.ClientSecret == "" {
			req.ClientSecret = extractString(nested, "client_secret", "clientSecret")
		}
		if req.Region == "" {
			req.Region = extractString(nested, "region")
		}
		if req.IDCRegion == "" {
			req.IDCRegion = extractString(nested, "idc_region", "idcRegion")
		}
		if req.AuthMethod == "" {
			req.AuthMethod = extractString(nested, "auth_method", "authMethod")
		}
		if req.ExpiresAt == "" {
			req.ExpiresAt = extractString(nested, "expires_at", "expiresAt", "expired")
		}
	}
	return req
}

func (s *server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var raw map[string]any
	if err := decodeJSONBody(r, 6<<20, &raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid body"})
		return
	}
	req := parseImportRequest(raw)
	if strings.TrimSpace(req.AccessToken) == "" && strings.TrimSpace(req.RefreshToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing refresh_token or access_token"})
		return
	}
	if strings.TrimSpace(req.Region) == "" {
		req.Region = s.cfg.Region
	}
	if strings.TrimSpace(req.IDCRegion) == "" {
		req.IDCRegion = req.Region
	}
	enabled := boolPtrValue(req.Enabled, true)
	authMethod := strings.ToLower(strings.TrimSpace(req.AuthMethod))
	if authMethod == "" {
		if strings.TrimSpace(req.ProfileARN) != "" {
			authMethod = authMethodSocial
		} else if strings.TrimSpace(req.ClientID) != "" || strings.TrimSpace(req.ClientSecret) != "" {
			authMethod = authMethodBuilder
		} else {
			authMethod = authMethodImport
		}
	}

	acc := &account{
		ID:             strings.TrimSpace(req.AccountID),
		DisplayName:    strings.TrimSpace(req.DisplayName),
		GroupName:      strings.TrimSpace(req.GroupName),
		Enabled:        enabled,
		AuthMethod:     authMethod,
		SocialProvider: normalizeSocialProvider(req.SocialProvider),
		AccessToken:    strings.TrimSpace(req.AccessToken),
		RefreshToken:   strings.TrimSpace(req.RefreshToken),
		ProfileARN:     strings.TrimSpace(req.ProfileARN),
		ClientID:       strings.TrimSpace(req.ClientID),
		ClientSecret:   strings.TrimSpace(req.ClientSecret),
		Region:         strings.TrimSpace(req.Region),
		IDCRegion:      strings.TrimSpace(req.IDCRegion),
		ExpiresAt:      strings.TrimSpace(req.ExpiresAt),
	}
	if acc.DisplayName == "" {
		acc.DisplayName = "imported-" + newUUID()[:8]
	}

	if strings.TrimSpace(acc.AccessToken) == "" && strings.TrimSpace(acc.RefreshToken) != "" {
		tmp, err := s.refreshAccessTokenForAccount(r.Context(), acc)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		acc = tmp
	}
	acc.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	saved, err := s.store.addOrUpdateAccount(acc, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account": saved})
}

func (s *server) refreshAccountTokens(ctx context.Context, accountID string) (*account, error) {
	acc, ok := s.store.getAccount(accountID)
	if !ok {
		return nil, errors.New("account not found")
	}
	refreshed, err := s.refreshAccessTokenForAccount(ctx, acc)
	if err != nil {
		_, _ = s.store.updateAccount(accountID, func(a *account) error {
			a.LastError = err.Error()
			return nil
		})
		return nil, err
	}
	refreshed.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	saved, err := s.store.addOrUpdateAccount(refreshed, false)
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *server) refreshAccessTokenForAccount(ctx context.Context, acc *account) (*account, error) {
	if acc == nil {
		return nil, errors.New("nil account")
	}
	if strings.TrimSpace(acc.RefreshToken) == "" {
		if strings.TrimSpace(acc.AccessToken) != "" {
			return acc, nil
		}
		return nil, errors.New("missing refresh token")
	}
	method := strings.ToLower(strings.TrimSpace(acc.AuthMethod))
	if method == "" {
		method = authMethodImport
	}

	ctxTimeout := time.Duration(s.cfg.TokenRefreshTimeoutSec) * time.Second
	if ctxTimeout <= 0 {
		ctxTimeout = 15 * time.Second
	}
	tx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	if method == authMethodSocial {
		region := nonEmpty(acc.Region, s.cfg.Region)
		endpoint := renderRegionTemplate(s.cfg.SocialRefreshURL, region)
		body, _ := json.Marshal(map[string]any{"refreshToken": strings.TrimSpace(acc.RefreshToken)})
		req, _ := http.NewRequestWithContext(tx, http.MethodPost, endpoint, bytes.NewReader(body))
		req.Header.Set("content-type", "application/json")
		req.Header.Set("user-agent", defaultUserAgent)
		resp, err := s.httpc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New(summarizeHTTPError(resp.StatusCode, respBody))
		}
		var out socialTokenResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return nil, errors.New("invalid social refresh response")
		}
		if strings.TrimSpace(out.AccessToken) == "" {
			return nil, errors.New("social refresh missing accessToken")
		}
		next := cloneAccount(acc)
		next.AccessToken = strings.TrimSpace(out.AccessToken)
		if strings.TrimSpace(out.RefreshToken) != "" {
			next.RefreshToken = strings.TrimSpace(out.RefreshToken)
		}
		if strings.TrimSpace(out.ProfileARN) != "" {
			next.ProfileARN = strings.TrimSpace(out.ProfileARN)
		}
		if out.ExpiresIn > 0 {
			next.ExpiresAt = time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second).Format(time.RFC3339)
		}
		next.LastError = ""
		return next, nil
	}

	region := nonEmpty(acc.IDCRegion, acc.Region)
	if region == "" {
		region = s.cfg.Region
	}
	endpoint := renderRegionTemplate(s.cfg.IDCRefreshURL, region)
	payload := map[string]any{
		"refreshToken": strings.TrimSpace(acc.RefreshToken),
		"grantType":    "refresh_token",
	}
	if strings.TrimSpace(acc.ClientID) != "" {
		payload["clientId"] = strings.TrimSpace(acc.ClientID)
	}
	if strings.TrimSpace(acc.ClientSecret) != "" {
		payload["clientSecret"] = strings.TrimSpace(acc.ClientSecret)
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(tx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", defaultUserAgent)
	resp, err := s.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(summarizeHTTPError(resp.StatusCode, respBody))
	}
	var out idcTokenResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, errors.New("invalid idc refresh response")
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, errors.New("idc refresh missing accessToken")
	}
	next := cloneAccount(acc)
	next.AccessToken = strings.TrimSpace(out.AccessToken)
	if strings.TrimSpace(out.RefreshToken) != "" {
		next.RefreshToken = strings.TrimSpace(out.RefreshToken)
	}
	if out.ExpiresIn > 0 {
		next.ExpiresAt = time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	next.LastError = ""
	return next, nil
}
