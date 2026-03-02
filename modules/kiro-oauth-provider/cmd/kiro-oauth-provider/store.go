package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type quotaCache struct {
	UsedPercent      float64 `json:"used_percent"`
	RemainingPercent float64 `json:"remaining_percent"`
	ResetAt          string  `json:"reset_at,omitempty"`
	FetchedAt        string  `json:"fetched_at,omitempty"`
}

type account struct {
	ID             string      `json:"id"`
	DisplayName    string      `json:"display_name,omitempty"`
	GroupName      string      `json:"group_name,omitempty"`
	Enabled        bool        `json:"enabled"`
	AuthMethod     string      `json:"auth_method,omitempty"`
	SocialProvider string      `json:"social_provider,omitempty"`
	AccessToken    string      `json:"access_token,omitempty"`
	RefreshToken   string      `json:"refresh_token,omitempty"`
	ProfileARN     string      `json:"profile_arn,omitempty"`
	ClientID       string      `json:"client_id,omitempty"`
	ClientSecret   string      `json:"client_secret,omitempty"`
	Region         string      `json:"region,omitempty"`
	IDCRegion      string      `json:"idc_region,omitempty"`
	ExpiresAt      string      `json:"expires_at,omitempty"`
	LastRefresh    string      `json:"last_refresh,omitempty"`
	LastError      string      `json:"last_error,omitempty"`
	CooldownUntil  string      `json:"cooldown_until,omitempty"`
	CreatedAt      string      `json:"created_at,omitempty"`
	UpdatedAt      string      `json:"updated_at,omitempty"`
	Quota          *quotaCache `json:"quota,omitempty"`
}

type accountStoreSnapshot struct {
	Accounts          []*account `json:"accounts"`
	ActiveAccountID   string     `json:"active_account_id,omitempty"`
	SelectionStrategy string     `json:"selection_strategy,omitempty"`
	RoundRobinCursor  int        `json:"round_robin_cursor,omitempty"`
}

type accountStore struct {
	mu    sync.RWMutex
	file  string
	state accountStoreSnapshot
}

func newAccountStore(filePath string) (*accountStore, error) {
	s := &accountStore{file: filePath}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *accountStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = accountStoreSnapshot{
		Accounts:          []*account{},
		SelectionStrategy: "fill_first",
	}

	b, err := os.ReadFile(s.file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	var snap accountStoreSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	if snap.Accounts == nil {
		snap.Accounts = []*account{}
	}
	snap.SelectionStrategy = normalizeSelectionStrategy(snap.SelectionStrategy)
	for _, a := range snap.Accounts {
		if a == nil {
			continue
		}
		a.ID = strings.TrimSpace(a.ID)
		if a.ID == "" {
			a.ID = newUUID()
		}
		if strings.TrimSpace(a.CreatedAt) == "" {
			a.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if strings.TrimSpace(a.UpdatedAt) == "" {
			a.UpdatedAt = a.CreatedAt
		}
	}
	s.state = snap
	return nil
}

func (s *accountStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file)
}

func cloneAccount(in *account) *account {
	if in == nil {
		return nil
	}
	cp := *in
	if in.Quota != nil {
		q := *in.Quota
		cp.Quota = &q
	}
	return &cp
}

func (s *accountStore) listAccounts() []*account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*account, 0, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		if a == nil {
			continue
		}
		out = append(out, cloneAccount(a))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Enabled != out[j].Enabled {
			return out[i].Enabled
		}
		return strings.ToLower(out[i].DisplayName+out[i].ID) < strings.ToLower(out[j].DisplayName+out[j].ID)
	})
	return out
}

func (s *accountStore) activeAccountID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.state.ActiveAccountID)
}

func (s *accountStore) setSelectionStrategy(strategy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.SelectionStrategy = normalizeSelectionStrategy(strategy)
	return s.saveLocked()
}

