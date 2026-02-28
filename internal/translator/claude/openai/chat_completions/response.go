// Package chat_completions provides response translation from Claude Messages API
// streaming/non-streaming format to OpenAI Chat Completions format.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package chat_completions

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var dataTag = []byte("data:")

// StreamState tracks state across streaming chunks.
type StreamState struct {
	CreatedAt            int64
	ResponseID           string
	FinishReason         string
	ToolCallsAccumulator map[int]*ToolCallAccum
}

// ToolCallAccum accumulates tool call data across streaming chunks.
type ToolCallAccum struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// ConvertClaudeResponseToOpenAI converts a single Claude SSE chunk to OpenAI chat.completion.chunk format.
func ConvertClaudeResponseToOpenAI(_ context.Context, modelName string, _, _ []byte, rawJSON []byte, param *any) []string {
	if *param == nil {
		*param = &StreamState{}
	}
	st := (*param).(*StreamState)

	if !bytes.HasPrefix(rawJSON, dataTag) {
		return nil
	}
	rawJSON = bytes.TrimSpace(rawJSON[5:])
	root := gjson.ParseBytes(rawJSON)
	ev := root.Get("type").String()

	tpl := `{"id":"","object":"chat.completion.chunk","created":0,"model":"","choices":[{"index":0,"delta":{},"finish_reason":null}]}`
	if modelName != "" {
		tpl, _ = sjson.Set(tpl, "model", modelName)
	}
	if st.ResponseID != "" {
		tpl, _ = sjson.Set(tpl, "id", st.ResponseID)
	}
	if st.CreatedAt > 0 {
		tpl, _ = sjson.Set(tpl, "created", st.CreatedAt)
	}

	switch ev {
	case "message_start":
		if msg := root.Get("message"); msg.Exists() {
			st.ResponseID = msg.Get("id").String()
			st.CreatedAt = time.Now().Unix()
			st.ToolCallsAccumulator = make(map[int]*ToolCallAccum)
			tpl, _ = sjson.Set(tpl, "id", st.ResponseID)
			tpl, _ = sjson.Set(tpl, "created", st.CreatedAt)
			tpl, _ = sjson.Set(tpl, "choices.0.delta.role", "assistant")
		}
		return []string{tpl}

	case "content_block_start":
		cb := root.Get("content_block")
		if !cb.Exists() {
			return nil
		}
		if cb.Get("type").String() == "tool_use" {
			idx := int(root.Get("index").Int())
			if st.ToolCallsAccumulator == nil {
				st.ToolCallsAccumulator = make(map[int]*ToolCallAccum)
			}
			st.ToolCallsAccumulator[idx] = &ToolCallAccum{
				ID:   cb.Get("id").String(),
				Name: cb.Get("name").String(),
			}
		}
		return nil

	case "content_block_delta":
		d := root.Get("delta")
		if !d.Exists() {
			return nil
		}
		dt := d.Get("type").String()
		switch dt {
		case "text_delta":
			if t := d.Get("text"); t.Exists() {
				tpl, _ = sjson.Set(tpl, "choices.0.delta.content", t.String())
				return []string{tpl}
			}
		case "thinking_delta":
			if t := d.Get("thinking"); t.Exists() {
				tpl, _ = sjson.Set(tpl, "choices.0.delta.reasoning_content", t.String())
				return []string{tpl}
			}
		case "input_json_delta":
			if pj := d.Get("partial_json"); pj.Exists() {
				idx := int(root.Get("index").Int())
				if st.ToolCallsAccumulator != nil {
					if acc, ok := st.ToolCallsAccumulator[idx]; ok {
						acc.Arguments.WriteString(pj.String())
					}
				}
			}
			return nil
		}
		return nil

	case "content_block_stop":
		idx := int(root.Get("index").Int())
		if st.ToolCallsAccumulator != nil {
			if acc, ok := st.ToolCallsAccumulator[idx]; ok {
				args := acc.Arguments.String()
				if args == "" {
					args = "{}"
				}
				tpl, _ = sjson.Set(tpl, "choices.0.delta.tool_calls.0.index", idx)
				tpl, _ = sjson.Set(tpl, "choices.0.delta.tool_calls.0.id", acc.ID)
				tpl, _ = sjson.Set(tpl, "choices.0.delta.tool_calls.0.type", "function")
				tpl, _ = sjson.Set(tpl, "choices.0.delta.tool_calls.0.function.name", acc.Name)
				tpl, _ = sjson.Set(tpl, "choices.0.delta.tool_calls.0.function.arguments", args)
				delete(st.ToolCallsAccumulator, idx)
				return []string{tpl}
			}
		}
		return nil

	case "message_delta":
		if d := root.Get("delta"); d.Exists() {
			if sr := d.Get("stop_reason"); sr.Exists() {
				st.FinishReason = mapStopReason(sr.String())
				tpl, _ = sjson.Set(tpl, "choices.0.finish_reason", st.FinishReason)
			}
		}
		if usage := root.Get("usage"); usage.Exists() {
			input := usage.Get("input_tokens").Int()
			output := usage.Get("output_tokens").Int()
			cached := usage.Get("cache_read_input_tokens").Int()
			tpl, _ = sjson.Set(tpl, "usage.prompt_tokens", input)
			tpl, _ = sjson.Set(tpl, "usage.completion_tokens", output)
			tpl, _ = sjson.Set(tpl, "usage.total_tokens", input+output)
			if cached > 0 {
				tpl, _ = sjson.Set(tpl, "usage.prompt_tokens_details.cached_tokens", cached)
			}
		}
		return []string{tpl}

	case "message_stop", "ping":
		return nil

	case "error":
		if e := root.Get("error"); e.Exists() {
			ej := `{"error":{"message":"","type":""}}`
			ej, _ = sjson.Set(ej, "error.message", e.Get("message").String())
			ej, _ = sjson.Set(ej, "error.type", e.Get("type").String())
			return []string{ej}
		}
		return nil
	}
	return nil
}

