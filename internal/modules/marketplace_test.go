package modules_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"lightbridge/internal/gateway"
	"lightbridge/internal/modules"
	"lightbridge/internal/providers"
	"lightbridge/internal/routing"
	"lightbridge/internal/store"
	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func buildSampleModuleZip(t *testing.T) ([]byte, string) {
	t.Helper()
	root := repoRoot(t)
	work := t.TempDir()
	binDir := filepath.Join(work, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	binaryPath := filepath.Join(binDir, "sample-module")
	sourcePath := filepath.Join(root, "tests/testdata/module-sample/cmd/sample-module")
	cmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build sample module: %v\n%s", err, string(output))
	}

	manifest := map[string]any{
		"id":               "sample-module",
		"name":             "Sample Module",
		"version":          "0.1.0",
		"license":          "MIT",
		"min_core_version": "0.1.0",
		"entrypoints": map[string]any{
			runtime.GOOS + "/" + runtime.GOARCH: map[string]any{
				"command": "bin/sample-module",
				"args":    []string{},
			},
		},
		"services": []map[string]any{{
			"kind":     "provider",
			"protocol": "http_openai",
			"health": map[string]any{
				"type": "http",
				"path": "/health",
			},
			"expose_provider_aliases": []string{"samplemod"},
		}},
		"config_schema":   map[string]any{"type": "object"},
		"config_defaults": map[string]any{},
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addFileToZip(t, zw, "manifest.json", manifestBytes, 0o644)
	addFileToZip(t, zw, "README.md", []byte("# Sample Module\n"), 0o644)
	addFileToZip(t, zw, "LICENSE", []byte("MIT"), 0o644)
	binaryBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	addFileToZip(t, zw, "bin/sample-module", binaryBytes, 0o755)
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

func addFileToZip(t *testing.T, zw *zip.Writer, name string, data []byte, mode os.FileMode) {
	t.Helper()
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(mode)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatalf("create zip entry %s: %v", name, err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write zip entry %s: %v", name, err)
	}
}

func setupGatewayForModules(t *testing.T, st *store.Store, dataDir, indexURL string, mgr *modules.Manager) *httptest.Server {
	t.Helper()
	resolver := routing.NewResolver(st, rand.New(rand.NewSource(9)))
	registry := providers.NewRegistry(
		providers.NewHTTPForwardAdapter(types.ProtocolOpenAI, nil),
		providers.NewHTTPForwardAdapter(types.ProtocolForward, nil),
		providers.NewHTTPForwardAdapter(types.ProtocolHTTPOpenAI, nil),
		providers.NewHTTPForwardAdapter(types.ProtocolHTTPRPC, nil),
		providers.NewOpenAIResponsesAdapter(nil),
		providers.NewGeminiAdapter(nil),
		providers.NewAnthropicAdapter(nil),
		providers.NewAzureOpenAIAdapter(nil),
		providers.NewGRPCChatAdapter(),
	)
	market := modules.NewMarketplace(st, dataDir, nil)
	srv, err := gateway.New(gateway.Config{ListenAddr: "127.0.0.1:0", ModuleIndexURL: indexURL}, st, resolver, registry, market, mgr, "test-secret")
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	return httptest.NewServer(srv.Handler())
}

func createClientKey(t *testing.T, st *store.Store) string {
	t.Helper()
	key := fmt.Sprintf("lbk_module_%d", time.Now().UnixNano())
	if err := st.CreateClientKey(context.Background(), types.ClientAPIKey{
		ID:        fmt.Sprintf("id_%d", time.Now().UnixNano()),
		Key:       key,
		Name:      "module-test",
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create client key: %v", err)
	}
	return key
}

func TestMarketplaceSHA256Mismatch(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, _ := buildSampleModuleZip(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	_, _, err := market.Install(context.Background(), types.ModuleEntry{
		ID:          "sample-module",
		Name:        "Sample Module",
		Version:     "0.1.0",
		DownloadURL: server.URL,
		SHA256:      strings.Repeat("0", 64),
		Protocols:   []string{"http_openai"},
	})
	if err == nil {
		t.Fatalf("expected sha mismatch error")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected sha mismatch, got %v", err)
	}
}

func TestInstallAndRunModuleThroughGateway(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	index := types.ModuleIndex{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		MinCoreVersion: "0.1.0",
		Modules: []types.ModuleEntry{{
			ID:          "sample-module",
			Name:        "Sample Module",
			Version:     "0.1.0",
			Description: "test module",
			License:     "MIT",
			Protocols:   []string{"http_openai"},
			DownloadURL: "",
			SHA256:      zipSHA,
		}},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			idx := index
			idx.Modules[0].DownloadURL = server.URL + "/module.zip"
			_ = json.NewEncoder(w).Encode(idx)
		case "/module.zip":
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	fetched, err := market.FetchIndex(context.Background(), server.URL+"/index.json")
	if err != nil {
		t.Fatalf("fetch index: %v", err)
	}
	entry := fetched.Modules[0]
	installed, _, err := market.Install(context.Background(), entry)
	if err != nil {
		t.Fatalf("install module: %v", err)
	}

	mgr := modules.NewManager(st, dataDir)
	rt, err := mgr.StartInstalledModule(context.Background(), installed.ID)
	if err != nil {
		t.Fatalf("start module: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.StopModule(context.Background(), installed.ID)
	})
	if rt.HTTPPort == 0 {
		t.Fatalf("expected runtime HTTP port")
	}

	provider, err := st.GetProvider(context.Background(), "samplemod")
	if err != nil {
		t.Fatalf("get module provider: %v", err)
	}
	if provider == nil || provider.Protocol != types.ProtocolHTTPOpenAI {
		t.Fatalf("expected samplemod provider protocol http_openai, got %+v", provider)
	}

	if err := st.UpsertModel(context.Background(), types.Model{ID: "gpt-4o-mini", DisplayName: "gpt-4o-mini", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(context.Background(), "gpt-4o-mini", []types.ModelRoute{{
		ModelID:       "gpt-4o-mini",
		ProviderID:    "samplemod",
		UpstreamModel: "sample-module-model",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	apiKey := createClientKey(t, st)
	gw := setupGatewayForModules(t, st, dataDir, server.URL+"/index.json", mgr)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/v1/models", nil)
	req.Header.Set("authorization", "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("models request: %v", err)
	}
	defer resp.Body.Close()
	modelsBody, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(modelsBody, []byte("gpt-4o-mini@samplemod")) {
		t.Fatalf("expected module variant in /v1/models: %s", string(modelsBody))
	}

	chatPayload := `{"model":"gpt-4o-mini@samplemod","messages":[{"role":"user","content":"hello"}]}`
	req, _ = http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(chatPayload))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("chat request: %v", err)
	}
	defer resp.Body.Close()
	chatBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(chatBody))
	}
	if !bytes.Contains(chatBody, []byte("module-ok")) {
		t.Fatalf("expected module response body, got %s", string(chatBody))
	}
}

func TestModuleStartNotTiedToContext(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	installed, _, err := market.Install(context.Background(), types.ModuleEntry{
		ID:          "sample-module",
		Name:        "Sample Module",
		Version:     "0.1.0",
		DownloadURL: server.URL,
		SHA256:      zipSHA,
		Protocols:   []string{"http_openai"},
	})
	if err != nil {
		t.Fatalf("install module: %v", err)
	}

	mgr := modules.NewManager(st, dataDir)
	ctx, cancel := context.WithCancel(context.Background())
	rt, err := mgr.StartInstalledModule(ctx, installed.ID)
	if err != nil {
		t.Fatalf("start module: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.StopModule(context.Background(), installed.ID)
	})

	cancel()
	time.Sleep(250 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", rt.HTTPPort))
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected health status: %d", resp.StatusCode)
	}
}

func TestModuleConfigPreservedOnRestart(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	installed, _, err := market.Install(context.Background(), types.ModuleEntry{
		ID:          "sample-module",
		Name:        "Sample Module",
		Version:     "0.1.0",
		DownloadURL: server.URL,
		SHA256:      zipSHA,
		Protocols:   []string{"http_openai"},
	})
	if err != nil {
		t.Fatalf("install module: %v", err)
	}

	mgr := modules.NewManager(st, dataDir)
	rt, err := mgr.StartInstalledModule(context.Background(), installed.ID)
	if err != nil {
		t.Fatalf("start module: %v", err)
	}
	if rt.HTTPPort == 0 {
		t.Fatalf("expected http port")
	}

	cfgPath := mgr.ModuleConfigPath(installed.ID)
	custom := []byte("{\"hello\":\"world\"}")
	if err := os.WriteFile(cfgPath, custom, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_ = mgr.StopModule(context.Background(), installed.ID)

	_, err = mgr.StartInstalledModule(context.Background(), installed.ID)
	if err != nil {
		t.Fatalf("restart module: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.StopModule(context.Background(), installed.ID)
	})

	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.TrimSpace(string(b)) != string(custom) {
		t.Fatalf("expected config preserved, got %s", string(b))
	}
}

func TestMarketplaceInstallPreservesEnabledFlag(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	zipBytes, zipSHA := buildSampleModuleZip(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	entry := types.ModuleEntry{
		ID:          "sample-module",
		Name:        "Sample Module",
		Version:     "0.1.0",
		DownloadURL: server.URL,
		SHA256:      zipSHA,
		Protocols:   []string{"http_openai"},
	}
	installed, _, err := market.Install(context.Background(), entry)
	if err != nil {
		t.Fatalf("install module: %v", err)
	}
	if err := st.SetModuleEnabled(context.Background(), installed.ID, false); err != nil {
		t.Fatalf("disable module: %v", err)
	}

	installed2, _, err := market.Install(context.Background(), entry)
	if err != nil {
		t.Fatalf("reinstall module: %v", err)
	}
	if installed2.Enabled {
		t.Fatalf("expected enabled flag preserved as false, got %+v", installed2)
	}
}
