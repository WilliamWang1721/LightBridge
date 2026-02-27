package modules

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lightbridge/internal/store"
	"lightbridge/internal/types"
)

var ErrSHA256Mismatch = errors.New("sha256 mismatch")
var errManifestFound = errors.New("manifest found")

type Marketplace struct {
	client  *http.Client
	store   *store.Store
	baseDir string

	mu    sync.Mutex
	cache map[string]cachedIndex
}

type cachedIndex struct {
	idx       *types.ModuleIndex
	expiresAt time.Time
}

func NewMarketplace(st *store.Store, baseDir string, client *http.Client) *Marketplace {
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	return &Marketplace{
		client:  client,
		store:   st,
		baseDir: baseDir,
		cache:   map[string]cachedIndex{},
	}
}

func (m *Marketplace) FetchIndex(ctx context.Context, indexURL string) (*types.ModuleIndex, error) {
	indexURL = strings.TrimSpace(indexURL)
	if isLocalIndexURL(indexURL) {
		return m.fetchLocalIndex(ctx)
	}

	if spec, ok := parseGitHubModulesSpec(indexURL); ok {
		return m.fetchGitHubIndex(ctx, spec)
	}

	if u, err := url.Parse(indexURL); err == nil && u != nil && u.Scheme == "file" {
		return nil, fmt.Errorf("file:// index is not supported; use %q to scan local MODULES dir", "local")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("index fetch failed with %d", resp.StatusCode)
	}
	var index types.ModuleIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}
	sort.Slice(index.Modules, func(i, j int) bool {
		return index.Modules[i].ID < index.Modules[j].ID
	})
	return &index, nil
}

func (m *Marketplace) Install(ctx context.Context, entry types.ModuleEntry) (*types.ModuleInstalled, *types.ModuleManifest, error) {
	if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.DownloadURL) == "" || strings.TrimSpace(entry.SHA256) == "" {
		return nil, nil, fmt.Errorf("invalid module entry")
	}

	tmp, err := os.CreateTemp("", "lightbridge-module-*.zip")
	if err != nil {
		return nil, nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()

	downloadURL := strings.TrimSpace(entry.DownloadURL)
	if filePath, ok := parseFileDownloadURL(downloadURL); ok {
		f, err := os.Open(filePath)
		if err != nil {
			_ = tmp.Close()
			return nil, nil, err
		}
		_, copyErr := io.Copy(io.MultiWriter(tmp, hasher), f)
		_ = f.Close()
		if copyErr != nil {
			_ = tmp.Close()
			return nil, nil, copyErr
		}
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if err != nil {
			_ = tmp.Close()
			return nil, nil, err
		}
		resp, err := m.client.Do(req)
		if err != nil {
			_ = tmp.Close()
			return nil, nil, err
		}
		if resp.StatusCode >= 400 {
			_ = resp.Body.Close()
			_ = tmp.Close()
			return nil, nil, fmt.Errorf("download failed with %d", resp.StatusCode)
		}
		_, copyErr := io.Copy(io.MultiWriter(tmp, hasher), resp.Body)
		_ = resp.Body.Close()
		if copyErr != nil {
			_ = tmp.Close()
			return nil, nil, copyErr
		}
	}
	if err := tmp.Close(); err != nil {
		return nil, nil, err
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actual, entry.SHA256) {
		return nil, nil, fmt.Errorf("%w: expected %s got %s", ErrSHA256Mismatch, entry.SHA256, actual)
	}

	installDir := filepath.Join(m.baseDir, "modules", entry.ID, entry.Version)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := unzip(tmpPath, installDir); err != nil {
		return nil, nil, err
	}

	manifestPath, err := findManifest(installDir)
	if err != nil {
		return nil, nil, err
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	var manifest types.ModuleManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, nil, err
	}
	if err := validateManifest(manifest); err != nil {
		return nil, nil, err
	}

	enabled := true
	if existing, err := m.store.GetInstalledModule(ctx, manifest.ID); err == nil && existing != nil {
		enabled = existing.Enabled
	}

	installed := &types.ModuleInstalled{
		ID:          manifest.ID,
		Version:     manifest.Version,
		InstallPath: filepath.Dir(manifestPath),
		Enabled:     enabled,
		Protocols:   strings.Join(entry.Protocols, ","),
		SHA256:      actual,
		InstalledAt: time.Now().UTC(),
	}
	if err := m.store.SaveInstalledModule(ctx, *installed); err != nil {
		return nil, nil, err
	}
	return installed, &manifest, nil
}