// ConvertClaudeResponseToOpenAINonStream converts Claude SSE lines to a single OpenAI chat.completion response.
func ConvertClaudeResponseToOpenAINonStream(_ context.Context, _ string, _, _ []byte, rawJSON []byte, _ *any) string {
	var chunks [][]byte
	for _, line := range bytes.Split(rawJSON, []byte("\n")) {
		if bytes.HasPrefix(line, dataTag) {
			chunks = append(chunks, bytes.TrimSpace(line[5:]))
		}
	}

	out := `{"id":"","object":"chat.completion","created":0,"model":"","choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`

	var (
		msgID        string
		model        string
		stopReason   string
		content      strings.Builder
		reasoning    strings.Builder
		toolCallAccs = make(map[int]*ToolCallAccum)
	)

	for _, ch := range chunks {
		root := gjson.ParseBytes(ch)
		ev := root.Get("type").String()
		switch ev {
		case "message_start":
			if msg := root.Get("message"); msg.Exists() {
				msgID = msg.Get("id").String()
				model = msg.Get("model").String()
			}
		case "content_block_start":
			cb := root.Get("content_block")
			if cb.Exists() && cb.Get("type").String() == "tool_use" {
				idx := int(root.Get("index").Int())
				toolCallAccs[idx] = &ToolCallAccum{
					ID:   cb.Get("id").String(),
					Name: cb.Get("name").String(),
				}
			}
		case "content_block_delta":
			d := root.Get("delta")
			if !d.Exists() {
				continue
			}
			switch d.Get("type").String() {
			case "text_delta":
				content.WriteString(d.Get("text").String())
			case "thinking_delta":
				reasoning.WriteString(d.Get("thinking").String())
			case "input_json_delta":
				idx := int(root.Get("index").Int())
				if acc, ok := toolCallAccs[idx]; ok {
					acc.Arguments.WriteString(d.Get("partial_json").String())
				}
			}
		case "message_delta":
			if sr := root.Get("delta.stop_reason"); sr.Exists() {
				stopReason = sr.String()
			}
			if usage := root.Get("usage"); usage.Exists() {
				input := usage.Get("input_tokens").Int()
				output := usage.Get("output_tokens").Int()
				cached := usage.Get("cache_read_input_tokens").Int()
				out, _ = sjson.Set(out, "usage.prompt_tokens", input)
				out, _ = sjson.Set(out, "usage.completion_tokens", output)
				out, _ = sjson.Set(out, "usage.total_tokens", input+output)
				if cached > 0 {
					out, _ = sjson.Set(out, "usage.prompt_tokens_details.cached_tokens", cached)
				}
			}
		}
	}

	out, _ = sjson.Set(out, "id", msgID)
	out, _ = sjson.Set(out, "created", time.Now().Unix())
	out, _ = sjson.Set(out, "model", model)
	out, _ = sjson.Set(out, "choices.0.message.content", content.String())

	if reasoning.Len() > 0 {
		out, _ = sjson.Set(out, "choices.0.message.reasoning_content", reasoning.String())
	}

	if len(toolCallAccs) > 0 {
		idx := 0
		for i := 0; i <= maxKey(toolCallAccs); i++ {
			acc, ok := toolCallAccs[i]
			if !ok {
				continue
			}
			args := acc.Arguments.String()
			if args == "" {
				args = "{}"
			}
			out, _ = sjson.Set(out, fmt.Sprintf("choices.0.message.tool_calls.%d.id", idx), acc.ID)
			out, _ = sjson.Set(out, fmt.Sprintf("choices.0.message.tool_calls.%d.type", idx), "function")
			out, _ = sjson.Set(out, fmt.Sprintf("choices.0.message.tool_calls.%d.function.name", idx), acc.Name)
			out, _ = sjson.Set(out, fmt.Sprintf("choices.0.message.tool_calls.%d.function.arguments", idx), args)
			idx++
		}
		out, _ = sjson.Set(out, "choices.0.finish_reason", "tool_calls")
	} else {
		out, _ = sjson.Set(out, "choices.0.finish_reason", mapStopReason(stopReason))
	}

	return out
}

func mapStopReason(r string) string {
	switch r {
	case "end_turn":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

func maxKey(m map[int]*ToolCallAccum) int {
	mx := -1
	for k := range m {
		if k > mx {
			mx = k
		}
	}
	return mx
}
