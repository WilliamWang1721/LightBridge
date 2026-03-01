package gateway

import "testing"

func TestUsageFromJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		raw       string
		wantInput int
		wantOut   int
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
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in, out := usageFromJSON([]byte(tc.raw))
			if in != tc.wantInput || out != tc.wantOut {
				t.Fatalf("usage mismatch: got input=%d output=%d, want input=%d output=%d", in, out, tc.wantInput, tc.wantOut)
			}
		})
	}
}

func TestUsageFromResponseSSE(t *testing.T) {
	t.Parallel()

	sse := "data: {\"id\":\"chunk-1\",\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":8}}\n\n" +
		"data: {\"id\":\"chunk-2\",\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":13}}\n\n" +
		"data: [DONE]\n\n"

	in, out := usageFromResponse("text/event-stream", []byte(sse))
	if in != 21 || out != 13 {
		t.Fatalf("usage mismatch from sse: got input=%d output=%d", in, out)
	}
}
