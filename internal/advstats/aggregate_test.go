package advstats

import (
	"testing"
	"time"
)

func TestAggregateIncludesAPIUsageAndSpecialBackends(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	logs := []RequestLog{
		{
			Timestamp:       now.Add(-30 * time.Minute).Format(time.RFC3339),
			ProviderID:      "codex",
			ModelID:         "gpt-4o",
			Path:            "/api/codex/v1/chat/completions",
			InputTokens:     100,
			OutputTokens:    50,
			ReasoningTokens: 10,
			CachedTokens:    20,
		},
		{
			Timestamp:       now.Add(-20 * time.Minute).Format(time.RFC3339),
			ProviderID:      "claude-code",
			ModelID:         "claude-3-5-sonnet",
			Path:            "/api/claude-code/v1/chat/completions",
			InputTokens:     80,
			OutputTokens:    20,
			ReasoningTokens: 0,
			CachedTokens:    0,
		},
		{
			Timestamp:       now.Add(-10 * time.Minute).Format(time.RFC3339),
			ProviderID:      "cherry-studio",
			ModelID:         "gpt-4o-mini",
			Path:            "/api/cherry-studio/v1/responses",
			InputTokens:     40,
			OutputTokens:    10,
			ReasoningTokens: 0,
			CachedTokens:    0,
		},
		{
			Timestamp:       now.Add(-5 * time.Minute).Format(time.RFC3339),
			ProviderID:      "forward",
			ModelID:         "gpt-4o-mini",
			Path:            "/v1/chat/completions",
			InputTokens:     20,
			OutputTokens:    5,
			ReasoningTokens: 0,
			CachedTokens:    0,
		},
		{
			Timestamp:       now.Add(-2 * time.Minute).Format(time.RFC3339),
			ProviderID:      "codex",
			ModelID:         "gpt-4o",
			Path:            "/v1/responses",
			InputTokens:     10,
			OutputTokens:    2,
			ReasoningTokens: 0,
			CachedTokens:    0,
		},
	}

	out := Aggregate(AggregateRequest{
		Start:      now.Add(-2 * time.Hour).Format(time.RFC3339),
		End:        now.Format(time.RFC3339),
		WindowLogs: logs,
		TodayLogs:  logs,
	}, now)

	codexAPI := findAPIUsage(out.APIUsage, "codex")
	if codexAPI == nil {
		t.Fatalf("expected codex API usage")
	}
	if codexAPI.BackendURL != "/api/codex/v1/*" {
		t.Fatalf("unexpected codex backend url: %q", codexAPI.BackendURL)
	}
	if codexAPI.Requests != 1 || codexAPI.TotalTokens != 150 {
		t.Fatalf("unexpected codex api usage: %+v", *codexAPI)
	}

	defaultV1 := findAPIUsage(out.APIUsage, "default-v1")
	if defaultV1 == nil {
		t.Fatalf("expected default-v1 API usage")
	}
	if defaultV1.Requests != 2 || defaultV1.TotalTokens != 37 {
		t.Fatalf("unexpected default-v1 usage: %+v", *defaultV1)
	}

	if len(out.SpecialBackends) != 3 {
		t.Fatalf("expected 3 special backends, got %d", len(out.SpecialBackends))
	}

	codexSpecial := findSpecialBackend(out.SpecialBackends, "codex")
	if codexSpecial == nil {
		t.Fatalf("expected codex special backend")
	}
	if codexSpecial.Summary.Requests != 2 || codexSpecial.Summary.TotalTokens != 162 {
		t.Fatalf("unexpected codex special summary: %+v", codexSpecial.Summary)
	}
	if len(codexSpecial.ModelUsage) == 0 || codexSpecial.ModelUsage[0].ModelID != "gpt-4o" {
		t.Fatalf("unexpected codex model usage: %+v", codexSpecial.ModelUsage)
	}

	cherrySpecial := findSpecialBackend(out.SpecialBackends, "cherry-studio")
	if cherrySpecial == nil {
		t.Fatalf("expected cherry-studio special backend")
	}
	if cherrySpecial.Summary.Requests != 1 || cherrySpecial.Summary.TotalTokens != 50 {
		t.Fatalf("unexpected cherry-studio summary: %+v", cherrySpecial.Summary)
	}
}

func TestExtractAppIDFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{path: "/api/codex/v1/chat/completions", want: "codex"},
		{path: "api/claude-code/v1/responses", want: "claude-code"},
		{path: "/API/cherry-studio/V1/responses", want: "cherry-studio"},
		{path: "/v1/chat/completions", want: ""},
		{path: "/openai/v1/chat/completions", want: ""},
	}
	for _, tc := range cases {
		if got := extractAppIDFromPath(tc.path); got != tc.want {
			t.Fatalf("extractAppIDFromPath(%q)=%q, want %q", tc.path, got, tc.want)
		}
	}
}

func findAPIUsage(list []APIUsage, backendID string) *APIUsage {
	for i := range list {
		if list[i].BackendID == backendID {
			return &list[i]
		}
	}
	return nil
}

func findSpecialBackend(list []SpecialBackendUsage, id string) *SpecialBackendUsage {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}
