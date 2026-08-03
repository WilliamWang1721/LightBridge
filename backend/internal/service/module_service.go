package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/modules"
	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

type ModuleService struct {
	store                   modules.Store
	installer               modules.Installer
	moduleVerifier          modules.InstalledVerifier
	providerRuntime         modules.ProviderRuntime
	outboundRuntime         modules.OutboundRuntime
	providerRegistry        *modules.ProviderRegistry
	moduleDataDir           string
	marketplaceRegistryPath string
	marketplaceRegistryURL  string
	marketplaceTimeout      time.Duration
}

const defaultMarketplaceTimeout = 60 * time.Second

func (s *ModuleService) SetOutboundRuntime(runtime modules.OutboundRuntime) {
	if s == nil {
		return
	}
	s.outboundRuntime = runtime
}

func NewModuleService(store modules.Store) *ModuleService {
	return &ModuleService{store: store, moduleDataDir: "data", marketplaceTimeout: defaultMarketplaceTimeout}
}

func ProvideModuleService(
	cfg *config.Config,
	store modules.Store,
	installer modules.Installer,
	providerRuntime modules.ProviderRuntime,
	outboundRuntime modules.OutboundRuntime,
	providerRegistry *modules.ProviderRegistry,
) *ModuleService {
	dataDir := "data"
	if cfg != nil && strings.TrimSpace(cfg.Modules.DataDir) != "" {
		dataDir = strings.TrimSpace(cfg.Modules.DataDir)
	}
	marketplaceTimeout := defaultMarketplaceTimeout
	if cfg != nil && cfg.Modules.MarketplaceTimeoutSeconds > 0 {
		marketplaceTimeout = time.Duration(cfg.Modules.MarketplaceTimeoutSeconds) * time.Second
	}
	marketplaceRegistryPath := ""
	marketplaceRegistryURL := ""
	if cfg != nil {
		marketplaceRegistryPath = strings.TrimSpace(cfg.Modules.MarketplaceRegistryPath)
		marketplaceRegistryURL = strings.TrimSpace(cfg.Modules.MarketplaceRegistryURL)
	}
	var moduleVerifier modules.InstalledVerifier
	if verifier, ok := installer.(modules.InstalledVerifier); ok {
		moduleVerifier = verifier
	}
	svc := &ModuleService{
		store:                   store,
		installer:               installer,
		moduleVerifier:          moduleVerifier,
		providerRuntime:         providerRuntime,
		outboundRuntime:         outboundRuntime,
		providerRegistry:        providerRegistry,
		moduleDataDir:           dataDir,
		marketplaceRegistryPath: marketplaceRegistryPath,
		marketplaceRegistryURL:  marketplaceRegistryURL,
		marketplaceTimeout:      marketplaceTimeout,
	}
	return svc
}

type ModuleUIManifestItem struct {
	ModuleID       string                    `json:"moduleId"`
	ModuleName     string                    `json:"moduleName"`
	ModuleNameI18n modules.LocalizedText     `json:"moduleNameI18n,omitempty"`
	Version        string                    `json:"version"`
	RemoteEntry    string                    `json:"remoteEntry"`
	Routes         []ModuleUIRouteSpec       `json:"routes,omitempty"`
	Menu           []ModuleUIMenuSpec        `json:"menu,omitempty"`
	AccountForms   []ModuleUIAccountFormSpec `json:"accountForms,omitempty"`
	EntityPanels   []ModuleUIEntityPanelSpec `json:"entityPanels,omitempty"`
}

type ModuleUIRouteSpec struct {
	Path          string                `json:"path"`
	Title         string                `json:"title"`
	TitleI18n     modules.LocalizedText `json:"title_i18n,omitempty"`
	RemoteEntry   string                `json:"remoteEntry"`
	ExposedModule string                `json:"exposedModule"`
	RequiresAdmin bool                  `json:"requiresAdmin,omitempty"`
}

type ModuleUIMenuSpec struct {
	Title     string                `json:"title"`
	TitleI18n modules.LocalizedText `json:"title_i18n,omitempty"`
	Path      string                `json:"path"`
	Group     string                `json:"group,omitempty"`
	Order     int                   `json:"order,omitempty"`
}

type ModuleUIAccountFormSpec struct {
	ProviderID       string                `json:"providerId"`
	ProviderName     string                `json:"providerName,omitempty"`
	ProviderNameI18n modules.LocalizedText `json:"providerNameI18n,omitempty"`
	ModuleID         string                `json:"moduleId,omitempty"`
	ModuleName       string                `json:"moduleName,omitempty"`
	ModuleNameI18n   modules.LocalizedText `json:"moduleNameI18n,omitempty"`
	ModuleVersion    string                `json:"moduleVersion,omitempty"`
	RemoteEntry      string                `json:"remoteEntry"`
	ExposedModule    string                `json:"exposedModule"`
}

type ModuleUIEntityPanelSpec struct {
	Entity        string                `json:"entity"`
	Title         string                `json:"title"`
	TitleI18n     modules.LocalizedText `json:"title_i18n,omitempty"`
	ModuleID      string                `json:"moduleId"`
	ModuleVersion string                `json:"moduleVersion"`
	RemoteEntry   string                `json:"remoteEntry"`
	ExposedModule string                `json:"exposedModule"`
	RequiresAdmin bool                  `json:"requiresAdmin,omitempty"`
	Order         int                   `json:"order,omitempty"`
}

type ModuleProviderAdapterStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ModulePermissionStatus struct {
	Permissions []modules.PermissionRecord `json:"permissions"`
	Approved    bool                       `json:"approved"`
}

