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
	"os"
	"path/filepath"
	"sort"
	"strings"
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
}

func NewMarketplace(st *store.Store, baseDir string, client *http.Client) *Marketplace {
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	return &Marketplace{
		client:  client,
		store:   st,
		baseDir: baseDir,
	}
}

func (m *Marketplace) FetchIndex(ctx context.Context, indexURL string) (*types.ModuleIndex, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.DownloadURL, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("download failed with %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "lightbridge-module-*.zip")
	if err != nil {
		return nil, nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		_ = tmp.Close()
		return nil, nil, err
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

	installed := &types.ModuleInstalled{
		ID:          manifest.ID,
		Version:     manifest.Version,
		InstallPath: filepath.Dir(manifestPath),
		Enabled:     true,
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
