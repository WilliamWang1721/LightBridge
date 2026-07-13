package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

const (
	UIThemeManifestFile = "lightbridge-ui.json"

	uiThemeMaxZipBytes       = 10 << 20
	uiThemeMaxExtractBytes   = 20 << 20
	uiThemeMaxCSSBytes       = 512 << 10
	uiThemeHTTPClientTimeout = 30 * time.Second
)

var (
	uiThemeIDPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	uiThemeConfigKeyRegexp = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
)

type UIThemeRepository interface {
	List(ctx context.Context) ([]UITheme, error)
	Get(ctx context.Context, id string) (*UITheme, error)
	GetActive(ctx context.Context) (*UITheme, error)
	Upsert(ctx context.Context, theme UITheme) error
	UpdateConfig(ctx context.Context, id string, config json.RawMessage) (*UITheme, error)
	Activate(ctx context.Context, id string) (*UITheme, error)
	Deactivate(ctx context.Context, id string) (*UITheme, error)
	Delete(ctx context.Context, id string) error
}

type UITheme struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Source    string          `json:"source"`
	EntryCSS  string          `json:"entry_css"`
	Preview   string          `json:"preview,omitempty"`
	Manifest  json.RawMessage `json:"manifest"`
	Config    json.RawMessage `json:"config"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type UIThemeManifest struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Version   string               `json:"version"`
	EntryCSS  string               `json:"entry_css"`
	Preview   string               `json:"preview,omitempty"`
	Config    []UIThemeConfigField `json:"config,omitempty"`
	MenuItems []UIThemeMenuItem    `json:"menu_items,omitempty"`
	Meta      map[string]any       `json:"meta,omitempty"`
}

type UIThemeConfigField struct {
	Key     string          `json:"key"`
	Label   string          `json:"label"`
	Type    string          `json:"type"`
	Default json.RawMessage `json:"default,omitempty"`
	Options []string        `json:"options,omitempty"`
}

type UIThemeMenuItem struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Visibility string `json:"visibility"`
	Type       string `json:"type"`
	Source     string `json:"source"`
	URL        string `json:"url,omitempty"`
	IconSVG    string `json:"icon_svg,omitempty"`
	SortOrder  int    `json:"sort_order"`
}

type UIThemeInstallResult struct {
	Theme UITheme `json:"theme"`
}

type UIThemeInjection struct {
	ThemeID string            `json:"theme_id"`
	CSSHref string            `json:"css_href"`
	Vars    map[string]string `json:"vars"`
}

type UIThemeService struct {
	repo       UIThemeRepository
	setting    SettingRepository
	dataDir    string
	pagesDir   string
	httpClient *http.Client
	onUpdate   func()
}

func NewUIThemeService(repo UIThemeRepository, settingRepo SettingRepository, cfg *config.Config) *UIThemeService {
	pagesBaseDir := "data"
	if cfg != nil && strings.TrimSpace(cfg.Pricing.DataDir) != "" {
		pagesBaseDir = strings.TrimSpace(cfg.Pricing.DataDir)
	}
	return &UIThemeService{
		repo:     repo,
		setting:  settingRepo,
		dataDir:  filepath.Join("data", "ui-themes"),
		pagesDir: filepath.Join(pagesBaseDir, "pages"),
		httpClient: &http.Client{
			Timeout: uiThemeHTTPClientTimeout,
		},
	}
}

func (s *UIThemeService) SetOnUpdateCallback(cb func()) {
	s.onUpdate = cb
}

func (s *UIThemeService) notifyUpdate() {
	if s != nil && s.onUpdate != nil {
		s.onUpdate()
	}
}

func (s *UIThemeService) List(ctx context.Context) ([]UITheme, error) {
	return s.repo.List(ctx)
}

func (s *UIThemeService) Get(ctx context.Context, id string) (*UITheme, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *UIThemeService) InstallZIP(ctx context.Context, data []byte, source string, replace bool) (*UIThemeInstallResult, error) {
	if len(data) == 0 {
		return nil, infraerrors.BadRequest("UI_THEME_EMPTY_ZIP", "theme zip is empty")
	}
	if len(data) > uiThemeMaxZipBytes {
		return nil, infraerrors.BadRequest("UI_THEME_ZIP_TOO_LARGE", "theme zip exceeds 10MB")
	}

	manifest, err := readManifestFromZip(data)
	if err != nil {
		return nil, err
	}
	if err := validateUIThemeManifest(manifest); err != nil {
		return nil, err
	}
	if !replace {
		if existing, err := s.repo.Get(ctx, manifest.ID); err == nil && existing != nil {
			return nil, infraerrors.Conflict("UI_THEME_EXISTS", "theme already exists; pass replace=true to overwrite")
		}
	}

	targetDir := filepath.Join(s.dataDir, manifest.ID)
	tmpDir := targetDir + ".tmp-" + time.Now().Format("20060102150405")
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return nil, fmt.Errorf("create theme data dir: %w", err)
	}
	if err := extractThemeZip(data, tmpDir, manifest); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	if err := os.RemoveAll(targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("remove previous theme files: %w", err)
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("install theme files: %w", err)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	defaultConfig := defaultUIThemeConfig(manifest)
	configBytes, err := json.Marshal(defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal default config: %w", err)
	}

	theme := UITheme{
		ID:       manifest.ID,
		Name:     manifest.Name,
		Version:  manifest.Version,
		Source:   strings.TrimSpace(source),
		EntryCSS: manifest.EntryCSS,
		Preview:  manifest.Preview,
		Manifest: manifestBytes,
		Config:   configBytes,
	}
	if err := s.repo.Upsert(ctx, theme); err != nil {
		return nil, err
	}
	stored, err := s.repo.Get(ctx, manifest.ID)
	if err != nil {
		return nil, err
	}
	s.notifyUpdate()
	return &UIThemeInstallResult{Theme: *stored}, nil
}

func (s *UIThemeService) ImportGitHub(ctx context.Context, repoURL string, replace bool) (*UIThemeInstallResult, error) {
	archiveURLs, err := githubArchiveURLs(repoURL)
	if err != nil {
		return nil, err
	}
	var lastStatus int
	for _, archiveURL := range archiveURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/zip, application/octet-stream")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download github theme archive: %w", err)
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, uiThemeMaxZipBytes+1))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if readErr != nil {
				return nil, fmt.Errorf("read github archive: %w", readErr)
			}
			return s.InstallZIP(ctx, data, repoURL, replace)
		}
		lastStatus = resp.StatusCode
	}
	return nil, infraerrors.BadRequest("UI_THEME_GITHUB_DOWNLOAD_FAILED", fmt.Sprintf("github archive download failed: HTTP %d", lastStatus))
}

func (s *UIThemeService) Activate(ctx context.Context, id string) (*UITheme, error) {
	theme, err := s.repo.Activate(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if err := s.applyThemeMenuItems(ctx, theme); err != nil {
		return nil, err
	}
	s.notifyUpdate()
	return theme, nil
}

func (s *UIThemeService) Deactivate(ctx context.Context, id string) (*UITheme, error) {
	theme, err := s.repo.Deactivate(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if err := s.removeThemeMenuItems(ctx, strings.TrimSpace(id)); err != nil {
		return nil, err
	}
	s.notifyUpdate()
	return theme, nil
}

func (s *UIThemeService) UpdateConfig(ctx context.Context, id string, config json.RawMessage) (*UITheme, error) {
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	var obj map[string]any
	if err := json.Unmarshal(config, &obj); err != nil {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_CONFIG", "config must be a JSON object")
	}
	theme, err := s.repo.UpdateConfig(ctx, strings.TrimSpace(id), config)
	if err != nil {
		return nil, err
	}
	s.notifyUpdate()
	return theme, nil
}

func (s *UIThemeService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return infraerrors.BadRequest("UI_THEME_INVALID_ID", "theme id is required")
	}
	if err := s.removeThemeMenuItems(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(s.dataDir, id)); err != nil {
		return err
	}
	s.notifyUpdate()
	return nil
}

func (s *UIThemeService) ActiveInjection(ctx context.Context) (*UIThemeInjection, error) {
	theme, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if theme == nil {
		return nil, nil
	}
	var config map[string]any
	if len(theme.Config) > 0 {
		_ = json.Unmarshal(theme.Config, &config)
	}
	return &UIThemeInjection{
		ThemeID: theme.ID,
		CSSHref: "/ui-themes/" + url.PathEscape(theme.ID) + "/" + path.Clean("/" + theme.EntryCSS)[1:],
		Vars:    uiThemeCSSVars(config),
	}, nil
}

func (s *UIThemeService) ResolveAssetPath(ctx context.Context, id, assetPath string) (string, string, error) {
	id = strings.TrimSpace(id)
	if !uiThemeIDPattern.MatchString(id) {
		return "", "", infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		return "", "", err
	}
	clean, err := cleanThemeAssetPath(assetPath)
	if err != nil {
		return "", "", err
	}
	fullPath := filepath.Join(s.dataDir, id, clean)
	root, _ := filepath.Abs(filepath.Join(s.dataDir, id))
	full, _ := filepath.Abs(fullPath)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", "", infraerrors.Forbidden("UI_THEME_ASSET_FORBIDDEN", "theme asset path is outside theme directory")
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return "", "", infraerrors.NotFound("UI_THEME_ASSET_NOT_FOUND", "theme asset not found")
	}
	return full, mime.TypeByExtension(strings.ToLower(filepath.Ext(full))), nil
}

func (s *UIThemeService) applyThemeMenuItems(ctx context.Context, theme *UITheme) error {
	if theme == nil || s.setting == nil {
		return nil
	}
	var manifest UIThemeManifest
	if err := json.Unmarshal(theme.Manifest, &manifest); err != nil {
		return nil
	}
	if len(manifest.MenuItems) == 0 {
		return s.removeThemeMenuItems(ctx, theme.ID)
	}
	raw, err := s.setting.GetValue(ctx, SettingKeyCustomMenuItems)
	if err != nil && err != ErrSettingNotFound {
		return err
	}
	existing := parseMenuItemsForTheme(raw)
	filtered := make([]map[string]any, 0, len(existing)+len(manifest.MenuItems))
	prefix := "theme-" + theme.ID + "-"
	for _, item := range existing {
		id, _ := item["id"].(string)
		if !strings.HasPrefix(id, prefix) {
			filtered = append(filtered, item)
		}
	}
	for _, item := range manifest.MenuItems {
		menu, err := buildThemeMenuItem(theme.ID, item)
		if err != nil {
			return err
		}
		if strings.TrimSpace(item.Type) == "markdown" {
			if err := s.installThemeMarkdownPage(theme.ID, item); err != nil {
				return err
			}
		}
		filtered = append(filtered, menu)
	}
	out, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	return s.setting.Set(ctx, SettingKeyCustomMenuItems, string(out))
}

func (s *UIThemeService) removeThemeMenuItems(ctx context.Context, themeID string) error {
	if s.setting == nil || strings.TrimSpace(themeID) == "" {
		return nil
	}
	raw, err := s.setting.GetValue(ctx, SettingKeyCustomMenuItems)
	if err != nil {
		if err == ErrSettingNotFound {
			return nil
		}
		return err
	}
	existing := parseMenuItemsForTheme(raw)
	prefix := "theme-" + themeID + "-"
	filtered := make([]map[string]any, 0, len(existing))
	for _, item := range existing {
		id, _ := item["id"].(string)
		if !strings.HasPrefix(id, prefix) {
			filtered = append(filtered, item)
		}
	}
	out, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	if err := s.setting.Set(ctx, SettingKeyCustomMenuItems, string(out)); err != nil {
		return err
	}
	return s.removeThemeMarkdownPages(themeID)
}

func (s *UIThemeService) installThemeMarkdownPage(themeID string, item UIThemeMenuItem) error {
	id := strings.ToLower(strings.TrimSpace(item.ID))
	source, err := cleanThemeAssetPath(item.Source)
	if err != nil {
		return err
	}
	sourcePath := filepath.Join(s.dataDir, themeID, source)
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read theme markdown page: %w", err)
	}
	if len(content) > 1<<20 {
		return infraerrors.BadRequest("UI_THEME_PAGE_TOO_LARGE", "theme markdown page exceeds 1MB")
	}
	if err := os.MkdirAll(s.pagesDir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(s.pagesDir, "theme-"+themeID+"-"+id+".md")
	return os.WriteFile(target, content, 0o644)
}

func (s *UIThemeService) removeThemeMarkdownPages(themeID string) error {
	entries, err := os.ReadDir(s.pagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	prefix := "theme-" + themeID + "-"
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if err := os.Remove(filepath.Join(s.pagesDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func readManifestFromZip(data []byte) (*UIThemeManifest, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_ZIP", "invalid theme zip")
	}
	for _, f := range zr.File {
		if path.Base(f.Name) != UIThemeManifestFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		var manifest UIThemeManifest
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, infraerrors.BadRequest("UI_THEME_INVALID_MANIFEST", "invalid lightbridge-ui.json")
		}
		return &manifest, nil
	}
	return nil, infraerrors.BadRequest("UI_THEME_MANIFEST_MISSING", "lightbridge-ui.json is required")
}

func validateUIThemeManifest(m *UIThemeManifest) error {
	if m == nil {
		return infraerrors.BadRequest("UI_THEME_INVALID_MANIFEST", "manifest is required")
	}
	m.ID = strings.ToLower(strings.TrimSpace(m.ID))
	m.Name = strings.TrimSpace(m.Name)
	m.Version = strings.TrimSpace(m.Version)
	m.EntryCSS = strings.TrimSpace(m.EntryCSS)
	m.Preview = strings.TrimSpace(m.Preview)
	if !uiThemeIDPattern.MatchString(m.ID) {
		return infraerrors.BadRequest("UI_THEME_INVALID_ID", "theme id must match ^[a-z0-9][a-z0-9_-]{0,31}$")
	}
	if m.Name == "" || len(m.Name) > 80 {
		return infraerrors.BadRequest("UI_THEME_INVALID_NAME", "theme name is required and must be <= 80 characters")
	}
	if m.Version == "" || len(m.Version) > 32 {
		return infraerrors.BadRequest("UI_THEME_INVALID_VERSION", "theme version is required and must be <= 32 characters")
	}
	if err := validateRelativeThemePath(m.EntryCSS, ".css"); err != nil {
		return err
	}
	if m.Preview != "" {
		if err := validateRelativeThemePath(m.Preview, ".png", ".jpg", ".jpeg", ".webp", ".svg"); err != nil {
			return err
		}
	}
	for _, field := range m.Config {
		if !uiThemeConfigKeyRegexp.MatchString(strings.TrimSpace(field.Key)) {
			return infraerrors.BadRequest("UI_THEME_INVALID_CONFIG_FIELD", "theme config key is invalid")
		}
		switch strings.TrimSpace(field.Type) {
		case "color", "text", "select", "number", "boolean":
		default:
			return infraerrors.BadRequest("UI_THEME_INVALID_CONFIG_FIELD", "theme config type is invalid")
		}
	}
	for _, item := range m.MenuItems {
		if _, err := buildThemeMenuItem(m.ID, item); err != nil {
			return err
		}
	}
	return nil
}

func extractThemeZip(data []byte, dest string, manifest *UIThemeManifest) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return infraerrors.BadRequest("UI_THEME_INVALID_ZIP", "invalid theme zip")
	}
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean, err := cleanThemeZipPath(f.Name)
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(clean))
		if !allowedThemeExt(ext) {
			return infraerrors.BadRequest("UI_THEME_UNSUPPORTED_FILE", "theme contains unsupported file type: "+ext)
		}
		total += int64(f.UncompressedSize64)
		if total > uiThemeMaxExtractBytes {
			return infraerrors.BadRequest("UI_THEME_TOO_LARGE", "theme extracted size exceeds 20MB")
		}
		if ext == ".css" && f.UncompressedSize64 > uiThemeMaxCSSBytes {
			return infraerrors.BadRequest("UI_THEME_CSS_TOO_LARGE", "theme css exceeds 512KB")
		}
		target := filepath.Join(dest, clean)
		root, _ := filepath.Abs(dest)
		full, _ := filepath.Abs(target)
		if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
			return infraerrors.BadRequest("UI_THEME_UNSAFE_PATH", "theme contains unsafe path")
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		content, readErr := io.ReadAll(io.LimitReader(rc, int64(f.UncompressedSize64)+1))
		_ = rc.Close()
		if readErr != nil {
			return readErr
		}
		if ext == ".css" {
			sanitized, err := sanitizeThemeCSS(content)
			if err != nil {
				return err
			}
			content = sanitized
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(dest, path.Clean("/" + manifest.EntryCSS)[1:])); err != nil {
		return infraerrors.BadRequest("UI_THEME_ENTRY_CSS_MISSING", "entry css file is missing")
	}
	return nil
}

func sanitizeThemeCSS(content []byte) ([]byte, error) {
	css := string(content)
	lower := strings.ToLower(css)
	for _, blocked := range []string{"expression(", "-moz-binding", "behavior:", "javascript:", "@import"} {
		if strings.Contains(lower, blocked) {
			return nil, infraerrors.BadRequest("UI_THEME_UNSAFE_CSS", "theme css contains unsafe construct: "+blocked)
		}
	}
	if regexp.MustCompile(`url\(\s*['"]?(https?:)?//`).MatchString(lower) || regexp.MustCompile(`url\(\s*['"]?data:`).MatchString(lower) {
		return nil, infraerrors.BadRequest("UI_THEME_UNSAFE_CSS", "theme css cannot reference external or data URLs")
	}
	return content, nil
}

