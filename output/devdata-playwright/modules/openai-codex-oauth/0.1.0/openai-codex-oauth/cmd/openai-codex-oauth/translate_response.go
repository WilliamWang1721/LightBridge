package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type codexEvent struct {
	Type      string         `json:"type"`
	Delta     string         `json:"delta,omitempty"`
	Arguments string         `json:"arguments,omitempty"`
	Response  *codexResponse `json:"response,omitempty"`
	Item      *codexItem     `json:"item,omitempty"`
}

type codexResponse struct {
	ID        string            `json:"id"`
	CreatedAt int64             `json:"created_at"`
	Model     string            `json:"model"`
	Status    string            `json:"status,omitempty"`
	Usage     *codexUsage       `json:"usage,omitempty"`
	Output    []codexOutputItem `json:"output,omitempty"`
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`

	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

type codexOutputItem struct {
	Type      string             `json:"type"`
	Content   []codexContentItem `json:"content,omitempty"` // message
	Summary   []codexSummaryItem `json:"summary,omitempty"` // reasoning
	CallID    string             `json:"call_id,omitempty"`
	Name      string             `json:"name,omitempty"`
	Arguments string             `json:"arguments,omitempty"`
}

type codexContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexSummaryItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type streamState struct {
	responseID string
	createdAt  int64
	model      string

	toolCallIndex             int
	hasReceivedArgumentsDelta bool
	hasToolCallAnnounced      bool
	revToolNameMap            map[string]string // short -> original
}

type openAIChatCompletionChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []openAIChunkChoice `json:"choices"`
	Usage   *openAIUsage        `json:"usage,omitempty"`
}

type openAIChunkChoice struct {
	Index              int              `json:"index"`
	Delta              openAIChunkDelta `json:"delta"`
	FinishReason       *string          `json:"finish_reason,omitempty"`
	NativeFinishReason *string          `json:"native_finish_reason,omitempty"`
}

type openAIChunkDelta struct {
	Role             *string               `json:"role,omitempty"`
	Content          *string               `json:"content,omitempty"`
	ReasoningContent *string               `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIToolCallDelta `json:"tool_calls,omitempty"`
}

type openAIToolCallDelta struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function openAIToolCallFuncDel `json:"function"`
}

type openAIToolCallFuncDel struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIChatCompletion struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
}

type openAIChoice struct {
	Index              int           `json:"index"`
	Message            openAIMessage `json:"message"`
	FinishReason       string        `json:"finish_reason,omitempty"`
	NativeFinishReason string        `json:"native_finish_reason,omitempty"`
}

