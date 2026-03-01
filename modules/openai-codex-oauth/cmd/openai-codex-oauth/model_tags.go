package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

const supportedModelTagValues = "auto, none, minimal, low, medium, high, xhigh"

func loadModelTagAliasesFromEnv() map[string]string {
	raw := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_MODEL_TAG_ALIASES"))
	if raw == "" {
		return nil
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		log.Printf("model tags: LIGHTBRIDGE_MODEL_TAG_ALIASES parse failed: %v", err)
		return nil
	}
	out := make(map[string]string, len(parsed))
	for k, v := range parsed {
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
		return "", "", true, fmt.Errorf("invalid model tag %q", modelRaw)
	}

	base = strings.TrimSpace(string(runes[:left]))
	tag := strings.TrimSpace(string(runes[left+1 : len(runes)-1]))
	if base == "" {
		return "", "", true, fmt.Errorf("model is empty after stripping tag")
	}
	if tag == "" {
		return "", "", true, fmt.Errorf("model tag is empty")
	}

	tagLower := strings.ToLower(tag)
	if mapped, ok := aliases[tagLower]; ok {
		tagLower = strings.ToLower(strings.TrimSpace(mapped))
	}

	if tagLower == "auto" {
		return base, "", true, nil
	}
	if !isOfficialReasoningEffort(tagLower) {
		return "", "", true, fmt.Errorf("unknown model tag %q (supported: %s)", tag, supportedModelTagValues)
	}
	return base, tagLower, true, nil
}

func isOfficialReasoningEffort(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}
