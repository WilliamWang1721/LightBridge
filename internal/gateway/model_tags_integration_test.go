package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
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

	"github.com/tidwall/gjson"
)

type upstreamSeenRequest struct {
	path string
	body []byte
}

func setupGatewayWithConfig(t *testing.T, st *store.Store, dataDir string, cfg gateway.Config) *httptest.Server {
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
	moduleMgr := modules.NewManager(st, dataDir)
	srv, err := gateway.New(cfg, st, resolver, registry, market, moduleMgr, "test-secret")
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	return httptest.NewServer(srv.Handler())
}

func TestModelTagReasoningEffortPatch(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	seenCh := make(chan upstreamSeenRequest, 10)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seenCh <- upstreamSeenRequest{path: r.URL.Path, body: b}
		w.Header().Set("content-type", "application/json")
		switch r.URL.Path {
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-test",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   gjson.GetBytes(b, "model").String(),
				"choices": []map[string]any{{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "ok"},
					"finish_reason": "stop",
				}},
			})
		case "/v1/responses":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "resp-test",
				"object":  "response",
				"created": time.Now().Unix(),
				"model":   gjson.GetBytes(b, "model").String(),
				"output":  []any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	config := map[string]any{
		"base_url":    upstream.URL,
		"api_key":     "upstream-secret",
		"model_remap": map[string]string{},
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
	if err := st.UpsertModel(ctx, types.Model{ID: "gpt-5.2", DisplayName: "gpt-5.2", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "gpt-5.2", []types.ModelRoute{{
		ModelID:       "gpt-5.2",
		ProviderID:    "forward",
		UpstreamModel: "gpt-5.2",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	ts := setupGatewayWithConfig(t, st, dir, gateway.Config{
		ListenAddr:      "127.0.0.1:0",
		ModuleIndexURL:  "",
		ModelTagAliases: map[string]string{"raisingfaults": "high"},
	})
	defer ts.Close()

	apiKey := createClientKey(t, st)
	doReq := func(path, body string) (*http.Response, []byte) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
		req.Header.Set("authorization", "Bearer "+apiKey)
		req.Header.Set("content-type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %s failed: %v", path, err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp, respBody
	}
	nextSeen := func() upstreamSeenRequest {
		t.Helper()
		select {
		case v := <-seenCh:
			return v
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for upstream request")
		}
		return upstreamSeenRequest{}
	}

	resp, body := doReq("/v1/chat/completions", `{"model":"gpt-5.2(high)","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat(high) status=%d body=%s", resp.StatusCode, string(body))
	}
	seen := nextSeen()
	if gjson.GetBytes(seen.body, "model").String() != "gpt-5.2" {
		t.Fatalf("expected stripped model, got %s", gjson.GetBytes(seen.body, "model").Raw)
	}
	if gjson.GetBytes(seen.body, "reasoning_effort").String() != "high" {
		t.Fatalf("expected reasoning_effort=high, got %s", gjson.GetBytes(seen.body, "reasoning_effort").Raw)
	}

	resp, body = doReq("/v1/chat/completions", `{"model":"gpt-5.2(auto)","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat(auto) status=%d body=%s", resp.StatusCode, string(body))
	}
	seen = nextSeen()
	if gjson.GetBytes(seen.body, "reasoning_effort").Exists() {
		t.Fatalf("expected no reasoning_effort for auto, got %s", gjson.GetBytes(seen.body, "reasoning_effort").Raw)
	}

	resp, body = doReq("/v1/chat/completions", `{"model":"gpt-5.2(high)","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"low","stream":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat(body wins) status=%d body=%s", resp.StatusCode, string(body))
	}
	seen = nextSeen()
	if gjson.GetBytes(seen.body, "reasoning_effort").String() != "low" {
		t.Fatalf("expected reasoning_effort body value low, got %s", gjson.GetBytes(seen.body, "reasoning_effort").Raw)
	}

	resp, body = doReq("/v1/responses", `{"model":"gpt-5.2(RaisingFaults)","input":"hi"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("responses(alias) status=%d body=%s", resp.StatusCode, string(body))
	}
	seen = nextSeen()
	if gjson.GetBytes(seen.body, "model").String() != "gpt-5.2" {
		t.Fatalf("expected stripped model for responses, got %s", gjson.GetBytes(seen.body, "model").Raw)
	}
	if gjson.GetBytes(seen.body, "reasoning.effort").String() != "high" {
		t.Fatalf("expected reasoning.effort=high, got %s", gjson.GetBytes(seen.body, "reasoning.effort").Raw)
	}

	resp, body = doReq("/v1/responses", `{"model":"gpt-5.2(high)","input":"hi","reasoning":{"effort":"low"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("responses(body wins) status=%d body=%s", resp.StatusCode, string(body))
	}
	seen = nextSeen()
	if gjson.GetBytes(seen.body, "reasoning.effort").String() != "low" {
		t.Fatalf("expected reasoning.effort body value low, got %s", gjson.GetBytes(seen.body, "reasoning.effort").Raw)
	}

	resp, body = doReq("/v1/chat/completions", `{"model":"gpt-5.2(unknown)","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown tag should fail with 400, got status=%d body=%s", resp.StatusCode, string(body))
	}
	if gjson.GetBytes(body, "error.code").String() != "invalid_model_tag" {
		t.Fatalf("expected invalid_model_tag, got %s", gjson.GetBytes(body, "error.code").Raw)
	}
	select {
	case leaked := <-seenCh:
		t.Fatalf("unexpected upstream call on invalid tag: path=%s body=%s", leaked.path, string(leaked.body))
	default:
	}
}
