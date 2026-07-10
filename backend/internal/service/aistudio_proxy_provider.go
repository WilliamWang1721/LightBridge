package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/service/aistudio_proxy"
)

// ProvideAistudioProxyManager builds the per-account aistudio-api subprocess
// manager, wiring it to AccountService for credential persistence and to the
// configured data dir for the runtime checkout location.
func ProvideAistudioProxyManager(cfg *config.Config, accountService *AccountService) *aistudio_proxy.Manager {
	dataDir := "data"
	runtimeDir := ""
	if cfg != nil {
		if strings.TrimSpace(cfg.Modules.DataDir) != "" {
			dataDir = strings.TrimSpace(cfg.Modules.DataDir)
		}
		runtimeDir = strings.TrimSpace(cfg.Modules.AistudioProxy.RuntimeDir)
	}
	if runtimeDir == "" {
		runtimeDir = filepath.Join(dataDir, "aistudio-proxy")
	}
	pythonBin := ""
	if cfg != nil {
		pythonBin = strings.TrimSpace(cfg.Modules.AistudioProxy.PythonBin)
	}
	return aistudio_proxy.NewManager(
		aistudio_proxy.Config{
			DataDir:       dataDir,
			RuntimeDir:    runtimeDir,
			PythonBin:     pythonBin,
			HealthTimeout: 0, // use manager default
		},
		&aistudioProxyCredUpdater{svc: accountService},
		&aistudioProxyCredReader{svc: accountService},
	)
}

// aistudioProxyCredUpdater implements aistudio_proxy.AccountCredentialUpdater.
type aistudioProxyCredUpdater struct {
	svc *AccountService
}

func (u *aistudioProxyCredUpdater) UpdateCredentials(ctx context.Context, accountID int64, credentials map[string]any) error {
	if u == nil || u.svc == nil {
		return nil
	}
	return u.svc.UpdateCredentials(ctx, accountID, credentials)
}

// aistudioProxyCredReader implements aistudio_proxy.AccountCredentialReader.
type aistudioProxyCredReader struct {
	svc *AccountService
}

func (r *aistudioProxyCredReader) GetCredentials(ctx context.Context, accountID int64) (map[string]any, error) {
	if r == nil || r.svc == nil {
		return nil, nil
	}
	return r.svc.GetCredentials(ctx, accountID)
}
