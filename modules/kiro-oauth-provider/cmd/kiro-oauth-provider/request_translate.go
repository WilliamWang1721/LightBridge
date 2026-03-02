package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type chatCompletionRequest struct {
	Model    string           `json:"model"`
	Messages []chatMessage    `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []map[string]any `json:"tools,omitempty"`
	System   any              `json:"system,omitempty"`
	Thinking map[string]any   `json:"thinking,omitempty"`
}

type chatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []map[string]any `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type kiroToolCall struct {
	ID   string
	Name string
	Args string
}

var modelMapping = map[string]string{
	"claude-haiku-4-5":           "claude-haiku-4.5",
	"claude-opus-4-6":            "claude-opus-4.6",
	"claude-sonnet-4-6":          "claude-sonnet-4.6",
	"claude-opus-4-5":            "claude-opus-4.5",
	"claude-opus-4-5-20251101":   "claude-opus-4.5",
	"claude-sonnet-4-5":          "CLAUDE_SONNET_4_5_20250929_V1_0",
	"claude-sonnet-4-5-20250929": "CLAUDE_SONNET_4_5_20250929_V1_0",
}

func mapModel(model string) string {
	m := strings.TrimSpace(model)
	if m == "" {
		return "CLAUDE_SONNET_4_5_20250929_V1_0"
	}
	if out, ok := modelMapping[m]; ok && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	return m
}

func contentToText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var b strings.Builder
		for _, part := range t {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(fmt.Sprint(m["type"])) == "text" {
				b.WriteString(fmt.Sprint(m["text"]))
			}
		}
		return b.String()
	case []map[string]any:
		var b strings.Builder
		for _, m := range t {
			if strings.TrimSpace(fmt.Sprint(m["type"])) == "text" {
				b.WriteString(fmt.Sprint(m["text"]))
			}
		}
		return b.String()
	default:
		return ""
	}
}

func extractToolResults(content any) []map[string]any {
	arr, ok := content.([]any)
	if !ok {
		if arr2, ok2 := content.([]map[string]any); ok2 {
			arr = make([]any, 0, len(arr2))
			for _, it := range arr2 {
				arr = append(arr, it)
			}
		} else {
			return nil
		}
	}
	out := make([]map[string]any, 0)
	seen := map[string]struct{}{}
	for _, part := range arr {
		m, ok := part.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(m["type"])) != "tool_result" {
			continue
		}
		toolUseID := strings.TrimSpace(fmt.Sprint(m["tool_use_id"]))
		if toolUseID == "" {
			continue
		}
		if _, dup := seen[toolUseID]; dup {
			continue
		}
		seen[toolUseID] = struct{}{}
		result := map[string]any{
			"toolUseId": toolUseID,
			"status":    "success",
			"content": []map[string]string{{
				"text": contentToText(m["content"]),
			}},
		}
		out = append(out, result)
	}
	return out
}

func extractImages(content any, keep bool) []map[string]any {
	if !keep {
		return nil
	}
	arr, ok := content.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0)
	for _, part := range arr {
		m, ok := part.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(m["type"])) != "image" {
			continue
		}
		source, _ := m["source"].(map[string]any)
		mediaType := strings.TrimSpace(fmt.Sprint(source["media_type"]))
		data := strings.TrimSpace(fmt.Sprint(source["data"]))
		if data == "" {
			continue
		}
		format := "png"
		if strings.Contains(mediaType, "/") {
			parts := strings.Split(mediaType, "/")
			format = strings.TrimSpace(parts[len(parts)-1])
		}
		out = append(out, map[string]any{
			"format": format,
			"source": map[string]any{"bytes": data},
		})
	}
	return out
}

func extractAssistantToolUses(msg chatMessage) []map[string]any {
	out := make([]map[string]any, 0)
	for _, tc := range msg.ToolCalls {
		if tc == nil {
			continue
		}
		typ := strings.TrimSpace(fmt.Sprint(tc["type"]))
		if typ != "function" {
			continue
		}
		fn, _ := tc["function"].(map[string]any)
		name := strings.TrimSpace(fmt.Sprint(fn["name"]))
		argsRaw := strings.TrimSpace(fmt.Sprint(fn["arguments"]))
		if name == "" {
			continue
		}
		var args map[string]any
		if strings.TrimSpace(argsRaw) != "" {
			_ = json.Unmarshal([]byte(argsRaw), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		out = append(out, map[string]any{
			"name":      name,
			"toolUseId": strings.TrimSpace(fmt.Sprint(tc["id"])),
			"input":     args,
		})
	}
	return out
}

func convertTools(tools []map[string]any) []map[string]any {
	if len(tools) == 0 {
		return []map[string]any{{
			"toolSpecification": map[string]any{
				"name":        "no_tool_available",
				"description": "placeholder tool",
				"inputSchema": map[string]any{"json": map[string]any{"type": "object", "properties": map[string]any{}}},
			},
		}}
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(t["type"])) != "function" {
			continue
		}
		fn, _ := t["function"].(map[string]any)
		name := strings.TrimSpace(fmt.Sprint(fn["name"]))
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(fmt.Sprint(fn["description"]))
		if len(desc) > 9216 {
			desc = desc[:9216] + "..."
		}
		if desc == "" {
			desc = "function tool"
		}
		params, _ := fn["parameters"].(map[string]any)
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"toolSpecification": map[string]any{
				"name":        name,
				"description": desc,
				"inputSchema": map[string]any{"json": params},
			},
		})
	}
	if len(out) == 0 {
		return []map[string]any{{
			"toolSpecification": map[string]any{
				"name":        "no_tool_available",
				"description": "placeholder tool",
				"inputSchema": map[string]any{"json": map[string]any{"type": "object", "properties": map[string]any{}}},
			},
		}}
	}
	return out
}