type ModuleMarketplaceResult struct {
	Modules []ModuleMarketplaceEntry `json:"modules"`
}

type ModuleMarketplaceEntry struct {
	ID               string                `json:"id"`
	Version          string                `json:"version"`
	Type             modules.ModuleType    `json:"type"`
	Name             string                `json:"name,omitempty"`
	NameI18n         modules.LocalizedText `json:"name_i18n,omitempty"`
	Description      string                `json:"description,omitempty"`
	DescriptionI18n  modules.LocalizedText `json:"description_i18n,omitempty"`
	DownloadURL      string                `json:"downloadUrl"`
	SHA256           string                `json:"sha256,omitempty"`
	Signature        string                `json:"signature,omitempty"`
	Core             string                `json:"core"`
	Capabilities     []modules.Capability  `json:"capabilities,omitempty"`
	Permissions      modules.PermissionSet `json:"permissions,omitempty"`
	InstalledStatus  modules.ModuleStatus  `json:"installedStatus,omitempty"`
	InstalledVersion string                `json:"installedVersion,omitempty"`
}

func (s *ModuleService) ListInstalled(ctx context.Context) ([]modules.InstalledModule, error) {
	return s.store.ListInstalled(ctx)
}

func (s *ModuleService) ProviderAdapters(context.Context) ([]ModuleProviderAdapterStatus, error) {
	if s.providerRegistry == nil {
		return []ModuleProviderAdapterStatus{}, nil
	}
	ids := s.providerRegistry.IDs()
	sort.Strings(ids)
	result := make([]ModuleProviderAdapterStatus, 0, len(ids))
	for _, id := range ids {
		result = append(result, ModuleProviderAdapterStatus{
			ID:     id,
			Status: "registered",
		})
	}
	return result, nil
}

func (s *ModuleService) StartEnabledModules(ctx context.Context) error {
	if s == nil || (s.providerRuntime == nil && s.outboundRuntime == nil) {
		return nil
	}
	installed, err := s.store.ListInstalled(ctx)
	if err != nil {
		return err
	}
	var failures []string
	for _, item := range installed {
		if item.Status != modules.ModuleStatusEnabled {
			continue
		}
		verified, err := s.verifiedModuleForStart(ctx, item)
		if err != nil {
			_ = s.store.SetStatus(ctx, item.ID, modules.ModuleStatusFailed, err.Error())
			failures = append(failures, item.ID+": "+err.Error())
			continue
		}
		if err := s.startModuleRuntime(ctx, *verified); err != nil {
			_ = s.store.SetStatus(ctx, item.ID, modules.ModuleStatusFailed, err.Error())
			failures = append(failures, item.ID+": "+err.Error())
		}
	}
	if len(failures) > 0 {
		return infraerrors.ServiceUnavailable("MODULE_RUNTIME_RESTORE_FAILED", "one or more enabled modules failed to start").WithCause(errors.New(strings.Join(failures, "; ")))
	}
	return nil
}

func (s *ModuleService) InstallArchive(ctx context.Context, archivePath string) (*modules.InstalledModule, error) {
	if strings.TrimSpace(archivePath) == "" {
		return nil, infraerrors.BadRequest("MODULE_ARCHIVE_PATH_REQUIRED", "module archive path is required")
	}
	if s.installer == nil {
		return nil, infraerrors.ServiceUnavailable("MODULE_INSTALLER_UNAVAILABLE", "module installer is not configured")
	}
	installed, err := s.installer.InstallArchive(ctx, archivePath)
	if err != nil {
		return nil, infraerrors.BadRequest("MODULE_INSTALL_FAILED", "module archive failed validation or installation").WithCause(err)
	}
	return installed, nil
}

// InstallFromMarketplace retains its historical internal name for compatibility.
// It is used only by managed provider provisioning; no marketplace install API
// is registered for administrators or end users.
func (s *ModuleService) InstallFromMarketplace(ctx context.Context, moduleID string, version string) (*modules.InstalledModule, error) {
	moduleID = strings.TrimSpace(moduleID)
	version = strings.TrimSpace(version)
	if moduleID == "" {
		return nil, infraerrors.BadRequest("MODULE_ID_REQUIRED", "module id is required")
	}
	if version == "" {
		return nil, infraerrors.BadRequest("MODULE_VERSION_REQUIRED", "module version is required")
	}
	registry, err := s.loadMarketplaceRegistry(ctx)
	if err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_NOT_CONFIGURED", "module marketplace registry is not configured")
	}
	var selected *ModuleMarketplaceEntry
	for idx := range registry.Modules {
		entry := registry.Modules[idx]
		entry.normalize()
		if entry.ID == moduleID && entry.Version == version {
			if err := validateMarketplaceEntry(entry); err != nil {
				return nil, infraerrors.BadRequest("MODULE_MARKETPLACE_INVALID_ENTRY", "module marketplace registry contains an invalid entry").WithCause(err)
			}
			selected = &entry
			break
		}
	}
	if selected == nil {
		return nil, infraerrors.NotFound("MODULE_MARKETPLACE_ENTRY_NOT_FOUND", "module version was not found in marketplace registry").
			WithMetadata(map[string]string{"module_id": moduleID, "version": version})
	}
	archivePath, cleanup, err := s.downloadMarketplacePackage(ctx, *selected)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if selected.SHA256 != "" {
		if err := verifyFileSHA256(archivePath, selected.SHA256); err != nil {
			return nil, infraerrors.BadRequest("MODULE_MARKETPLACE_SHA256_MISMATCH", "downloaded module package checksum does not match registry").WithCause(err)
		}
	}
	return s.InstallArchive(ctx, archivePath)
}

