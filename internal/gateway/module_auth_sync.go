package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lightbridge/internal/types"
)

// syncModuleAuthFromProviderConfig imports provider-bound credentials/accounts into
// OAuth helper modules before handling a request. This keeps module runtime state
// aligned with the currently selected provider.
func (s *Server) syncModuleAuthFromProviderConfig(ctx context.Context, provider *types.Provider) (string, error) {
	if s == nil || provider == nil {
		return "", nil
	}
	raw := strings.TrimSpace(provider.ConfigJSON)
	if raw == "" {
		return "", nil
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return "", nil
	}

	moduleID := strings.ToLower(strings.TrimSpace(moduleConfigString(cfg["module_id"])))
	if moduleID == "" {
		return "", nil
	}

	switch moduleID {
	case codexOAuthModuleID:
		return s.syncCodexProviderAuth(ctx, provider, cfg)
	case kiroOAuthModuleID:
		return s.syncKiroProviderAuth(ctx, provider, cfg)
	default:
		return "", nil
	}
}

func (s *Server) syncCodexProviderAuth(ctx context.Context, provider *types.Provider, cfg map[string]any) (string, error) {
	accessToken := strings.TrimSpace(moduleConfigString(cfg["access_token"]))
	refreshToken := strings.TrimSpace(moduleConfigString(cfg["refresh_token"]))
	if accessToken == "" && refreshToken == "" {
		return "", nil
	}

	payload := map[string]any{}
	if accessToken != "" {
		payload["access_token"] = accessToken
	}
	if refreshToken != "" {
		payload["refresh_token"] = refreshToken
	}
	if accountID := strings.TrimSpace(moduleConfigString(cfg["account_id"])); accountID != "" {
		payload["account_id"] = accountID
	}
	if email := strings.TrimSpace(moduleConfigString(cfg["email"])); email != "" {
		payload["email"] = email
	}
	if expired := strings.TrimSpace(moduleConfigString(cfg["expired"])); expired != "" {
		payload["expired"] = expired
	}

	if _, _, err := s.postProviderModuleJSON(ctx, provider.Endpoint, "/auth/import", payload); err != nil {
		return "", err
	}
	return codexOAuthModuleID, nil
}

func (s *Server) syncKiroProviderAuth(ctx context.Context, provider *types.Provider, cfg map[string]any) (string, error) {
	account := selectKiroAccountFromConfig(cfg)
	if len(account) == 0 {
		return "", nil
	}

	payload := map[string]any{}
	for _, field := range []string{
		"account_id",
		"display_name",
		"group_name",
		"auth_method",
		"social_provider",
		"access_token",
		"refresh_token",
		"profile_arn",
		"client_id",
		"client_secret",
		"region",
		"idc_region",
		"expires_at",
	} {
		if v := strings.TrimSpace(moduleConfigString(account[field])); v != "" {
			payload[field] = v
		}
	}
	payload["enabled"] = moduleConfigBoolDefault(account["enabled"], true)
	if _, _, err := s.postProviderModuleJSON(ctx, provider.Endpoint, "/auth/import", payload); err != nil {
		return "", err
	}

	accountID := strings.TrimSpace(moduleConfigString(account["account_id"], account["id"]))
	if accountID != "" {
		_, _, _ = s.postProviderModuleJSON(ctx, provider.Endpoint, "/auth/accounts/activate", map[string]any{
			"account_id": accountID,
		})
	}
	return kiroOAuthModuleID, nil
}

