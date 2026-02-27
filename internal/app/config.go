package app

import (
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddr      string
	DataDir         string
	DatabasePath    string
	ModuleIndexURL  string
	CookieSecretKey string
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
	addr := "127.0.0.1:3210"
	if v := os.Getenv("LIGHTBRIDGE_ADDR"); v != "" {
		addr = v
	}
	moduleIndex := "local"
	if v := os.Getenv("LIGHTBRIDGE_MODULE_INDEX"); v != "" {
		moduleIndex = v
	}
	secret := os.Getenv("LIGHTBRIDGE_COOKIE_SECRET")

	return Config{
		ListenAddr:      addr,
		DataDir:         dataDir,
		DatabasePath:    filepath.Join(dataDir, "lightbridge.db"),
		ModuleIndexURL:  moduleIndex,
		CookieSecretKey: secret,
	}, nil
}
