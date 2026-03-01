package gateway_test

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
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"lightbridge/internal/gateway"
	"lightbridge/internal/modules"
	"lightbridge/internal/providers"
	"lightbridge/internal/routing"
	"lightbridge/internal/store"
	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
	"lightbridge/internal/util"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
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

func setupGateway(t *testing.T, st *store.Store, dataDir string, moduleIndexURL string) *httptest.Server {
	t.Helper()
	resolver := routing.NewResolver(st, rand.New(rand.NewSource(7)))
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
	moduleMgr := modules.NewManager(st, dataDir)
	srv, err := gateway.New(gateway.Config{ListenAddr: "127.0.0.1:0", ModuleIndexURL: moduleIndexURL}, st, resolver, registry, market, moduleMgr, "test-secret")
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	return httptest.NewServer(srv.Handler())
}

func createClientKey(t *testing.T, st *store.Store) string {
	t.Helper()
	key := fmt.Sprintf("lbk_test_%d", time.Now().UnixNano())
	err := st.CreateClientKey(context.Background(), types.ClientAPIKey{
		ID:        fmt.Sprintf("k_%d", time.Now().UnixNano()),
		Key:       key,
		Name:      "test",
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create client key: %v", err)
	}
	return key
}

func TestAutoStartEnabledModulesOnNoHealthyProvider(t *testing.T) {
	st, dataDir := testutil.NewStore(t)
	ctx := context.Background()

	// Simulate a setup where built-in providers are present but unusable.
	for _, id := range []string{"forward", "anthropic"} {
		p, err := st.GetProvider(ctx, id)
		if err != nil || p == nil {
			t.Fatalf("get provider %s: %v", id, err)
		}
		p.Enabled = false
		p.Health = "down"
		if err := st.UpsertProvider(ctx, *p); err != nil {
			t.Fatalf("disable provider %s: %v", id, err)
		}
	}

	zipBytes, zipSHA := buildSampleModuleZip(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer up.Close()

	market := modules.NewMarketplace(st, dataDir, nil)
	installed, _, err := market.Install(ctx, types.ModuleEntry{
		ID:          "sample-module",
		Name:        "Sample Module",
		Version:     "0.1.0",
		DownloadURL: up.URL,
		SHA256:      zipSHA,
		Protocols:   []string{"http_openai"},
	})
	if err != nil {
		t.Fatalf("install module: %v", err)
	}

	mgr := modules.NewManager(st, dataDir)
	t.Cleanup(func() { mgr.StopAll(context.Background()) })

	resolver := routing.NewResolver(st, rand.New(rand.NewSource(11)))
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
	srv, err := gateway.New(gateway.Config{ListenAddr: "127.0.0.1:0", ModuleIndexURL: ""}, st, resolver, registry, market, mgr, "test-secret")
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	apiKey := createClientKey(t, st)
	body := `{"model":"some-unknown-model","messages":[{"role":"user","content":"ping"}],"stream":false}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("chat request: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected autostart to succeed, status=%d body=%s", resp.StatusCode, string(b))
	}
	if !bytes.Contains(b, []byte("module-ok")) {
		t.Fatalf("expected module response, got %s", string(b))
	}

	// Make sure the module was actually started.
	if _, err := st.GetProvider(ctx, "samplemod"); err != nil {
		t.Fatalf("get started provider: %v", err)
	}

	_ = mgr.StopModule(context.Background(), installed.ID)
}

func TestForwardProviderPassThrough(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	var seenModel atomic.Value
	var seenAuth atomic.Value
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if model, ok := payload["model"].(string); ok {
				seenModel.Store(model)
			}
			seenAuth.Store(r.Header.Get("Authorization"))
			if stream, _ := payload["stream"].(bool); stream {
				w.Header().Set("content-type", "text/event-stream")
				fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n")
				fmt.Fprintf(w, "data: [DONE]\n\n")
				return
			}
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-1",
				"object":  "chat.completion",
				"model":   payload["model"],
				"created": time.Now().Unix(),
				"choices": []map[string]any{{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "pong"},
					"finish_reason": "stop",
				}},
			})
		case "/v1/models":
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	config := map[string]any{
		"base_url":      upstream.URL,
		"api_key":       "upstream-secret",
		"extra_headers": map[string]string{"x-test": "1"},
		"model_remap":   map[string]string{},
	}
	configBytes, _ := json.Marshal(config)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "forward",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolForward,
		Endpoint:   upstream.URL,
		ConfigJSON: string(configBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert forward: %v", err)
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "my-gpt", DisplayName: "my-gpt", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "my-gpt", []types.ModelRoute{{
		ModelID:       "my-gpt",
		ProviderID:    "forward",
		UpstreamModel: "gpt-upstream",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	apiKey := createClientKey(t, st)
	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	body := `{"model":"my-gpt","messages":[{"role":"user","content":"ping"}],"stream":false}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("chat request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status: %d body=%s", resp.StatusCode, string(b))
	}
	if got := seenModel.Load(); got != "gpt-upstream" {
		t.Fatalf("expected upstream model remapped to gpt-upstream, got %v", got)
	}
	if got := seenAuth.Load(); got != "Bearer upstream-secret" {
		t.Fatalf("expected upstream auth header, got %v", got)
	}

	streamReqBody := `{"model":"my-gpt","messages":[{"role":"user","content":"ping"}],"stream":true}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(streamReqBody))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	streamBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(streamBody), "[DONE]") || !strings.Contains(string(streamBody), "\"content\":\"hi\"") {
		t.Fatalf("unexpected stream body: %s", string(streamBody))
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/v1/models", nil)
	req.Header.Set("authorization", "Bearer "+apiKey)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("models request: %v", err)
	}
	defer resp.Body.Close()
	modelsBody, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(modelsBody, []byte("my-gpt@forward")) {
		t.Fatalf("expected variant model in list, got: %s", string(modelsBody))
	}

	logs, err := st.ListRequestLogs(ctx, 20)
	if err != nil {
		t.Fatalf("list request logs: %v", err)
	}
	foundChatLog := false
	for _, item := range logs {
		if strings.TrimSpace(item.Path) != "/v1/chat/completions" {
			continue
		}
		foundChatLog = true
		if got := strings.TrimSpace(item.ModelID); got != "gpt-upstream" {
			t.Fatalf("expected logged model to be upstream model gpt-upstream, got %q", got)
		}
	}
	if !foundChatLog {
		t.Fatalf("expected at least one /v1/chat/completions log entry")
	}
}

func TestAnthropicProviderConversion(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	var seenAnthropicModel atomic.Value
	var seenSystem atomic.Value
	anthropic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if m, ok := payload["model"].(string); ok {
			seenAnthropicModel.Store(m)
		}
		if system, ok := payload["system"].(string); ok {
			seenSystem.Store(system)
		}
		if stream, _ := payload["stream"].(bool); stream {
			w.Header().Set("content-type", "text/event-stream")
			fmt.Fprint(w, "event: message_start\n")
			fmt.Fprint(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_stream\",\"model\":\"claude-upstream\",\"usage\":{\"input_tokens\":5}}}\n\n")
			fmt.Fprint(w, "event: content_block_start\n")
			fmt.Fprint(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
			fmt.Fprint(w, "event: content_block_delta\n")
			fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n")
			fmt.Fprint(w, "event: content_block_stop\n")
			fmt.Fprint(w, "data: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
			fmt.Fprint(w, "event: message_delta\n")
			fmt.Fprint(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\n")
			fmt.Fprint(w, "event: message_stop\n")
			fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_1",
			"model": payload["model"],
			"content": []map[string]any{{
				"type": "text",
				"text": "anthropic-ok",
			}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 3, "output_tokens": 4},
		})
	}))
	defer anthropic.Close()

	cfg := map[string]any{"base_url": anthropic.URL, "api_key": "anthropic-key"}
	cfgBytes, _ := json.Marshal(cfg)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "anthropic",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolAnthropic,
		Endpoint:   anthropic.URL,
		ConfigJSON: string(cfgBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert anthropic provider: %v", err)
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "claude-opus-4-5", DisplayName: "claude-opus-4-5", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "claude-opus-4-5", []types.ModelRoute{{
		ModelID:       "claude-opus-4-5",
		ProviderID:    "anthropic",
		UpstreamModel: "claude-upstream",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	apiKey := createClientKey(t, st)
	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	nonStreamBody := `{"model":"claude-opus-4-5","messages":[{"role":"system","content":"be concise"},{"role":"user","content":"hello"}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(nonStreamBody))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("anthropic non-stream request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(bodyBytes, []byte("anthropic-ok")) {
		t.Fatalf("expected converted completion body, got %s", string(bodyBytes))
	}
	if got := seenAnthropicModel.Load(); got != "claude-upstream" {
		t.Fatalf("expected mapped upstream anthropic model, got %v", got)
	}
	if got := seenSystem.Load(); got != "be concise" {
		t.Fatalf("expected system prompt forwarded, got %v", got)
	}

	streamBody := `{"model":"claude-opus-4-5","messages":[{"role":"user","content":"stream"}],"stream":true}`
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(streamBody))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("anthropic stream request: %v", err)
	}
	defer resp.Body.Close()
	streamResp, _ := io.ReadAll(resp.Body)
	str := string(streamResp)
	if !strings.Contains(str, "chat.completion.chunk") || !strings.Contains(str, "[DONE]") || !strings.Contains(str, "hello") {
		t.Fatalf("unexpected anthropic stream conversion: %s", str)
	}
}

