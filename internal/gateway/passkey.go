package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func rpIDFromOrigin(origin string) string {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u == nil {
		return ""
	}
	return strings.TrimSpace(u.Hostname())
}

// webAuthnOriginFromRequest prefers the browser-provided Origin header (when present)
// so WebAuthn rpId/origin match the actual page origin even behind reverse proxies.
// It falls back to baseURLFromRequest.
func webAuthnOriginFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && origin != "null" {
		if u, err := url.Parse(origin); err == nil && u != nil {
			scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
			host := strings.TrimSpace(u.Host)
			if (scheme == "http" || scheme == "https") && host != "" {
				return scheme + "://" + host
			}
		}
	}
	return baseURLFromRequest(r)
}

func (s *Server) handlePasskeyAuthBeginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	origin := webAuthnOriginFromRequest(r)
	rpID := rpIDFromOrigin(origin)
	payload, _ := json.Marshal(map[string]any{
		"username": strings.TrimSpace(req.Username),
		"rp_id":    rpID,
		"origin":   origin,
	})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodPost, "/passkey/auth/begin", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handlePasskeyAuthFinishAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	var req struct {
		Remember bool `json:"remember"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	status, respBody, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodPost, "/passkey/auth/finish", body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if status < 200 || status >= 300 {
		writeProxyResponse(w, status, hdr, respBody)
		return
	}
	var moduleResp struct {
		OK       bool   `json:"ok"`
		Username string `json:"username"`
	}
	_ = json.Unmarshal(respBody, &moduleResp)
	if strings.TrimSpace(moduleResp.Username) == "" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "invalid module response"})
		return
	}
	s.finalizePrimaryLogin(w, r, strings.TrimSpace(moduleResp.Username), req.Remember, "passkey")
}

func (s *Server) handlePasskeyLoginPage(w http.ResponseWriter, r *http.Request) {
	hasAdmin, err := s.store.HasAdmin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !hasAdmin {
		http.Redirect(w, r, "/admin/setup", http.StatusFound)
		return
	}
	passkeyInstalled := false
	if mod, err := s.store.GetInstalledModule(r.Context(), passkeyLoginModuleID); err == nil && mod != nil && mod.Enabled {
		passkeyInstalled = true
	}
	twoFAInstalled := false
	if mod, err := s.store.GetInstalledModule(r.Context(), totp2FAModuleID); err == nil && mod != nil && mod.Enabled {
		twoFAInstalled = true
	}
	s.renderPage(w, "login_passkey", map[string]any{
		"Page":             "Passkey Login",
		"PasskeyInstalled": passkeyInstalled,
		"TwoFAInstalled":   twoFAInstalled,
	})
}

func (s *Server) adminUsernameFromSession(r *http.Request) (string, bool) {
	username, ok := s.sessions.username(r)
	username = strings.TrimSpace(username)
	if !ok || username == "" {
		return "", false
	}
	return username, true
}

func (s *Server) handlePasskeyRegisterBeginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "passkey module is not enabled"})
		return
	}
	username, ok := s.adminUsernameFromSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	origin := webAuthnOriginFromRequest(r)
	rpID := rpIDFromOrigin(origin)
	payload, _ := json.Marshal(map[string]any{
		"username": username,
		"rp_id":    rpID,
		"rp_name":  "LightBridge",
		"origin":   origin,
	})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodPost, "/passkey/register/begin", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handlePasskeyRegisterFinishAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "passkey module is not enabled"})
		return
	}
	username, ok := s.adminUsernameFromSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	var req struct {
		State      string `json:"state"`
		Credential any    `json:"credential"`
		Label      string `json:"label"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.State) == "" || req.Credential == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "state and credential are required"})
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"username":   username,
		"state":      strings.TrimSpace(req.State),
		"credential": req.Credential,
		"label":      strings.TrimSpace(req.Label),
	})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodPost, "/passkey/register/finish", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handlePasskeyCredentialsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "passkey module is not enabled"})
		return
	}
	username, ok := s.adminUsernameFromSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	p := "/passkey/credentials?username=" + url.QueryEscape(username)
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodGet, p, nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handlePasskeyCredentialDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !s.isModuleInstalledAndEnabled(r.Context(), passkeyLoginModuleID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "passkey module is not enabled"})
		return
	}
	username, ok := s.adminUsernameFromSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	var req struct {
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.CredentialID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "credential_id is required"})
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"username":      username,
		"credential_id": strings.TrimSpace(req.CredentialID),
	})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), passkeyLoginModuleID, http.MethodPost, "/passkey/credentials/delete", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}
