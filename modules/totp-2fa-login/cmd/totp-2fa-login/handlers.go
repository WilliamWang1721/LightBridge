package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type pendingEnroll struct {
	Username    string
	Label       string
	Secret      string
	Issuer      string
	AccountName string
	ExpiresAt   time.Time
}

type enrollBeginReq struct {
	Username    string `json:"username"`
	Label       string `json:"label"`
	Issuer      string `json:"issuer"`
	AccountName string `json:"account_name"`
}

func (s *server) handleEnrollBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req enrollBeginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Label = strings.TrimSpace(req.Label)
	req.Issuer = strings.TrimSpace(req.Issuer)
	req.AccountName = strings.TrimSpace(req.AccountName)
	if req.Username == "" {
		writeErr(w, http.StatusBadRequest, "missing username")
		return
	}
	if req.Issuer == "" {
		req.Issuer = "LightBridge"
	}
	if req.AccountName == "" {
		req.AccountName = req.Username
	}
	if len(req.Label) > 64 {
		req.Label = req.Label[:64]
	}

	secret, err := newTOTPSecret()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	state, err := s.newEnrollState(pendingEnroll{
		Username:    req.Username,
		Label:       req.Label,
		Secret:      secret,
		Issuer:      req.Issuer,
		AccountName: req.AccountName,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	otpauth := buildOTPAuthURI(req.Issuer, req.AccountName, secret)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"state":        state,
		"secret":       secret,
		"otpauth_uri":  otpauth,
		"issuer":       req.Issuer,
		"account_name": req.AccountName,
	})
}

func (s *server) handleEnrollConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		State string `json:"state"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.State = strings.TrimSpace(req.State)
	if req.State == "" {
		writeErr(w, http.StatusBadRequest, "missing state")
		return
	}
	st, ok := s.getEnrollState(req.State)
	if !ok {
		writeErr(w, http.StatusBadRequest, "state expired or invalid")
		return
	}
	okCode, _ := verifyTOTP(st.Secret, req.Code, time.Now())
	if !okCode {
		writeErr(w, http.StatusUnauthorized, "invalid 2fa code")
		return
	}

	devIDBytes, err := randomBytes(12)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	deviceID := base64urlEncode(devIDBytes)
	label := strings.TrimSpace(st.Label)
	if label == "" {
		label = "Authenticator"
	}
	if len(label) > 64 {
		label = label[:64]
	}
	dev := totpDevice{
		DeviceID:   deviceID,
		Label:      label,
		Secret:     st.Secret,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		LastUsedAt: "",
	}
	if err := s.addDevice(st.Username, dev); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	s.deleteEnrollState(req.State)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"username":  st.Username,
		"device_id": dev.DeviceID,
		"label":     dev.Label,
	})
}

func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeErr(w, http.StatusBadRequest, "missing username")
		return
	}
	dev, _, ok, err := s.verifyForUser(req.Username, req.Code, time.Now())
	if err != nil {
		if errors.Is(err, errInvalidCode) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusUnauthorized, "invalid 2fa code")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"username":  req.Username,
		"device_id": dev.DeviceID,
		"label":     dev.Label,
	})
}

func (s *server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		writeErr(w, http.StatusBadRequest, "missing username")
		return
	}
	list := s.listDevices(username)
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		out = append(out, map[string]any{
			"device_id":    item.DeviceID,
			"label":        item.Label,
			"created_at":   item.CreatedAt,
			"last_used_at": item.LastUsedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *server) handleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.Username == "" || req.DeviceID == "" {
		writeErr(w, http.StatusBadRequest, "missing username/device_id")
		return
	}
	deleted, err := s.deleteDevice(req.Username, req.DeviceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted})
}

var errInvalidCode = errors.New("invalid code")

func (s *server) newEnrollState(st pendingEnroll) (string, error) {
	if strings.TrimSpace(st.Username) == "" || strings.TrimSpace(st.Secret) == "" {
		return "", errors.New("invalid enroll state")
	}
	id, err := randomBytes(24)
	if err != nil {
		return "", err
	}
	token := base64urlEncode(id)
	st.ExpiresAt = time.Now().Add(s.stateTTL)

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.enrollStates == nil {
		s.enrollStates = map[string]pendingEnroll{}
	}
	now := time.Now()
	for k, v := range s.enrollStates {
		if now.After(v.ExpiresAt.Add(s.stateMaxSkew)) {
			delete(s.enrollStates, k)
		}
	}
	s.enrollStates[token] = st
	return token, nil
}

func (s *server) getEnrollState(token string) (pendingEnroll, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return pendingEnroll{}, false
	}
	now := time.Now()
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	st, ok := s.enrollStates[token]
	if !ok {
		return pendingEnroll{}, false
	}
	if now.After(st.ExpiresAt.Add(s.stateMaxSkew)) {
		delete(s.enrollStates, token)
		return pendingEnroll{}, false
	}
	return st, true
}

func (s *server) deleteEnrollState(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	delete(s.enrollStates, token)
}
