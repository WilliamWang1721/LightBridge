package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type pendingState struct {
	Kind      string
	Username  string
	Challenge []byte
	RPID      string
	Origin    string
	ExpiresAt time.Time
}

type beginResp struct {
	State     string `json:"state"`
	PublicKey any    `json:"publicKey"`
}

type registerBeginReq struct {
	Username string `json:"username"`
	RPID     string `json:"rp_id"`
	RPName   string `json:"rp_name"`
	Origin   string `json:"origin"`
}

type authBeginReq struct {
	Username string `json:"username"`
	RPID     string `json:"rp_id"`
	Origin   string `json:"origin"`
}

type finishReq struct {
	Username   string               `json:"username"`
	State      string               `json:"state"`
	Credential publicKeyCredential  `json:"credential"`
	Label      string               `json:"label"`
	Remember   bool                 `json:"remember"`
}

type publicKeyCredential struct {
	ID       string                    `json:"id"`
	RawID    string                    `json:"rawId"`
	Type     string                    `json:"type"`
	Response publicKeyCredentialResp   `json:"response"`
}

type publicKeyCredentialResp struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`

	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}

type credentialSummary struct {
	CredentialID string `json:"credential_id"`
	Label        string `json:"label"`
	CreatedAt    string `json:"created_at"`
	LastUsedAt   string `json:"last_used_at"`
}

func (s *server) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req registerBeginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.RPID = strings.TrimSpace(req.RPID)
	req.Origin = strings.TrimSpace(req.Origin)
	req.RPName = strings.TrimSpace(req.RPName)
	if req.Username == "" || req.RPID == "" || req.Origin == "" {
		writeErr(w, http.StatusBadRequest, "missing username/rp_id/origin")
		return
	}
	if req.RPName == "" {
		req.RPName = "LightBridge"
	}

	chal, _ := randomBytes(32)
	state, err := s.newState("register", req.Username, req.RPID, req.Origin, chal)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	userHandle := userHandleForUsername(req.Username)
	excl := make([]publicKeyCredentialDescriptor, 0)
	for _, item := range s.listPasskeys(req.Username) {
		if strings.TrimSpace(item.CredentialID) == "" {
			continue
		}
		excl = append(excl, publicKeyCredentialDescriptor{Type: "public-key", ID: item.CredentialID})
	}

	opts := publicKeyCredentialCreationOptions{
		Challenge: base64urlEncode(chal),
		RP: rpEntity{
			Name: req.RPName,
			ID:   req.RPID,
		},
		User: userEntity{
			ID:          base64urlEncode(userHandle),
			Name:        req.Username,
			DisplayName: req.Username,
		},
		PubKeyCredParams: []pubKeyCredParam{
			{Type: "public-key", Alg: -7},   // ES256
			{Type: "public-key", Alg: -257}, // RS256 (optional)
		},
		Timeout:     60_000,
		Attestation: "none",
		AuthenticatorSelection: authenticatorSelection{
			ResidentKey:     "preferred",
			UserVerification: "preferred",
		},
		ExcludeCredentials: excl,
	}
	writeJSON(w, http.StatusOK, beginResp{State: state, PublicKey: opts})
}

func (s *server) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	_ = r.Body.Close()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var req finishReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.State = strings.TrimSpace(req.State)
	if req.Username == "" || req.State == "" {
		writeErr(w, http.StatusBadRequest, "missing username/state")
		return
	}
	st, ok := s.takeState(req.State)
	if !ok {
		writeErr(w, http.StatusBadRequest, "state expired or invalid")
		return
	}
	if st.Kind != "register" || st.Username != req.Username {
		writeErr(w, http.StatusBadRequest, "state mismatch")
		return
	}

	if strings.TrimSpace(req.Credential.Type) != "public-key" {
		writeErr(w, http.StatusBadRequest, "invalid credential type")
		return
	}

	clientDataJSON, err := base64urlDecode(req.Credential.Response.ClientDataJSON)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid clientDataJSON")
		return
	}
	cd, err := parseClientDataJSON(clientDataJSON)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid clientDataJSON")
		return
	}
	if err := verifyClientData(cd, "webauthn.create", st.Origin, st.Challenge); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}

	attObj, err := base64urlDecode(req.Credential.Response.AttestationObject)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid attestationObject")
		return
	}
	authData, err := parseAttestationObject(attObj)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	parsed, err := parseAttestedAuthData(authData)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := verifyRPIDHash(parsed.RPIDHash, st.RPID); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}
	if !isUserPresent(parsed.Flags) {
		writeErr(w, http.StatusForbidden, "user not present")
		return
	}

	credID := base64urlEncode(parsed.Credential)
	if credID == "" || len(parsed.PublicKeyCB) == 0 {
		writeErr(w, http.StatusBadRequest, "invalid credential")
		return
	}

	label := strings.TrimSpace(req.Label)
	if len(label) > 64 {
		label = label[:64]
	}
	rec := passkeyRecord{
		CredentialID:  credID,
		PublicKeyCOSE: base64urlEncode(parsed.PublicKeyCB),
		SignCount:     parsed.SignCnt,
		UserHandle:    base64urlEncode(userHandleForUsername(req.Username)),
		Label:         label,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		LastUsedAt:    "",
	}
	if err := s.addPasskey(req.Username, rec); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"username":      req.Username,
		"credential_id": credID,
	})
}

func (s *server) handleAuthBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req authBeginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.RPID = strings.TrimSpace(req.RPID)
	req.Origin = strings.TrimSpace(req.Origin)
	if req.RPID == "" || req.Origin == "" {
		writeErr(w, http.StatusBadRequest, "missing rp_id/origin")
		return
	}

	allow := make([]publicKeyCredentialDescriptor, 0)
	if req.Username != "" {
		for _, item := range s.listPasskeys(req.Username) {
			if strings.TrimSpace(item.CredentialID) == "" {
				continue
			}
			allow = append(allow, publicKeyCredentialDescriptor{Type: "public-key", ID: item.CredentialID})
		}
		if len(allow) == 0 {
			writeErr(w, http.StatusNotFound, "no passkey for this user")
			return
		}
	}

	chal, _ := randomBytes(32)
	state, err := s.newState("auth", req.Username, req.RPID, req.Origin, chal)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	opts := publicKeyCredentialRequestOptions{
		Challenge:       base64urlEncode(chal),
		RPID:            req.RPID,
		Timeout:         60_000,
		UserVerification: "preferred",
	}
	if len(allow) > 0 {
		opts.AllowCredentials = allow
	}
	writeJSON(w, http.StatusOK, beginResp{State: state, PublicKey: opts})
}

func (s *server) handleAuthFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	_ = r.Body.Close()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var req finishReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.State = strings.TrimSpace(req.State)
	if req.State == "" {
		writeErr(w, http.StatusBadRequest, "missing state")
		return
	}
	st, ok := s.takeState(req.State)
	if !ok {
		writeErr(w, http.StatusBadRequest, "state expired or invalid")
		return
	}
	if st.Kind != "auth" {
		writeErr(w, http.StatusBadRequest, "state mismatch")
		return
	}

	rawID := strings.TrimSpace(req.Credential.RawID)
	if rawID == "" {
		rawID = strings.TrimSpace(req.Credential.ID)
	}
	if rawID == "" {
		writeErr(w, http.StatusBadRequest, "missing credential id")
		return
	}
	// Normalize by decoding + re-encoding.
	rawIDBytes, err := base64urlDecode(rawID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid credential id")
		return
	}
	credID := base64urlEncode(rawIDBytes)

	username, rec, ok := s.getByCredentialID(credID)
	if !ok {
		writeErr(w, http.StatusNotFound, "credential not found")
		return
	}
	if st.Username != "" && st.Username != username {
		writeErr(w, http.StatusForbidden, "credential does not belong to user")
		return
	}

	clientDataJSON, err := base64urlDecode(req.Credential.Response.ClientDataJSON)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid clientDataJSON")
		return
	}
	cd, err := parseClientDataJSON(clientDataJSON)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid clientDataJSON")
		return
	}
	if err := verifyClientData(cd, "webauthn.get", st.Origin, st.Challenge); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}

	authenticatorData, err := base64urlDecode(req.Credential.Response.AuthenticatorData)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid authenticatorData")
		return
	}
	parsed, err := parseAuthenticatorData(authenticatorData)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := verifyRPIDHash(parsed.RPIDHash, st.RPID); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}
	if !isUserPresent(parsed.Flags) {
		writeErr(w, http.StatusForbidden, "user not present")
		return
	}

	sig, err := base64urlDecode(req.Credential.Response.Signature)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid signature")
		return
	}
	pubKeyBytes, err := base64urlDecode(rec.PublicKeyCOSE)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "invalid stored public key")
		return
	}
	pub, err := parseCOSEES256PublicKey(pubKeyBytes)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := verifyAssertionSignature(pub, authenticatorData, clientDataJSON, sig); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}

	// Sign counter check (best-effort; some authenticators always return 0).
	if parsed.SignCnt != 0 && rec.SignCount != 0 && parsed.SignCnt <= rec.SignCount {
		writeErr(w, http.StatusForbidden, "sign count did not increase")
		return
	}
	if parsed.SignCnt > rec.SignCount {
		_ = s.updateSignCount(credID, parsed.SignCnt)
	} else {
		_ = s.updateSignCount(credID, rec.SignCount)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"username": username,
		"uv":       isUserVerified(parsed.Flags),
	})
}

func (s *server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		writeErr(w, http.StatusBadRequest, "missing username")
		return
	}
	list := s.listPasskeys(username)
	out := make([]credentialSummary, 0, len(list))
	for _, item := range list {
		out = append(out, credentialSummary{
			CredentialID: item.CredentialID,
			Label:        item.Label,
			CreatedAt:    item.CreatedAt,
			LastUsedAt:   item.LastUsedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *server) handleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username     string `json:"username"`
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.CredentialID = strings.TrimSpace(req.CredentialID)
	if req.Username == "" || req.CredentialID == "" {
		writeErr(w, http.StatusBadRequest, "missing username/credential_id")
		return
	}
	deleted, err := s.deletePasskey(req.Username, req.CredentialID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"deleted": deleted,
	})
}

func (s *server) newState(kind, username, rpID, origin string, challenge []byte) (string, error) {
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(rpID) == "" || strings.TrimSpace(origin) == "" || len(challenge) < 16 {
		return "", errors.New("invalid state params")
	}
	id, err := randomBytes(24)
	if err != nil {
		return "", err
	}
	token := base64urlEncode(id)
	now := time.Now()
	st := pendingState{
		Kind:      kind,
		Username:  strings.TrimSpace(username),
		Challenge: append([]byte(nil), challenge...),
		RPID:      strings.TrimSpace(rpID),
		Origin:    strings.TrimSpace(origin),
		ExpiresAt: now.Add(s.stateTTL),
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.states == nil {
		s.states = map[string]pendingState{}
	}
	// Best-effort cleanup.
	for k, v := range s.states {
		if now.After(v.ExpiresAt.Add(s.stateMaxSkew)) {
			delete(s.states, k)
		}
	}
	s.states[token] = st
	return token, nil
}

func (s *server) takeState(token string) (pendingState, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return pendingState{}, false
	}
	now := time.Now()
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	st, ok := s.states[token]
	if ok {
		delete(s.states, token) // one-time
	}
	if !ok {
		return pendingState{}, false
	}
	if now.After(st.ExpiresAt.Add(s.stateMaxSkew)) {
		return pendingState{}, false
	}
	return st, true
}