func (s *ModuleService) AutoInstallManagedProviderModules(ctx context.Context) error {
	if s == nil || s.store == nil || s.installer == nil {
		return nil
	}
	registry, err := s.loadMarketplaceRegistry(ctx)
	if err != nil {
		return err
	}
	if registry == nil {
		return nil
	}
	installed, err := s.store.ListInstalled(ctx)
	if err != nil {
		return err
	}
	installedByID := make(map[string]modules.InstalledModule, len(installed))
	for _, item := range installed {
		installedByID[item.ID] = item
	}
	latestByID := make(map[string]ModuleMarketplaceEntry)
	for _, entry := range registry.Modules {
		entry.normalize()
		if err := validateMarketplaceEntry(entry); err != nil {
			return infraerrors.BadRequest("MODULE_MARKETPLACE_INVALID_ENTRY", "module marketplace registry contains an invalid entry").WithCause(err)
		}
		if !isManagedProviderMarketplaceEntry(entry) {
			continue
		}
		current, ok := latestByID[entry.ID]
		if !ok {
			latestByID[entry.ID] = entry
			continue
		}
		cmp, err := compareModuleVersionStrings(entry.Version, current.Version)
		if err != nil {
			return infraerrors.BadRequest("MODULE_VERSION_INVALID", "managed provider module versions must be semantic versions").WithCause(err)
		}
		if cmp > 0 {
			latestByID[entry.ID] = entry
		}
	}
	ids := make([]string, 0, len(latestByID))
	for id := range latestByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var failures []error
	for _, id := range ids {
		entry := latestByID[id]
		if err := s.ensureManagedProviderModuleInstalled(ctx, entry, installedByID[id]); err != nil {
			failures = append(failures, fmt.Errorf("%s@%s: %w", entry.ID, entry.Version, err))
		}
	}
	return errors.Join(failures...)
}

func (s *ModuleService) ensureManagedProviderModuleInstalled(ctx context.Context, entry ModuleMarketplaceEntry, installed modules.InstalledModule) error {
	if installed.ID != "" && installed.Status != modules.ModuleStatusUninstalled && installed.Status != modules.ModuleStatusPurged {
		cmp, err := compareModuleVersionStrings(entry.Version, installed.Version)
		if err == nil && cmp <= 0 {
			if installed.Status == modules.ModuleStatusEnabled {
				return nil
			}
			if err := s.store.ApprovePermissions(ctx, installed.ID); err != nil {
				return err
			}
			return s.store.SetStatus(ctx, installed.ID, modules.ModuleStatusEnabled, "")
		}
	}
	next, err := s.InstallFromMarketplace(ctx, entry.ID, entry.Version)
	if err != nil {
		return err
	}
	if err := s.store.ApprovePermissions(ctx, next.ID); err != nil {
		return err
	}
	return s.store.SetStatus(ctx, next.ID, modules.ModuleStatusEnabled, "")
}

var managedProviderModuleIDs = map[string]struct{}{
	"anthropic":       {},
	"anthropic-oauth": {},
	"claude":          {},
	"claude-oauth":    {},
	"gemini":          {},
	"gemini-oauth":    {},
	"openai":          {},
	"openai-oauth":    {},
}

func isManagedProviderMarketplaceEntry(entry ModuleMarketplaceEntry) bool {
	if entry.Type != modules.ModuleTypeProvider {
		return false
	}
	if _, ok := managedProviderModuleIDs[strings.ToLower(strings.TrimSpace(entry.ID))]; !ok {
		return false
	}
	for _, capability := range entry.Capabilities {
		if capability == modules.CapabilityProviderAdapter {
			return true
		}
	}
	return false
}

func (s *ModuleService) UpgradeFromMarketplace(ctx context.Context, moduleID string, version string) (*modules.InstalledModule, error) {
	return s.changeMarketplaceVersion(ctx, moduleID, version, true)
}

func (s *ModuleService) RollbackFromMarketplace(ctx context.Context, moduleID string, version string) (*modules.InstalledModule, error) {
	return s.changeMarketplaceVersion(ctx, moduleID, version, false)
}

func (s *ModuleService) changeMarketplaceVersion(ctx context.Context, moduleID string, version string, upgrade bool) (*modules.InstalledModule, error) {
	moduleID = strings.TrimSpace(moduleID)
	version = strings.TrimSpace(version)
	current, err := s.getMutableModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if current.Status == modules.ModuleStatusUninstalled {
		return nil, infraerrors.Conflict("MODULE_UNINSTALLED", "module must be reinstalled before its version can be changed")
	}
	if version == "" {
		return nil, infraerrors.BadRequest("MODULE_VERSION_REQUIRED", "module version is required")
	}
	if version == current.Version {
		return nil, infraerrors.Conflict("MODULE_VERSION_UNCHANGED", "target module version is already installed")
	}
	cmp, err := compareModuleVersionStrings(version, current.Version)
	if err != nil {
		return nil, infraerrors.BadRequest("MODULE_VERSION_INVALID", "module versions must be semantic versions for upgrade or rollback").WithCause(err)
	}
	if upgrade && cmp <= 0 {
		return nil, infraerrors.Conflict("MODULE_UPGRADE_TARGET_NOT_NEWER", "upgrade target version must be newer than the installed version")
	}
	if !upgrade && cmp >= 0 {
		return nil, infraerrors.Conflict("MODULE_ROLLBACK_TARGET_NOT_OLDER", "rollback target version must be older than the installed version")
	}
	if current.Status == modules.ModuleStatusEnabled {
		if err := s.stopModuleRuntime(ctx, moduleID); err != nil {
			return nil, err
		}
		if err := s.store.SetStatus(ctx, moduleID, modules.ModuleStatusDisabled, "module runtime stopped before version change"); err != nil {
			return nil, mapModuleStoreError(err, moduleID)
		}
	}
	installed, err := s.InstallFromMarketplace(ctx, moduleID, version)
	if err != nil {
		_ = s.store.SetStatus(ctx, moduleID, modules.ModuleStatusFailed, err.Error())
		return nil, err
	}
	return installed, nil
}

