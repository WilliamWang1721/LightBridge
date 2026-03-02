package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"lightbridge/internal/store"
	"lightbridge/internal/types"
)

type Manager struct {
	store   *store.Store
	dataDir string
	client  *http.Client

	mu        sync.Mutex
	processes map[string]*moduleProcess
}

func NewManager(st *store.Store, dataDir string) *Manager {
	return &Manager{
		store:     st,
		dataDir:   dataDir,
		client:    &http.Client{Timeout: 5 * time.Second},
		processes: map[string]*moduleProcess{},
	}
}

func (m *Manager) ModuleDataDir(moduleID string) string {
	return filepath.Join(m.dataDir, "module_data", moduleID)
}

func (m *Manager) ModuleConfigPath(moduleID string) string {
	return filepath.Join(m.ModuleDataDir(moduleID), "config.json")
}

func (m *Manager) ModuleInstallRoot(moduleID string) string {
	return filepath.Join(m.dataDir, "modules", moduleID)
}

func (m *Manager) ReadModuleConfig(moduleID string, defaults map[string]any) (map[string]any, error) {
	configPath := m.ModuleConfigPath(moduleID)
	if err := ensureJSONFile(configPath, defaults); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

func (m *Manager) WriteModuleConfig(moduleID string, cfg map[string]any) error {
	if cfg == nil {
		cfg = map[string]any{}
	}
	dir := m.ModuleDataDir(moduleID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.ModuleConfigPath(moduleID), b, 0o644)
}

func (m *Manager) LoadInstalledManifest(ctx context.Context, moduleID string) (*types.ModuleManifest, error) {
	installed, err := m.store.GetInstalledModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if installed == nil {
		return nil, fmt.Errorf("module %s not installed", moduleID)
	}
	manifest, _, err := m.loadManifest(installed.InstallPath)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

type moduleProcess struct {
	cmd     *exec.Cmd
	done    chan struct{}
	aliases []string
}

func (m *Manager) StartInstalledModule(ctx context.Context, moduleID string) (*types.ModuleRuntime, error) {
	m.mu.Lock()
	if _, ok := m.processes[moduleID]; ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("module %s already running", moduleID)
	}
	proc := &moduleProcess{done: make(chan struct{})}
	m.processes[moduleID] = proc
	m.mu.Unlock()

	installed, err := m.store.GetInstalledModule(ctx, moduleID)
	if err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}
	if installed == nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, fmt.Errorf("module %s not installed", moduleID)
	}
	if !installed.Enabled {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, fmt.Errorf("module %s is disabled", moduleID)
	}

	manifest, manifestPath, err := m.loadManifest(installed.InstallPath)
	if err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}
	if manifest.ID != installed.ID {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, fmt.Errorf("manifest id mismatch: %s", manifest.ID)
	}
	ep, err := resolveEntrypoint(manifest.Entrypoints, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}

	httpPort, err := findFreePort()
	if err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}
	grpcPort, err := findFreePort()
	if err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}
	moduleDataDir := filepath.Join(m.dataDir, "module_data", manifest.ID)
	if err := os.MkdirAll(moduleDataDir, 0o755); err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}
	configPath := filepath.Join(moduleDataDir, "config.json")
	if err := ensureJSONFile(configPath, manifest.ConfigDefaults); err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}

	cmdPath := ep.Command
	if !filepath.IsAbs(cmdPath) {
		cmdPath = filepath.Join(filepath.Dir(manifestPath), cmdPath)
	}
	cmd := exec.Command(cmdPath, ep.Args...)
	cmd.Dir = filepath.Dir(manifestPath)
	cmd.Env = append(os.Environ(),
		"LIGHTBRIDGE_MODULE_ID="+manifest.ID,
		"LIGHTBRIDGE_DATA_DIR="+moduleDataDir,
		"LIGHTBRIDGE_CONFIG_PATH="+configPath,
		fmt.Sprintf("LIGHTBRIDGE_HTTP_PORT=%d", httpPort),
		fmt.Sprintf("LIGHTBRIDGE_GRPC_PORT=%d", grpcPort),
		"LIGHTBRIDGE_LOG_LEVEL=info",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
		return nil, err
	}

	aliases := collectExposedAliases(manifest.Services)
	if helper := helperProviderAliases(moduleID); len(helper) > 0 {
		seen := map[string]struct{}{}
		for _, a := range aliases {
			seen[strings.TrimSpace(a)] = struct{}{}
		}
		for _, a := range helper {
			a = strings.TrimSpace(a)
			if a == "" {
				continue
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			aliases = append(aliases, a)
		}
	}
	m.mu.Lock()
	proc.cmd = cmd
	proc.aliases = aliases
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		bg := context.Background()
		_ = m.store.DeleteModuleRuntime(bg, moduleID)
		for _, alias := range aliases {
			_ = m.store.UpdateProviderHealth(bg, alias, "down")
		}
		m.mu.Lock()
		delete(m.processes, moduleID)
		m.mu.Unlock()
		close(proc.done)
	}()

	if err := m.waitHealth(ctx, manifest.Services, httpPort, grpcPort); err != nil {
		_ = m.StopModule(context.Background(), moduleID)
		return nil, err
	}

	now := time.Now().UTC()
	rt := &types.ModuleRuntime{
		ModuleID:    moduleID,
		PID:         cmd.Process.Pid,
		HTTPPort:    httpPort,
		GRPCPort:    grpcPort,
		Status:      "running",
		LastStartAt: now,
	}
	if err := m.store.SaveModuleRuntime(ctx, *rt); err != nil {
		return nil, err
	}

	if err := m.registerProviderAliases(ctx, manifest.Services, moduleID, httpPort, grpcPort); err != nil {
		return nil, err
	}
	return rt, nil
}