func unzip(srcZip, dstDir string) error {
	zr, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, file := range zr.File {
		targetPath := filepath.Join(dstDir, file.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return err
		}
		if err := out.Close(); err != nil {
			_ = rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}
	return nil
}

func findManifest(installDir string) (string, error) {
	var found string
	err := filepath.WalkDir(installDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "manifest.json") {
			found = path
			return errManifestFound
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errManifestFound) {
			return found, nil
		}
		return "", err
	}
	if found == "" {
		return "", errors.New("manifest.json not found in module archive")
	}
	return found, nil
}

func validateManifest(m types.ModuleManifest) error {
	if strings.TrimSpace(m.ID) == "" {
		return errors.New("manifest.id is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("manifest.version is required")
	}
	if len(m.Entrypoints) == 0 {
		return errors.New("manifest.entrypoints is required")
	}
	if len(m.Services) == 0 {
		return errors.New("manifest.services is required")
	}
	for _, svc := range m.Services {
		if svc.Kind != "provider" {
			return fmt.Errorf("unsupported service kind %s", svc.Kind)
		}
		switch svc.Protocol {
		case types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC, types.ProtocolGRPCChat:
		default:
			return fmt.Errorf("unsupported service protocol %s", svc.Protocol)
		}
	}
	return nil
}

func isLocalIndexURL(indexURL string) bool {
	indexURL = strings.ToLower(strings.TrimSpace(indexURL))
	return indexURL == "" || indexURL == "local"
}

type gitHubModulesSpec struct {
	Owner string
	Repo  string
	Ref   string
	Path  string
}

func (s gitHubModulesSpec) cacheKey() string {
	owner := strings.ToLower(strings.TrimSpace(s.Owner))
	repo := strings.ToLower(strings.TrimSpace(s.Repo))
	ref := strings.TrimSpace(s.Ref)
	path := strings.Trim(strings.TrimSpace(s.Path), "/")
	return "github:" + owner + "/" + repo + "@" + ref + "/" + path
}

func parseGitHubModulesSpec(indexURL string) (gitHubModulesSpec, bool) {
	indexURL = strings.TrimSpace(indexURL)
	if indexURL == "" {
		return gitHubModulesSpec{}, false
	}

	parseRefSuffix := func(raw string) (base, ref string) {
		raw = strings.TrimSpace(raw)
		ref = "main"
		if i := strings.LastIndexAny(raw, "@#"); i >= 0 && i < len(raw)-1 {
			ref = strings.TrimSpace(raw[i+1:])
			raw = strings.TrimSpace(raw[:i])
		}
		if ref == "" {
			ref = "main"
		}
		return raw, ref
	}

	if strings.HasPrefix(strings.ToLower(indexURL), "github:") || strings.HasPrefix(strings.ToLower(indexURL), "gh:") {
		raw := indexURL
		if strings.HasPrefix(strings.ToLower(raw), "github:") {
			raw = raw[len("github:"):]
		} else {
			raw = raw[len("gh:"):]
		}
		raw = strings.TrimPrefix(strings.TrimSpace(raw), "//")

		base, ref := parseRefSuffix(raw)
		parts := strings.Split(strings.Trim(base, "/"), "/")
		if len(parts) < 2 {
			return gitHubModulesSpec{}, false
		}
		path := strings.Join(parts[2:], "/")
		if strings.TrimSpace(path) == "" {
			path = "MODULES"
		}
		return gitHubModulesSpec{Owner: parts[0], Repo: parts[1], Ref: ref, Path: path}, true
	}

	u, err := url.Parse(indexURL)
	if err != nil || u == nil {
		return gitHubModulesSpec{}, false
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	switch host {
	case "github.com":
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(segs) < 2 {
			return gitHubModulesSpec{}, false
		}
		owner, repo := segs[0], segs[1]
		ref := "main"
		path := "MODULES"
		if len(segs) >= 4 && segs[2] == "tree" {
			ref = segs[3]
			if len(segs) > 4 {
				path = strings.Join(segs[4:], "/")
			}
			if strings.TrimSpace(path) == "" {
				path = "MODULES"
			}
		} else {
			return gitHubModulesSpec{}, false
		}
		return gitHubModulesSpec{Owner: owner, Repo: repo, Ref: ref, Path: path}, true
	case "api.github.com":
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		// /repos/{owner}/{repo}/contents/{path}
		if len(segs) < 5 {
			return gitHubModulesSpec{}, false
		}
		if segs[0] != "repos" || segs[3] != "contents" {
			return gitHubModulesSpec{}, false
		}
		owner, repo := segs[1], segs[2]
		path := strings.Join(segs[4:], "/")
		if strings.TrimSpace(path) == "" {
			path = "MODULES"
		}
		ref := strings.TrimSpace(u.Query().Get("ref"))
		if ref == "" {
			ref = "main"
		}
		return gitHubModulesSpec{Owner: owner, Repo: repo, Ref: ref, Path: path}, true
	default:
		return gitHubModulesSpec{}, false
	}
}

func (m *Marketplace) fetchGitHubIndex(ctx context.Context, spec gitHubModulesSpec) (*types.ModuleIndex, error) {
	key := spec.cacheKey()
	if idx, ok := m.getCachedIndex(key); ok {
		return idx, nil
	}

	idx, err := m.buildGitHubIndex(ctx, spec)
	if err != nil {
		return nil, err
	}
	m.setCachedIndex(key, idx, 2*time.Minute)
	return idx, nil
}

func (m *Marketplace) getCachedIndex(key string) (*types.ModuleIndex, bool) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if ent, ok := m.cache[key]; ok {
		if ent.idx != nil && ent.expiresAt.After(now) {
			return ent.idx, true
		}
		delete(m.cache, key)
	}
	return nil, false
}

func (m *Marketplace) setCachedIndex(key string, idx *types.ModuleIndex, ttl time.Duration) {
	if idx == nil || ttl <= 0 {
		return
	}
	m.mu.Lock()
	m.cache[key] = cachedIndex{idx: idx, expiresAt: time.Now().Add(ttl)}
	m.mu.Unlock()
}

type gitHubContentItem struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
}

