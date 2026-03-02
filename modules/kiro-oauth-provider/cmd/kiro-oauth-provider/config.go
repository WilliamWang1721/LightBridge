package main

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
)

type config struct {
	BaseURL                string   `json:"base_url"`
	AmazonQBaseURL         string   `json:"amazonq_base_url"`
	AuthServiceEndpoint    string   `json:"auth_service_endpoint"`
	AWSOIDCEndpoint        string   `json:"aws_oidc_endpoint"`
	SocialRefreshURL       string   `json:"social_refresh_url"`
	IDCRefreshURL          string   `json:"idc_refresh_url"`
	BuilderIDStartURL      string   `json:"builder_id_start_url"`
	Region                 string   `json:"region"`
	NearExpiryMinutes      int      `json:"near_expiry_minutes"`
	SelectionStrategy      string   `json:"selection_strategy"`
	QuotaLowPercent        float64  `json:"quota_low_percent"`
	TimeoutSeconds         int      `json:"timeout_seconds"`
	TokenRefreshTimeoutSec int      `json:"token_refresh_timeout_sec"`
	Models                 []string `json:"models"`
}

type configFile struct {
	BaseURL                *string  `json:"base_url"`
	AmazonQBaseURL         *string  `json:"amazonq_base_url"`
	AuthServiceEndpoint    *string  `json:"auth_service_endpoint"`
	AWSOIDCEndpoint        *string  `json:"aws_oidc_endpoint"`
	SocialRefreshURL       *string  `json:"social_refresh_url"`
	IDCRefreshURL          *string  `json:"idc_refresh_url"`
	BuilderIDStartURL      *string  `json:"builder_id_start_url"`
	Region                 *string  `json:"region"`
	NearExpiryMinutes      *int     `json:"near_expiry_minutes"`
	SelectionStrategy      *string  `json:"selection_strategy"`
	QuotaLowPercent        *float64 `json:"quota_low_percent"`
	TimeoutSeconds         *int     `json:"timeout_seconds"`
	TokenRefreshTimeoutSec *int     `json:"token_refresh_timeout_sec"`
	Models                 []string `json:"models"`
}

var kiroDefaultModels = []string{
	"claude-haiku-4-5",
	"claude-opus-4-6",
	"claude-sonnet-4-6",
	"claude-opus-4-5",
	"claude-opus-4-5-20251101",
	"claude-sonnet-4-5",
	"claude-sonnet-4-5-20250929",
	"claude-sonnet-4-20250514",
	"claude-3-7-sonnet-20250219",
}

func copyDefaultModels() []string {
	out := make([]string, len(kiroDefaultModels))
	copy(out, kiroDefaultModels)
	return out
}

