package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ListenAddr      string
	DataDir         string
	DatabasePath    string
	ModuleIndexURL  string
	CookieSecretKey string
	ModelTagAliases map[string]string
}

func DefaultConfig() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, err
	}
	dataDir := filepath.Join(configDir, "LightBridge")
	if v := os.Getenv("LIGHTBRIDGE_DATA_DIR"); v != "" {
		dataDir = v
	}
	if abs, err := filepath.Abs(dataDir); err == nil && abs != "" {
		dataDir = abs
	}
	addr := "127.0.0.1:3210"
	if v := os.Getenv("LIGHTBRIDGE_ADDR"); v != "" {
		addr = v
	}
	// Default to a remote Marketplace source (static index.json on raw.githubusercontent.com).
	// Keep "local" available for development/offline fallback via env override.
	moduleIndex := "https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json"
	if v := os.Getenv("LIGHTBRIDGE_MODULE_INDEX"); v != "" {
		moduleIndex = v
	}
	secret := os.Getenv("LIGHTBRIDGE_COOKIE_SECRET")
	modelTagAliases := loadModelTagAliasesFromEnv()

	return Config{
		ListenAddr:      addr,
		DataDir:         dataDir,
		DatabasePath:    filepath.Join(dataDir, "lightbridge.db"),
		ModuleIndexURL:  moduleIndex,
		CookieSecretKey: secret,
		ModelTagAliases: modelTagAliases,
	}, nil
}

func loadModelTagAliasesFromEnv() map[string]string {
	raw := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_MODEL_TAG_ALIASES"))
	if raw == "" {
		return nil
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		log.Printf("config: LIGHTBRIDGE_MODEL_TAG_ALIASES parse failed: %v", err)
		return nil
	}
	out := make(map[string]string, len(parsed))
	for k, v := range parsed {
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.ToLower(strings.TrimSpace(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
