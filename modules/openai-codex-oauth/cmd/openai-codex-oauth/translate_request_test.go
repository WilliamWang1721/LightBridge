package main

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestBuildCodexRequest_ModelTagReasoningEffort(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		aliases          map[string]string
		wantModel        string
		wantEffort       string
		wantEffortExists bool
	}{
		{
			name:             "tag_high_injected",
			body:             `{"model":"gpt-5.2(high)","messages":[{"role":"user","content":"hi"}]}`,
			wantModel:        "gpt-5.2",
			wantEffort:       "high",
			wantEffortExists: true,
		},
		{
			name:             "body_reasoning_wins",
			body:             `{"model":"gpt-5.2(high)","reasoning_effort":"low","messages":[{"role":"user","content":"hi"}]}`,
			wantModel:        "gpt-5.2",
			wantEffort:       "low",
			wantEffortExists: true,
		},
		{
			name:             "auto_no_effort",
			body:             `{"model":"gpt-5.2(auto)","messages":[{"role":"user","content":"hi"}]}`,
			wantModel:        "gpt-5.2",
			wantEffortExists: false,
		},
		{
			name:             "no_tag_no_effort",
			body:             `{"model":"gpt-5.2","messages":[{"role":"user","content":"hi"}]}`,
			wantModel:        "gpt-5.2",
			wantEffortExists: false,
		},
		{
			name:             "alias_high",
			body:             `{"model":"gpt-5.2(RaisingFaults)","messages":[{"role":"user","content":"hi"}]}`,
			aliases:          map[string]string{"raisingfaults": "high"},
			wantModel:        "gpt-5.2",
			wantEffort:       "high",
			wantEffortExists: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, _, _, err := buildCodexRequest([]byte(tc.body), false, tc.aliases)
			if err != nil {
				t.Fatalf("buildCodexRequest failed: %v", err)
			}

			if got := gjson.GetBytes(out, "model").String(); got != tc.wantModel {
				t.Fatalf("expected model=%q got=%q", tc.wantModel, got)
			}
			effort := gjson.GetBytes(out, "reasoning.effort")
			if tc.wantEffortExists && !effort.Exists() {
				t.Fatalf("expected reasoning.effort")
			}
			if !tc.wantEffortExists && effort.Exists() {
				t.Fatalf("did not expect reasoning.effort, got=%s", effort.Raw)
			}
			if tc.wantEffortExists && effort.String() != tc.wantEffort {
				t.Fatalf("expected effort=%q got=%q", tc.wantEffort, effort.String())
			}
		})
	}
}

func TestBuildCodexRequest_InvalidModelTag(t *testing.T) {
	_, _, _, err := buildCodexRequest([]byte(`{"model":"gpt-5.2(unknown)","messages":[{"role":"user","content":"hi"}]}`), false, nil)
	if err == nil {
		t.Fatal("expected invalid tag error")
	}
}
