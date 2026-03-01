package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIRequestToCodex_ReasoningEffortOptional(t *testing.T) {
	in := []byte(`{"model":"gpt-5.2","messages":[{"role":"user","content":"hi"}]}`)
	out := ConvertOpenAIRequestToCodex("gpt-5.2", in, false)
	if v := gjson.GetBytes(out, "reasoning.effort"); v.Exists() {
		t.Fatalf("expected no reasoning.effort when request omits reasoning_effort, got %s", v.Raw)
	}
}

func TestConvertOpenAIRequestToCodex_ReasoningEffortMapped(t *testing.T) {
	in := []byte(`{"model":"gpt-5.2","reasoning_effort":"high","messages":[{"role":"user","content":"hi"}]}`)
	out := ConvertOpenAIRequestToCodex("gpt-5.2", in, false)
	if got := gjson.GetBytes(out, "reasoning.effort").String(); got != "high" {
		t.Fatalf("expected reasoning.effort=high, got %q", got)
	}
}
