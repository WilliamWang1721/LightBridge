package main

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

type parsedEvent struct {
	Type                string
	Content             string
	ToolName            string
	ToolUseID           string
	ToolInput           string
	ToolStop            bool
	ContextUsagePercent float64
}

type streamToolState struct {
	ID   string
	Name string
	Args strings.Builder
}

func findJSONEnd(text string, start int) int {
	if start < 0 || start >= len(text) || text[start] != '{' {
		return -1
	}
	brace := 0
	inString := false
	escapeNext := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if escapeNext {
			escapeNext = false
			continue
		}
		if ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			brace++
		case '}':
			brace--
			if brace == 0 {
				return i
			}
		}
	}
	return -1
}

func parseAwsEventStreamBuffer(buffer string) (events []parsedEvent, remaining string) {
	remaining = buffer
	searchStart := 0
	for {
		contentStart := strings.Index(remaining[searchStart:], `{"content":`)
		if contentStart >= 0 {
			contentStart += searchStart
		}
		nameStart := strings.Index(remaining[searchStart:], `{"name":`)
		if nameStart >= 0 {
			nameStart += searchStart
		}
		followupStart := strings.Index(remaining[searchStart:], `{"followupPrompt":`)
		if followupStart >= 0 {
			followupStart += searchStart
		}
		inputStart := strings.Index(remaining[searchStart:], `{"input":`)
		if inputStart >= 0 {
			inputStart += searchStart
		}
		stopStart := strings.Index(remaining[searchStart:], `{"stop":`)
		if stopStart >= 0 {
			stopStart += searchStart
		}
		ctxStart := strings.Index(remaining[searchStart:], `{"contextUsagePercentage":`)
		if ctxStart >= 0 {
			ctxStart += searchStart
		}

		candidates := []int{}
		for _, pos := range []int{contentStart, nameStart, followupStart, inputStart, stopStart, ctxStart} {
			if pos >= 0 {
				candidates = append(candidates, pos)
			}
		}
		if len(candidates) == 0 {
			break
		}
		jsonStart := candidates[0]
		for _, c := range candidates[1:] {
			if c < jsonStart {
				jsonStart = c
			}
		}
		jsonEnd := findJSONEnd(remaining, jsonStart)
		if jsonEnd < 0 {
			remaining = remaining[jsonStart:]
			return events, remaining
		}
		chunk := remaining[jsonStart : jsonEnd+1]
		var obj map[string]any
		if err := json.Unmarshal([]byte(chunk), &obj); err == nil {
			if v, ok := obj["content"]; ok && obj["followupPrompt"] == nil {
				events = append(events, parsedEvent{Type: "content", Content: strings.TrimSpace(toString(v, false))})
			} else if obj["name"] != nil && obj["toolUseId"] != nil {
				events = append(events, parsedEvent{
					Type:      "toolUse",
					ToolName:  strings.TrimSpace(toString(obj["name"], false)),
					ToolUseID: strings.TrimSpace(toString(obj["toolUseId"], false)),
					ToolInput: toString(obj["input"], false),
					ToolStop:  toBool(obj["stop"]),
				})
			} else if obj["input"] != nil && obj["name"] == nil {
				events = append(events, parsedEvent{Type: "toolUseInput", ToolInput: toString(obj["input"], false)})
			} else if obj["stop"] != nil && obj["contextUsagePercentage"] == nil {
				events = append(events, parsedEvent{Type: "toolUseStop", ToolStop: toBool(obj["stop"])})
			} else if obj["contextUsagePercentage"] != nil {
				events = append(events, parsedEvent{Type: "contextUsage", ContextUsagePercent: toFloat(obj["contextUsagePercentage"])})
			}
		}
		searchStart = jsonEnd + 1
		if searchStart >= len(remaining) {
			remaining = ""
			return events, remaining
		}
	}
	if searchStart > 0 && searchStart < len(remaining) {
		remaining = remaining[searchStart:]
	}
	return events, remaining
}

