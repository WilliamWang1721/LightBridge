package gateway

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	errInvalidModelTag = errors.New("invalid_model_tag")
	errMissingModelTag = errors.New("missing_model")
)

const supportedModelTagValues = "auto, none, minimal, low, medium, high, xhigh"

func normalizeModelTagAliases(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.ToLower(strings.TrimSpace(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseModelTag(model string, aliases map[string]string) (base string, effort string, hasTag bool, err error) {
	modelRaw := strings.TrimSpace(model)
	if modelRaw == "" {
		return "", "", false, nil
	}

	runes := []rune(modelRaw)
	last := runes[len(runes)-1]
	if last != ')' && last != '）' {
		return modelRaw, "", false, nil
	}
	hasTag = true

	left := -1
	for i := len(runes) - 2; i >= 0; i-- {
		if runes[i] == '(' || runes[i] == '（' {
			left = i
			break
		}
	}
	if left < 0 {
		return "", "", true, fmt.Errorf("%w: malformed model tag on %q", errInvalidModelTag, modelRaw)
	}

	base = strings.TrimSpace(string(runes[:left]))
	tag := strings.TrimSpace(string(runes[left+1 : len(runes)-1]))
	if base == "" {
		return "", "", true, fmt.Errorf("%w: model is empty after stripping tag", errMissingModelTag)
	}
	if tag == "" {
		return "", "", true, fmt.Errorf("%w: empty model tag in %q", errInvalidModelTag, modelRaw)
	}

	tagLower := strings.ToLower(tag)
	if mapped, ok := aliases[tagLower]; ok {
		tagLower = strings.ToLower(strings.TrimSpace(mapped))
	}

	if tagLower == "auto" {
		return base, "", true, nil
	}
	if !isOfficialReasoningEffort(tagLower) {
		return "", "", true, fmt.Errorf(
			"%w: unknown model tag %q (supported: %s; aliases via LIGHTBRIDGE_MODEL_TAG_ALIASES)",
			errInvalidModelTag,
			tag,
			supportedModelTagValues,
		)
	}
	return base, tagLower, true, nil
}

func patchReasoningEffort(body []byte, endpointKind string, baseModel string, effort string, precedenceBodyWins bool) ([]byte, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return body, nil
	}
	if !gjson.ValidBytes(trimmed) {
		return nil, errors.New("invalid json body")
	}

	out := body
	var err error
	if gjson.GetBytes(out, "model").Exists() {
		out, err = sjson.SetBytes(out, "model", baseModel)
		if err != nil {
			return nil, err
		}
	}

	effort = strings.TrimSpace(effort)
	if effort == "" {
		return out, nil
	}

	switch endpointKind {
	case endpointKindChatCompletions:
		if precedenceBodyWins && gjson.GetBytes(out, "reasoning_effort").Exists() {
			return out, nil
		}
		out, err = sjson.SetBytes(out, "reasoning_effort", effort)
		if err != nil {
			return nil, err
		}
	case endpointKindResponses:
		if precedenceBodyWins && gjson.GetBytes(out, "reasoning.effort").Exists() {
			return out, nil
		}
		out, err = sjson.SetBytes(out, "reasoning.effort", effort)
		if err != nil {
			return nil, err
		}
	default:
	}

	return out, nil
}

func isOfficialReasoningEffort(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}