func compareModuleVersionStrings(left string, right string) (int, error) {
	leftVersion, err := parseModuleSemanticVersion(left)
	if err != nil {
		return 0, fmt.Errorf("invalid target version %q: %w", left, err)
	}
	rightVersion, err := parseModuleSemanticVersion(right)
	if err != nil {
		return 0, fmt.Errorf("invalid installed version %q: %w", right, err)
	}
	return leftVersion.compare(rightVersion), nil
}

type moduleSemanticVersion struct {
	major int
	minor int
	patch int
}

func parseModuleSemanticVersion(value string) (moduleSemanticVersion, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if idx := strings.IndexAny(value, "-+"); idx >= 0 {
		value = value[:idx]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return moduleSemanticVersion{}, fmt.Errorf("expected semver major.minor.patch")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return moduleSemanticVersion{}, fmt.Errorf("invalid semver major version")
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return moduleSemanticVersion{}, fmt.Errorf("invalid semver minor version")
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return moduleSemanticVersion{}, fmt.Errorf("invalid semver patch version")
	}
	if major < 0 || minor < 0 || patch < 0 {
		return moduleSemanticVersion{}, fmt.Errorf("semver parts must be non-negative")
	}
	return moduleSemanticVersion{major: major, minor: minor, patch: patch}, nil
}

func (v moduleSemanticVersion) compare(other moduleSemanticVersion) int {
	switch {
	case v.major != other.major:
		return v.major - other.major
	case v.minor != other.minor:
		return v.minor - other.minor
	default:
		return v.patch - other.patch
	}
}

func (e *ModuleMarketplaceEntry) normalize() {
	e.ID = strings.TrimSpace(e.ID)
	e.Version = strings.TrimSpace(e.Version)
	e.Name = strings.TrimSpace(e.Name)
	e.Description = strings.TrimSpace(e.Description)
	e.DownloadURL = strings.TrimSpace(e.DownloadURL)
	e.SHA256 = strings.TrimSpace(e.SHA256)
	e.Signature = strings.TrimSpace(e.Signature)
	e.Core = strings.TrimSpace(e.Core)
	for idx := range e.Capabilities {
		e.Capabilities[idx] = modules.Capability(strings.TrimSpace(string(e.Capabilities[idx])))
	}
}

func validateMarketplaceEntry(entry ModuleMarketplaceEntry) error {
	if entry.ID == "" {
		return errors.New("module id is required")
	}
	if entry.Version == "" {
		return errors.New("module version is required")
	}
	if entry.Type == "" {
		return errors.New("module type is required")
	}
	if entry.Core == "" {
		return errors.New("core compatibility range is required")
	}
	if entry.DownloadURL == "" {
		return errors.New("downloadUrl is required")
	}
	if len(entry.Capabilities) == 0 {
		return errors.New("at least one capability is required")
	}
	for _, capability := range entry.Capabilities {
		if !modules.IsAllowedCapability(capability) {
			return fmt.Errorf("unsupported capability %q", capability)
		}
	}
	manifest := modules.Manifest{
		APIVersion:      modules.ManifestAPIVersionV1Alpha1,
		ID:              entry.ID,
		Name:            entry.Name,
		NameI18n:        entry.NameI18n,
		Description:     entry.Description,
		DescriptionI18n: entry.DescriptionI18n,
		Type:            entry.Type,
		Version:         entry.Version,
		Core:            modules.CoreSpec{Compatible: entry.Core},
		Capabilities:    append([]modules.Capability(nil), entry.Capabilities...),
		Permissions:     entry.Permissions,
	}
	if manifest.Name == "" {
		manifest.Name = entry.ID
	}
	if err := modules.ValidateManifest(manifest); err == nil {
		return nil
	} else if !strings.Contains(err.Error(), "provider.adapter requires backend spec") && !strings.Contains(err.Error(), "outbound.adapter requires backend spec") {
		return err
	}
	return nil
}

func (s *ModuleService) loadMarketplaceRegistry(ctx context.Context) (*ModuleMarketplaceResult, error) {
	if s == nil {
		return nil, nil
	}
	if strings.TrimSpace(s.marketplaceRegistryPath) != "" {
		return s.readMarketplaceRegistryFile(strings.TrimSpace(s.marketplaceRegistryPath))
	}
	if strings.TrimSpace(s.marketplaceRegistryURL) != "" {
		return s.readMarketplaceRegistryURL(ctx, strings.TrimSpace(s.marketplaceRegistryURL))
	}
	return nil, nil
}

func (s *ModuleService) readMarketplaceRegistryFile(filePath string) (*ModuleMarketplaceResult, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, nil
	}
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_REGISTRY_UNAVAILABLE", "module marketplace registry file could not be read").WithCause(err)
	}
	return decodeMarketplaceRegistry(content)
}