func (m *Marketplace) buildGitHubIndex(ctx context.Context, spec gitHubModulesSpec) (*types.ModuleIndex, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("LIGHTBRIDGE_GITHUB_API_BASE")), "/")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}

	owner := strings.TrimSpace(spec.Owner)
	repo := strings.TrimSpace(spec.Repo)
	ref := strings.TrimSpace(spec.Ref)
	dir := strings.Trim(strings.TrimSpace(spec.Path), "/")
	if owner == "" || repo == "" {
		return nil, errors.New("invalid github spec")
	}
	if ref == "" {
		ref = "main"
	}
	if dir == "" {
		dir = "MODULES"
	}

	// Build: GET /repos/{owner}/{repo}/contents/{dir}?ref={ref}
	escapedDir := escapePath(dir)
	u, err := url.Parse(apiBase)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/contents/" + escapedDir
	q := u.Query()
	q.Set("ref", ref)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/vnd.github+json")
	req.Header.Set("user-agent", "lightbridge")
	if tok := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_GITHUB_TOKEN")); tok != "" {
		req.Header.Set("authorization", "Bearer "+tok)
	} else if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("authorization", "Bearer "+tok)
	} else if tok := strings.TrimSpace(os.Getenv("GH_TOKEN")); tok != "" {
		req.Header.Set("authorization", "Bearer "+tok)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("github index fetch failed (%d): %s", resp.StatusCode, msg)
	}

	var items []gitHubContentItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("github index decode failed: %w", err)
	}

	zips := make([]gitHubContentItem, 0)
	for _, it := range items {
		if strings.ToLower(strings.TrimSpace(it.Type)) != "file" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(it.Name)), ".zip") {
			continue
		}
		if strings.TrimSpace(it.DownloadURL) == "" {
			continue
		}
		zips = append(zips, it)
	}

	modules := make([]types.ModuleEntry, 0, len(zips))
	for _, it := range zips {
		entry, err := m.moduleEntryFromRemoteZip(ctx, it.DownloadURL, "GitHub module package", "github", "https://github.com/"+owner+"/"+repo)
		if err != nil {
			continue
		}
		modules = append(modules, entry)
	}

	sort.Slice(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })
	return &types.ModuleIndex{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		MinCoreVersion: "0.1.0",
		Modules:        modules,
	}, nil
}

func (m *Marketplace) moduleEntryFromRemoteZip(ctx context.Context, downloadURL, description, tagPrefix, homepage string) (types.ModuleEntry, error) {
	downloadURL = strings.TrimSpace(downloadURL)
	if downloadURL == "" {
		return types.ModuleEntry{}, errors.New("missing download url")
	}

	tmp, err := os.CreateTemp("", "lightbridge-gh-module-*.zip")
	if err != nil {
		return types.ModuleEntry{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		_ = tmp.Close()
		return types.ModuleEntry{}, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		_ = tmp.Close()
		return types.ModuleEntry{}, err
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		_ = tmp.Close()
		return types.ModuleEntry{}, fmt.Errorf("download failed with %d", resp.StatusCode)
	}
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		_ = resp.Body.Close()
		_ = tmp.Close()
		return types.ModuleEntry{}, err
	}
	_ = resp.Body.Close()
	if err := tmp.Close(); err != nil {
		return types.ModuleEntry{}, err
	}

	sha := hex.EncodeToString(hasher.Sum(nil))
	manifest, err := readManifestFromZip(tmpPath)
	if err != nil {
		return types.ModuleEntry{}, err
	}
	if err := validateManifest(*manifest); err != nil {
		return types.ModuleEntry{}, err
	}

	id := strings.TrimSpace(manifest.ID)
	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		name = id
	}
	tags := inferRemoteTags(tagPrefix, id)
	protos := protocolsFromManifest(*manifest)

	return types.ModuleEntry{
		ID:          id,
		Name:        name,
		Version:     strings.TrimSpace(manifest.Version),
		Description: description,
		License:     strings.TrimSpace(manifest.License),
		Tags:        tags,
		Protocols:   protos,
		DownloadURL: downloadURL,
		SHA256:      sha,
		Homepage:    homepage,
	}, nil
}

