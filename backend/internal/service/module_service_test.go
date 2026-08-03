package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/modules"
	"github.com/stretchr/testify/require"
)

type moduleServiceMemoryStore struct {
	items        []modules.InstalledModule
	permissions  map[string][]modules.PermissionRecord
	statusCalls  []moduleStatusCall
	approveCalls []string
}

type moduleStatusCall struct {
	id     string
	status modules.ModuleStatus
	errMsg string
}

func (s *moduleServiceMemoryStore) ListInstalled(context.Context) ([]modules.InstalledModule, error) {
	return append([]modules.InstalledModule(nil), s.items...), nil
}

func (s *moduleServiceMemoryStore) GetInstalled(_ context.Context, id string) (*modules.InstalledModule, error) {
	for idx := range s.items {
		if s.items[idx].ID == id {
			item := s.items[idx]
			return &item, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *moduleServiceMemoryStore) SaveInstalled(_ context.Context, module modules.InstalledModule) error {
	for idx := range s.items {
		if s.items[idx].ID == module.ID {
			s.items[idx] = module
			return nil
		}
	}
	s.items = append(s.items, module)
	return nil
}

func (s *moduleServiceMemoryStore) SavePermissions(_ context.Context, moduleID string, permissions []modules.PermissionRecord) error {
	if s.permissions == nil {
		s.permissions = map[string][]modules.PermissionRecord{}
	}
	s.permissions[moduleID] = append([]modules.PermissionRecord(nil), permissions...)
	return nil
}

func (s *moduleServiceMemoryStore) ListPermissions(_ context.Context, moduleID string) ([]modules.PermissionRecord, error) {
	return append([]modules.PermissionRecord(nil), s.permissions[moduleID]...), nil
}

func (s *moduleServiceMemoryStore) ApprovePermissions(_ context.Context, moduleID string) error {
	s.approveCalls = append(s.approveCalls, moduleID)
	for idx := range s.permissions[moduleID] {
		s.permissions[moduleID][idx].Approved = true
		now := time.Now().UTC()
		s.permissions[moduleID][idx].ApprovedAt = &now
	}
	return nil
}

func (s *moduleServiceMemoryStore) SetStatus(_ context.Context, id string, status modules.ModuleStatus, errMsg string) error {
	s.statusCalls = append(s.statusCalls, moduleStatusCall{id: id, status: status, errMsg: errMsg})
	for idx := range s.items {
		if s.items[idx].ID == id {
			s.items[idx].Status = status
			s.items[idx].LastError = errMsg
		}
	}
	return nil
}

type fakeOutboundRuntime struct {
	started []string
	stopped []string
}

func (r *fakeOutboundRuntime) StartOutbound(_ context.Context, module modules.InstalledModule) error {
	r.started = append(r.started, module.ID)
	return nil
}

func (r *fakeOutboundRuntime) StopOutbound(_ context.Context, id string) error {
	r.stopped = append(r.stopped, id)
	return nil
}

type passthroughModuleVerifier struct{}

func (passthroughModuleVerifier) VerifyInstalled(_ context.Context, module modules.InstalledModule) (*modules.InstalledModule, error) {
	return &module, nil
}

type fakeMarketplaceInstaller struct {
	store     modules.Store
	archive   map[string]modules.InstalledModule
	installed []string
}

func (i *fakeMarketplaceInstaller) InstallArchive(ctx context.Context, archivePath string) (*modules.InstalledModule, error) {
	if i.archive == nil {
		return nil, errors.New("archive map is not configured")
	}
	name := filepath.Base(archivePath)
	module, ok := i.archive[name]
	if !ok {
		return nil, fmt.Errorf("unexpected archive %s", name)
	}
	module.Status = modules.ModuleStatusInstalled
	if module.Manifest.ID == "" {
		module.Manifest.ID = module.ID
	}
	if module.Manifest.Version == "" {
		module.Manifest.Version = module.Version
	}
	if module.Manifest.Type == "" {
		module.Manifest.Type = module.Type
	}
	i.installed = append(i.installed, module.ID)
	if i.store != nil {
		if err := i.store.SaveInstalled(ctx, module); err != nil {
			return nil, err
		}
		if err := i.store.SavePermissions(ctx, module.ID, nil); err != nil {
			return nil, err
		}
	}
	return &module, nil
}

func TestMarketplaceTimeoutDefaultsOverridesAndFallback(t *testing.T) {
	const wantDefault = 60 * time.Second

	t.Run("new service uses the default", func(t *testing.T) {
		svc := NewModuleService(nil)
		require.Equal(t, wantDefault, svc.effectiveMarketplaceTimeout())
	})

	t.Run("configured value overrides the default", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Modules.MarketplaceTimeoutSeconds = 90
		svc := ProvideModuleService(cfg, nil, nil, nil, nil, nil)
		require.Equal(t, 90*time.Second, svc.effectiveMarketplaceTimeout())
	})

	t.Run("zero-value service falls back to the default", func(t *testing.T) {
		var svc *ModuleService
		require.Equal(t, wantDefault, svc.effectiveMarketplaceTimeout())
	})
}

func TestDecodeMarketplaceRegistryPreservesLocalizedModuleText(t *testing.T) {
	registry, err := decodeMarketplaceRegistry([]byte(`{
		"modules": [{
			"id": "openai",
			"version": "0.1.1",
			"type": "provider",
			"name": "OpenAI OAuth Provider",
			"name_i18n": {
				"en": "OpenAI OAuth Provider",
				"zh-CN": "OpenAI OAuth 提供商"
			},
			"description": "OpenAI provider module",
			"description_i18n": {
				"en": "OpenAI provider module",
				"zh-CN": "OpenAI OAuth 提供商模块"
			},
			"downloadUrl": "https://example.test/lightbridge-module-openai-0.1.1.tar.zst",
			"core": ">=0.1.0 <0.2.0",
			"capabilities": ["provider.adapter"]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode marketplace registry: %v", err)
	}
	if got := registry.Modules[0].NameI18n["zh-CN"]; got != "OpenAI OAuth 提供商" {
		t.Fatalf("expected zh-CN module name, got %q", got)
	}
	if got := registry.Modules[0].DescriptionI18n["zh-CN"]; got != "OpenAI OAuth 提供商模块" {
		t.Fatalf("expected zh-CN module description, got %q", got)
	}
	if err := validateMarketplaceEntry(registry.Modules[0]); err != nil {
		t.Fatalf("localized marketplace entry should validate: %v", err)
	}
}

func TestAutoInstallManagedProviderModulesInstallsOnlyProviders(t *testing.T) {
	dir := t.TempDir()
	openAIArchive := writeModuleArchivePlaceholder(t, dir, "lightbridge-module-openai-0.1.1.tar.zst")
	anthropicArchive := writeModuleArchivePlaceholder(t, dir, "lightbridge-module-anthropic-oauth-0.1.0.tar.zst")
	proxyArchive := writeModuleArchivePlaceholder(t, dir, "lightbridge-module-proxy-0.1.0.tar.zst")
	registryPath := writeModuleMarketplaceRegistry(t, fmt.Sprintf(`{
		"modules": [{
			"id": "openai",
			"version": "0.1.1",
			"type": "provider",
			"name": "OpenAI OAuth Provider",
			"downloadUrl": %q,
			"core": ">=0.1.0 <0.2.0",
			"capabilities": ["provider.adapter"]
		}, {
			"id": "anthropic-oauth",
			"version": "0.1.0",
			"type": "provider",
			"name": "Anthropic OAuth Provider",
			"downloadUrl": %q,
			"core": ">=0.1.0 <0.2.0",
			"capabilities": ["provider.adapter"]
		}, {
			"id": "lightbridge.proxy",
			"version": "0.1.0",
			"type": "outbound",
			"name": "LightBridge Proxy",
			"downloadUrl": %q,
			"core": ">=0.1.0 <0.2.0",
			"capabilities": ["outbound.adapter"]
		}]
	}`, openAIArchive, anthropicArchive, proxyArchive))
	store := &moduleServiceMemoryStore{}
	installer := &fakeMarketplaceInstaller{
		store: store,
		archive: map[string]modules.InstalledModule{
			"lightbridge-module-openai-0.1.1.tar.zst": {
				ID:      "openai",
				Name:    "OpenAI OAuth Provider",
				Type:    modules.ModuleTypeProvider,
				Version: "0.1.1",
			},
			"lightbridge-module-anthropic-oauth-0.1.0.tar.zst": {
				ID:      "anthropic-oauth",
				Name:    "Anthropic OAuth Provider",
				Type:    modules.ModuleTypeProvider,
				Version: "0.1.0",
			},
		},
	}
	svc := NewModuleService(store)
	svc.installer = installer
	svc.marketplaceRegistryPath = registryPath

	if err := svc.AutoInstallManagedProviderModules(context.Background()); err != nil {
		t.Fatalf("auto install managed provider modules: %v", err)
	}
	if len(installer.installed) != 2 {
		t.Fatalf("expected two provider installs, got %#v", installer.installed)
	}
	if _, ok := store.permissions["lightbridge.proxy"]; ok {
		t.Fatalf("proxy module should not be auto-installed")
	}
	for _, id := range []string{"anthropic-oauth", "openai"} {
		item, err := store.GetInstalled(context.Background(), id)
		if err != nil {
			t.Fatalf("expected %s to be installed: %v", id, err)
		}
		if item.Status != modules.ModuleStatusEnabled {
			t.Fatalf("expected %s to be enabled, got %s", id, item.Status)
		}
	}
}

func TestUIManifestIncludesLocalizedRouteMenuAndAccountFormText(t *testing.T) {
	now := time.Now()
	store := &moduleServiceMemoryStore{items: []modules.InstalledModule{{
		ID:      "anthropic-oauth",
		Name:    "Anthropic OAuth Provider",
		Type:    modules.ModuleTypeProvider,
		Version: "0.1.0",
		Status:  modules.ModuleStatusEnabled,
		Manifest: modules.Manifest{
			NameI18n: modules.LocalizedText{
				"en":    "Anthropic OAuth Provider",
				"zh-CN": "Anthropic OAuth 提供商",
			},
			Frontend: &modules.FrontendSpec{
				Entry: "frontend/remoteEntry.js",
				Routes: []modules.FrontendRouteSpec{{
					Path:  "/admin/providers/anthropic-oauth-module",
					Title: "Anthropic OAuth Provider",
					TitleI18n: modules.LocalizedText{
						"zh-CN": "Anthropic OAuth 提供商",
					},
					ExposedModule: "./AnthropicOAuthProviderSettings",
				}},
				Menu: []modules.FrontendMenuSpec{{
					Title: "Anthropic OAuth Provider",
					TitleI18n: modules.LocalizedText{
						"zh-CN": "Anthropic OAuth 提供商",
					},
					Path: "/admin/providers/anthropic-oauth-module",
				}},
				AccountForms: []modules.FrontendAccountFormSpec{{
					ProviderID:    "anthropic-oauth",
					ExposedModule: "./AnthropicOAuthAccountForm",
				}},
				EntityPanels: []modules.FrontendEntityPanelSpec{{
					Entity:        "account",
					Title:         "Provider status",
					ExposedModule: "./AnthropicOAuthAccountPanel",
				}},
			},
		},
		InstalledAt: now,
	}}}
	svc := NewModuleService(store)

	items, err := svc.UIManifest(context.Background())
	if err != nil {
		t.Fatalf("load UI manifest: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one UI manifest item, got %d", len(items))
	}
	if got := items[0].ModuleNameI18n["zh-CN"]; got != "Anthropic OAuth 提供商" {
		t.Fatalf("expected localized module name, got %q", got)
	}
	if got := items[0].Routes[0].TitleI18n["zh-CN"]; got != "Anthropic OAuth 提供商" {
		t.Fatalf("expected localized route title, got %q", got)
	}
	if got := items[0].Menu[0].TitleI18n["zh-CN"]; got != "Anthropic OAuth 提供商" {
		t.Fatalf("expected localized menu title, got %q", got)
	}
	if got := items[0].AccountForms[0].ProviderNameI18n["zh-CN"]; got != "Anthropic OAuth 提供商" {
		t.Fatalf("expected localized account form provider name, got %q", got)
	}
	if got := items[0].EntityPanels[0].Entity; got != "account" {
		t.Fatalf("expected account entity panel, got %q", got)
	}
}

func TestStartEnabledModulesStartsOutboundModules(t *testing.T) {
	store := &moduleServiceMemoryStore{items: []modules.InstalledModule{{
		ID:      "lightbridge.proxy",
		Name:    "LightBridge Proxy",
		Type:    modules.ModuleTypeOutbound,
		Version: "0.1.0",
		Status:  modules.ModuleStatusEnabled,
	}}}
	runtime := &fakeOutboundRuntime{}
	svc := NewModuleService(store)
	svc.moduleVerifier = passthroughModuleVerifier{}
	svc.SetOutboundRuntime(runtime)

	if err := svc.StartEnabledModules(context.Background()); err != nil {
		t.Fatalf("start enabled modules: %v", err)
	}
	if len(runtime.started) != 1 || runtime.started[0] != "lightbridge.proxy" {
		t.Fatalf("expected outbound runtime to start lightbridge.proxy, got %#v", runtime.started)
	}
}

func TestDisableStopsOutboundRuntime(t *testing.T) {
	store := &moduleServiceMemoryStore{items: []modules.InstalledModule{{
		ID:      "lightbridge.proxy",
		Name:    "LightBridge Proxy",
		Type:    modules.ModuleTypeOutbound,
		Version: "0.1.0",
		Status:  modules.ModuleStatusEnabled,
	}}}
	runtime := &fakeOutboundRuntime{}
	svc := NewModuleService(store)
	svc.SetOutboundRuntime(runtime)

	if _, err := svc.Disable(context.Background(), "lightbridge.proxy"); err != nil {
		t.Fatalf("disable module: %v", err)
	}
	if len(runtime.stopped) != 1 || runtime.stopped[0] != "lightbridge.proxy" {
		t.Fatalf("expected outbound runtime to stop lightbridge.proxy, got %#v", runtime.stopped)
	}
}

func writeModuleMarketplaceRegistry(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return path
}

func writeModuleArchivePlaceholder(t *testing.T, dir string, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("placeholder"), 0o600); err != nil {
		t.Fatalf("write archive placeholder: %v", err)
	}
	return "file://" + path
}

func TestResolveEnabledAssetRestrictsVersionAndSymlinkEscapes(t *testing.T) {
	dataDir := t.TempDir()
	moduleID := "example.module"
	version := "1.0.0"
	installPath := modules.InstallDir(dataDir, moduleID, version)
	require.NoError(t, os.MkdirAll(installPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(installPath, "remoteEntry.js"), []byte("ok"), 0o644))

	store := &moduleServiceMemoryStore{items: []modules.InstalledModule{{
		ID: moduleID, Version: version, Status: modules.ModuleStatusEnabled, InstallPath: installPath,
	}}}
	svc := NewModuleService(store)
	svc.moduleDataDir = dataDir

	asset, err := svc.ResolveEnabledAsset(context.Background(), moduleID, version, "remoteEntry.js")
	require.NoError(t, err)
	expectedAsset, err := filepath.EvalSymlinks(filepath.Join(installPath, "remoteEntry.js"))
	require.NoError(t, err)
	require.Equal(t, expectedAsset, asset)

	_, err = svc.ResolveEnabledAsset(context.Background(), moduleID, "0.9.0", "remoteEntry.js")
	require.Error(t, err)
	_, err = svc.ResolveEnabledAsset(context.Background(), moduleID, version, "../remoteEntry.js")
	require.Error(t, err)

	outside := filepath.Join(dataDir, "outside.js")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o644))
	if err := os.Symlink(outside, filepath.Join(installPath, "escape.js")); err == nil {
		_, err = svc.ResolveEnabledAsset(context.Background(), moduleID, version, "escape.js")
		require.Error(t, err)
	}

	store.items[0].Status = modules.ModuleStatusDisabled
	_, err = svc.ResolveEnabledAsset(context.Background(), moduleID, version, "remoteEntry.js")
	require.Error(t, err)
}