func defaultUIThemeConfig(m *UIThemeManifest) map[string]any {
	result := make(map[string]any, len(m.Config))
	for _, field := range m.Config {
		if len(field.Default) == 0 {
			continue
		}
		var v any
		if err := json.Unmarshal(field.Default, &v); err == nil {
			result[field.Key] = v
		}
	}
	return result
}

func uiThemeCSSVars(config map[string]any) map[string]string {
	result := map[string]string{}
	for key, value := range config {
		if !uiThemeConfigKeyRegexp.MatchString(key) {
			continue
		}
		var rendered string
		switch v := value.(type) {
		case string:
			rendered = v
		case float64, bool:
			rendered = fmt.Sprint(v)
		default:
			continue
		}
		if strings.ContainsAny(rendered, "<>{}") || strings.Contains(strings.ToLower(rendered), "javascript:") {
			continue
		}
		result["--theme-"+strings.ReplaceAll(strings.ToLower(key), "_", "-")] = rendered
	}
	return result
}

func githubArchiveURLs(raw string) ([]string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_GITHUB_URL", "github repository URL is invalid")
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" && host != "www.github.com" {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_GITHUB_URL", "only github.com repository URLs are supported")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_GITHUB_URL", "github repository URL must include owner and repo")
	}
	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_GITHUB_URL", "github repository URL must include owner and repo")
	}
	base := "https://github.com/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/archive/refs/heads/"
	return []string{base + "main.zip", base + "master.zip"}, nil
}

