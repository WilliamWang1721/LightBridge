package service

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

var (
	ErrNoUpdateAvailable        = infraerrors.Conflict("ALREADY_UP_TO_DATE", "no update available; current version is latest")
	ErrInPlaceUpdateUnsupported = infraerrors.Conflict("IN_PLACE_UPDATE_UNSUPPORTED", "in-place updates are not supported for container deployments; run `docker compose pull` then `docker compose up -d`")
)

const (
	updateCacheKey = "update_check_cache"
	updateCacheTTL = 1200 // 20 minutes
	githubRepo     = "WilliamWang1721/LightBridge"

	// Security: allowed download domains for updates
	allowedDownloadHost = "github.com"
	allowedAssetHost    = "objects.githubusercontent.com"

	// Security: max download size (500MB)
	maxDownloadSize = 500 * 1024 * 1024

	// DeploymentTypeBinary represents a host-managed binary installation.
	DeploymentTypeBinary = "binary"
	// DeploymentTypeContainer represents a container-managed installation.
	DeploymentTypeContainer = "container"

	// Explicit deployment markers take precedence over filesystem heuristics.
	updateDeploymentTypeEnv = "LIGHTBRIDGE_DEPLOYMENT_TYPE"
	updateContainerEnv      = "LIGHTBRIDGE_CONTAINER"
)

// UpdateCache defines cache operations for update service
type UpdateCache interface {
	GetUpdateInfo(ctx context.Context) (string, error)
	SetUpdateInfo(ctx context.Context, data string, ttl time.Duration) error
}

// GitHubReleaseClient 获取 GitHub release 信息的接口
type GitHubReleaseClient interface {
	FetchLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error)
	FetchReleases(ctx context.Context, repo string, limit int) ([]GitHubRelease, error)
	DownloadFile(ctx context.Context, url, dest string, maxSize int64) error
	FetchChecksumFile(ctx context.Context, url string) ([]byte, error)
}

// UpdateService handles software updates
type UpdateService struct {
	cache                  UpdateCache
	githubClient           GitHubReleaseClient
	currentVersion         string
	buildType              string // "source" for manual builds, "release" for CI builds
	deploymentTypeResolver func() string
}

type VersionRelease struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	Current     bool   `json:"current"`
	Latest      bool   `json:"latest"`
}

// NewUpdateService creates a new UpdateService
func NewUpdateService(cache UpdateCache, githubClient GitHubReleaseClient, version, buildType string) *UpdateService {
	return &UpdateService{
		cache:                  cache,
		githubClient:           githubClient,
		currentVersion:         version,
		buildType:              buildType,
		deploymentTypeResolver: resolveUpdateDeploymentType,
	}
}

// UpdateInfo contains update information
type UpdateInfo struct {
	CurrentVersion string             `json:"current_version"`
	LatestVersion  string             `json:"latest_version"`
	HasUpdate      bool               `json:"has_update"`
	ReleaseInfo    *ReleaseInfo       `json:"release_info,omitempty"`
	Cached         bool               `json:"cached"`
	Warning        string             `json:"warning,omitempty"`
	BuildType      string             `json:"build_type"` // "source" or "release"
	Capabilities   UpdateCapabilities `json:"capabilities"`
}

// UpdateCapabilities describes which local lifecycle actions are safe for the
// current deployment. The fields are intentionally independent of build type:
// a source build is still host-managed, while a container must always be
// upgraded by replacing its image.
type UpdateCapabilities struct {
	DeploymentType   string `json:"deployment_type"`
	CanInPlaceUpdate bool   `json:"can_in_place_update"`
	CanRollback      bool   `json:"can_rollback"`
	CanRestart       bool   `json:"can_restart"`
}

// Capabilities returns the current deployment's safe local update actions.
func (s *UpdateService) Capabilities() UpdateCapabilities {
	deploymentType := DeploymentTypeBinary
	if s.deploymentTypeResolver != nil {
		deploymentType = s.deploymentTypeResolver()
	}
	if deploymentType != DeploymentTypeContainer {
		deploymentType = DeploymentTypeBinary
	}

	canOperateInPlace := deploymentType != DeploymentTypeContainer
	return UpdateCapabilities{
		DeploymentType:   deploymentType,
		CanInPlaceUpdate: canOperateInPlace,
		CanRollback:      canOperateInPlace,
		CanRestart:       canOperateInPlace,
	}
}

