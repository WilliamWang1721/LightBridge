package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lightbridge/internal/util"
)

const (
	totp2FAModuleID               = "totp-2fa-login"
	settingAdmin2FAEnabled        = "admin_2fa_enabled"
	settingAdmin2FARequirePwd     = "admin_2fa_require_for_password"
	settingAdmin2FARequirePasskey = "admin_2fa_require_for_passkey"
	settingAdmin2FAAllowTOTPOnly  = "admin_2fa_allow_totp_only"
)

type twoFAPolicy struct {
	Enabled            bool `json:"enabled"`
	RequireForPassword bool `json:"require_for_password"`
	RequireForPasskey  bool `json:"require_for_passkey"`
	AllowTOTPOnlyLogin bool `json:"allow_totp_only_login"`
}

type authTicket struct {
	Username      string
	Remember      bool
	PrimaryMethod string
	ExpiresAt     time.Time
}

func parseBoolSetting(v string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func boolSetting(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func (s *Server) isModuleInstalledAndEnabled(ctx context.Context, moduleID string) bool {
	mod, err := s.store.GetInstalledModule(ctx, moduleID)
	if err != nil || mod == nil {
		return false
	}
	return mod.Enabled
}

func (s *Server) getTwoFAPolicy(ctx context.Context) (twoFAPolicy, error) {
	policy := twoFAPolicy{
		Enabled:            false,
		RequireForPassword: true,
		RequireForPasskey:  false,
		AllowTOTPOnlyLogin: false,
	}

	if v, ok, err := s.store.GetSetting(ctx, settingAdmin2FAEnabled); err != nil {
		return policy, err
	} else if ok {
		policy.Enabled = parseBoolSetting(v, policy.Enabled)
	}
	if v, ok, err := s.store.GetSetting(ctx, settingAdmin2FARequirePwd); err != nil {
		return policy, err
	} else if ok {
		policy.RequireForPassword = parseBoolSetting(v, policy.RequireForPassword)
	}
	if v, ok, err := s.store.GetSetting(ctx, settingAdmin2FARequirePasskey); err != nil {
		return policy, err
	} else if ok {
		policy.RequireForPasskey = parseBoolSetting(v, policy.RequireForPasskey)
	}
	if v, ok, err := s.store.GetSetting(ctx, settingAdmin2FAAllowTOTPOnly); err != nil {
		return policy, err
	} else if ok {
		policy.AllowTOTPOnlyLogin = parseBoolSetting(v, policy.AllowTOTPOnlyLogin)
	}

	if !policy.Enabled {
		policy.RequireForPassword = false
		policy.RequireForPasskey = false
		policy.AllowTOTPOnlyLogin = false
	}
	return policy, nil
}

func (s *Server) saveTwoFAPolicy(ctx context.Context, policy twoFAPolicy) error {
	if !policy.Enabled {
		policy.RequireForPassword = false
		policy.RequireForPasskey = false
		policy.AllowTOTPOnlyLogin = false
	}
	if err := s.store.SetSetting(ctx, settingAdmin2FAEnabled, boolSetting(policy.Enabled)); err != nil {
		return err
	}
	if err := s.store.SetSetting(ctx, settingAdmin2FARequirePwd, boolSetting(policy.RequireForPassword)); err != nil {
		return err
	}
	if err := s.store.SetSetting(ctx, settingAdmin2FARequirePasskey, boolSetting(policy.RequireForPasskey)); err != nil {
		return err
	}
	if err := s.store.SetSetting(ctx, settingAdmin2FAAllowTOTPOnly, boolSetting(policy.AllowTOTPOnlyLogin)); err != nil {
		return err
	}
	return nil
}

func moduleBodyError(status int, body []byte) error {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &payload)
	msg := strings.TrimSpace(payload.Error)
	if msg == "" {
		msg = strings.TrimSpace(payload.Message)
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	if msg == "" {
		msg = fmt.Sprintf("module request failed (%d)", status)
	}
	return errors.New(msg)
}

func (s *Server) getTwoFADeviceCount(ctx context.Context, username string) (int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, errors.New("missing username")
	}
	p := "/totp/devices?username=" + url.QueryEscape(username)
	status, body, _, err := s.proxyModuleHTTP(ctx, totp2FAModuleID, http.MethodGet, p, nil)
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return 0, moduleBodyError(status, body)
	}
	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	return len(resp.Data), nil
}

func (s *Server) verifyTwoFACode(ctx context.Context, username, code string) error {
	payload, _ := json.Marshal(map[string]any{
		"username": strings.TrimSpace(username),
		"code":     strings.TrimSpace(code),
	})
	status, body, _, err := s.proxyModuleHTTP(ctx, totp2FAModuleID, http.MethodPost, "/totp/verify", payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return moduleBodyError(status, body)
	}
	return nil
}

