package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type passkeyStore struct {
	Users map[string][]passkeyRecord `json:"users"`
}

type passkeyRecord struct {
	CredentialID  string `json:"credential_id"`
	PublicKeyCOSE string `json:"public_key_cose"`
	SignCount     uint32 `json:"sign_count"`
	UserHandle    string `json:"user_handle"`
	Label         string `json:"label"`
	CreatedAt     string `json:"created_at"`
	LastUsedAt    string `json:"last_used_at"`
}

func (s *server) loadStore() error {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	b, err := os.ReadFile(s.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.store = passkeyStore{Users: map[string][]passkeyRecord{}}
			return err
		}
		return err
	}
	var st passkeyStore
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if st.Users == nil {
		st.Users = map[string][]passkeyRecord{}
	}
	s.store = st
	return nil
}

func (s *server) saveStoreLocked() error {
	if s.store.Users == nil {
		s.store.Users = map[string][]passkeyRecord{}
	}
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.storePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.storePath)
}

func (s *server) listPasskeys(username string) []passkeyRecord {
	username = strings.TrimSpace(username)
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	out := append([]passkeyRecord(nil), s.store.Users[username]...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func (s *server) anyPasskeyCount() int {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	n := 0
	for _, list := range s.store.Users {
		n += len(list)
	}
	return n
}

func (s *server) getByCredentialID(credID string) (username string, rec passkeyRecord, ok bool) {
	credID = strings.TrimSpace(credID)
	if credID == "" {
		return "", passkeyRecord{}, false
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	for u, list := range s.store.Users {
		for _, item := range list {
			if item.CredentialID == credID {
				return u, item, true
			}
		}
	}
	return "", passkeyRecord{}, false
}

func (s *server) addPasskey(username string, rec passkeyRecord) error {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(rec.CredentialID) == "" || strings.TrimSpace(rec.PublicKeyCOSE) == "" {
		return errors.New("invalid passkey record")
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	for u, list := range s.store.Users {
		for _, item := range list {
			if item.CredentialID == rec.CredentialID {
				if u == username {
					return errors.New("credential already registered")
				}
				return errors.New("credential id already belongs to another user")
			}
		}
	}
	if s.store.Users == nil {
		s.store.Users = map[string][]passkeyRecord{}
	}
	if strings.TrimSpace(rec.CreatedAt) == "" {
		rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	s.store.Users[username] = append(s.store.Users[username], rec)
	return s.saveStoreLocked()
}

func (s *server) deletePasskey(username, credID string) (bool, error) {
	username = strings.TrimSpace(username)
	credID = strings.TrimSpace(credID)
	if username == "" || credID == "" {
		return false, errors.New("missing username/credential_id")
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	list := s.store.Users[username]
	if len(list) == 0 {
		return false, nil
	}
	out := list[:0]
	deleted := false
	for _, item := range list {
		if item.CredentialID == credID {
			deleted = true
			continue
		}
		out = append(out, item)
	}
	if !deleted {
		return false, nil
	}
	if len(out) == 0 {
		delete(s.store.Users, username)
	} else {
		s.store.Users[username] = out
	}
	return true, s.saveStoreLocked()
}

func (s *server) updateSignCount(credID string, signCount uint32) error {
	credID = strings.TrimSpace(credID)
	if credID == "" {
		return errors.New("missing credential id")
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	for u, list := range s.store.Users {
		changed := false
		for i := range list {
			if list[i].CredentialID != credID {
				continue
			}
			list[i].SignCount = signCount
			list[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339)
			changed = true
			break
		}
		if changed {
			s.store.Users[u] = list
			return s.saveStoreLocked()
		}
	}
	return errors.New("credential not found")
}