func cleanThemeZipPath(raw string) (string, error) {
	clean := path.Clean(strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/")))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", infraerrors.BadRequest("UI_THEME_UNSAFE_PATH", "theme contains unsafe path")
	}
	parts := strings.Split(clean, "/")
	if len(parts) > 1 && strings.Contains(parts[0], "-") {
		clean = strings.Join(parts[1:], "/")
	}
	if clean == "" || strings.HasPrefix(clean, ".") || strings.Contains(clean, "/.") {
		return "", infraerrors.BadRequest("UI_THEME_UNSAFE_PATH", "theme contains hidden or unsafe path")
	}
	return clean, nil
}

func cleanThemeAssetPath(raw string) (string, error) {
	clean := path.Clean(strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/")))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", infraerrors.Forbidden("UI_THEME_ASSET_FORBIDDEN", "unsafe theme asset path")
	}
	if !allowedThemeExt(strings.ToLower(filepath.Ext(clean))) {
		return "", infraerrors.Forbidden("UI_THEME_ASSET_FORBIDDEN", "theme asset type is not allowed")
	}
	return clean, nil
}

func validateRelativeThemePath(p string, exts ...string) error {
	clean, err := cleanThemeAssetPath(p)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(clean))
	for _, allowed := range exts {
		if ext == allowed {
			return nil
		}
	}
	return infraerrors.BadRequest("UI_THEME_INVALID_PATH", "theme path has invalid extension")
}

