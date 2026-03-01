package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"lightbridge/internal/types"
)

const voucherConfigSettingKey = "voucher_config_v1"

type voucherModelMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type voucherAppConfig struct {
	KeyID         string                `json:"key_id,omitempty"`
	ModelMappings []voucherModelMapping `json:"model_mappings,omitempty"`
}

type voucherConfig struct {
	BaseURL          string                      `json:"base_url"`
	BaseOrigin       string                      `json:"base_origin,omitempty"`
	DefaultInterface string                      `json:"default_interface,omitempty"`
	Apps             map[string]voucherAppConfig `json:"apps"`
}

func defaultVoucherConfig() voucherConfig {
	return voucherConfig{
		BaseURL:          "",
		BaseOrigin:       "",
		DefaultInterface: types.ProtocolOpenAI,
		Apps: map[string]voucherAppConfig{
			"codex":         {},
			"claude-code":   {},
			"opencode":      {},
			"gemini-cli":    {},
			"cherry-studio": {},
		},
	}
}

func normalizeVoucherConfig(in voucherConfig) voucherConfig {
	out := in
	out.BaseOrigin = strings.TrimRight(strings.TrimSpace(out.BaseOrigin), "/")
	if out.BaseOrigin == "" {
		out.BaseOrigin = strings.TrimRight(strings.TrimSpace(out.BaseURL), "/")
	}
	out.BaseURL = out.BaseOrigin // keep compatibility for old readers.
	out.DefaultInterface = strings.TrimSpace(out.DefaultInterface)
	if out.DefaultInterface == "" {
		out.DefaultInterface = types.ProtocolOpenAI
	}

	if out.Apps == nil {
		out.Apps = map[string]voucherAppConfig{}
	}
	// Ensure default apps exist.
	for k := range defaultVoucherConfig().Apps {
		if _, ok := out.Apps[k]; !ok {
			out.Apps[k] = voucherAppConfig{}
		}
	}

	for appID, app := range out.Apps {
		appIDNorm := strings.ToLower(strings.TrimSpace(appID))
		if appIDNorm == "" {
			delete(out.Apps, appID)
			continue
		}
		if appIDNorm != appID {
			delete(out.Apps, appID)
		}

		app.KeyID = strings.TrimSpace(app.KeyID)
		// Cap mappings and normalize entries.
		if len(app.ModelMappings) > 200 {
			app.ModelMappings = app.ModelMappings[:200]
		}
		next := make([]voucherModelMapping, 0, len(app.ModelMappings))
		seen := map[string]struct{}{}
		for _, m := range app.ModelMappings {
			from := strings.TrimSpace(m.From)
			to := strings.TrimSpace(m.To)
			if from == "" || to == "" {
				continue
			}
			key := from + "->" + to
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			next = append(next, voucherModelMapping{From: from, To: to})
		}
		app.ModelMappings = next
		out.Apps[appIDNorm] = app
	}

	return out
}

func (s *Server) getVoucherConfig(ctx context.Context) voucherConfig {
	const ttl = 2 * time.Second

	now := time.Now()
	s.voucherMu.RLock()
	if s.voucherCfgOnce && now.Sub(s.voucherCfgAt) < ttl {
		cfg := s.voucherCfg
		s.voucherMu.RUnlock()
		return cfg
	}
	s.voucherMu.RUnlock()

	s.voucherMu.Lock()
	defer s.voucherMu.Unlock()
	if s.voucherCfgOnce && now.Sub(s.voucherCfgAt) < ttl {
		return s.voucherCfg
	}

	cfg := defaultVoucherConfig()
	raw, ok, err := s.store.GetSetting(ctx, voucherConfigSettingKey)
	if err == nil && ok && strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	cfg = normalizeVoucherConfig(cfg)
	s.voucherCfg = cfg
	s.voucherCfgAt = now
	s.voucherCfgOnce = true
	return cfg
}

func (s *Server) setVoucherConfig(ctx context.Context, cfg voucherConfig) error {
	cfg = normalizeVoucherConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := s.store.SetSetting(ctx, voucherConfigSettingKey, string(b)); err != nil {
		return err
	}
	s.voucherMu.Lock()
	s.voucherCfg = cfg
	s.voucherCfgAt = time.Now()
	s.voucherCfgOnce = true
	s.voucherMu.Unlock()
	return nil
}

func (s *Server) mapModelForApp(ctx context.Context, appID, requestedModel string) string {
	appID = strings.ToLower(strings.TrimSpace(appID))
	model := strings.TrimSpace(requestedModel)
	if appID == "" || model == "" {
		return requestedModel
	}
	cfg := s.getVoucherConfig(ctx)
	app := cfg.Apps[appID]
	for _, m := range app.ModelMappings {
		if strings.TrimSpace(m.From) == model && strings.TrimSpace(m.To) != "" {
			return strings.TrimSpace(m.To)
		}
	}
	return requestedModel
}