func (s *accountStore) addOrUpdateAccount(a *account, setActive bool) (*account, error) {
	if a == nil {
		return nil, errors.New("nil account")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	a.ID = strings.TrimSpace(a.ID)
	if a.ID == "" {
		a.ID = newUUID()
	}
	a.DisplayName = strings.TrimSpace(a.DisplayName)
	a.GroupName = strings.TrimSpace(a.GroupName)
	a.AuthMethod = strings.TrimSpace(a.AuthMethod)
	a.Region = strings.TrimSpace(a.Region)
	a.IDCRegion = strings.TrimSpace(a.IDCRegion)
	if a.Region == "" {
		a.Region = "us-east-1"
	}
	if a.IDCRegion == "" {
		a.IDCRegion = a.Region
	}
	if strings.TrimSpace(a.CreatedAt) == "" {
		a.CreatedAt = now
	}
	a.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	updated := false
	for i, cur := range s.state.Accounts {
		if cur == nil {
			continue
		}
		if strings.EqualFold(cur.ID, a.ID) {
			if strings.TrimSpace(a.CreatedAt) == "" {
				a.CreatedAt = cur.CreatedAt
			}
			s.state.Accounts[i] = cloneAccount(a)
			updated = true
			break
		}
	}
	if !updated {
		s.state.Accounts = append(s.state.Accounts, cloneAccount(a))
	}

	if setActive || strings.TrimSpace(s.state.ActiveAccountID) == "" {
		s.state.ActiveAccountID = a.ID
	}
	if strings.TrimSpace(s.state.SelectionStrategy) == "" {
		s.state.SelectionStrategy = "fill_first"
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return cloneAccount(a), nil
}

func (s *accountStore) getAccount(id string) (*account, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cur := range s.state.Accounts {
		if cur == nil {
			continue
		}
		if strings.EqualFold(cur.ID, id) {
			return cloneAccount(cur), true
		}
	}
	return nil, false
}

func (s *accountStore) updateAccount(id string, fn func(a *account) error) (*account, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("missing account id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, cur := range s.state.Accounts {
		if cur == nil || !strings.EqualFold(cur.ID, id) {
			continue
		}
		cp := cloneAccount(cur)
		if err := fn(cp); err != nil {
			return nil, err
		}
		cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.state.Accounts[i] = cp
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
		return cloneAccount(cp), nil
	}
	return nil, errors.New("account not found")
}

func (s *accountStore) setAccountEnabled(id string, enabled bool) error {
	_, err := s.updateAccount(id, func(a *account) error {
		a.Enabled = enabled
		return nil
	})
	return err
}

func (s *accountStore) deleteAccount(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("missing account id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*account, 0, len(s.state.Accounts))
	found := false
	for _, cur := range s.state.Accounts {
		if cur == nil {
			continue
		}
		if strings.EqualFold(cur.ID, id) {
			found = true
			continue
		}
		out = append(out, cur)
	}
	if !found {
		return nil
	}
	s.state.Accounts = out
	if strings.EqualFold(strings.TrimSpace(s.state.ActiveAccountID), id) {
		s.state.ActiveAccountID = ""
		for _, a := range out {
			if a != nil && a.Enabled {
				s.state.ActiveAccountID = a.ID
				break
			}
		}
	}
	return s.saveLocked()
}

func (s *accountStore) setActiveAccount(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("missing account id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cur := range s.state.Accounts {
		if cur == nil {
			continue
		}
		if strings.EqualFold(cur.ID, id) {
			s.state.ActiveAccountID = cur.ID
			return s.saveLocked()
		}
	}
	return errors.New("account not found")
}

func parseRFC3339OrZero(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func accountInCooldown(a *account, now time.Time) bool {
	if a == nil {
		return false
	}
	t := parseRFC3339OrZero(a.CooldownUntil)
	if t.IsZero() {
		return false
	}
	return t.After(now)
}

func (s *accountStore) selectAccount(strategy string) (*account, string) {
	strategy = normalizeSelectionStrategy(strategy)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	candidates := make([]*account, 0, len(s.state.Accounts))
	for _, a := range s.state.Accounts {
		if a == nil || !a.Enabled {
			continue
		}
		if accountInCooldown(a, now) {
			continue
		}
		if strings.TrimSpace(a.AccessToken) == "" && strings.TrimSpace(a.RefreshToken) == "" {
			continue
		}
		candidates = append(candidates, a)
	}
	if len(candidates) == 0 {
		return nil, "no_healthy_account"
	}

	pick := candidates[0]
	if strategy == "round_robin" {
		idx := s.state.RoundRobinCursor % len(candidates)
		if idx < 0 {
			idx = 0
		}
		pick = candidates[idx]
		s.state.RoundRobinCursor = (idx + 1) % len(candidates)
		_ = s.saveLocked()
		return cloneAccount(pick), ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		li := -1.0
		if candidates[i].Quota != nil {
			li = candidates[i].Quota.RemainingPercent
		}
		lj := -1.0
		if candidates[j].Quota != nil {
			lj = candidates[j].Quota.RemainingPercent
		}
		if li == lj {
			return parseRFC3339OrZero(candidates[i].UpdatedAt).After(parseRFC3339OrZero(candidates[j].UpdatedAt))
		}
		return li > lj
	})
	pick = candidates[0]
	return cloneAccount(pick), ""
}

func (s *accountStore) markCooldown(id string, until time.Time, reason string) error {
	_, err := s.updateAccount(id, func(a *account) error {
		a.CooldownUntil = until.UTC().Format(time.RFC3339)
		a.LastError = strings.TrimSpace(reason)
		return nil
	})
	return err
}

func (s *accountStore) clearCooldown(id string) error {
	_, err := s.updateAccount(id, func(a *account) error {
		a.CooldownUntil = ""
		return nil
	})
	return err
}

func (s *accountStore) setQuotaCache(id string, q *quotaCache) error {
	_, err := s.updateAccount(id, func(a *account) error {
		a.Quota = q
		if q != nil {
			a.CooldownUntil = ""
		}
		return nil
	})
	return err
}

func (s *accountStore) anyAccountEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.state.Accounts {
		if a != nil && a.Enabled {
			return true
		}
	}
	return false
}

func (s *accountStore) sanitizedAccounts() []map[string]any {
	items := s.listAccounts()
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		if a == nil {
			continue
		}
		row := map[string]any{
			"id":                a.ID,
			"display_name":      a.DisplayName,
			"group_name":        a.GroupName,
			"enabled":           a.Enabled,
			"auth_method":       a.AuthMethod,
			"social_provider":   a.SocialProvider,
			"region":            a.Region,
			"idc_region":        a.IDCRegion,
			"expires_at":        a.ExpiresAt,
			"last_refresh":      a.LastRefresh,
			"last_error":        a.LastError,
			"cooldown_until":    a.CooldownUntil,
			"profile_arn":       a.ProfileARN,
			"has_access_token":  strings.TrimSpace(a.AccessToken) != "",
			"has_refresh_token": strings.TrimSpace(a.RefreshToken) != "",
		}
		if strings.TrimSpace(a.AccessToken) != "" {
			row["access_token"] = a.AccessToken
		}
		if strings.TrimSpace(a.RefreshToken) != "" {
			row["refresh_token"] = a.RefreshToken
		}
		if a.Quota != nil {
			row["quota"] = map[string]any{
				"used_percent":      a.Quota.UsedPercent,
				"remaining_percent": a.Quota.RemainingPercent,
				"reset_at":          a.Quota.ResetAt,
				"fetched_at":        a.Quota.FetchedAt,
			}
		}
		out = append(out, row)
	}
	return out
}