func inferRemoteTags(prefix, id string) []string {
	id = strings.ToLower(strings.TrimSpace(id))
	tags := []string{strings.ToLower(strings.TrimSpace(prefix)), "provider"}
	if strings.Contains(id, "oauth") || strings.Contains(id, "auth") {
		tags = append(tags, "auth")
	}
	if strings.Contains(id, "tool") {
		tags = append(tags, "tool")
	}
	return tags
}

func escapePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	segs := strings.Split(p, "/")
	for i := range segs {
		segs[i] = url.PathEscape(segs[i])
	}
	return strings.Join(segs, "/")
}

func (m *Marketplace) localModulesDir() string {
	if v := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_MODULES_DIR")); v != "" {
		return v
	}
	if st, err := os.Stat("MODULES"); err == nil && st.IsDir() {
		return "MODULES"
	}
	return filepath.Join(m.baseDir, "MODULES")
}

func (m *Marketplace) fetchLocalIndex(ctx context.Context) (*types.ModuleIndex, error) {
	_ = ctx
	dir := m.localModulesDir()
	st, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &types.ModuleIndex{
				GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
				MinCoreVersion: "0.1.0",
				Modules:        []types.ModuleEntry{},
			}, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("local modules dir is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	mods := make([]types.ModuleEntry, 0)
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".zip") {
			continue
		}
		zipPath := filepath.Join(dir, name)
		entry, err := moduleEntryFromZip(zipPath)
		if err != nil {
			continue
		}
		mods = append(mods, entry)
	}

	sort.Slice(mods, func(i, j int) bool { return mods[i].ID < mods[j].ID })
	return &types.ModuleIndex{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		MinCoreVersion: "0.1.0",
		Modules:        mods,
	}, nil
}

func moduleEntryFromZip(zipPath string) (types.ModuleEntry, error) {
	abs, err := filepath.Abs(zipPath)
	if err == nil {
		zipPath = abs
	}

	sha, err := sha256File(zipPath)
	if err != nil {
		return types.ModuleEntry{}, err
	}

	manifest, err := readManifestFromZip(zipPath)
	if err != nil {
		return types.ModuleEntry{}, err
	}
	if err := validateManifest(*manifest); err != nil {
		return types.ModuleEntry{}, err
	}

	protos := protocolsFromManifest(*manifest)
	tags := inferLocalTags(manifest.ID)

	downloadURL := (&url.URL{Scheme: "file", Path: zipPath}).String()
	return types.ModuleEntry{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: "Local module package",
		License:     manifest.License,
		Tags:        tags,
		Protocols:   protos,
		DownloadURL: downloadURL,
		SHA256:      sha,
		Homepage:    "",
	}, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readManifestFromZip(zipPath string) (*types.ModuleManifest, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if !strings.EqualFold(filepath.Base(f.Name), "manifest.json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, 2<<20))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		var manifest types.ModuleManifest
		if err := json.Unmarshal(b, &manifest); err != nil {
			return nil, err
		}
		return &manifest, nil
	}
	return nil, errors.New("manifest.json not found in module zip")
}

func protocolsFromManifest(m types.ModuleManifest) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, svc := range m.Services {
		if svc.Kind != "provider" {
			continue
		}
		p := strings.TrimSpace(svc.Protocol)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func inferLocalTags(id string) []string {
	id = strings.ToLower(strings.TrimSpace(id))
	tags := []string{"local", "provider"}
	if strings.Contains(id, "oauth") || strings.Contains(id, "auth") {
		tags = append(tags, "auth")
	}
	if strings.Contains(id, "tool") {
		tags = append(tags, "tool")
	}
	return tags
}

func parseFileDownloadURL(downloadURL string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(downloadURL))
	if err != nil || u == nil || u.Scheme != "file" {
		return "", false
	}
	p, err := url.PathUnescape(u.Path)
	if err != nil {
		p = u.Path
	}
	// Windows file URIs can look like file:///C:/path.
	if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
		p = strings.TrimPrefix(p, "/")
	}
	if strings.TrimSpace(p) == "" {
		return "", false
	}
	return p, true
}