func mergeKiroModels(configured []string) []string {
	if len(configured) == 0 {
		return copyDefaultModels()
	}
	seen := make(map[string]string, len(kiroDefaultModels)+len(configured))
	for _, m := range kiroDefaultModels {
		id := strings.TrimSpace(m)
		if id == "" {
			continue
		}
		seen[strings.ToLower(id)] = id
	}
	for _, m := range configured {
		id := strings.TrimSpace(m)
		if id == "" {
			continue
		}
		seen[strings.ToLower(id)] = id
	}
	out := make([]string, 0, len(seen))
	for _, id := range seen {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func normalizeSelectionStrategy(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "round_robin", "round-robin", "rr":
		return "round_robin"
	default:
		return "fill_first"
	}
}

func defaultConfig() config {
	return config{
		BaseURL:                "https://q.{{region}}.amazonaws.com/generateAssistantResponse",
		AmazonQBaseURL:         "https://q.{{region}}.amazonaws.com/generateAssistantResponse",
		AuthServiceEndpoint:    "https://prod.us-east-1.auth.desktop.kiro.dev",
		AWSOIDCEndpoint:        "https://oidc.{{region}}.amazonaws.com",
		SocialRefreshURL:       "https://prod.{{region}}.auth.desktop.kiro.dev/refreshToken",
		IDCRefreshURL:          "https://oidc.{{region}}.amazonaws.com/token",
		BuilderIDStartURL:      "https://view.awsapps.com/start",
		Region:                 "us-east-1",
		NearExpiryMinutes:      30,
		SelectionStrategy:      "fill_first",
		QuotaLowPercent:        10,
		TimeoutSeconds:         120,
		TokenRefreshTimeoutSec: 15,
		Models:                 copyDefaultModels(),
	}
}

func loadConfig(path string) (config, error) {
	cfg := defaultConfig()
	path = strings.TrimSpace(path)
	if path == "" {
		return cfg, errors.New("LIGHTBRIDGE_CONFIG_PATH is empty; using defaults")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	var tmp configFile
	if err := json.Unmarshal(b, &tmp); err != nil {
		return cfg, err
	}
	if tmp.BaseURL != nil && strings.TrimSpace(*tmp.BaseURL) != "" {
		cfg.BaseURL = strings.TrimSpace(*tmp.BaseURL)
	}
	if tmp.AmazonQBaseURL != nil && strings.TrimSpace(*tmp.AmazonQBaseURL) != "" {
		cfg.AmazonQBaseURL = strings.TrimSpace(*tmp.AmazonQBaseURL)
	}
	if tmp.AuthServiceEndpoint != nil && strings.TrimSpace(*tmp.AuthServiceEndpoint) != "" {
		cfg.AuthServiceEndpoint = strings.TrimSpace(*tmp.AuthServiceEndpoint)
	}
	if tmp.AWSOIDCEndpoint != nil && strings.TrimSpace(*tmp.AWSOIDCEndpoint) != "" {
		cfg.AWSOIDCEndpoint = strings.TrimSpace(*tmp.AWSOIDCEndpoint)
	}
	if tmp.SocialRefreshURL != nil && strings.TrimSpace(*tmp.SocialRefreshURL) != "" {
		cfg.SocialRefreshURL = strings.TrimSpace(*tmp.SocialRefreshURL)
	}
	if tmp.IDCRefreshURL != nil && strings.TrimSpace(*tmp.IDCRefreshURL) != "" {
		cfg.IDCRefreshURL = strings.TrimSpace(*tmp.IDCRefreshURL)
	}
	if tmp.BuilderIDStartURL != nil && strings.TrimSpace(*tmp.BuilderIDStartURL) != "" {
		cfg.BuilderIDStartURL = strings.TrimSpace(*tmp.BuilderIDStartURL)
	}
	if tmp.Region != nil && strings.TrimSpace(*tmp.Region) != "" {
		cfg.Region = strings.TrimSpace(*tmp.Region)
	}
	if tmp.NearExpiryMinutes != nil && *tmp.NearExpiryMinutes >= 0 {
		cfg.NearExpiryMinutes = *tmp.NearExpiryMinutes
	}
	if tmp.SelectionStrategy != nil {
		cfg.SelectionStrategy = normalizeSelectionStrategy(*tmp.SelectionStrategy)
	}
	if tmp.QuotaLowPercent != nil && *tmp.QuotaLowPercent >= 0 && *tmp.QuotaLowPercent <= 100 {
		cfg.QuotaLowPercent = *tmp.QuotaLowPercent
	}
	if tmp.TimeoutSeconds != nil && *tmp.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = *tmp.TimeoutSeconds
	}
	if tmp.TokenRefreshTimeoutSec != nil && *tmp.TokenRefreshTimeoutSec > 0 {
		cfg.TokenRefreshTimeoutSec = *tmp.TokenRefreshTimeoutSec
	}
	if tmp.Models != nil {
		cfg.Models = mergeKiroModels(tmp.Models)
	}
	return cfg, nil
}

func renderRegionTemplate(tpl, region string) string {
	v := strings.TrimSpace(tpl)
	if v == "" {
		return ""
	}
	if strings.TrimSpace(region) == "" {
		region = "us-east-1"
	}
	return strings.ReplaceAll(v, "{{region}}", strings.TrimSpace(region))
}
