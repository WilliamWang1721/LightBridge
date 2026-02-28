package gateway

import (
	"context"
	"strings"
	"time"
)

// startEnabledModulesBestEffort attempts to start all enabled modules.
// It is intentionally best-effort: failures are ignored so a single bad module
// doesn't prevent other modules from starting.
func (s *Server) startEnabledModulesBestEffort() {
	if s == nil || s.store == nil || s.moduleMgr == nil {
		return
	}

	bg, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	mods, err := s.store.ListInstalledModules(bg)
	if err != nil {
		return
	}

	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		startCtx, startCancel := context.WithTimeout(context.Background(), 20*time.Second)
		_, startErr := s.moduleMgr.StartInstalledModule(startCtx, mod.ID)
		startCancel()
		if startErr == nil {
			continue
		}
		// Ignore already-running modules.
		if strings.Contains(strings.ToLower(startErr.Error()), "already running") {
			continue
		}
	}
}