func (m *Manager) StopModule(ctx context.Context, moduleID string) error {
	var (
		proc    *moduleProcess
		aliases []string
	)

	m.mu.Lock()
	proc = m.processes[moduleID]
	if proc != nil {
		aliases = append([]string(nil), proc.aliases...)
	}
	m.mu.Unlock()

	for _, alias := range aliases {
		_ = m.store.UpdateProviderHealth(ctx, alias, "down")
	}

	if proc != nil && proc.cmd != nil && proc.cmd.Process != nil {
		_ = proc.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-proc.done:
		case <-time.After(2 * time.Second):
			_ = proc.cmd.Process.Kill()
			select {
			case <-proc.done:
			case <-time.After(2 * time.Second):
			}
		}
	}

	_ = m.store.DeleteModuleRuntime(ctx, moduleID)
	return nil
}

func (m *Manager) StopAll(ctx context.Context) {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.StopModule(ctx, id)
	}
}

func (m *Manager) loadManifest(installPath string) (*types.ModuleManifest, string, error) {
	manifestPath := filepath.Join(installPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}
	var manifest types.ModuleManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, "", err
	}
	return &manifest, manifestPath, nil
}

func ensureJSONFile(path string, defaults map[string]any) error {
	if _, err := os.Stat(path); err == nil {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var tmp any
		if err := json.Unmarshal(b, &tmp); err != nil {
			return fmt.Errorf("invalid json in %s: %w", filepath.Base(path), err)
		}
		if tmp != nil {
			if _, ok := tmp.(map[string]any); !ok {
				return fmt.Errorf("%s must be a JSON object", filepath.Base(path))
			}
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if defaults == nil {
		defaults = map[string]any{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(defaults, "", "  ")
	return os.WriteFile(path, b, 0o644)
}

func resolveEntrypoint(entrypoints map[string]types.ManifestEntrypoint, goos, goarch string) (types.ManifestEntrypoint, error) {
	keys := []string{goos + "/" + goarch, goos, "default"}
	for _, key := range keys {
		if ep, ok := entrypoints[key]; ok {
			if strings.TrimSpace(ep.Command) == "" {
				return types.ManifestEntrypoint{}, fmt.Errorf("entrypoint %s command is empty", key)
			}
			return ep, nil
		}
	}
	return types.ManifestEntrypoint{}, errors.New("no compatible entrypoint")
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func collectExposedAliases(services []types.ManifestService) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, svc := range services {
		if svc.Kind != "provider" {
			continue
		}
		for _, alias := range svc.ExposeProviderAliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if _, ok := seen[alias]; ok {
				continue
			}
			seen[alias] = struct{}{}
			out = append(out, alias)
		}
	}
	return out
}

func helperProviderAliases(moduleID string) []string {
	id := strings.ToLower(strings.TrimSpace(moduleID))
	switch id {
	case "kiro-oauth-provider":
		// The Kiro module is treated as an OAuth helper in the UI, but it still exposes
		// an OpenAI-compatible chat endpoint for the `kiro` provider.
		return []string{"kiro"}
	default:
		return nil
	}
}

func (m *Manager) waitHealth(ctx context.Context, services []types.ManifestService, httpPort, grpcPort int) error {
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		allHealthy := true
		for _, svc := range services {
			if svc.Kind != "provider" {
				continue
			}
			switch svc.Protocol {
			case types.ProtocolOpenAI, types.ProtocolOpenAIResponses, types.ProtocolGemini, types.ProtocolAnthropic, types.ProtocolAzureOpenAI, types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC, types.ProtocolCodex:
				path := svc.Health.Path
				if path == "" {
					path = "/health"
				}
				if err := m.httpHealth(httpPort, path); err != nil {
					allHealthy = false
				}
			case types.ProtocolGRPCChat:
				if err := grpcHealth(grpcPort); err != nil {
					allHealthy = false
				}
			}
		}
		if allHealthy {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("module health check timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (m *Manager) httpHealth(port int, path string) error {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	resp, err := m.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health status %d", resp.StatusCode)
	}
	return nil
}

func grpcHealth(port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("127.0.0.1:%d", port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()
	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("grpc health status: %s", resp.Status.String())
	}
	return nil
}

func (m *Manager) registerProviderAliases(ctx context.Context, services []types.ManifestService, moduleID string, httpPort, grpcPort int) error {
	skipCreate := skipAutoProviderAliasRegistration(moduleID)
	for _, svc := range services {
		if svc.Kind != "provider" {
			continue
		}
		endpoint := ""
		switch svc.Protocol {
		case types.ProtocolOpenAI, types.ProtocolOpenAIResponses, types.ProtocolGemini, types.ProtocolAnthropic, types.ProtocolAzureOpenAI, types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC, types.ProtocolCodex:
			endpoint = fmt.Sprintf("http://127.0.0.1:%d", httpPort)
		case types.ProtocolGRPCChat:
			endpoint = fmt.Sprintf("127.0.0.1:%d", grpcPort)
		}
		aliasList := append([]string(nil), svc.ExposeProviderAliases...)
		if helper := helperProviderAliases(moduleID); len(helper) > 0 {
			aliasList = append(aliasList, helper...)
		}
		seenAlias := map[string]struct{}{}
		for _, alias := range aliasList {
			if strings.TrimSpace(alias) == "" {
				continue
			}
			alias = strings.TrimSpace(alias)
			if _, ok := seenAlias[alias]; ok {
				continue
			}
			seenAlias[alias] = struct{}{}

			existing, err := m.store.GetProvider(ctx, alias)
			if err != nil {
				return err
			}
			if skipCreate && existing == nil {
				// Helper modules should not auto-create provider aliases on startup, but we still
				// refresh the endpoint/protocol for existing module providers because ports can
				// change across restarts.
				continue
			}

			provider := types.Provider{
				ID:         alias,
				Type:       types.ProviderTypeModule,
				Protocol:   svc.Protocol,
				Endpoint:   endpoint,
				ConfigJSON: "{}",
				Enabled:    true,
				Health:     "healthy",
			}

			// Preserve user intent where possible:
			// - If the provider already exists and is not a module provider, do not clobber it.
			// - If the provider exists as a module provider, preserve Enabled + ConfigJSON, but still
			//   refresh Endpoint/Protocol (ports can change across restarts).
			if existing != nil {
				if strings.TrimSpace(existing.Type) != "" && existing.Type != types.ProviderTypeModule {
					continue
				}
				if strings.TrimSpace(existing.DisplayName) != "" {
					provider.DisplayName = existing.DisplayName
				}
				if strings.TrimSpace(existing.GroupName) != "" {
					provider.GroupName = existing.GroupName
				}
				provider.Enabled = existing.Enabled
				if strings.TrimSpace(existing.ConfigJSON) != "" {
					provider.ConfigJSON = existing.ConfigJSON
				}
			}

			if err := m.store.UpsertProvider(ctx, provider); err != nil {
				return err
			}
		}
	}
	return nil
}

func skipAutoProviderAliasRegistration(moduleID string) bool {
	id := strings.ToLower(strings.TrimSpace(moduleID))
	switch id {
	case "kiro-oauth-provider":
		// Kiro module is an OAuth/account helper. It should not auto-create
		// a provider alias on module startup.
		return true
	default:
		return false
	}
}

func (m *Manager) StartEnabledModules(ctx context.Context) error {
	modules, err := m.store.ListInstalledModules(ctx)
	if err != nil {
		return err
	}
	for _, mod := range modules {
		if !mod.Enabled {
			continue
		}
		if _, err := m.StartInstalledModule(ctx, mod.ID); err != nil {
			return fmt.Errorf("start module %s: %w", mod.ID, err)
		}
	}
	return nil
}