func (s *ModuleService) readMarketplaceRegistryURL(ctx context.Context, rawURL string) (*ModuleMarketplaceResult, error) {
	parsed, err := neturl.Parse(rawURL)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, infraerrors.BadRequest("MODULE_MARKETPLACE_REGISTRY_URL_INVALID", "module marketplace registry URL must use http or https").WithCause(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, infraerrors.BadRequest("MODULE_MARKETPLACE_REGISTRY_URL_INVALID", "module marketplace registry URL is invalid").WithCause(err)
	}
	client := &http.Client{Timeout: s.effectiveMarketplaceTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_REGISTRY_UNAVAILABLE", "module marketplace registry URL could not be read").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_REGISTRY_UNAVAILABLE", "module marketplace registry URL returned an unsuccessful status").
			WithMetadata(map[string]string{"status": resp.Status})
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_REGISTRY_UNAVAILABLE", "module marketplace registry response could not be read").WithCause(err)
	}
	return decodeMarketplaceRegistry(content)
}

func decodeMarketplaceRegistry(content []byte) (*ModuleMarketplaceResult, error) {
	var registry ModuleMarketplaceResult
	if err := json.Unmarshal(content, &registry); err != nil {
		return nil, infraerrors.BadRequest("MODULE_MARKETPLACE_REGISTRY_INVALID", "module marketplace registry JSON is invalid").WithCause(err)
	}
	if registry.Modules == nil {
		registry.Modules = []ModuleMarketplaceEntry{}
	}
	return &registry, nil
}

func (s *ModuleService) downloadMarketplacePackage(ctx context.Context, entry ModuleMarketplaceEntry) (string, func(), error) {
	parsed, err := neturl.Parse(entry.DownloadURL)
	if err != nil {
		return "", func() {}, infraerrors.BadRequest("MODULE_MARKETPLACE_DOWNLOAD_URL_INVALID", "module download URL is invalid").WithCause(err)
	}
	tempDir, err := os.MkdirTemp("", "lightbridge-module-marketplace-*")
	if err != nil {
		return "", func() {}, infraerrors.InternalServer("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package download workspace could not be created").WithCause(err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	targetPath := filepath.Join(tempDir, safeMarketplacePackageFilename(entry))
	if parsed.Scheme == "" || parsed.Scheme == "file" {
		sourcePath := parsed.Path
		if parsed.Scheme == "" {
			sourcePath = entry.DownloadURL
		}
		if err := copyLocalFile(sourcePath, targetPath); err != nil {
			cleanup()
			return "", func() {}, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "local module package could not be read").WithCause(err)
		}
		return targetPath, cleanup, nil
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		cleanup()
		return "", func() {}, infraerrors.BadRequest("MODULE_MARKETPLACE_DOWNLOAD_URL_INVALID", "module download URL must use http, https, file, or a local path")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		cleanup()
		return "", func() {}, infraerrors.BadRequest("MODULE_MARKETPLACE_DOWNLOAD_URL_INVALID", "module download URL is invalid").WithCause(err)
	}
	client := &http.Client{Timeout: s.effectiveMarketplaceTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		cleanup()
		return "", func() {}, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package could not be downloaded").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		cleanup()
		return "", func() {}, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package download returned an unsuccessful status").
			WithMetadata(map[string]string{"status": resp.Status})
	}
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		cleanup()
		return "", func() {}, infraerrors.InternalServer("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package download file could not be created").WithCause(err)
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		cleanup()
		return "", func() {}, infraerrors.ServiceUnavailable("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package response could not be saved").WithCause(copyErr)
	}
	if closeErr != nil {
		cleanup()
		return "", func() {}, infraerrors.InternalServer("MODULE_MARKETPLACE_DOWNLOAD_FAILED", "module package download file could not be closed").WithCause(closeErr)
	}
	return targetPath, cleanup, nil
}

func (s *ModuleService) effectiveMarketplaceTimeout() time.Duration {
	if s == nil || s.marketplaceTimeout <= 0 {
		return defaultMarketplaceTimeout
	}
	return s.marketplaceTimeout
}

func safeMarketplacePackageFilename(entry ModuleMarketplaceEntry) string {
	name := filepath.Base(strings.TrimSpace(entry.DownloadURL))
	if name == "." || name == "/" || name == "" {
		name = fmt.Sprintf("lightbridge-module-%s-%s.tar.zst", entry.ID, entry.Version)
	}
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	if name == "" {
		return "module.tar.zst"
	}
	return name
}

func copyLocalFile(sourcePath string, targetPath string) error {
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" || sourcePath == "." {
		return errors.New("local package path is empty")
	}
	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", sourcePath)
	}
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verifyFileSHA256(filePath string, expected string) error {
	expected = strings.TrimSpace(strings.ToLower(expected))
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil || len(expectedBytes) != sha256.Size {
		return fmt.Errorf("invalid sha256 %q", expected)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("sha256 mismatch: expected %s got %s", expected, actual)
	}
	return nil
}

func (s *ModuleService) GetInstalled(ctx context.Context, id string) (*modules.InstalledModule, error) {
	if strings.TrimSpace(id) == "" {
		return nil, infraerrors.BadRequest("MODULE_ID_REQUIRED", "module id is required")
	}
	module, err := s.store.GetInstalled(ctx, id)
	if err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	return module, nil
}