func allowedThemeExt(ext string) bool {
	switch ext {
	case ".json", ".css", ".md", ".png", ".jpg", ".jpeg", ".svg", ".webp", ".woff", ".woff2":
		return true
	default:
		return false
	}
}

func parseMenuItemsForTheme(raw string) []map[string]any {
	var items []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &items); err != nil {
		return []map[string]any{}
	}
	return items
}

func buildThemeMenuItem(themeID string, item UIThemeMenuItem) (map[string]any, error) {
	id := strings.TrimSpace(item.ID)
	if id == "" || !uiThemeIDPattern.MatchString(strings.ToLower(id)) {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_MENU_ITEM", "theme menu item id is invalid")
	}
	label := strings.TrimSpace(item.Label)
	if label == "" || len(label) > 50 {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_MENU_ITEM", "theme menu item label is required and must be <= 50 characters")
	}
	visibility := strings.TrimSpace(item.Visibility)
	if visibility == "" {
		visibility = "user"
	}
	if visibility != "user" && visibility != "admin" {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_MENU_ITEM", "theme menu item visibility must be user or admin")
	}
	itemType := strings.TrimSpace(item.Type)
	var targetURL string
	var pageSlug string
	switch itemType {
	case "markdown":
		if err := validateRelativeThemePath(item.Source, ".md"); err != nil {
			return nil, err
		}
		pageSlug = "theme-" + themeID + "-" + id
		targetURL = "md:" + pageSlug
	case "iframe":
		u, err := url.Parse(strings.TrimSpace(item.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, infraerrors.BadRequest("UI_THEME_INVALID_MENU_ITEM", "iframe menu item URL must be absolute http(s)")
		}
		targetURL = u.String()
	default:
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_MENU_ITEM", "theme menu item type must be markdown or iframe")
	}
	return map[string]any{
		"id":         "theme-" + themeID + "-" + id,
		"label":      label,
		"icon_svg":   item.IconSVG,
		"url":        targetURL,
		"page_slug":  pageSlug,
		"visibility": visibility,
		"sort_order": item.SortOrder,
	}, nil
}

func UIThemeConfigHash(config json.RawMessage) string {
	sum := sha256.Sum256(config)
	return hex.EncodeToString(sum[:8])
}

func SortedUIThemeVars(vars map[string]string) [][2]string {
	pairs := make([][2]string, 0, len(vars))
	for k, v := range vars {
		pairs = append(pairs, [2]string{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })
	return pairs
}