func (s *Server) shouldRequireTwoFA(ctx context.Context, username, primaryMethod string) (bool, error) {
	if !s.isModuleInstalledAndEnabled(ctx, totp2FAModuleID) {
		return false, nil
	}
	policy, err := s.getTwoFAPolicy(ctx)
	if err != nil {
		return false, err
	}
	if !policy.Enabled {
		return false, nil
	}

	require := false
	switch strings.ToLower(strings.TrimSpace(primaryMethod)) {
	case "password":
		require = policy.RequireForPassword
	case "passkey":
		require = policy.RequireForPasskey
	}
	if !require {
		return false, nil
	}
	count, err := s.getTwoFADeviceCount(ctx, username)
	if err != nil {
		return false, err
	}
	if count <= 0 {
		return false, errors.New("2FA is required but no authenticator is enrolled")
	}
	return true, nil
}

func (s *Server) issueAuthTicket(username string, remember bool, primaryMethod string) (string, error) {
	tok, err := util.RandomToken(36)
	if err != nil {
		return "", err
	}
	now := time.Now()
	ticket := authTicket{
		Username:      strings.TrimSpace(username),
		Remember:      remember,
		PrimaryMethod: strings.TrimSpace(primaryMethod),
		ExpiresAt:     now.Add(3 * time.Minute),
	}

	s.authTicketMu.Lock()
	defer s.authTicketMu.Unlock()
	if s.authTickets == nil {
		s.authTickets = map[string]authTicket{}
	}
	for k, v := range s.authTickets {
		if now.After(v.ExpiresAt) {
			delete(s.authTickets, k)
		}
	}
	s.authTickets[tok] = ticket
	return tok, nil
}

func (s *Server) getAuthTicket(token string) (authTicket, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return authTicket{}, false
	}
	now := time.Now()
	s.authTicketMu.Lock()
	defer s.authTicketMu.Unlock()
	t, ok := s.authTickets[token]
	if !ok {
		return authTicket{}, false
	}
	if now.After(t.ExpiresAt) {
		delete(s.authTickets, token)
		return authTicket{}, false
	}
	return t, true
}

func (s *Server) deleteAuthTicket(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	s.authTicketMu.Lock()
	defer s.authTicketMu.Unlock()
	delete(s.authTickets, token)
}

func (s *Server) finalizePrimaryLogin(w http.ResponseWriter, r *http.Request, username string, remember bool, primaryMethod string) {
	require2FA, err := s.shouldRequireTwoFA(r.Context(), username, primaryMethod)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": err.Error()})
		return
	}
	if require2FA {
		ticket, err := s.issueAuthTicket(username, remember, primaryMethod)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"requires_2fa": true,
			"ticket":       ticket,
			"next":         "/admin/dashboard",
		})
		return
	}
	if err := s.sessions.newSession(w, strings.TrimSpace(username), remember); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "next": "/admin/dashboard"})
}

func (s *Server) handleTwoFAChallengeVerifyAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	ticket, ok := s.getAuthTicket(req.Ticket)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "ticket expired or invalid"})
		return
	}
	if err := s.verifyTwoFACode(r.Context(), ticket.Username, req.Code); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	s.deleteAuthTicket(req.Ticket)
	if err := s.sessions.newSession(w, ticket.Username, ticket.Remember); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "next": "/admin/dashboard"})
}