func (s *UpdateService) ensureInPlaceUpdateSupported() error {
	if !s.Capabilities().CanInPlaceUpdate {
		return ErrInPlaceUpdateUnsupported
	}
	return nil
}

// DetectUpdateDeploymentType determines the deployment type without retaining
// process-global state. Its injectable dependencies keep container detection
// deterministic in tests and allow an explicit operator setting to override
// marker-file heuristics.
func DetectUpdateDeploymentType(
	lookupEnv func(string) (string, bool),
	fileExists func(string) bool,
) string {
	if lookupEnv != nil {
		for _, key := range []string{updateDeploymentTypeEnv, updateContainerEnv} {
			if raw, ok := lookupEnv(key); ok {
				switch strings.ToLower(strings.TrimSpace(raw)) {
				case "1", "true", "yes", "container", "docker", "podman", "kubernetes", "k8s":
					return DeploymentTypeContainer
				case "0", "false", "no", "binary", "host", "bare-metal", "systemd":
					return DeploymentTypeBinary
				}
			}
		}
	}

	if fileExists != nil {
		for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
			if fileExists(marker) {
				return DeploymentTypeContainer
			}
		}
	}

	return DeploymentTypeBinary
}

func resolveUpdateDeploymentType() string {
	return DetectUpdateDeploymentType(os.LookupEnv, func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	})
}

// ReleaseInfo contains GitHub release details
type ReleaseInfo struct {
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets,omitempty"`
}

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
}

