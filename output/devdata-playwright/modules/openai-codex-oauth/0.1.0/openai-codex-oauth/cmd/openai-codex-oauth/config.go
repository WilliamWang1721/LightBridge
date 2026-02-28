package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type config struct {
	BaseURL           string   `json:"base_url"`
	NearExpiryMinutes int      `json:"near_expiry_minutes"`
	ClientVersion     string   `json:"client_version"`
	UserAgent         string   `json:"user_agent"`
	BetaFeatures      string   `json:"beta_features"`
	WebSearchEligible bool     `json:"web_search_eligible"`
	Models            []string `json:"models"`
}

type configFile struct {
	BaseURL           *string  `json:"base_url"`
	NearExpiryMinutes *int     `json:"near_expiry_minutes"`
	ClientVersion     *string  `json:"client_version"`
	UserAgent         *string  `json:"user_agent"`
	BetaFeatures      *string  `json:"beta_features"`
	WebSearchEligible *bool    `json:"web_search_eligible"`
	Models            []string `json:"models"`
}

func defaultConfig() config {
	return config{
		BaseURL:           "https://chatgpt.com/backend-api/codex",
		NearExpiryMinutes: 20,
		ClientVersion:     "0.101.0",
		UserAgent:         "codex_cli_rs/0.101.0 (Windows 10.0.26100; x86_64) WindowsTerminal",
		BetaFeatures:      "powershell_utf8",
		WebSearchEligible: true,
		Models:            []string{"gpt-5-codex", "gpt-5-codex-mini", "gpt-5.1-codex", "gpt-5.1-codex-mini", "gpt-5.2-codex"},
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
	if tmp.NearExpiryMinutes != nil && *tmp.NearExpiryMinutes >= 0 {
		cfg.NearExpiryMinutes = *tmp.NearExpiryMinutes
	}
	if tmp.ClientVersion != nil && strings.TrimSpace(*tmp.ClientVersion) != "" {
		cfg.ClientVersion = strings.TrimSpace(*tmp.ClientVersion)
	}
	if tmp.UserAgent != nil && strings.TrimSpace(*tmp.UserAgent) != "" {
		cfg.UserAgent = strings.TrimSpace(*tmp.UserAgent)
	}
	if tmp.BetaFeatures != nil && strings.TrimSpace(*tmp.BetaFeatures) != "" {
		cfg.BetaFeatures = strings.TrimSpace(*tmp.BetaFeatures)
	}
	if tmp.WebSearchEligible != nil {
		cfg.WebSearchEligible = *tmp.WebSearchEligible
	}
	if tmp.Models != nil {
		cfg.Models = tmp.Models
	}
	return cfg, nil
}