func parseBracketToolCalls(raw string) []kiroToolCall {
	re := regexp.MustCompile(`\[Called\s+([A-Za-z0-9_\-\.]+)\s+with\s+args:\s*(\{[^\]]*\})\]`)
	matches := re.FindAllStringSubmatch(raw, -1)
	out := make([]kiroToolCall, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		name := strings.TrimSpace(m[1])
		args := strings.TrimSpace(m[2])
		if name == "" {
			continue
		}
		out = append(out, kiroToolCall{ID: newUUID(), Name: name, Args: args})
	}
	return out
}

func toString(v any, trim bool) string {
	var out string
	switch t := v.(type) {
	case string:
		out = t
	default:
		b, _ := json.Marshal(t)
		out = string(b)
	}
	if trim {
		return strings.TrimSpace(out)
	}
	return out
}

func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func toFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}

func parseKiroMixedResponse(raw string) (text string, toolCalls []kiroToolCall, contextUsage float64) {
	buffer := raw
	state := &streamToolState{}
	for len(buffer) > 0 {
		events, remain := parseAwsEventStreamBuffer(buffer)
		if len(events) == 0 {
			break
		}
		for _, ev := range events {
			switch ev.Type {
			case "content":
				text += ev.Content
			case "toolUse":
				if state.ID != "" {
					toolCalls = append(toolCalls, kiroToolCall{ID: state.ID, Name: state.Name, Args: strings.TrimSpace(state.Args.String())})
				}
				state = &streamToolState{ID: nonEmpty(ev.ToolUseID, newUUID()), Name: ev.ToolName}
				state.Args.WriteString(ev.ToolInput)
				if ev.ToolStop {
					toolCalls = append(toolCalls, kiroToolCall{ID: state.ID, Name: state.Name, Args: strings.TrimSpace(state.Args.String())})
					state = &streamToolState{}
				}
			case "toolUseInput":
				if state.ID != "" {
					state.Args.WriteString(ev.ToolInput)
				}
			case "toolUseStop":
				if state.ID != "" {
					toolCalls = append(toolCalls, kiroToolCall{ID: state.ID, Name: state.Name, Args: strings.TrimSpace(state.Args.String())})
					state = &streamToolState{}
				}
			case "contextUsage":
				contextUsage = ev.ContextUsagePercent
			}
		}
		if remain == buffer {
			break
		}
		buffer = remain
	}
	if state != nil && state.ID != "" {
		toolCalls = append(toolCalls, kiroToolCall{ID: state.ID, Name: state.Name, Args: strings.TrimSpace(state.Args.String())})
	}
	if extra := parseBracketToolCalls(raw); len(extra) > 0 {
		toolCalls = append(toolCalls, extra...)
	}
	return strings.TrimSpace(text), dedupeToolCalls(toolCalls), contextUsage
}

func dedupeToolCalls(in []kiroToolCall) []kiroToolCall {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]kiroToolCall, 0, len(in))
	for _, tc := range in {
		key := strings.TrimSpace(tc.Name) + "|" + strings.TrimSpace(tc.Args)
		if strings.TrimSpace(tc.ID) != "" {
			key = strings.TrimSpace(tc.ID)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(tc.ID) == "" {
			tc.ID = newUUID()
		}
		out = append(out, tc)
	}
	return out
}

func splitThinking(content string) (reasoning string, normal string) {
	raw := content
	start := strings.Index(raw, "<thinking>")
	if start < 0 {
		return "", content
	}
	end := strings.Index(raw[start+len("<thinking>"):], "</thinking>")
	if end < 0 {
		return "", content
	}
	end += start + len("<thinking>")
	reasoning = strings.TrimSpace(raw[start+len("<thinking>") : end])
	before := raw[:start]
	after := raw[end+len("</thinking>"):]
	normal = strings.TrimSpace(before + "\n" + after)
	return reasoning, normal
}
