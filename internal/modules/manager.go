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
	processes map[string]*exec.Cmd
}

func NewManager(st *store.Store, dataDir string) *Manager {
	return &Manager{
		store:     st,
		dataDir:   dataDir,
		client:    &http.Client{Timeout: 5 * time.Second},
		processes: map[string]*exec.Cmd{},
	}
}

func (m *Manager) StartInstalledModule(ctx context.Context, moduleID string) (*types.ModuleRuntime, error) {
	installed, err := m.store.GetInstalledModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if installed == nil {
		return nil, fmt.Errorf("module %s not installed", moduleID)
	}
	if !installed.Enabled {
		return nil, fmt.Errorf("module %s is disabled", moduleID)
	}

	manifest, manifestPath, err := m.loadManifest(installed.InstallPath)
	if err != nil {
		return nil, err
	}
	ep, err := resolveEntrypoint(manifest.Entrypoints, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	httpPort, err := findFreePort()
	if err != nil {
		return nil, err
	}
	grpcPort, err := findFreePort()
	if err != nil {
		return nil, err
	}
	moduleDataDir := filepath.Join(m.dataDir, "module_data", manifest.ID)
	if err := os.MkdirAll(moduleDataDir, 0o755); err != nil {
		return nil, err
	}
	configPath := filepath.Join(moduleDataDir, "config.json")
	if manifest.ConfigDefaults == nil {
		manifest.ConfigDefaults = map[string]any{}
	}
	configBytes, _ := json.MarshalIndent(manifest.ConfigDefaults, "", "  ")
	if err := os.WriteFile(configPath, configBytes, 0o644); err != nil {
		return nil, err
	}

	cmdPath := ep.Command
	if !filepath.IsAbs(cmdPath) {
		cmdPath = filepath.Join(filepath.Dir(manifestPath), cmdPath)
	}
	cmd := exec.CommandContext(ctx, cmdPath, ep.Args...)
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
		return nil, err
	}

	m.mu.Lock()
	m.processes[moduleID] = cmd
	m.mu.Unlock()

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
	m.mu.Lock()
	cmd, ok := m.processes[moduleID]
	if ok {
		delete(m.processes, moduleID)
	}
	m.mu.Unlock()
	if ok && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			_, _ = cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		}
	}
	_ = m.store.DeleteModuleRuntime(ctx, moduleID)
	return nil
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
			case types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC:
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
	for _, svc := range services {
		if svc.Kind != "provider" {
			continue
		}
		endpoint := ""
		switch svc.Protocol {
		case types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC:
			endpoint = fmt.Sprintf("http://127.0.0.1:%d", httpPort)
		case types.ProtocolGRPCChat:
			endpoint = fmt.Sprintf("127.0.0.1:%d", grpcPort)
		}
		for _, alias := range svc.ExposeProviderAliases {
			if strings.TrimSpace(alias) == "" {
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
			if err := m.store.UpsertProvider(ctx, provider); err != nil {
				return err
			}
		}
	}
	return nil
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
