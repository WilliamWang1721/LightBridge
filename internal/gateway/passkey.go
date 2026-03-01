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
	origin := baseURLFromRequest(r)
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