func thinkingPrefix(thinking map[string]any) string {
	if len(thinking) == 0 {
		return ""
	}
	typ := strings.ToLower(strings.TrimSpace(fmt.Sprint(thinking["type"])))
	switch typ {
	case "enabled":
		budget := 20000
		if n, ok := thinking["budget_tokens"].(float64); ok {
			budget = int(n)
		}
		if budget < 1024 {
			budget = 1024
		}
		if budget > 24576 {
			budget = 24576
		}
		return fmt.Sprintf("<thinking_mode>enabled</thinking_mode><max_thinking_length>%d</max_thinking_length>", budget)
	case "adaptive":
		effort := strings.ToLower(strings.TrimSpace(fmt.Sprint(thinking["effort"])))
		if effort != "low" && effort != "medium" && effort != "high" {
			effort = "high"
		}
		return fmt.Sprintf("<thinking_mode>adaptive</thinking_mode><thinking_effort>%s</thinking_effort>", effort)
	default:
		return ""
	}
}

func buildKiroRequest(req chatCompletionRequest, acc *account) (map[string]any, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("messages is required")
	}
	modelID := mapModel(req.Model)

	messages := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "" {
			continue
		}
		messages = append(messages, m)
	}
	if len(messages) == 0 {
		return nil, errors.New("messages is required")
	}

	systemPrompt := contentToText(req.System)
	prefix := thinkingPrefix(req.Thinking)
	if prefix != "" {
		if strings.TrimSpace(systemPrompt) == "" {
			systemPrompt = prefix
		} else if !strings.Contains(systemPrompt, "<thinking_mode>") {
			systemPrompt = prefix + "\n" + systemPrompt
		}
	}

	last := messages[len(messages)-1]
	history := make([]map[string]any, 0, len(messages))

	for i := 0; i < len(messages)-1; i++ {
		msg := messages[i]
		distance := (len(messages) - 1) - i
		keepImages := distance <= 5
		if msg.Role == "user" {
			u := map[string]any{
				"content": contentToText(msg.Content),
				"modelId": modelID,
				"origin":  "AI_EDITOR",
			}
			ctx := map[string]any{}
			if tr := extractToolResults(msg.Content); len(tr) > 0 {
				ctx["toolResults"] = tr
			}
			if imgs := extractImages(msg.Content, keepImages); len(imgs) > 0 {
				u["images"] = imgs
			}
			if len(ctx) > 0 {
				u["userInputMessageContext"] = ctx
			}
			history = append(history, map[string]any{"userInputMessage": u})
			continue
		}

		if msg.Role == "assistant" {
			a := map[string]any{"content": contentToText(msg.Content)}
			if tools := extractAssistantToolUses(msg); len(tools) > 0 {
				a["toolUses"] = tools
			}
			history = append(history, map[string]any{"assistantResponseMessage": a})
		}
	}

	currentContent := contentToText(last.Content)
	if strings.TrimSpace(systemPrompt) != "" {
		if len(history) > 0 {
			first := history[0]
			if uim, ok := first["userInputMessage"].(map[string]any); ok {
				orig := strings.TrimSpace(fmt.Sprint(uim["content"]))
				if orig == "" {
					uim["content"] = systemPrompt
				} else {
					uim["content"] = systemPrompt + "\n\n" + orig
				}
				history[0] = map[string]any{"userInputMessage": uim}
			}
		} else {
			history = append(history, map[string]any{
				"userInputMessage": map[string]any{
					"content": systemPrompt,
					"modelId": modelID,
					"origin":  "AI_EDITOR",
				},
			})
		}
	}

	if strings.ToLower(strings.TrimSpace(last.Role)) == "assistant" {
		a := map[string]any{"content": currentContent}
		if tools := extractAssistantToolUses(last); len(tools) > 0 {
			a["toolUses"] = tools
		}
		history = append(history, map[string]any{"assistantResponseMessage": a})
		currentContent = "Continue"
	} else if strings.TrimSpace(currentContent) == "" {
		if tr := extractToolResults(last.Content); len(tr) > 0 {
			currentContent = "Tool results provided."
		} else {
			currentContent = "Continue"
		}
	}

	ctx := map[string]any{
		"tools": convertTools(req.Tools),
	}
	if tr := extractToolResults(last.Content); len(tr) > 0 {
		ctx["toolResults"] = tr
	}

	current := map[string]any{
		"content": currentContent,
		"modelId": modelID,
		"origin":  "AI_EDITOR",
	}
	if imgs := extractImages(last.Content, true); len(imgs) > 0 {
		current["images"] = imgs
	}
	if len(ctx) > 0 {
		current["userInputMessageContext"] = ctx
	}

	request := map[string]any{
		"conversationState": map[string]any{
			"chatTriggerType": "MANUAL",
			"conversationId":  newUUID(),
			"currentMessage": map[string]any{
				"userInputMessage": current,
			},
		},
	}
	if len(history) > 0 {
		request["conversationState"].(map[string]any)["history"] = history
	}
	if acc != nil && strings.EqualFold(strings.TrimSpace(acc.AuthMethod), authMethodSocial) && strings.TrimSpace(acc.ProfileARN) != "" {
		request["profileArn"] = strings.TrimSpace(acc.ProfileARN)
	}
	return request, nil
}