func (s *Server) handleTwoFATOTPOnlyLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "2FA module is not enabled"})
		return
	}
	policy, err := s.getTwoFAPolicy(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !policy.Enabled || !policy.AllowTOTPOnlyLogin {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "2FA-only login is disabled"})
		return
	}
	var req struct {
		Username string `json:"username"`
		Code     string `json:"code"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "username is required"})
		return
	}
	if _, err := s.store.GetAdminPasswordHash(r.Context(), req.Username); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid username or 2fa code"})
		return
	}
	if err := s.verifyTwoFACode(r.Context(), req.Username, req.Code); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid username or 2fa code"})
		return
	}
	if err := s.sessions.newSession(w, req.Username, req.Remember); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "next": "/admin/dashboard"})
}

func (s *Server) handleAuthMethodsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	methods := make([]map[string]any, 0, 2)

	if s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID) {
		methods = append(methods, map[string]any{
			"id":    "passkey",
			"label": "使用 Passkey 登录",
		})
	}

	if s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		policy, err := s.getTwoFAPolicy(r.Context())
		if err == nil && policy.Enabled && policy.AllowTOTPOnlyLogin {
			canUse := true
			if username != "" {
				if cnt, err := s.getTwoFADeviceCount(r.Context(), username); err != nil || cnt <= 0 {
					canUse = false
				}
			}
			if canUse {
				methods = append(methods, map[string]any{
					"id":    "totp",
					"label": "使用 2FA 代码登录",
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"methods": methods,
	})
}

func (s *Server) handleTwoFAPolicyAPI(w http.ResponseWriter, r *http.Request) {
	username, _ := s.sessions.username(r)
	username = strings.TrimSpace(username)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		policy, err := s.getTwoFAPolicy(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		moduleEnabled := s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID)
		count := 0
		if moduleEnabled {
			if c, err := s.getTwoFADeviceCount(r.Context(), username); err == nil {
				count = c
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"policy":         policy,
			"module_enabled": moduleEnabled,
			"device_count":   count,
		})
	case http.MethodPost:
		if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "2FA module is not enabled"})
			return
		}
		current, err := s.getTwoFAPolicy(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		var req struct {
			Enabled            *bool `json:"enabled"`
			RequireForPassword *bool `json:"require_for_password"`
			RequireForPasskey  *bool `json:"require_for_passkey"`
			AllowTOTPOnlyLogin *bool `json:"allow_totp_only_login"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if req.Enabled != nil {
			current.Enabled = *req.Enabled
		}
		if req.RequireForPassword != nil {
			current.RequireForPassword = *req.RequireForPassword
		}
		if req.RequireForPasskey != nil {
			current.RequireForPasskey = *req.RequireForPasskey
		}
		if req.AllowTOTPOnlyLogin != nil {
			current.AllowTOTPOnlyLogin = *req.AllowTOTPOnlyLogin
		}

		if current.Enabled {
			if !current.RequireForPassword && !current.RequireForPasskey && !current.AllowTOTPOnlyLogin {
				current.RequireForPassword = true
			}
			count, err := s.getTwoFADeviceCount(r.Context(), username)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
				return
			}
			if count <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请先添加至少一个 2FA 验证器"})
				return
			}
		}

		if err := s.saveTwoFAPolicy(r.Context(), current); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "policy": current})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleTwoFAEnrollBeginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	username, _ := s.sessions.username(r)
	username = strings.TrimSpace(username)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "2FA module is not enabled"})
		return
	}
	var req struct {
		Label string `json:"label"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
	payload, _ := json.Marshal(map[string]any{
		"username":     username,
		"label":        strings.TrimSpace(req.Label),
		"issuer":       "LightBridge",
		"account_name": username,
	})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), totp2FAModuleID, http.MethodPost, "/totp/enroll/begin", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handleTwoFAEnrollConfirmAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "2FA module is not enabled"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	status, respBody, hdr, err := s.proxyModuleHTTP(r.Context(), totp2FAModuleID, http.MethodPost, "/totp/enroll/confirm", body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, respBody)
}

func (s *Server) handleTwoFADevicesAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	username, _ := s.sessions.username(r)
	username = strings.TrimSpace(username)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "2FA module is not enabled"})
		return
	}
	p := "/totp/devices?username=" + url.QueryEscape(username)
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), totp2FAModuleID, http.MethodGet, p, nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handleTwoFADeviceDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	username, _ := s.sessions.username(r)
	username = strings.TrimSpace(username)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "2FA module is not enabled"})
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"username":  username,
		"device_id": strings.TrimSpace(req.DeviceID),
	})
	status, body, _, err := s.proxyModuleHTTP(r.Context(), totp2FAModuleID, http.MethodPost, "/totp/devices/delete", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if status < 200 || status >= 300 {
		writeJSON(w, status, map[string]any{"error": moduleBodyError(status, body).Error()})
		return
	}

	policyReset := false
	if count, err := s.getTwoFADeviceCount(r.Context(), username); err == nil && count == 0 {
		if p, err := s.getTwoFAPolicy(r.Context()); err == nil {
			if p.Enabled || p.AllowTOTPOnlyLogin || p.RequireForPasskey || p.RequireForPassword {
				_ = s.saveTwoFAPolicy(r.Context(), twoFAPolicy{})
				policyReset = true
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "policy_reset": policyReset})
}

func (s *Server) authSettingsPageData(r *http.Request) map[string]any {
	username, _ := s.sessions.username(r)
	if strings.TrimSpace(username) == "" {
		username = "Admin"
	}
	twoFAInstalled := s.isModuleInstalledAndEnabled(r.Context(), totp2FAModuleID)
	passkeyInstalled := s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID)
	return map[string]any{
		"Username":         username,
		"TwoFAInstalled":   twoFAInstalled,
		"PasskeyInstalled": passkeyInstalled,
	}
}

func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	data := s.authSettingsPageData(r)
	data["Page"] = "Settings"
	s.renderPage(w, "settings_index", data)
}

func (s *Server) handleSettingsAuthPage(w http.ResponseWriter, r *http.Request) {
	data := s.authSettingsPageData(r)
	data["Page"] = "Authentication Settings"
	s.renderPage(w, "settings_auth", data)
}

func (s *Server) handleSettingsAuthPasskeyPage(w http.ResponseWriter, r *http.Request) {
	data := s.authSettingsPageData(r)
	data["Page"] = "Passkey Settings"
	s.renderPage(w, "settings_auth_passkey", data)
}

func (s *Server) handleSettingsAuth2FAPage(w http.ResponseWriter, r *http.Request) {
	data := s.authSettingsPageData(r)
	data["Page"] = "2FA Settings"
	s.renderPage(w, "settings", data)
}

func (s *Server) handleSettingsAuthPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := s.authSettingsPageData(r)
	data["Page"] = "Password Settings"
	s.renderPage(w, "settings_auth_password", data)
}