func (s *ModuleService) Permissions(ctx context.Context, id string) (*ModulePermissionStatus, error) {
	if strings.TrimSpace(id) == "" {
		return nil, infraerrors.BadRequest("MODULE_ID_REQUIRED", "module id is required")
	}
	if _, err := s.store.GetInstalled(ctx, id); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	permissions, err := s.store.ListPermissions(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ModulePermissionStatus{
		Permissions: permissions,
		Approved:    modulePermissionsApproved(permissions),
	}, nil
}

func (s *ModuleService) ApprovePermissions(ctx context.Context, id string) (*ModulePermissionStatus, error) {
	if strings.TrimSpace(id) == "" {
		return nil, infraerrors.BadRequest("MODULE_ID_REQUIRED", "module id is required")
	}
	if _, err := s.store.GetInstalled(ctx, id); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	if err := s.store.ApprovePermissions(ctx, id); err != nil {
		return nil, err
	}
	return s.Permissions(ctx, id)
}

func (s *ModuleService) Enable(ctx context.Context, id string) (*modules.InstalledModule, error) {
	current, err := s.getMutableModule(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == modules.ModuleStatusEnabled {
		return current, nil
	}
	if current.Status == modules.ModuleStatusUninstalled {
		return nil, infraerrors.Conflict("MODULE_UNINSTALLED", "module must be reinstalled before it can be enabled")
	}
	if err := s.requirePermissionsApproved(ctx, id); err != nil {
		return nil, err
	}
	if s.providerRuntime != nil || s.outboundRuntime != nil {
		verified, err := s.verifiedModuleForStart(ctx, *current)
		if err != nil {
			_ = s.store.SetStatus(ctx, id, modules.ModuleStatusFailed, err.Error())
			return nil, infraerrors.ServiceUnavailable("MODULE_PACKAGE_VERIFY_FAILED", "module package failed verification before runtime start").WithCause(err)
		}
		if err := s.startModuleRuntime(ctx, *verified); err != nil {
			_ = s.store.SetStatus(ctx, id, modules.ModuleStatusFailed, err.Error())
			return nil, infraerrors.ServiceUnavailable("MODULE_RUNTIME_START_FAILED", "module runtime failed to start").WithCause(err)
		}
	}
	if err := s.store.SetStatus(ctx, id, modules.ModuleStatusEnabled, ""); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	return s.GetInstalled(ctx, id)
}

func (s *ModuleService) Disable(ctx context.Context, id string) (*modules.InstalledModule, error) {
	current, err := s.getMutableModule(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == modules.ModuleStatusDisabled {
		return current, nil
	}
	if err := s.stopModuleRuntime(ctx, id); err != nil {
		return nil, err
	}
	if err := s.store.SetStatus(ctx, id, modules.ModuleStatusDisabled, ""); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	return s.GetInstalled(ctx, id)
}

func (s *ModuleService) Uninstall(ctx context.Context, id string) (*modules.InstalledModule, error) {
	current, err := s.getMutableModule(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.stopModuleRuntime(ctx, id); err != nil {
		return nil, err
	}
	if err := s.removeModuleFiles(*current); err != nil {
		return nil, err
	}
	if err := s.store.SetStatus(ctx, id, modules.ModuleStatusUninstalled, ""); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	return s.GetInstalled(ctx, id)
}

func (s *ModuleService) Purge(ctx context.Context, id string) (*modules.InstalledModule, error) {
	current, err := s.getMutableModule(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.stopModuleRuntime(ctx, id); err != nil {
		return nil, err
	}
	dataDir := "data"
	if s != nil && strings.TrimSpace(s.moduleDataDir) != "" {
		dataDir = strings.TrimSpace(s.moduleDataDir)
	}
	installPath, err := validateModuleInstallPath(dataDir, current.ID, current.Version, current.InstallPath)
	if err != nil {
		return nil, infraerrors.InternalServer("MODULE_FILE_DELETE_FAILED", "module files could not be deleted").WithCause(err)
	}
	if purger, ok := s.store.(modules.DataPurger); ok {
		if err := purger.PurgeModuleData(ctx, *current); err != nil {
			return nil, infraerrors.InternalServer("MODULE_DATA_PURGE_FAILED", "module private data purge failed").WithCause(err)
		}
	}
	if err := removeValidatedModuleInstallPath(installPath); err != nil {
		return nil, infraerrors.InternalServer("MODULE_FILE_DELETE_FAILED", "module files could not be deleted").WithCause(err)
	}
	if err := s.store.SetStatus(ctx, id, modules.ModuleStatusPurged, ""); err != nil {
		return nil, mapModuleStoreError(err, id)
	}
	return s.GetInstalled(ctx, id)
}

func (s *ModuleService) removeModuleFiles(module modules.InstalledModule) error {
	dataDir := "data"
	if s != nil && strings.TrimSpace(s.moduleDataDir) != "" {
		dataDir = strings.TrimSpace(s.moduleDataDir)
	}
	if err := removeModuleInstallPath(dataDir, module.ID, module.Version, module.InstallPath); err != nil {
		return infraerrors.InternalServer("MODULE_FILE_DELETE_FAILED", "module files could not be deleted").WithCause(err)
	}
	return nil
}

func removeModuleInstallPath(dataDir string, moduleID string, version string, installPath string) error {
	targetPath, err := validateModuleInstallPath(dataDir, moduleID, version, installPath)
	if err != nil {
		return err
	}
	return removeValidatedModuleInstallPath(targetPath)
}

func validateModuleInstallPath(dataDir string, moduleID string, version string, installPath string) (string, error) {
	if strings.TrimSpace(installPath) == "" {
		return "", errors.New("module install path is empty")
	}
	if strings.TrimSpace(moduleID) == "" || strings.ContainsAny(moduleID, `/\`) {
		return "", errors.New("module id is invalid for file deletion")
	}
	if strings.TrimSpace(version) == "" || strings.ContainsAny(version, `/\`) {
		return "", errors.New("module version is invalid for file deletion")
	}
	expectedPath := filepath.Clean(modules.InstallDir(dataDir, moduleID, version))
	cleanInstallPath := filepath.Clean(installPath)
	absExpected, err := filepath.Abs(expectedPath)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(cleanInstallPath)
	if err != nil {
		return "", err
	}
	if absTarget != absExpected {
		return "", errors.New("module install path does not match expected module directory")
	}
	modulesRoot, err := filepath.Abs(filepath.Join(dataDir, "modules"))
	if err != nil {
		return "", err
	}
	if !pathInsideDirectory(modulesRoot, absTarget) {
		return "", errors.New("module install path is outside modules directory")
	}
	info, err := os.Lstat(absTarget)
	if errors.Is(err, os.ErrNotExist) {
		return absTarget, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("module install path must not be a symlink")
	}
	if !info.IsDir() {
		return "", errors.New("module install path must be a directory")
	}
	realRoot, err := filepath.EvalSymlinks(modulesRoot)
	if err != nil {
		return "", err
	}
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", err
	}
	if !pathInsideDirectory(realRoot, realTarget) {
		return "", errors.New("module install path resolves outside modules directory")
	}
	return absTarget, nil
}

func removeValidatedModuleInstallPath(targetPath string) error {
	if strings.TrimSpace(targetPath) == "" {
		return errors.New("module install path is empty")
	}
	return os.RemoveAll(targetPath)
}

func pathInsideDirectory(root string, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if root == target {
		return false
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (s *ModuleService) stopModuleRuntime(ctx context.Context, id string) error {
	var failures []error
	if s.providerRuntime != nil {
		if err := s.providerRuntime.StopProvider(ctx, id); err != nil {
			failures = append(failures, err)
		}
	}
	if s.outboundRuntime != nil {
		if err := s.outboundRuntime.StopOutbound(ctx, id); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return infraerrors.ServiceUnavailable("MODULE_RUNTIME_STOP_FAILED", "module runtime failed to stop").WithCause(errors.Join(failures...))
	}
	return nil
}

func (s *ModuleService) startModuleRuntime(ctx context.Context, module modules.InstalledModule) error {
	switch module.Type {
	case modules.ModuleTypeProvider:
		if s.providerRuntime == nil {
			return nil
		}
		return s.providerRuntime.StartProvider(ctx, module)
	case modules.ModuleTypeOutbound:
		if s.outboundRuntime == nil {
			return nil
		}
		return s.outboundRuntime.StartOutbound(ctx, module)
	default:
		return fmt.Errorf("unsupported module type %q", module.Type)
	}
}

func (s *ModuleService) verifiedModuleForStart(ctx context.Context, module modules.InstalledModule) (*modules.InstalledModule, error) {
	if s == nil || s.moduleVerifier == nil {
		return nil, errors.New("module package verifier is not configured")
	}
	verified, err := s.moduleVerifier.VerifyInstalled(ctx, module)
	if err != nil {
		return nil, err
	}
	if verified == nil {
		return nil, errors.New("module package verifier returned no module")
	}
	if s.store != nil {
		if err := s.store.SaveInstalled(ctx, *verified); err != nil {
			return nil, fmt.Errorf("sync verified module record: %w", err)
		}
	}
	return verified, nil
}

func (s *ModuleService) requirePermissionsApproved(ctx context.Context, id string) error {
	permissions, err := s.store.ListPermissions(ctx, id)
	if err != nil {
		return err
	}
	if modulePermissionsApproved(permissions) {
		return nil
	}
	return infraerrors.Conflict("MODULE_PERMISSIONS_NOT_APPROVED", "module permissions must be approved before enabling")
}

func modulePermissionsApproved(permissions []modules.PermissionRecord) bool {
	for _, permission := range permissions {
		if !permission.Approved {
			return false
		}
	}
	return true
}

func (s *ModuleService) UIManifest(ctx context.Context) ([]ModuleUIManifestItem, error) {
	installed, err := s.store.ListInstalled(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ModuleUIManifestItem, 0)
	for _, item := range installed {
		if item.Status != modules.ModuleStatusEnabled || item.Manifest.Frontend == nil {
			continue
		}
		frontend := item.Manifest.Frontend
		remoteEntry := publicModuleAssetPath(item.ID, item.Version, frontend.Entry)
		uiItem := ModuleUIManifestItem{
			ModuleID:       item.ID,
			ModuleName:     item.Name,
			ModuleNameI18n: item.Manifest.NameI18n,
			Version:        item.Version,
			RemoteEntry:    remoteEntry,
		}
		for _, route := range frontend.Routes {
			uiItem.Routes = append(uiItem.Routes, ModuleUIRouteSpec{
				Path:          route.Path,
				Title:         route.Title,
				TitleI18n:     route.TitleI18n,
				RemoteEntry:   remoteEntry,
				ExposedModule: route.ExposedModule,
				RequiresAdmin: route.RequiresAdmin,
			})
		}
		for _, menu := range frontend.Menu {
			uiItem.Menu = append(uiItem.Menu, ModuleUIMenuSpec{
				Title:     menu.Title,
				TitleI18n: menu.TitleI18n,
				Path:      menu.Path,
				Group:     menu.Group,
				Order:     menu.Order,
			})
		}
		for _, form := range frontend.AccountForms {
			uiItem.AccountForms = append(uiItem.AccountForms, ModuleUIAccountFormSpec{
				ProviderID:       form.ProviderID,
				ProviderName:     item.Name,
				ProviderNameI18n: item.Manifest.NameI18n,
				ModuleID:         item.ID,
				ModuleName:       item.Name,
				ModuleNameI18n:   item.Manifest.NameI18n,
				ModuleVersion:    item.Version,
				RemoteEntry:      remoteEntry,
				ExposedModule:    form.ExposedModule,
			})
		}
		for _, panel := range frontend.EntityPanels {
			uiItem.EntityPanels = append(uiItem.EntityPanels, ModuleUIEntityPanelSpec{
				Entity:        panel.Entity,
				Title:         panel.Title,
				TitleI18n:     panel.TitleI18n,
				ModuleID:      item.ID,
				ModuleVersion: item.Version,
				RemoteEntry:   remoteEntry,
				ExposedModule: panel.ExposedModule,
				RequiresAdmin: panel.RequiresAdmin,
				Order:         panel.Order,
			})
		}
		result = append(result, uiItem)
	}
	return result, nil
}

func (s *ModuleService) ProviderAccountForms(ctx context.Context) ([]ModuleUIAccountFormSpec, error) {
	uiManifest, err := s.UIManifest(ctx)
	if err != nil {
		return nil, err
	}
	var result []ModuleUIAccountFormSpec
	for _, item := range uiManifest {
		for _, form := range item.AccountForms {
			if form.ModuleID == "" {
				form.ModuleID = item.ModuleID
			}
			if form.ModuleName == "" {
				form.ModuleName = item.ModuleName
			}
			if form.ModuleVersion == "" {
				form.ModuleVersion = item.Version
			}
			if form.ProviderName == "" {
				form.ProviderName = item.ModuleName
			}
			result = append(result, form)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ProviderName == result[j].ProviderName {
			return result[i].ProviderID < result[j].ProviderID
		}
		return result[i].ProviderName < result[j].ProviderName
	})
	return result, nil
}

func (s *ModuleService) getMutableModule(ctx context.Context, id string) (*modules.InstalledModule, error) {
	module, err := s.GetInstalled(ctx, id)
	if err != nil {
		return nil, err
	}
	if module.Status == modules.ModuleStatusPurged {
		return nil, infraerrors.Conflict("MODULE_PURGED", "purged module cannot be changed")
	}
	return module, nil
}

func mapModuleStoreError(err error, id string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.NotFound("MODULE_NOT_FOUND", "module not found").WithMetadata(map[string]string{"module_id": id})
	}
	return err
}

func publicModuleAssetPath(moduleID, version, rel string) string {
	cleanRel := path.Clean("/" + filepath.ToSlash(strings.TrimPrefix(rel, "./")))
	if cleanRel == "/" {
		cleanRel = ""
	}
	return path.Join("/modules", moduleID, version, cleanRel)
}

// StopEnabledModuleRuntimes stops runtime resources for modules whose desired
// status remains enabled. It is used during process shutdown or when the whole
// module-runtime feature is disabled; persisted module status is intentionally
// unchanged so the next eligible boot can restore the same modules.
func (s *ModuleService) StopEnabledModuleRuntimes(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	installed, err := s.store.ListInstalled(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, item := range installed {
		if item.Status != modules.ModuleStatusEnabled {
			continue
		}
		if err := s.stopModuleRuntime(ctx, item.ID); err != nil {
			failures = append(failures, fmt.Errorf("stop module %s: %w", item.ID, err))
		}
	}
	return errors.Join(failures...)
}

// ResolveEnabledAsset resolves a frontend asset for the currently enabled
// module version. Both the declared install root and the final file target are
// evaluated through symlinks and must remain inside the module package.
func (s *ModuleService) ResolveEnabledAsset(ctx context.Context, moduleID, version, relativePath string) (string, error) {
	moduleID = strings.TrimSpace(moduleID)
	version = strings.TrimSpace(version)
	if moduleID == "" || strings.ContainsAny(moduleID, `/\\`) {
		return "", infraerrors.BadRequest("MODULE_ASSET_INVALID", "module id is invalid")
	}
	if version == "" || strings.ContainsAny(version, `/\\`) {
		return "", infraerrors.BadRequest("MODULE_ASSET_INVALID", "module version is invalid")
	}
	cleanRelative := filepath.Clean(strings.TrimPrefix(relativePath, "/"))
	if cleanRelative == "." || cleanRelative == "" || filepath.IsAbs(cleanRelative) || cleanRelative == ".." || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) {
		return "", infraerrors.BadRequest("MODULE_ASSET_INVALID", "module asset path is invalid")
	}

	installed, err := s.store.GetInstalled(ctx, moduleID)
	if err != nil {
		return "", mapModuleStoreError(err, moduleID)
	}
	if installed.Status != modules.ModuleStatusEnabled || installed.Version != version {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found")
	}

	dataDir := "data"
	if s != nil && strings.TrimSpace(s.moduleDataDir) != "" {
		dataDir = strings.TrimSpace(s.moduleDataDir)
	}
	installPath, err := validateModuleInstallPath(dataDir, installed.ID, installed.Version, installed.InstallPath)
	if err != nil {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found").WithCause(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(installPath)
	if err != nil {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found").WithCause(err)
	}
	candidate := filepath.Join(installPath, cleanRelative)
	resolvedTarget, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found").WithCause(err)
	}
	within, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) || filepath.IsAbs(within) {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found")
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil || !info.Mode().IsRegular() {
		return "", infraerrors.NotFound("MODULE_ASSET_NOT_FOUND", "module asset was not found")
	}
	return resolvedTarget, nil
}
