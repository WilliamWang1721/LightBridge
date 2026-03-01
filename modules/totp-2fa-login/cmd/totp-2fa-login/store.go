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

type totpStore struct {
	Users map[string][]totpDevice `json:"users"`
}

type totpDevice struct {
	DeviceID     string `json:"device_id"`
	Label        string `json:"label"`
	Secret       string `json:"secret"`
	CreatedAt    string `json:"created_at"`
	LastUsedAt   string `json:"last_used_at"`
	LastUsedStep int64  `json:"last_used_step"`
}

func (s *server) loadStore() error {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	b, err := os.ReadFile(s.storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.store = totpStore{Users: map[string][]totpDevice{}}
			return err
		}
		return err
	}
	var st totpStore
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if st.Users == nil {
		st.Users = map[string][]totpDevice{}
	}
	s.store = st
	return nil
}

func (s *server) saveStoreLocked() error {
	if s.store.Users == nil {
		s.store.Users = map[string][]totpDevice{}
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

func (s *server) listDevices(username string) []totpDevice {
	username = strings.TrimSpace(username)
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	out := append([]totpDevice(nil), s.store.Users[username]...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func (s *server) addDevice(username string, dev totpDevice) error {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(dev.DeviceID) == "" || strings.TrimSpace(dev.Secret) == "" {
		return errors.New("invalid device")
	}
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.store.Users == nil {
		s.store.Users = map[string][]totpDevice{}
	}
	list := s.store.Users[username]
	for _, item := range list {
		if item.DeviceID == dev.DeviceID {
			return errors.New("device id exists")
		}
	}
	if strings.TrimSpace(dev.CreatedAt) == "" {
		dev.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	s.store.Users[username] = append(list, dev)
	return s.saveStoreLocked()
}

func (s *server) deleteDevice(username, deviceID string) (bool, error) {
	username = strings.TrimSpace(username)
	deviceID = strings.TrimSpace(deviceID)
	if username == "" || deviceID == "" {
		return false, errors.New("missing username/device_id")
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
		if item.DeviceID == deviceID {
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

func (s *server) verifyForUser(username, code string, now time.Time) (totpDevice, int64, bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return totpDevice{}, 0, false, errors.New("missing username")
	}
	code = cleanCode(code)
	if code == "" {
		return totpDevice{}, 0, false, errInvalidCode
	}

	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	list := s.store.Users[username]
	if len(list) == 0 {
		return totpDevice{}, 0, false, nil
	}

	matched := -1
	matchedStep := int64(0)
	for i := range list {
		ok, step := verifyTOTP(list[i].Secret, code, now)
		if !ok {
			continue
		}
		if step <= list[i].LastUsedStep {
			continue
		}
		matched = i
		matchedStep = step
		break
	}
	if matched < 0 {
		return totpDevice{}, 0, false, nil
	}
	list[matched].LastUsedStep = matchedStep
	list[matched].LastUsedAt = now.UTC().Format(time.RFC3339)
	s.store.Users[username] = list
	if err := s.saveStoreLocked(); err != nil {
		return totpDevice{}, 0, false, err
	}
	return list[matched], matchedStep, true, nil
}
