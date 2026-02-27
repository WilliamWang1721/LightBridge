package modules_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lightbridge/internal/modules"
	"lightbridge/internal/testutil"
)

func TestGitHubModulesIndexScansZipFiles(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/modrepo/contents/MODULES":
			if r.URL.Query().Get("ref") != "main" {
				http.Error(w, "bad ref", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"type":         "file",
				"name":         "sample-module.zip",
				"download_url": ts.URL + "/download/sample-module.zip",
			}})
		case "/download/sample-module.zip":
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	t.Setenv("LIGHTBRIDGE_GITHUB_API_BASE", ts.URL)
	market := modules.NewMarketplace(st, dataDir, nil)

	idx, err := market.FetchIndex(context.Background(), "github:acme/modrepo/MODULES@main")
	if err != nil {
		t.Fatalf("fetch github index: %v", err)
	}
	if idx == nil || len(idx.Modules) != 1 {
		t.Fatalf("expected 1 module, got %+v", idx)
	}
	got := idx.Modules[0]
	if got.ID != "sample-module" {
		t.Fatalf("expected sample-module id, got %q", got.ID)
	}
	if got.SHA256 != zipSHA {
		t.Fatalf("expected sha %s, got %s", zipSHA, got.SHA256)
	}
	if got.DownloadURL == "" {
		t.Fatalf("expected download_url to be set")
	}
}
