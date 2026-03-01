package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type chatCompletionRequest struct {
	Model           string           `json:"model"`
	Messages        []chatMessage    `json:"messages"`
	Stream          bool             `json:"stream"`
	Tools           []map[string]any `json:"tools,omitempty"`
	ToolChoice      any              `json:"tool_choice,omitempty"`
	ReasoningEffort any              `json:"reasoning_effort,omitempty"`
	Metadata        map[string]any   `json:"metadata,omitempty"`
	PromptCacheKey  string           `json:"prompt_cache_key,omitempty"`
}

type chatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []map[string]any `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

func buildCodexRequest(openAIReqBytes []byte, clientWantsStream bool, modelTagAliases map[string]string) (codexReqBytes []byte, revToolName map[string]string, promptCacheKey string, err error) {
	var in chatCompletionRequest
	if err := json.Unmarshal(openAIReqBytes, &in); err != nil {
		return nil, nil, "", errors.New("invalid JSON")
	}
	in.Model = strings.TrimSpace(in.Model)
	if in.Model == "" {
		return nil, nil, "", errors.New("model is required")
	}

	baseModel, tagEffort, hasTag, tagErr := parseModelTag(in.Model, modelTagAliases)
	if tagErr != nil {
		return nil, nil, "", tagErr
	}
	if hasTag {
		in.Model = baseModel
	}

	effort := ""
	if v, ok := in.ReasoningEffort.(string); ok && strings.TrimSpace(v) != "" {
		effort = strings.TrimSpace(v)
	} else if strings.TrimSpace(tagEffort) != "" {
		effort = strings.TrimSpace(tagEffort)
	}

	toolNameMap, rev := buildToolNameMaps(in.Tools)

	input := make([]any, 0, len(in.Messages))
	for _, msg := range in.Messages {
		role := strings.TrimSpace(msg.Role)
		switch role {
		case "tool":
			callID := strings.TrimSpace(msg.ToolCallID)
			if callID == "" {
				continue
			}
			output := contentAsString(msg.Content)
			input = append(input, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  output,
			})
			continue
		default:
		}

		cRole := role
		if cRole == "system" {
			cRole = "developer"
		}

		parts := make([]any, 0, 1)
		if text := contentAsString(msg.Content); strings.TrimSpace(text) != "" {
			partType := "input_text"
			if role == "assistant" {
				partType = "output_text"
			}
			parts = append(parts, map[string]any{
				"type": partType,
				"text": text,
			})
		} else if arr, ok := msg.Content.([]any); ok {
			for _, it := range arr {
				m, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if strings.TrimSpace(asString(m["type"])) != "text" {
					continue
				}
				text := strings.TrimSpace(asString(m["text"]))
				if text == "" {
					continue
				}
				partType := "input_text"
				if role == "assistant" {
					partType = "output_text"
				}
				parts = append(parts, map[string]any{
					"type": partType,
					"text": text,
				})
			}
		}

		if len(parts) > 0 {
			input = append(input, map[string]any{
				"type":    "message",
				"role":    cRole,
				"content": parts,
			})
		}

		// Convert assistant tool calls to top-level function_call items.
		if role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				if strings.TrimSpace(asString(tc["type"])) != "function" {
					continue
				}
				callID := strings.TrimSpace(asString(tc["id"]))
				fn, _ := tc["function"].(map[string]any)
				name := strings.TrimSpace(asString(fn["name"]))
				if name == "" {
					continue
				}
				if short, ok := toolNameMap[name]; ok {
					name = short
				} else {
					name = shortenNameIfNeeded(name)
				}
				args := asString(fn["arguments"])
				input = append(input, map[string]any{
					"type":      "function_call",
					"call_id":   callID,
					"name":      name,
					"arguments": args,
				})
			}
		}
	}

	out := map[string]any{
		"model":               in.Model,
		"stream":              true, // upstream always SSE
		"instructions":        "",
		"input":               input,
		"parallel_tool_calls": true,
		"reasoning":           map[string]any{"summary": "auto"},
		"include":             []string{"reasoning.encrypted_content"},
		"store":               false,
	}
	if effort != "" {
		out["reasoning"].(map[string]any)["effort"] = effort
	}

	if tools := convertToolsToCodex(in.Tools, toolNameMap); len(tools) > 0 {
		out["tools"] = tools
	}
	if tc := convertToolChoiceToCodex(in.ToolChoice, toolNameMap); tc != nil {
		out["tool_choice"] = tc
	}

	promptCacheKey = strings.TrimSpace(in.PromptCacheKey)
	if promptCacheKey != "" {
		out["prompt_cache_key"] = promptCacheKey
	}

	b, err := json.Marshal(out)
	if err != nil {
		return nil, nil, "", err
	}
	return b, rev, promptCacheKey, nil
}

func injectPromptCacheKey(codexReqBytes []byte, key string) []byte {
	key = strings.TrimSpace(key)
	if key == "" {
		return codexReqBytes
	}
	var m map[string]any
	if err := json.Unmarshal(codexReqBytes, &m); err != nil {
		return codexReqBytes
	}
	if _, exists := m["prompt_cache_key"]; exists {
		return codexReqBytes
	}
	m["prompt_cache_key"] = key
	b, err := json.Marshal(m)
	if err != nil {
		return codexReqBytes
	}
	return b
}

func contentAsString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		// from encoding/json default
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

func buildToolNameMaps(tools []map[string]any) (origToShort map[string]string, shortToOrig map[string]string) {
	names := make([]string, 0)
	for _, t := range tools {
		if strings.TrimSpace(asString(t["type"])) != "function" {
			continue
		}
		fn, ok := t["function"].(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(asString(fn["name"]))
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	origToShort = buildShortNameMap(names)
	shortToOrig = map[string]string{}
	for orig, short := range origToShort {
		shortToOrig[short] = orig
	}
	return origToShort, shortToOrig
}

func convertToolsToCodex(tools []map[string]any, origToShort map[string]string) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		tt := strings.TrimSpace(asString(t["type"]))
		if tt == "" {
			continue
		}
		if tt != "function" {
			out = append(out, t)
			continue
		}
		fn, ok := t["function"].(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(asString(fn["name"]))
		if name == "" {
			continue
		}
		if short, ok := origToShort[name]; ok {
			name = short
		} else {
			name = shortenNameIfNeeded(name)
		}

		item := map[string]any{
			"type": "function",
			"name": name,
		}
		if desc := strings.TrimSpace(asString(fn["description"])); desc != "" {
			item["description"] = desc
		}
		if params, ok := fn["parameters"]; ok && params != nil {
			item["parameters"] = params
		}
		if strict, ok := fn["strict"].(bool); ok {
			item["strict"] = strict
		}
		out = append(out, item)
	}
	return out
}

func convertToolChoiceToCodex(tc any, origToShort map[string]string) any {
	if tc == nil {
		return nil
	}
	switch v := tc.(type) {
	case string:
		return v
	case map[string]any:
		tt := strings.TrimSpace(asString(v["type"]))
		if tt == "function" {
			fn, _ := v["function"].(map[string]any)
			name := strings.TrimSpace(asString(fn["name"]))
			if name != "" {
				if short, ok := origToShort[name]; ok {
					name = short
				} else {
					name = shortenNameIfNeeded(name)
				}
			}
			out := map[string]any{
				"type": "function",
			}
			if name != "" {
				out["name"] = name
			}
			return out
		}
		// Built-in tool choices already match Responses API.
		return v
	default:
		return nil
	}
}

func shortenNameIfNeeded(name string) string {
	const limit = 64
	name = strings.TrimSpace(name)
	if len(name) <= limit {
		return name
	}
	if strings.HasPrefix(name, "mcp__") {
		idx := strings.LastIndex(name, "__")
		if idx > 0 {
			cand := "mcp__" + name[idx+2:]
			if len(cand) > limit {
				return cand[:limit]
			}
			return cand
		}
	}
	return name[:limit]
}

func buildShortNameMap(names []string) map[string]string {
	const limit = 64
	used := map[string]struct{}{}
	out := map[string]string{}

	baseCandidate := func(n string) string {
		if len(n) <= limit {
			return n
		}
		if strings.HasPrefix(n, "mcp__") {
			idx := strings.LastIndex(n, "__")
			if idx > 0 {
				cand := "mcp__" + n[idx+2:]
				if len(cand) > limit {
					cand = cand[:limit]
				}
				return cand
			}
		}
		return n[:limit]
	}

	makeUnique := func(cand string) string {
		if _, ok := used[cand]; !ok {
			return cand
		}
		base := cand
		for i := 1; ; i++ {
			suffix := "_" + strconv.Itoa(i)
			allowed := limit - len(suffix)
			if allowed < 0 {
				allowed = 0
			}
			tmp := base
			if len(tmp) > allowed {
				tmp = tmp[:allowed]
			}
			tmp = tmp + suffix
			if _, ok := used[tmp]; !ok {
				return tmp
			}
		}
	}

	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		cand := baseCandidate(n)
		uniq := makeUnique(cand)
		used[uniq] = struct{}{}
		out[n] = uniq
	}
	return out
}