func selectKiroAccountFromConfig(cfg map[string]any) map[string]any {
	activeAccountID := strings.TrimSpace(moduleConfigString(cfg["active_account_id"]))
	accounts, _ := cfg["accounts"].([]any)

	var firstEnabled map[string]any
	for _, row := range accounts {
		item, _ := row.(map[string]any)
		if len(item) == 0 {
			continue
		}
		if !moduleConfigBoolDefault(item["enabled"], true) {
			continue
		}
		id := strings.TrimSpace(moduleConfigString(item["id"], item["account_id"]))
		if id == "" {
			continue
		}
		if activeAccountID != "" && strings.EqualFold(id, activeAccountID) {
			return kiroAccountPayloadFromStatus(item)
		}
		if len(firstEnabled) == 0 {
			firstEnabled = item
		}
	}
	if len(firstEnabled) > 0 {
		return kiroAccountPayloadFromStatus(firstEnabled)
	}

	accessToken := strings.TrimSpace(moduleConfigString(cfg["access_token"]))
	refreshToken := strings.TrimSpace(moduleConfigString(cfg["refresh_token"]))
	if accessToken == "" && refreshToken == "" {
		return nil
	}

	out := map[string]any{
		"account_id":      strings.TrimSpace(moduleConfigString(cfg["account_id"])),
		"display_name":    strings.TrimSpace(moduleConfigString(cfg["display_name"])),
		"group_name":      strings.TrimSpace(moduleConfigString(cfg["group_name"])),
		"auth_method":     strings.TrimSpace(moduleConfigString(cfg["auth_method"])),
		"social_provider": strings.TrimSpace(moduleConfigString(cfg["social_provider"])),
		"access_token":    accessToken,
		"refresh_token":   refreshToken,
		"profile_arn":     strings.TrimSpace(moduleConfigString(cfg["profile_arn"])),
		"client_id":       strings.TrimSpace(moduleConfigString(cfg["client_id"])),
		"client_secret":   strings.TrimSpace(moduleConfigString(cfg["client_secret"])),
		"region":          strings.TrimSpace(moduleConfigString(cfg["region"])),
		"idc_region":      strings.TrimSpace(moduleConfigString(cfg["idc_region"])),
		"expires_at":      strings.TrimSpace(moduleConfigString(cfg["expires_at"])),
		"enabled":         true,
	}
	return out
}

func kiroAccountPayloadFromStatus(in map[string]any) map[string]any {
	out := map[string]any{
		"account_id":      strings.TrimSpace(moduleConfigString(in["id"], in["account_id"])),
		"display_name":    strings.TrimSpace(moduleConfigString(in["display_name"])),
		"group_name":      strings.TrimSpace(moduleConfigString(in["group_name"])),
		"auth_method":     strings.TrimSpace(moduleConfigString(in["auth_method"])),
		"social_provider": strings.TrimSpace(moduleConfigString(in["social_provider"])),
		"access_token":    strings.TrimSpace(moduleConfigString(in["access_token"])),
		"refresh_token":   strings.TrimSpace(moduleConfigString(in["refresh_token"])),
		"profile_arn":     strings.TrimSpace(moduleConfigString(in["profile_arn"])),
		"client_id":       strings.TrimSpace(moduleConfigString(in["client_id"])),
		"client_secret":   strings.TrimSpace(moduleConfigString(in["client_secret"])),
		"region":          strings.TrimSpace(moduleConfigString(in["region"])),
		"idc_region":      strings.TrimSpace(moduleConfigString(in["idc_region"])),
		"expires_at":      strings.TrimSpace(moduleConfigString(in["expires_at"])),
		"enabled":         moduleConfigBoolDefault(in["enabled"], true),
	}
	return out
}

func (s *Server) postProviderModuleJSON(ctx context.Context, endpoint, reqPath string, payload map[string]any) (int, []byte, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return 0, nil, fmt.Errorf("module endpoint is empty")
	}
	u, err := joinUpstreamURL(endpoint, reqPath)
	if err != nil {
		return 0, nil, err
	}

	var body []byte
	if payload == nil {
		payload = map[string]any{}
	}
	body, _ = json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	httpc := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return resp.StatusCode, respBody, fmt.Errorf("module auth sync failed (%s): %s", reqPath, msg)
	}
	return resp.StatusCode, respBody, nil
}

func (s *Server) resetModuleAuthCacheBestEffort(provider *types.Provider, moduleID string) {
	if s == nil || provider == nil {
		return
	}
	moduleID = strings.ToLower(strings.TrimSpace(moduleID))
	if moduleID != codexOAuthModuleID && moduleID != kiroOAuthModuleID {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, _ = s.postProviderModuleJSON(ctx, provider.Endpoint, "/auth/reset", map[string]any{})
}

func moduleConfigString(values ...any) string {
	for _, raw := range values {
		switch v := raw.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case json.Number:
			if s := strings.TrimSpace(v.String()); s != "" {
				return s
			}
		case float64:
			if v != 0 {
				return fmt.Sprintf("%.0f", v)
			}
		case float32:
			if v != 0 {
				return fmt.Sprintf("%.0f", v)
			}
		case int:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		case int64:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		case int32:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		case uint:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		case uint64:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		case uint32:
			if v != 0 {
				return fmt.Sprintf("%d", v)
			}
		}
	}
	return ""
}

func moduleConfigBoolDefault(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		default:
			return def
		}
	case float64:
		return t != 0
	case float32:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case int32:
		return t != 0
	default:
		return def
	}
}
