package gateway

import (
	"errors"
	"testing"

	"github.com/tidwall/gjson"
)

func TestParseModelTag(t *testing.T) {
	aliases := map[string]string{
		"raisingfaults": "high",
	}
	tests := []struct {
		name       string
		model      string
		aliases    map[string]string
		wantBase   string
		wantEffort string
		wantHasTag bool
		wantErrIs  error
	}{
		{
			name:       "official_high",
			model:      "gpt-5.2(high)",
			wantBase:   "gpt-5.2",
			wantEffort: "high",
			wantHasTag: true,
		},
		{
			name:       "provider_variant",
			model:      "gpt-5.2@forward(low)",
			wantBase:   "gpt-5.2@forward",
			wantEffort: "low",
			wantHasTag: true,
		},
		{
			name:       "full_width_and_case_insensitive",
			model:      "gpt-5.2（High）",
			wantBase:   "gpt-5.2",
			wantEffort: "high",
			wantHasTag: true,
		},
		{
			name:       "alias_to_high",
			model:      "gpt-5.2(RaisingFaults)",
			aliases:    aliases,
			wantBase:   "gpt-5.2",
			wantEffort: "high",
			wantHasTag: true,
		},
		{
			name:      "unknown_tag",
			model:     "gpt-5.2(unknown)",
			wantErrIs: errInvalidModelTag,
		},
		{
			name:       "auto",
			model:      "gpt-5.2(auto)",
			wantBase:   "gpt-5.2",
			wantEffort: "",
			wantHasTag: true,
		},
		{
			name:      "missing_base",
			model:     "(high)",
			wantErrIs: errMissingModelTag,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			aliasMap := tc.aliases
			if aliasMap == nil {
				aliasMap = aliases
			}
			base, effort, hasTag, err := parseModelTag(tc.model, aliasMap)
			if tc.wantErrIs != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErrIs)
				}
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("expected error %v, got %v", tc.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if base != tc.wantBase || effort != tc.wantEffort || hasTag != tc.wantHasTag {
				t.Fatalf("base=%q effort=%q hasTag=%v", base, effort, hasTag)
			}
		})
	}
}

func TestPatchReasoningEffort(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		endpointKind       string
		baseModel          string
		effort             string
		precedenceBodyWins bool
		wantModel          string
		wantEffortPath     string
		wantEffortExists   bool
	}{
		{
			name:               "chat_inject",
			body:               `{"model":"gpt-5.2(high)","messages":[]}`,
			endpointKind:       endpointKindChatCompletions,
			baseModel:          "gpt-5.2",
			effort:             "high",
			precedenceBodyWins: true,
			wantModel:          "gpt-5.2",
			wantEffortPath:     "reasoning_effort",
			wantEffortExists:   true,
		},
		{
			name:               "chat_body_effort_wins",
			body:               `{"model":"gpt-5.2(high)","messages":[],"reasoning_effort":"low"}`,
			endpointKind:       endpointKindChatCompletions,
			baseModel:          "gpt-5.2",
			effort:             "high",
			precedenceBodyWins: true,
			wantModel:          "gpt-5.2",
			wantEffortPath:     "reasoning_effort",
			wantEffortExists:   true,
		},
		{
			name:               "responses_inject",
			body:               `{"model":"gpt-5.2(high)","input":"hi"}`,
			endpointKind:       endpointKindResponses,
			baseModel:          "gpt-5.2",
			effort:             "high",
			precedenceBodyWins: true,
			wantModel:          "gpt-5.2",
			wantEffortPath:     "reasoning.effort",
			wantEffortExists:   true,
		},
		{
			name:               "auto_no_effort",
			body:               `{"model":"gpt-5.2(auto)","messages":[]}`,
			endpointKind:       endpointKindChatCompletions,
			baseModel:          "gpt-5.2",
			effort:             "",
			precedenceBodyWins: true,
			wantModel:          "gpt-5.2",
			wantEffortPath:     "reasoning_effort",
			wantEffortExists:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := patchReasoningEffort([]byte(tc.body), tc.endpointKind, tc.baseModel, tc.effort, tc.precedenceBodyWins)
			if err != nil {
				t.Fatalf("patch failed: %v", err)
			}
			if got := gjson.GetBytes(out, "model").String(); got != tc.wantModel {
				t.Fatalf("expected model %q, got %q", tc.wantModel, got)
			}
			ev := gjson.GetBytes(out, tc.wantEffortPath)
			if tc.wantEffortExists && !ev.Exists() {
				t.Fatalf("expected effort at %q", tc.wantEffortPath)
			}
			if !tc.wantEffortExists && ev.Exists() {
				t.Fatalf("did not expect effort at %q, got %s", tc.wantEffortPath, ev.Raw)
			}
			if tc.name == "chat_body_effort_wins" && ev.String() != "low" {
				t.Fatalf("expected body effort low, got %q", ev.String())
			}
		})
	}
}
