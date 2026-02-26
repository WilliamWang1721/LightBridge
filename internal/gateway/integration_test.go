package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
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
)

func setupGateway(t *testing.T, st *store.Store, dataDir string, moduleIndexURL string) *httptest.Server {
	t.Helper()
	resolver := routing.NewResolver(st, rand.New(rand.NewSource(7)))
	registry := providers.NewRegistry(
		providers.NewHTTPForwardAdapter(types.ProtocolForward, nil),
		providers.NewHTTPForwardAdapter(types.ProtocolHTTPOpenAI, nil),
		providers.NewHTTPForwardAdapter(types.ProtocolHTTPRPC, nil),
		providers.NewAnthropicAdapter(nil),
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
			fmt.Fprint(w, "data: {\"type\":\"message_start\"}\n\n")
			fmt.Fprint(w, "event: content_block_delta\n")
			fmt.Fprint(w, "data: {\"delta\":{\"text\":\"hello\"}}\n\n")
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