type openAIMessage struct {
	Role             string           `json:"role"`
	Content          *string          `json:"content"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolCallFunc `json:"function"`
}

type openAIToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptTokensDetails *struct {
		CachedTokens int `json:"cached_tokens,omitempty"`
	} `json:"prompt_tokens_details,omitempty"`

	CompletionTokensDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	} `json:"completion_tokens_details,omitempty"`
}

func convertCodexEventToOpenAIChunks(data []byte, st *streamState) (chunks [][]byte, done bool, err error) {
	if st == nil {
		return nil, false, errors.New("nil state")
	}
	var ev codexEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, false, err
	}

	switch ev.Type {
	case "response.created":
		if ev.Response != nil {
			if strings.TrimSpace(ev.Response.ID) != "" {
				st.responseID = strings.TrimSpace(ev.Response.ID)
			}
			if ev.Response.CreatedAt > 0 {
				st.createdAt = ev.Response.CreatedAt
			}
			if strings.TrimSpace(ev.Response.Model) != "" {
				st.model = strings.TrimSpace(ev.Response.Model)
			}
		}
		if st.createdAt == 0 {
			st.createdAt = time.Now().Unix()
		}
		return nil, false, nil
	}

	chunk := openAIChatCompletionChunk{
		ID:      st.responseID,
		Object:  "chat.completion.chunk",
		Created: st.createdAt,
		Model:   st.model,
		Choices: []openAIChunkChoice{{
			Index: 0,
			Delta: openAIChunkDelta{},
		}},
	}

	if ev.Response != nil && ev.Response.Usage != nil {
		chunk.Usage = convertUsage(ev.Response.Usage)
	}

	role := "assistant"
	setRole := func() { chunk.Choices[0].Delta.Role = &role }

	switch ev.Type {
	case "response.output_text.delta":
		setRole()
		d := ev.Delta
		chunk.Choices[0].Delta.Content = &d
	case "response.reasoning_summary_text.delta":
		setRole()
		d := ev.Delta
		chunk.Choices[0].Delta.ReasoningContent = &d
	case "response.reasoning_summary_text.done":
		setRole()
		d := "\n\n"
		chunk.Choices[0].Delta.ReasoningContent = &d
	case "response.output_item.added":
		if ev.Item == nil || ev.Item.Type != "function_call" {
			return nil, false, nil
		}
		st.toolCallIndex++
		st.hasToolCallAnnounced = true

		name := restoreToolName(strings.TrimSpace(ev.Item.Name), st.revToolNameMap)
		setRole()
		chunk.Choices[0].Delta.ToolCalls = []openAIToolCallDelta{{
			Index: st.toolCallIndex,
			ID:    strings.TrimSpace(ev.Item.CallID),
			Type:  "function",
			Function: openAIToolCallFuncDel{
				Name:      name,
				Arguments: "",
			},
		}}
	case "response.function_call_arguments.delta":
		st.hasReceivedArgumentsDelta = true
		chunk.Choices[0].Delta.ToolCalls = []openAIToolCallDelta{{
			Index: st.toolCallIndex,
			Function: openAIToolCallFuncDel{
				Arguments: ev.Delta,
			},
		}}
	case "response.function_call_arguments.done":
		if st.hasReceivedArgumentsDelta {
			return nil, false, nil
		}
		chunk.Choices[0].Delta.ToolCalls = []openAIToolCallDelta{{
			Index: st.toolCallIndex,
			Function: openAIToolCallFuncDel{
				Arguments: ev.Arguments,
			},
		}}
	case "response.output_item.done":
		if ev.Item == nil || ev.Item.Type != "function_call" {
			return nil, false, nil
		}
		if st.hasToolCallAnnounced {
			st.hasToolCallAnnounced = false
			return nil, false, nil
		}
		st.toolCallIndex++
		name := restoreToolName(strings.TrimSpace(ev.Item.Name), st.revToolNameMap)
		setRole()
		chunk.Choices[0].Delta.ToolCalls = []openAIToolCallDelta{{
			Index: st.toolCallIndex,
			ID:    strings.TrimSpace(ev.Item.CallID),
			Type:  "function",
			Function: openAIToolCallFuncDel{
				Name:      name,
				Arguments: ev.Item.Arguments,
			},
		}}
	case "response.completed":
		finish := "stop"
		if st.toolCallIndex >= 0 {
			finish = "tool_calls"
		}
		chunk.Choices[0].FinishReason = &finish
		chunk.Choices[0].NativeFinishReason = &finish
		done = true
	default:
		return nil, false, nil
	}

	b, err := json.Marshal(chunk)
	if err != nil {
		return nil, false, err
	}
	return [][]byte{b}, done, nil
}

func convertCodexCompletedToOpenAICompletion(ev *codexEvent, requestedModel string, revToolName map[string]string) (*openAIChatCompletion, error) {
	if ev == nil || ev.Response == nil {
		return nil, errors.New("missing response")
	}
	resp := ev.Response
	model := strings.TrimSpace(resp.Model)
	if model == "" {
		model = strings.TrimSpace(requestedModel)
	}
	created := resp.CreatedAt
	if created == 0 {
		created = time.Now().Unix()
	}

	var contentText string
	var reasoningText string
	toolCalls := make([]openAIToolCall, 0)

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, c := range item.Content {
				if c.Type == "output_text" && strings.TrimSpace(c.Text) != "" {
					contentText = c.Text
					break
				}
			}
		case "reasoning":
			for _, s := range item.Summary {
				if s.Type == "summary_text" && strings.TrimSpace(s.Text) != "" {
					reasoningText = s.Text
					break
				}
			}
		case "function_call":
			name := restoreToolName(strings.TrimSpace(item.Name), revToolName)
			toolCalls = append(toolCalls, openAIToolCall{
				ID:   strings.TrimSpace(item.CallID),
				Type: "function",
				Function: openAIToolCallFunc{
					Name:      name,
					Arguments: item.Arguments,
				},
			})
		}
	}

	var contentPtr *string
	if strings.TrimSpace(contentText) != "" {
		contentPtr = &contentText
	} else {
		// OpenAI uses null content when tool_calls are returned.
		contentPtr = nil
	}

	var reasoningPtr *string
	if strings.TrimSpace(reasoningText) != "" {
		reasoningPtr = &reasoningText
	}

	msg := openAIMessage{
		Role:             "assistant",
		Content:          contentPtr,
		ReasoningContent: reasoningPtr,
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	finish := "stop"
	if len(toolCalls) > 0 {
		finish = "tool_calls"
	}

	out := &openAIChatCompletion{
		ID:      strings.TrimSpace(resp.ID),
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []openAIChoice{{
			Index:              0,
			Message:            msg,
			FinishReason:       finish,
			NativeFinishReason: finish,
		}},
	}
	if resp.Usage != nil {
		out.Usage = convertUsage(resp.Usage)
	}
	return out, nil
}

func convertUsage(u *codexUsage) *openAIUsage {
	if u == nil {
		return nil
	}
	out := &openAIUsage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
	}
	if u.InputTokensDetails.CachedTokens > 0 {
		out.PromptTokensDetails = &struct {
			CachedTokens int `json:"cached_tokens,omitempty"`
		}{CachedTokens: u.InputTokensDetails.CachedTokens}
	}
	if u.OutputTokensDetails.ReasoningTokens > 0 {
		out.CompletionTokensDetails = &struct {
			ReasoningTokens int `json:"reasoning_tokens,omitempty"`
		}{ReasoningTokens: u.OutputTokensDetails.ReasoningTokens}
	}
	return out
}

func restoreToolName(name string, rev map[string]string) string {
	name = strings.TrimSpace(name)
	if name == "" || rev == nil {
		return name
	}
	if orig, ok := rev[name]; ok {
		return orig
	}
	return name
}
