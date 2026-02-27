package modules_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lightbridge/internal/modules"
	"lightbridge/internal/testutil"
)

func TestLocalMarketplaceScansModulesDirAndInstallsFromFileURL(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	modDir := t.TempDir()
	zipPath := filepath.Join(modDir, "sample-module.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0o644); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	t.Setenv("LIGHTBRIDGE_MODULES_DIR", modDir)

	market := modules.NewMarketplace(st, dataDir, nil)
	idx, err := market.FetchIndex(context.Background(), "local")
	if err != nil {
		t.Fatalf("fetch local index: %v", err)
	}
	if idx == nil || len(idx.Modules) != 1 {
		t.Fatalf("expected 1 module, got %+v", idx)
	}

	entry := idx.Modules[0]
	if entry.ID != "sample-module" {
		t.Fatalf("expected sample-module id, got %q", entry.ID)
	}
	if entry.SHA256 != zipSHA {
		t.Fatalf("expected sha %s, got %s", zipSHA, entry.SHA256)
	}
	if !strings.HasPrefix(entry.DownloadURL, "file://") {
		t.Fatalf("expected file:// download_url, got %q", entry.DownloadURL)
	}

	installed, manifest, err := market.Install(context.Background(), entry)
	if err != nil {
		t.Fatalf("install from file url: %v", err)
	}
	if installed == nil || manifest == nil {
		t.Fatalf("expected installed+manifest, got installed=%v manifest=%v", installed, manifest)
	}
	if installed.SHA256 != zipSHA {
		t.Fatalf("expected installed sha %s, got %s", zipSHA, installed.SHA256)
	}
}