func TestOpenAIResponsesPrefixAndXAPIKeyAuth(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	var seenPath atomic.Value
	var seenAuth atomic.Value
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath.Store(r.URL.Path)
		seenAuth.Store(r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-2",
			"object":  "chat.completion",
			"model":   payload["model"],
			"created": time.Now().Unix(),
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "ok"},
				"finish_reason": "stop",
			}},
		})
	}))
	defer upstream.Close()

	cfg := map[string]any{
		"base_url": upstream.URL,
		"api_key":  "upstream-secret",
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "forward",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolForward,
		Endpoint:   upstream.URL,
		ConfigJSON: string(cfgBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert forward: %v", err)
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "demo-model", DisplayName: "demo-model", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "demo-model", []types.ModelRoute{{
		ModelID:       "demo-model",
		ProviderID:    "forward",
		UpstreamModel: "gpt-upstream",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	apiKey := createClientKey(t, st)
	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	reqBody := `{"model":"demo-model","messages":[{"role":"user","content":"ping"}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/openai-responses/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(b))
	}
	if got := seenPath.Load(); got != "/v1/chat/completions" {
		t.Fatalf("expected upstream path /v1/chat/completions, got %v", got)
	}
	if got := seenAuth.Load(); got != "Bearer upstream-secret" {
		t.Fatalf("expected upstream auth Bearer upstream-secret, got %v", got)
	}
}

func TestGeminiNativeIngressProxy(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	var seenPath atomic.Value
	var seenAPIKey atomic.Value
	geminiUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath.Store(r.URL.Path)
		seenAPIKey.Store(r.URL.Query().Get("key"))
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-pro"}`))
	}))
	defer geminiUpstream.Close()

	cfg := map[string]any{
		"base_url": geminiUpstream.URL,
		"api_key":  "gemini-provider-key",
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "gemini",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolGemini,
		Endpoint:   geminiUpstream.URL,
		ConfigJSON: string(cfgBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert gemini provider: %v", err)
	}

	apiKey := createClientKey(t, st)
	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	body := `{"contents":[{"parts":[{"text":"hello"}]}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/gemini/v1beta/models/gemini-2.5-pro:generateContent", strings.NewReader(body))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gemini native request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(b))
	}
	if got := seenPath.Load(); got != "/v1beta/models/gemini-2.5-pro:generateContent" {
		t.Fatalf("unexpected upstream path: %v", got)
	}
	if got := seenAPIKey.Load(); got != "gemini-provider-key" {
		t.Fatalf("expected provider key in query, got %v", got)
	}
}

func TestGeminiNativeIngressReturnsStructuredNotSupportedForCrossProtocolRoute(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	var upstreamCalls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	cfg := map[string]any{
		"base_url": upstream.URL,
		"api_key":  "unused",
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "forward",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolForward,
		Endpoint:   upstream.URL,
		ConfigJSON: string(cfgBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert forward provider: %v", err)
	}

	if err := st.UpsertModel(ctx, types.Model{ID: "gem-route", DisplayName: "gem-route", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "gem-route", []types.ModelRoute{{
		ModelID:       "gem-route",
		ProviderID:    "forward",
		UpstreamModel: "gpt-4o-mini",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	apiKey := createClientKey(t, st)
	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	body := `{"contents":[{"parts":[{"text":"hello"}]}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/gemini/v1beta/models/gem-route:generateContent", strings.NewReader(body))
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gemini native request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 501, status=%d body=%s", resp.StatusCode, string(b))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if got := strings.TrimSpace(fmt.Sprint(errObj["type"])); got != "not_supported" {
		t.Fatalf("unexpected error type: %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(errObj["source_protocol"])); got != "gemini" {
		t.Fatalf("unexpected source protocol: %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(errObj["target_protocol"])); got != "openai" {
		t.Fatalf("unexpected target protocol: %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(errObj["endpoint_kind"])); got != "generate_content" {
		t.Fatalf("unexpected endpoint kind: %v", got)
	}
	if upstreamCalls.Load() != 0 {
		t.Fatalf("upstream should not be called on not_supported route")
	}
}

func upsertInstalledModuleForTest(t *testing.T, st *store.Store, moduleID string, enabled bool) {
	t.Helper()
	err := st.SaveInstalledModule(context.Background(), types.ModuleInstalled{
		ID:          moduleID,
		Version:     "0.1.0",
		InstallPath: "/tmp/" + moduleID,
		Enabled:     enabled,
		Protocols:   "http_rpc",
		SHA256:      "",
		InstalledAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("upsert installed module %s: %v", moduleID, err)
	}
}

func saveModuleRuntimeFromBaseURLForTest(t *testing.T, st *store.Store, moduleID, baseURL string) {
	t.Helper()
	u, err := neturl.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		t.Fatalf("parse base url %q: %v", baseURL, err)
	}
	_, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port %q: %v", u.Host, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("atoi port %q: %v", portStr, err)
	}
	err = st.SaveModuleRuntime(context.Background(), types.ModuleRuntime{
		ModuleID:    moduleID,
		PID:         1,
		HTTPPort:    port,
		GRPCPort:    0,
		Status:      "running",
		LastStartAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save module runtime %s: %v", moduleID, err)
	}
}

func decodeJSONMap(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("decode json body: %v", err)
	}
	return out
}

func containsMethodID(methods []any, want string) bool {
	for _, item := range methods {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(m["id"])) == want {
			return true
		}
	}
	return false
}

func TestAuthMethodsExposePasskeyAndConditionalTOTP(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	totpModule := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/totp/devices":
			username := strings.TrimSpace(r.URL.Query().Get("username"))
			data := []map[string]any{}
			if username == "with-device" {
				data = append(data, map[string]any{
					"device_id":    "dev_1",
					"label":        "Authenticator",
					"created_at":   time.Now().UTC().Format(time.RFC3339),
					"last_used_at": "",
				})
			}
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		default:
			http.NotFound(w, r)
		}
	}))
	defer totpModule.Close()

	upsertInstalledModuleForTest(t, st, "passkey-login", true)
	upsertInstalledModuleForTest(t, st, "totp-2fa-login", true)
	saveModuleRuntimeFromBaseURLForTest(t, st, "totp-2fa-login", totpModule.URL)

	if err := st.SetSetting(ctx, "admin_2fa_enabled", "1"); err != nil {
		t.Fatalf("set policy enabled: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_require_for_password", "0"); err != nil {
		t.Fatalf("set policy require_for_password: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_require_for_passkey", "0"); err != nil {
		t.Fatalf("set policy require_for_passkey: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_allow_totp_only", "1"); err != nil {
		t.Fatalf("set policy allow_totp_only: %v", err)
	}

	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/admin/api/auth/methods")
	if err != nil {
		t.Fatalf("auth methods request(no username): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth methods(no username) status=%d", resp.StatusCode)
	}
	body := decodeJSONMap(t, resp.Body)
	methods, _ := body["methods"].([]any)
	if !containsMethodID(methods, "passkey") {
		t.Fatalf("expected passkey method in no-username response: %+v", body)
	}
	if !containsMethodID(methods, "totp") {
		t.Fatalf("expected totp method in no-username response: %+v", body)
	}

	resp, err = http.Get(ts.URL + "/admin/api/auth/methods?username=with-device")
	if err != nil {
		t.Fatalf("auth methods request(with-device): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth methods(with-device) status=%d", resp.StatusCode)
	}
	body = decodeJSONMap(t, resp.Body)
	methods, _ = body["methods"].([]any)
	if !containsMethodID(methods, "passkey") || !containsMethodID(methods, "totp") {
		t.Fatalf("expected passkey+totp for with-device user, got %+v", body)
	}

	resp, err = http.Get(ts.URL + "/admin/api/auth/methods?username=without-device")
	if err != nil {
		t.Fatalf("auth methods request(without-device): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth methods(without-device) status=%d", resp.StatusCode)
	}
	body = decodeJSONMap(t, resp.Body)
	methods, _ = body["methods"].([]any)
	if !containsMethodID(methods, "passkey") {
		t.Fatalf("expected passkey for without-device user, got %+v", body)
	}
	if containsMethodID(methods, "totp") {
		t.Fatalf("did not expect totp for without-device user, got %+v", body)
	}
}

func TestPasswordLoginRequiresTwoFAChallenge(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	totpModule := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/totp/devices":
			username := strings.TrimSpace(r.URL.Query().Get("username"))
			data := []map[string]any{}
			if username == "admin" {
				data = append(data, map[string]any{
					"device_id":    "dev_1",
					"label":        "Authenticator",
					"created_at":   time.Now().UTC().Format(time.RFC3339),
					"last_used_at": "",
				})
			}
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		case "/totp/verify":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			username := strings.TrimSpace(fmt.Sprint(req["username"]))
			code := strings.TrimSpace(fmt.Sprint(req["code"]))
			if username == "admin" && code == "123456" {
				w.Header().Set("content-type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "username": "admin"})
				return
			}
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid 2fa code"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer totpModule.Close()

	upsertInstalledModuleForTest(t, st, "totp-2fa-login", true)
	saveModuleRuntimeFromBaseURLForTest(t, st, "totp-2fa-login", totpModule.URL)

	if err := st.SetSetting(ctx, "admin_2fa_enabled", "1"); err != nil {
		t.Fatalf("set policy enabled: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_require_for_password", "1"); err != nil {
		t.Fatalf("set policy require_for_password: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_require_for_passkey", "0"); err != nil {
		t.Fatalf("set policy require_for_passkey: %v", err)
	}
	if err := st.SetSetting(ctx, "admin_2fa_allow_totp_only", "0"); err != nil {
		t.Fatalf("set policy allow_totp_only: %v", err)
	}

	passHash, err := util.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := st.CreateAdmin(ctx, "admin", passHash); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	ts := setupGateway(t, st, dir, "")
	defer ts.Close()

	loginPayload := map[string]any{
		"username": "admin",
		"password": "secret123",
		"remember": true,
	}
	loginBytes, _ := json.Marshal(loginPayload)
	resp, err := http.Post(ts.URL+"/admin/api/login", "application/json", bytes.NewReader(loginBytes))
	if err != nil {
		t.Fatalf("password login request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("password login status=%d body=%s", resp.StatusCode, string(b))
	}
	body := decodeJSONMap(t, resp.Body)
	if ok, _ := body["ok"].(bool); !ok {
		t.Fatalf("expected ok=true in login response, got %+v", body)
	}
	if required, _ := body["requires_2fa"].(bool); !required {
		t.Fatalf("expected requires_2fa=true, got %+v", body)
	}
	ticket := strings.TrimSpace(fmt.Sprint(body["ticket"]))
	if ticket == "" {
		t.Fatalf("expected non-empty ticket in login response, got %+v", body)
	}

	badVerifyPayload := map[string]any{"ticket": ticket, "code": "000000"}
	badVerifyBytes, _ := json.Marshal(badVerifyPayload)
	resp, err = http.Post(ts.URL+"/admin/api/2fa/challenge/verify", "application/json", bytes.NewReader(badVerifyBytes))
	if err != nil {
		t.Fatalf("2fa verify bad request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for bad 2fa code, status=%d body=%s", resp.StatusCode, string(b))
	}

	verifyPayload := map[string]any{"ticket": ticket, "code": "123456"}
	verifyBytes, _ := json.Marshal(verifyPayload)
	resp, err = http.Post(ts.URL+"/admin/api/2fa/challenge/verify", "application/json", bytes.NewReader(verifyBytes))
	if err != nil {
		t.Fatalf("2fa verify request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("2fa verify status=%d body=%s", resp.StatusCode, string(b))
	}
	if got := resp.Header.Get("Set-Cookie"); !strings.Contains(got, "lightbridge_admin=") {
		t.Fatalf("expected session cookie in 2fa verify response, got %q", got)
	}
	body = decodeJSONMap(t, resp.Body)
	if next := strings.TrimSpace(fmt.Sprint(body["next"])); next != "/admin/dashboard" {
		t.Fatalf("expected next=/admin/dashboard, got %+v", body)
	}

	replayPayload := map[string]any{"ticket": ticket, "code": "123456"}
	replayBytes, _ := json.Marshal(replayPayload)
	resp, err = http.Post(ts.URL+"/admin/api/2fa/challenge/verify", "application/json", bytes.NewReader(replayBytes))
	if err != nil {
		t.Fatalf("2fa replay verify request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for consumed ticket, status=%d body=%s", resp.StatusCode, string(b))
	}
}