// GitHubRelease represents GitHub API response
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	Assets      []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckUpdate checks for available updates
func (s *UpdateService) CheckUpdate(ctx context.Context, force bool) (*UpdateInfo, error) {
	// Try cache first
	if !force {
		if cached, err := s.getFromCache(ctx); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Fetch from GitHub
	info, err := s.fetchLatestRelease(ctx)
	if err != nil {
		// Return cached on error
		if cached, cacheErr := s.getFromCache(ctx); cacheErr == nil && cached != nil {
			cached.Warning = "Using cached data: " + err.Error()
			return cached, nil
		}
		return &UpdateInfo{
			CurrentVersion: s.currentVersion,
			LatestVersion:  s.currentVersion,
			HasUpdate:      false,
			Warning:        err.Error(),
			BuildType:      s.buildType,
			Capabilities:   s.Capabilities(),
		}, nil
	}

	// Cache result
	s.saveToCache(ctx, info)
	return info, nil
}

func (s *UpdateService) ListVersionReleases(ctx context.Context, force bool) ([]VersionRelease, *UpdateInfo, error) {
	info, err := s.CheckUpdate(ctx, force)
	if err != nil {
		return nil, nil, err
	}
	releases, err := s.githubClient.FetchReleases(ctx, githubRepo, 100)
	if err != nil {
		return nil, info, err
	}
	out := make([]VersionRelease, 0, len(releases))
	current := strings.TrimPrefix(strings.TrimSpace(info.CurrentVersion), "v")
	latest := strings.TrimPrefix(strings.TrimSpace(info.LatestVersion), "v")
	for _, release := range releases {
		version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
		if version == "" {
			continue
		}
		out = append(out, VersionRelease{
			Version:     version,
			Name:        release.Name,
			Body:        release.Body,
			PublishedAt: release.PublishedAt,
			HTMLURL:     release.HTMLURL,
			Prerelease:  release.Prerelease,
			Draft:       release.Draft,
			Current:     current == version,
			Latest:      latest == version,
		})
	}
	return out, info, nil
}

// PerformUpdate downloads and applies the update
// Uses atomic file replacement pattern for safe in-place updates
func (s *UpdateService) PerformUpdate(ctx context.Context) error {
	return s.PerformUpdateToVersion(ctx, "")
}

func (s *UpdateService) PerformUpdateToVersion(ctx context.Context, targetVersion string) error {
	if err := s.ensureInPlaceUpdateSupported(); err != nil {
		return err
	}

	info, err := s.updateInfoForTargetVersion(ctx, targetVersion)
	if err != nil {
		return err
	}

	if !info.HasUpdate {
		return ErrNoUpdateAvailable
	}

	asset, checksumURL, directBinary := s.selectUpdateAsset(info)
	if asset == nil {
		return fmt.Errorf("no compatible release found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// SECURITY: Validate download URL is from trusted domain
	if err := validateDownloadURL(asset.DownloadURL); err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if checksumURL != "" {
		if err := validateDownloadURL(checksumURL); err != nil {
			return fmt.Errorf("invalid checksum URL: %w", err)
		}
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	// Create temp directory in the SAME directory as executable
	// This ensures os.Rename is atomic (same filesystem)
	tempDir, err := os.MkdirTemp(exeDir, ".LightBridge-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Download archive
	downloadPath := filepath.Join(tempDir, filepath.Base(asset.Name))
	if err := s.downloadFile(ctx, asset.DownloadURL, downloadPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Verify checksum if available
	if checksumURL != "" {
		if err := s.verifyChecksum(ctx, downloadPath, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	newBinaryPath := filepath.Join(tempDir, "LightBridge")
	if directBinary {
		if err := os.Rename(downloadPath, newBinaryPath); err != nil {
			return fmt.Errorf("prepare binary failed: %w", err)
		}
	} else {
		// Extract binary from archive
		if err := s.extractBinary(downloadPath, newBinaryPath); err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}
	}

	// Set executable permission before replacement
	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}

	// Atomic replacement using rename pattern:
	// 1. Rename current -> backup (atomic on Unix)
	// 2. Rename new -> current (atomic on Unix, same filesystem)
	// If step 2 fails, restore backup
	backupPath := exePath + ".backup"

	// Remove old backup if exists
	_ = os.Remove(backupPath)

	// Step 1: Move current binary to backup
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Step 2: Move new binary to target location (atomic, same filesystem)
	if err := os.Rename(newBinaryPath, exePath); err != nil {
		// Restore backup on failure
		if restoreErr := os.Rename(backupPath, exePath); restoreErr != nil {
			return fmt.Errorf("replace failed and restore failed: %w (restore error: %v)", err, restoreErr)
		}
		return fmt.Errorf("replace failed (restored backup): %w", err)
	}

	// Success - backup file is kept for rollback capability
	// It will be cleaned up on next successful update
	return nil
}

func (s *UpdateService) updateInfoForTargetVersion(ctx context.Context, targetVersion string) (*UpdateInfo, error) {
	targetVersion = strings.TrimPrefix(strings.TrimSpace(targetVersion), "v")
	if targetVersion == "" {
		return s.CheckUpdate(ctx, true)
	}
	releases, err := s.githubClient.FetchReleases(ctx, githubRepo, 100)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
		if version != targetVersion {
			continue
		}
		assets := make([]Asset, len(release.Assets))
		for i, a := range release.Assets {
			assets[i] = Asset{
				Name:        a.Name,
				DownloadURL: a.BrowserDownloadURL,
				Size:        a.Size,
			}
		}
		return &UpdateInfo{
			CurrentVersion: s.currentVersion,
			LatestVersion:  targetVersion,
			// 指定目标版本时，只要与当前版本字符串不同即允许安装（覆盖 preview↔production 同号互装）。
			HasUpdate: normalizeVersionString(s.currentVersion) != normalizeVersionString(targetVersion),
			ReleaseInfo: &ReleaseInfo{
				Name:        release.Name,
				Body:        release.Body,
				PublishedAt: release.PublishedAt,
				HTMLURL:     release.HTMLURL,
				Prerelease:  release.Prerelease,
				Assets:      assets,
			},
			Cached:       false,
			BuildType:    s.buildType,
			Capabilities: s.Capabilities(),
		}, nil
	}
	return nil, fmt.Errorf("release version %s was not found", targetVersion)
}

// Rollback restores the previous version
func (s *UpdateService) Rollback() error {
	if err := s.ensureInPlaceUpdateSupported(); err != nil {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	backupFile := exePath + ".backup"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("no backup found")
	}

	// Replace current with backup
	if err := os.Rename(backupFile, exePath); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

func (s *UpdateService) fetchLatestRelease(ctx context.Context) (*UpdateInfo, error) {
	release, err := s.githubClient.FetchLatestRelease(ctx, githubRepo)
	if err != nil {
		return nil, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	assets := make([]Asset, len(release.Assets))
	for i, a := range release.Assets {
		assets[i] = Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
		}
	}

	return &UpdateInfo{
		CurrentVersion: s.currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      compareVersions(s.currentVersion, latestVersion) < 0,
		ReleaseInfo: &ReleaseInfo{
			Name:        release.Name,
			Body:        release.Body,
			PublishedAt: release.PublishedAt,
			HTMLURL:     release.HTMLURL,
			Prerelease:  release.Prerelease,
			Assets:      assets,
		},
		Cached:       false,
		BuildType:    s.buildType,
		Capabilities: s.Capabilities(),
	}, nil
}

func (s *UpdateService) downloadFile(ctx context.Context, downloadURL, dest string) error {
	return s.githubClient.DownloadFile(ctx, downloadURL, dest, maxDownloadSize)
}

func (s *UpdateService) getArchiveName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("%s_%s", osName, arch)
}

func (s *UpdateService) selectUpdateAsset(info *UpdateInfo) (*Asset, string, bool) {
	if info == nil || info.ReleaseInfo == nil {
		return nil, "", false
	}

	var checksumURL string
	for _, asset := range info.ReleaseInfo.Assets {
		if asset.Name == "checksums.txt" {
			checksumURL = asset.DownloadURL
			break
		}
	}

	if info.ReleaseInfo.Prerelease {
		if asset := s.findPreviewBinaryAsset(info); asset != nil {
			return asset, checksumURL, true
		}
	}

	archiveName := s.getArchiveName()
	for i := range info.ReleaseInfo.Assets {
		asset := &info.ReleaseInfo.Assets[i]
		if strings.Contains(asset.Name, archiveName) && strings.HasSuffix(asset.Name, ".tar.gz") {
			return asset, checksumURL, false
		}
	}
	for i := range info.ReleaseInfo.Assets {
		asset := &info.ReleaseInfo.Assets[i]
		if strings.Contains(asset.Name, archiveName) && !strings.HasSuffix(asset.Name, ".txt") {
			return asset, checksumURL, false
		}
	}
	return nil, checksumURL, false
}

func (s *UpdateService) findPreviewBinaryAsset(info *UpdateInfo) *Asset {
	version := normalizeVersionString(info.LatestVersion)
	expectedName := fmt.Sprintf("LightBridge_%s_%s", version, s.getArchiveName())
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}
	for i := range info.ReleaseInfo.Assets {
		if info.ReleaseInfo.Assets[i].Name == expectedName {
			return &info.ReleaseInfo.Assets[i]
		}
	}
	return nil
}

// validateDownloadURL checks if the URL is from an allowed domain
// SECURITY: This prevents SSRF and ensures downloads only come from trusted GitHub domains
func validateDownloadURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Must be HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	// Check against allowed hosts
	host := parsedURL.Host
	// GitHub release URLs can be from github.com or objects.githubusercontent.com
	if host != allowedDownloadHost &&
		!strings.HasSuffix(host, "."+allowedDownloadHost) &&
		host != allowedAssetHost &&
		!strings.HasSuffix(host, "."+allowedAssetHost) {
		return fmt.Errorf("download from untrusted host: %s", host)
	}

	return nil
}

func (s *UpdateService) verifyChecksum(ctx context.Context, filePath, checksumURL string) error {
	// Download checksums file
	checksumData, err := s.githubClient.FetchChecksumFile(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Calculate file hash
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	// Find expected hash in checksums file
	fileName := filepath.Base(filePath)
	scanner := bufio.NewScanner(strings.NewReader(string(checksumData)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == fileName {
			if parts[0] == actualHash {
				return nil
			}
			return fmt.Errorf("checksum mismatch: expected %s, got %s", parts[0], actualHash)
		}
	}

	return fmt.Errorf("checksum not found for %s", fileName)
}

func (s *UpdateService) extractBinary(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var reader io.Reader = f

	// Handle gzip compression
	if strings.HasSuffix(archivePath, ".gz") || strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer func() { _ = gzr.Close() }()
		reader = gzr
	}

	// Handle tar archive
	if strings.Contains(archivePath, ".tar") {
		tr := tar.NewReader(reader)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			// SECURITY: Prevent Zip Slip / Path Traversal attack
			// Only allow files with safe base names, no directory traversal
			baseName := filepath.Base(hdr.Name)

			// Check for path traversal attempts
			if strings.Contains(hdr.Name, "..") {
				return fmt.Errorf("path traversal attempt detected: %s", hdr.Name)
			}

			// Validate the entry is a regular file
			if hdr.Typeflag != tar.TypeReg {
				continue // Skip directories and special files
			}

			// Only extract the specific binary we need
			if baseName == "LightBridge" || baseName == "LightBridge.exe" {
				// Additional security: limit file size (max 500MB)
				const maxBinarySize = 500 * 1024 * 1024
				if hdr.Size > maxBinarySize {
					return fmt.Errorf("binary too large: %d bytes (max %d)", hdr.Size, maxBinarySize)
				}

				out, err := os.Create(destPath)
				if err != nil {
					return err
				}

				// Use LimitReader to prevent decompression bombs
				limited := io.LimitReader(tr, maxBinarySize)
				if _, err := io.Copy(out, limited); err != nil {
					_ = out.Close()
					return err
				}
				if err := out.Close(); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("binary not found in archive")
	}

	// Direct copy for non-tar files (with size limit)
	const maxBinarySize = 500 * 1024 * 1024
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	limited := io.LimitReader(reader, maxBinarySize)
	if _, err := io.Copy(out, limited); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func (s *UpdateService) getFromCache(ctx context.Context) (*UpdateInfo, error) {
	data, err := s.cache.GetUpdateInfo(ctx)
	if err != nil {
		return nil, err
	}

	var cached struct {
		Latest      string       `json:"latest"`
		ReleaseInfo *ReleaseInfo `json:"release_info"`
		Timestamp   int64        `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	if time.Now().Unix()-cached.Timestamp > updateCacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	return &UpdateInfo{
		CurrentVersion: s.currentVersion,
		LatestVersion:  cached.Latest,
		HasUpdate:      compareVersions(s.currentVersion, cached.Latest) < 0,
		ReleaseInfo:    cached.ReleaseInfo,
		Cached:         true,
		BuildType:      s.buildType,
		Capabilities:   s.Capabilities(),
	}, nil
}

func (s *UpdateService) saveToCache(ctx context.Context, info *UpdateInfo) {
	cacheData := struct {
		Latest      string       `json:"latest"`
		ReleaseInfo *ReleaseInfo `json:"release_info"`
		Timestamp   int64        `json:"timestamp"`
	}{
		Latest:      info.LatestVersion,
		ReleaseInfo: info.ReleaseInfo,
		Timestamp:   time.Now().Unix(),
	}

	data, _ := json.Marshal(cacheData)
	_ = s.cache.SetUpdateInfo(ctx, string(data), time.Duration(updateCacheTTL)*time.Second)
}

// compareVersions compares semantic versions, including prerelease identifiers.
func compareVersions(current, latest string) int {
	currentParts, currentPrerelease := parseSemanticVersion(current)
	latestParts, latestPrerelease := parseSemanticVersion(latest)

	for i := 0; i < 3; i++ {
		if currentParts[i] < latestParts[i] {
			return -1
		}
		if currentParts[i] > latestParts[i] {
			return 1
		}
	}
	return comparePrerelease(currentPrerelease, latestPrerelease)
}

// normalizeVersionString 去除前缀 v 并裁掉首尾空白，用于完整版本字符串（含预发布后缀）的精确比较。
func normalizeVersionString(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func parseSemanticVersion(v string) ([3]int, string) {
	v = normalizeVersionString(v)
	if idx := strings.IndexByte(v, '+'); idx >= 0 {
		v = v[:idx]
	}
	prerelease := ""
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		prerelease = v[idx+1:]
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	result := [3]int{0, 0, 0}
	for i := 0; i < len(parts) && i < 3; i++ {
		if parsed, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = parsed
		}
	}
	return result, prerelease
}

func comparePrerelease(current, latest string) int {
	if current == latest {
		return 0
	}
	if current == "" {
		return 1
	}
	if latest == "" {
		return -1
	}

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")
	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}
	for i := 0; i < maxLen; i++ {
		if i >= len(currentParts) {
			return -1
		}
		if i >= len(latestParts) {
			return 1
		}
		if cmp := comparePrereleaseIdentifier(currentParts[i], latestParts[i]); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func comparePrereleaseIdentifier(current, latest string) int {
	currentNum, currentIsNum := parseNumericPrereleaseIdentifier(current)
	latestNum, latestIsNum := parseNumericPrereleaseIdentifier(latest)
	if currentIsNum && latestIsNum {
		if currentNum < latestNum {
			return -1
		}
		if currentNum > latestNum {
			return 1
		}
		return 0
	}
	if currentIsNum {
		return -1
	}
	if latestIsNum {
		return 1
	}
	if current < latest {
		return -1
	}
	if current > latest {
		return 1
	}
	return 0
}

func parseNumericPrereleaseIdentifier(v string) (int, bool) {
	if v == "" {
		return 0, false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
