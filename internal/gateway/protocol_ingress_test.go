package gateway

import "testing"

func TestParseIngressRoute(t *testing.T) {
	tests := []struct {
		path  string
		ok    bool
		proto string
		app   string
		proxy string
		kind  string
		force bool
	}{
		{path: "/v1/chat/completions", ok: true, proto: "openai", proxy: "/v1/chat/completions", kind: endpointKindChatCompletions},
		{path: "/openai/v1/models", ok: true, proto: "openai", proxy: "/v1/models", kind: endpointKindModels},
		{path: "/openai/codex/v1/chat/completions", ok: true, proto: "openai", app: "codex", proxy: "/v1/chat/completions", kind: endpointKindChatCompletions},
		{path: "/openai-responses/v1/responses", ok: true, proto: "openai_responses", proxy: "/v1/responses", kind: endpointKindResponses},
		{path: "/anthropic/v1/messages", ok: true, proto: "anthropic", proxy: "/v1/messages", kind: endpointKindMessages, force: true},
		{path: "/gemini/v1beta/models/gemini-2.5-pro:generateContent", ok: true, proto: "gemini", proxy: "/v1beta/models/gemini-2.5-pro:generateContent", kind: endpointKindGenerateContent, force: true},
		{path: "/gemini/myapp/v1beta/models/gemini-2.5-pro:streamGenerateContent", ok: true, proto: "gemini", app: "myapp", proxy: "/v1beta/models/gemini-2.5-pro:streamGenerateContent", kind: endpointKindStreamGenerateContent, force: true},
		{path: "/gemini/v1/chat/completions", ok: true, proto: "openai", proxy: "/v1/chat/completions", kind: endpointKindChatCompletions},
		{path: "/azure/openai/v1/chat/completions", ok: true, proto: "azure_openai", proxy: "/v1/chat/completions", kind: endpointKindChatCompletions},
		{path: "/azure/openai/cherry-studio/deployments/gpt4/chat/completions", ok: true, proto: "azure_openai", app: "cherry-studio", proxy: "/azure/openai/deployments/gpt4/chat/completions", kind: "azure_legacy", force: true},
		{path: "/unknown/v1/chat/completions", ok: false},
	}

	for _, tc := range tests {
		got, ok := parseIngressRoute(tc.path)
		if ok != tc.ok {
			t.Fatalf("path=%s expected ok=%v got=%v", tc.path, tc.ok, ok)
		}
		if !ok {
			continue
		}
		if got.Protocol != tc.proto || got.AppID != tc.app || got.ProxyPath != tc.proxy || got.EndpointKind != tc.kind || got.ForceProviderByType != tc.force {
			t.Fatalf("path=%s got=%+v", tc.path, got)
		}
	}
}

func TestRequestModelFromPath(t *testing.T) {
	tests := []struct {
		path  string
		model string
	}{
		{path: "/v1beta/models/gemini-2.5-pro:generateContent", model: "gemini-2.5-pro"},
		{path: "/azure/openai/deployments/gpt-4o/chat/completions", model: "gpt-4o"},
		{path: "/openai/deployments/my-dep/embeddings", model: "my-dep"},
		{path: "/v1/chat/completions", model: ""},
	}
	for _, tc := range tests {
		if got := requestModelFromPath(tc.path); got != tc.model {
			t.Fatalf("path=%s expected model=%s got=%s", tc.path, tc.model, got)
		}
	}
}
