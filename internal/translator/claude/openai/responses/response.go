// Package responses provides response translation from Claude Messages API
// streaming format to OpenAI Responses API streaming format.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package responses

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

type responsesState struct {
	Seq          int
	ResponseID   string
	CreatedAt    int64
	CurrentMsgID string
	CurrentFCID  string
	InTextBlock  bool
	InFuncBlock  bool
	FuncArgsBuf  map[int]*strings.Builder
	FuncNames    map[int]string
	FuncCallIDs  map[int]string
	TextBuf      strings.Builder
	ReasoningActive    bool
	ReasoningItemID    string
	ReasoningBuf       strings.Builder
	ReasoningPartAdded bool
	ReasoningIndex     int
	InputTokens  int64
	OutputTokens int64
	UsageSeen    bool
}

func emitEvent(event string, payload string) string {
	return fmt.Sprintf("event: %s\ndata: %s", event, payload)
}

// ConvertClaudeResponseToOpenAIResponses converts Claude SSE to OpenAI Responses SSE events.
func ConvertClaudeResponseToOpenAIResponses(_ context.Context, _ string, originalRequestRawJSON, _ []byte, rawJSON []byte, param *any) []string {
	if *param == nil {
		*param = &responsesState{
			FuncArgsBuf: make(map[int]*strings.Builder),
			FuncNames:   make(map[int]string),
			FuncCallIDs: make(map[int]string),
		}
	}
	st := (*param).(*responsesState)

	if !bytes.HasPrefix(rawJSON, dataTag) {
		return nil
	}
	rawJSON = bytes.TrimSpace(rawJSON[5:])
	root := gjson.ParseBytes(rawJSON)
	ev := root.Get("type").String()
	var out []string

	nextSeq := func() int { st.Seq++; return st.Seq }

	switch ev {
	case "message_start":
		if msg := root.Get("message"); msg.Exists() {
			st.ResponseID = msg.Get("id").String()
			st.CreatedAt = time.Now().Unix()
			st.TextBuf.Reset()
			st.ReasoningBuf.Reset()
			st.ReasoningActive = false
			st.FuncArgsBuf = make(map[int]*strings.Builder)
			st.FuncNames = make(map[int]string)
			st.FuncCallIDs = make(map[int]string)
			if usage := msg.Get("usage"); usage.Exists() {
				st.InputTokens = usage.Get("input_tokens").Int()
				st.UsageSeen = true
			}
			created := `{"type":"response.created","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"in_progress","output":[]}}`
			created, _ = sjson.Set(created, "sequence_number", nextSeq())
			created, _ = sjson.Set(created, "response.id", st.ResponseID)
			created, _ = sjson.Set(created, "response.created_at", st.CreatedAt)
			out = append(out, emitEvent("response.created", created))
			inprog := `{"type":"response.in_progress","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"in_progress"}}`
			inprog, _ = sjson.Set(inprog, "sequence_number", nextSeq())
			inprog, _ = sjson.Set(inprog, "response.id", st.ResponseID)
			inprog, _ = sjson.Set(inprog, "response.created_at", st.CreatedAt)
			out = append(out, emitEvent("response.in_progress", inprog))
		}

	case "content_block_start":
		cb := root.Get("content_block")
		if !cb.Exists() {
			return out
		}
		idx := int(root.Get("index").Int())
		typ := cb.Get("type").String()
		if typ == "text" {
			st.InTextBlock = true
			st.CurrentMsgID = fmt.Sprintf("msg_%s_0", st.ResponseID)
			item := `{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"message","status":"in_progress","content":[],"role":"assistant"}}`
			item, _ = sjson.Set(item, "sequence_number", nextSeq())
			item, _ = sjson.Set(item, "item.id", st.CurrentMsgID)
			out = append(out, emitEvent("response.output_item.added", item))
			part := `{"type":"response.content_part.added","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"part":{"type":"output_text","text":""}}`
			part, _ = sjson.Set(part, "sequence_number", nextSeq())
			part, _ = sjson.Set(part, "item_id", st.CurrentMsgID)
			out = append(out, emitEvent("response.content_part.added", part))
		} else if typ == "tool_use" {
			st.InFuncBlock = true
			st.CurrentFCID = cb.Get("id").String()
			name := cb.Get("name").String()
			item := `{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"function_call","status":"in_progress","arguments":"","call_id":"","name":""}}`
			item, _ = sjson.Set(item, "sequence_number", nextSeq())
			item, _ = sjson.Set(item, "output_index", idx)
			item, _ = sjson.Set(item, "item.id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			item, _ = sjson.Set(item, "item.call_id", st.CurrentFCID)
			item, _ = sjson.Set(item, "item.name", name)
			out = append(out, emitEvent("response.output_item.added", item))
			if st.FuncArgsBuf[idx] == nil {
				st.FuncArgsBuf[idx] = &strings.Builder{}
			}
			st.FuncCallIDs[idx] = st.CurrentFCID
			st.FuncNames[idx] = name
		} else if typ == "thinking" {
			st.ReasoningActive = true
			st.ReasoningIndex = idx
			st.ReasoningBuf.Reset()
			st.ReasoningItemID = fmt.Sprintf("rs_%s_%d", st.ResponseID, idx)
			item := `{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"reasoning","status":"in_progress","summary":[]}}`
			item, _ = sjson.Set(item, "sequence_number", nextSeq())
			item, _ = sjson.Set(item, "output_index", idx)
			item, _ = sjson.Set(item, "item.id", st.ReasoningItemID)
			out = append(out, emitEvent("response.output_item.added", item))
			part := `{"type":"response.reasoning_summary_part.added","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`
			part, _ = sjson.Set(part, "sequence_number", nextSeq())
			part, _ = sjson.Set(part, "item_id", st.ReasoningItemID)
			part, _ = sjson.Set(part, "output_index", idx)
			out = append(out, emitEvent("response.reasoning_summary_part.added", part))
			st.ReasoningPartAdded = true
		}

	case "content_block_delta":
		d := root.Get("delta")
		if !d.Exists() {
			return out
		}
		dt := d.Get("type").String()
		if dt == "text_delta" {
			if t := d.Get("text"); t.Exists() {
				msg := `{"type":"response.output_text.delta","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"delta":""}`
				msg, _ = sjson.Set(msg, "sequence_number", nextSeq())
				msg, _ = sjson.Set(msg, "item_id", st.CurrentMsgID)
				msg, _ = sjson.Set(msg, "delta", t.String())
				out = append(out, emitEvent("response.output_text.delta", msg))
				st.TextBuf.WriteString(t.String())
			}
		} else if dt == "input_json_delta" {
			idx := int(root.Get("index").Int())
			if pj := d.Get("partial_json"); pj.Exists() {
				if st.FuncArgsBuf[idx] == nil {
					st.FuncArgsBuf[idx] = &strings.Builder{}
				}
				st.FuncArgsBuf[idx].WriteString(pj.String())
				msg := `{"type":"response.function_call_arguments.delta","sequence_number":0,"item_id":"","output_index":0,"delta":""}`
				msg, _ = sjson.Set(msg, "sequence_number", nextSeq())
				msg, _ = sjson.Set(msg, "item_id", fmt.Sprintf("fc_%s", st.CurrentFCID))
				msg, _ = sjson.Set(msg, "output_index", idx)
				msg, _ = sjson.Set(msg, "delta", pj.String())
				out = append(out, emitEvent("response.function_call_arguments.delta", msg))
			}
		} else if dt == "thinking_delta" {
			if st.ReasoningActive {
				if t := d.Get("thinking"); t.Exists() {
					st.ReasoningBuf.WriteString(t.String())
					msg := `{"type":"response.reasoning_summary_text.delta","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"delta":""}`
					msg, _ = sjson.Set(msg, "sequence_number", nextSeq())
					msg, _ = sjson.Set(msg, "item_id", st.ReasoningItemID)
					msg, _ = sjson.Set(msg, "output_index", st.ReasoningIndex)
					msg, _ = sjson.Set(msg, "delta", t.String())
					out = append(out, emitEvent("response.reasoning_summary_text.delta", msg))
				}
			}
		}

	case "content_block_stop":
		idx := int(root.Get("index").Int())
		if st.InTextBlock {
			done := `{"type":"response.output_text.done","sequence_number":0,"item_id":"","output_index":0,"content_index":0,"text":""}`
			done, _ = sjson.Set(done, "sequence_number", nextSeq())
			done, _ = sjson.Set(done, "item_id", st.CurrentMsgID)
			out = append(out, emitEvent("response.output_text.done", done))
			final := `{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{"id":"","type":"message","status":"completed","content":[{"type":"output_text","text":""}],"role":"assistant"}}`
			final, _ = sjson.Set(final, "sequence_number", nextSeq())
			final, _ = sjson.Set(final, "item.id", st.CurrentMsgID)
			out = append(out, emitEvent("response.output_item.done", final))
			st.InTextBlock = false
		} else if st.InFuncBlock {
			args := "{}"
			if buf := st.FuncArgsBuf[idx]; buf != nil && buf.Len() > 0 {
				args = buf.String()
			}
			fcDone := `{"type":"response.function_call_arguments.done","sequence_number":0,"item_id":"","output_index":0,"arguments":""}`
			fcDone, _ = sjson.Set(fcDone, "sequence_number", nextSeq())
			fcDone, _ = sjson.Set(fcDone, "item_id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			fcDone, _ = sjson.Set(fcDone, "output_index", idx)
			fcDone, _ = sjson.Set(fcDone, "arguments", args)
			out = append(out, emitEvent("response.function_call_arguments.done", fcDone))
			itemDone := `{"type":"response.output_item.done","sequence_number":0,"output_index":0,"item":{"id":"","type":"function_call","status":"completed","arguments":"","call_id":"","name":""}}`
			itemDone, _ = sjson.Set(itemDone, "sequence_number", nextSeq())
			itemDone, _ = sjson.Set(itemDone, "output_index", idx)
			itemDone, _ = sjson.Set(itemDone, "item.id", fmt.Sprintf("fc_%s", st.CurrentFCID))
			itemDone, _ = sjson.Set(itemDone, "item.arguments", args)
			itemDone, _ = sjson.Set(itemDone, "item.call_id", st.CurrentFCID)
			itemDone, _ = sjson.Set(itemDone, "item.name", st.FuncNames[idx])
			out = append(out, emitEvent("response.output_item.done", itemDone))
			st.InFuncBlock = false
		} else if st.ReasoningActive {
			full := st.ReasoningBuf.String()
			textDone := `{"type":"response.reasoning_summary_text.done","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"text":""}`
			textDone, _ = sjson.Set(textDone, "sequence_number", nextSeq())
			textDone, _ = sjson.Set(textDone, "item_id", st.ReasoningItemID)
			textDone, _ = sjson.Set(textDone, "output_index", st.ReasoningIndex)
			textDone, _ = sjson.Set(textDone, "text", full)
			out = append(out, emitEvent("response.reasoning_summary_text.done", textDone))
			st.ReasoningActive = false
			st.ReasoningPartAdded = false
		}

	case "message_delta":
		if usage := root.Get("usage"); usage.Exists() {
			if v := usage.Get("output_tokens"); v.Exists() {
				st.OutputTokens = v.Int()
				st.UsageSeen = true
			}
			if v := usage.Get("input_tokens"); v.Exists() {
				st.InputTokens = v.Int()
			}
		}

	case "message_stop":
		completed := `{"type":"response.completed","sequence_number":0,"response":{"id":"","object":"response","created_at":0,"status":"completed","output":[]}}`
		completed, _ = sjson.Set(completed, "sequence_number", nextSeq())
		completed, _ = sjson.Set(completed, "response.id", st.ResponseID)
		completed, _ = sjson.Set(completed, "response.created_at", st.CreatedAt)

		// Echo request fields
		if len(originalRequestRawJSON) > 0 {
			req := gjson.ParseBytes(originalRequestRawJSON)
			for _, key := range []string{"instructions", "model", "tools", "tool_choice", "reasoning", "store", "metadata"} {
				if v := req.Get(key); v.Exists() {
					completed, _ = sjson.Set(completed, "response."+key, v.Value())
				}
			}
		}

		// Build output array
		outputsW := `{"arr":[]}`
		if st.ReasoningBuf.Len() > 0 {
			item := `{"id":"","type":"reasoning","summary":[{"type":"summary_text","text":""}]}`
			item, _ = sjson.Set(item, "id", st.ReasoningItemID)
			item, _ = sjson.Set(item, "summary.0.text", st.ReasoningBuf.String())
			outputsW, _ = sjson.SetRaw(outputsW, "arr.-1", item)
		}
		if st.TextBuf.Len() > 0 || st.CurrentMsgID != "" {
			item := `{"id":"","type":"message","status":"completed","content":[{"type":"output_text","text":""}],"role":"assistant"}`
			item, _ = sjson.Set(item, "id", st.CurrentMsgID)
			item, _ = sjson.Set(item, "content.0.text", st.TextBuf.String())
			outputsW, _ = sjson.SetRaw(outputsW, "arr.-1", item)
		}
		if len(st.FuncArgsBuf) > 0 {
			for idx := 0; idx <= maxKeyInt(st.FuncArgsBuf); idx++ {
				buf, ok := st.FuncArgsBuf[idx]
				if !ok {
					continue
				}
				args := "{}"
				if buf.Len() > 0 {
					args = buf.String()
				}
				callID := st.FuncCallIDs[idx]
				name := st.FuncNames[idx]
				item := `{"id":"","type":"function_call","status":"completed","arguments":"","call_id":"","name":""}`
				item, _ = sjson.Set(item, "id", fmt.Sprintf("fc_%s", callID))
				item, _ = sjson.Set(item, "arguments", args)
				item, _ = sjson.Set(item, "call_id", callID)
				item, _ = sjson.Set(item, "name", name)
				outputsW, _ = sjson.SetRaw(outputsW, "arr.-1", item)
			}
		}
		if gjson.Get(outputsW, "arr.#").Int() > 0 {
			completed, _ = sjson.SetRaw(completed, "response.output", gjson.Get(outputsW, "arr").Raw)
		}

		if st.UsageSeen {
			completed, _ = sjson.Set(completed, "response.usage.input_tokens", st.InputTokens)
			completed, _ = sjson.Set(completed, "response.usage.output_tokens", st.OutputTokens)
			completed, _ = sjson.Set(completed, "response.usage.total_tokens", st.InputTokens+st.OutputTokens)
		}
		out = append(out, emitEvent("response.completed", completed))
	}

	return out
}

// ConvertClaudeResponseToOpenAIResponsesNonStream aggregates Claude SSE into a single OpenAI Responses JSON.
func ConvertClaudeResponseToOpenAIResponsesNonStream(_ context.Context, _ string, originalRequestRawJSON, _ []byte, rawJSON []byte, _ *any) string {
	// Delegate to streaming aggregation
	var param any
	var allOut []string
	for _, line := range bytes.Split(rawJSON, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if !bytes.HasPrefix(trimmed, dataTag) {
			continue
		}
		results := ConvertClaudeResponseToOpenAIResponses(context.Background(), "", originalRequestRawJSON, nil, trimmed, &param)
		allOut = append(allOut, results...)
	}
	// Find the response.completed event and extract its data
	for _, ev := range allOut {
		if strings.HasPrefix(ev, "event: response.completed\ndata: ") {
			return strings.TrimPrefix(ev, "event: response.completed\ndata: ")
		}
	}
	return ""
}

func maxKeyInt(m map[int]*strings.Builder) int {
	mx := -1
	for k := range m {
		if k > mx {
			mx = k
		}
	}
	return mx
}
