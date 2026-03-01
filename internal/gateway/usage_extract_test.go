package gateway

import "testing"

func TestUsageFromJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		raw           string
		wantInput     int
		wantOut       int
		wantReasoning int
		wantCached    int
	}{
		{
			name:      "chat completions usage",
			raw:       `{"usage":{"prompt_tokens":120,"completion_tokens":45,"total_tokens":165}}`,
			wantInput: 120,
			wantOut:   45,
		},
		{
			name:      "responses usage",
			raw:       `{"usage":{"input_tokens":88,"output_tokens":12,"total_tokens":100}}`,
			wantInput: 88,
			wantOut:   12,
		},
		{
			name:      "wrapped response usage",
			raw:       `{"response":{"usage":{"input_tokens":9,"output_tokens":4}}}`,
			wantInput: 9,
			wantOut:   4,
		},
		{
			name:      "total tokens fallback",
			raw:       `{"usage":{"total_tokens":33}}`,
			wantInput: 0,
			wantOut:   33,
		},
		{
			name:          "usage details",
			raw:           `{"usage":{"prompt_tokens":80,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":12},"completion_tokens_details":{"reasoning_tokens":7}}}`,
			wantInput:     80,
			wantOut:       20,
			wantReasoning: 7,
			wantCached:    12,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stats := usageFromJSON([]byte(tc.raw))
			if stats.InputTokens != tc.wantInput || stats.OutputTokens != tc.wantOut || stats.ReasoningTokens != tc.wantReasoning || stats.CachedTokens != tc.wantCached {
				t.Fatalf(
					"usage mismatch: got input=%d output=%d reasoning=%d cached=%d, want input=%d output=%d reasoning=%d cached=%d",
					stats.InputTokens,
					stats.OutputTokens,
					stats.ReasoningTokens,
					stats.CachedTokens,
					tc.wantInput,
					tc.wantOut,
					tc.wantReasoning,
					tc.wantCached,
				)
			}
		})
	}
}

func TestUsageFromResponseSSE(t *testing.T) {
	t.Parallel()

	sse := "data: {\"id\":\"chunk-1\",\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":8,\"prompt_tokens_details\":{\"cached_tokens\":2}}}\n\n" +
		"data: {\"id\":\"chunk-2\",\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":13,\"completion_tokens_details\":{\"reasoning_tokens\":5}}}\n\n" +
		"data: [DONE]\n\n"

	stats := usageFromResponse("text/event-stream", []byte(sse))
	if stats.InputTokens != 21 || stats.OutputTokens != 13 || stats.ReasoningTokens != 5 || stats.CachedTokens != 2 {
		t.Fatalf("usage mismatch from sse: got %+v", stats)
	}
}
