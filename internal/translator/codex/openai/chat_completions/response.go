// Package chat_completions provides response translation functionality for Codex to OpenAI API compatibility.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package chat_completions

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var dataTag = []byte("data:")

type ConvertCodexToOpenAIParams struct {
	ResponseID                string
	CreatedAt                 int64
	Model                     string
	FunctionCallIndex         int
	HasReceivedArgumentsDelta bool
	HasToolCallAnnounced      bool
}

// ConvertCodexResponseToOpenAI translates a single chunk of a streaming response from the
// Codex API format to the OpenAI Chat Completions streaming format.
func ConvertCodexResponseToOpenAI(_ context.Context, modelName string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []string {
	if *param == nil {
		*param = &ConvertCodexToOpenAIParams{
			Model:                     modelName,
			CreatedAt:                 0,
			ResponseID:                "",
			FunctionCallIndex:         -1,
			HasReceivedArgumentsDelta: false,
			HasToolCallAnnounced:      false,
		}
	}

	if !bytes.HasPrefix(rawJSON, dataTag) {
		return []string{}
	}
	rawJSON = bytes.TrimSpace(rawJSON[5:])

	template := `{"id":"","object":"chat.completion.chunk","created":12345,"model":"model","choices":[{"index":0,"delta":{"role":null,"content":null,"reasoning_content":null,"tool_calls":null},"finish_reason":null,"native_finish_reason":null}]}`

	rootResult := gjson.ParseBytes(rawJSON)

	typeResult := rootResult.Get("type")
	dataType := typeResult.String()
	if dataType == "response.created" {
		(*param).(*ConvertCodexToOpenAIParams).ResponseID = rootResult.Get("response.id").String()
		(*param).(*ConvertCodexToOpenAIParams).CreatedAt = rootResult.Get("response.created_at").Int()
		(*param).(*ConvertCodexToOpenAIParams).Model = rootResult.Get("response.model").String()
		return []string{}
	}

	// Extract and set the model version.
	if modelResult := gjson.GetBytes(rawJSON, "model"); modelResult.Exists() {
		template, _ = sjson.Set(template, "model", modelResult.String())
	} else if strings.TrimSpace((*param).(*ConvertCodexToOpenAIParams).Model) != "" {
		// Some Codex SSE delta events omit "model"; fall back to the model captured from response.created.
		template, _ = sjson.Set(template, "model", (*param).(*ConvertCodexToOpenAIParams).Model)
	} else if strings.TrimSpace(modelName) != "" {
		template, _ = sjson.Set(template, "model", modelName)
	}

	template, _ = sjson.Set(template, "created", (*param).(*ConvertCodexToOpenAIParams).CreatedAt)
	template, _ = sjson.Set(template, "id", (*param).(*ConvertCodexToOpenAIParams).ResponseID)

	// Extract and set usage metadata (token counts).
	if usageResult := gjson.GetBytes(rawJSON, "response.usage"); usageResult.Exists() {
		if outputTokensResult := usageResult.Get("output_tokens"); outputTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.completion_tokens", outputTokensResult.Int())
		}
		if totalTokensResult := usageResult.Get("total_tokens"); totalTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.total_tokens", totalTokensResult.Int())
		}
		if inputTokensResult := usageResult.Get("input_tokens"); inputTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.prompt_tokens", inputTokensResult.Int())
		}
		if cachedTokensResult := usageResult.Get("input_tokens_details.cached_tokens"); cachedTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.prompt_tokens_details.cached_tokens", cachedTokensResult.Int())
		}
		if reasoningTokensResult := usageResult.Get("output_tokens_details.reasoning_tokens"); reasoningTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.completion_tokens_details.reasoning_tokens", reasoningTokensResult.Int())
		}
	}

	if dataType == "response.reasoning_summary_text.delta" {
		if deltaResult := rootResult.Get("delta"); deltaResult.Exists() {
			template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
			template, _ = sjson.Set(template, "choices.0.delta.reasoning_content", deltaResult.String())
		}
	} else if dataType == "response.reasoning_summary_text.done" {
		template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
		template, _ = sjson.Set(template, "choices.0.delta.reasoning_content", "\n\n")
	} else if dataType == "response.output_text.delta" {
		if deltaResult := rootResult.Get("delta"); deltaResult.Exists() {
			template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
			template, _ = sjson.Set(template, "choices.0.delta.content", deltaResult.String())
		}
	} else if dataType == "response.completed" {
		finishReason := "stop"
		if (*param).(*ConvertCodexToOpenAIParams).FunctionCallIndex != -1 {
			finishReason = "tool_calls"
		}
		template, _ = sjson.Set(template, "choices.0.finish_reason", finishReason)
		template, _ = sjson.Set(template, "choices.0.native_finish_reason", finishReason)
	} else if dataType == "response.output_item.added" {
		itemResult := rootResult.Get("item")
		if !itemResult.Exists() {
			return []string{}
		}

		itemType := itemResult.Get("type").String()
		if itemType == "function_call" {
			if !(*param).(*ConvertCodexToOpenAIParams).HasToolCallAnnounced {
				(*param).(*ConvertCodexToOpenAIParams).FunctionCallIndex++
				(*param).(*ConvertCodexToOpenAIParams).HasToolCallAnnounced = true
			}

			functionCallItemTemplate := `{"index":0,"id":"","type":"function","function":{"name":"","arguments":""}}`
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "index", (*param).(*ConvertCodexToOpenAIParams).FunctionCallIndex)
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "id", itemResult.Get("call_id").String())

			// Restore original tool name if it was shortened.
			name := itemResult.Get("name").String()
			rev := buildReverseMapFromOriginalOpenAI(originalRequestRawJSON)
			if orig, ok := rev[name]; ok {
				name = orig
			}
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "function.name", name)

			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "function.arguments", itemResult.Get("arguments").String())
			template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
			template, _ = sjson.SetRaw(template, "choices.0.delta.tool_calls.-1", functionCallItemTemplate)

		} else {
			return []string{}
		}
	} else if dataType == "response.output_item.delta" {
		itemResult := rootResult.Get("item")
		if !itemResult.Exists() {
			return []string{}
		}

		itemType := itemResult.Get("type").String()
		if itemType == "function_call" {
			arguments := itemResult.Get("arguments").String()
			if arguments != "" {
				(*param).(*ConvertCodexToOpenAIParams).HasReceivedArgumentsDelta = true
			}

			functionCallItemTemplate := `{"index":0,"id":"","type":"function","function":{"name":"","arguments":""}}`
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "index", (*param).(*ConvertCodexToOpenAIParams).FunctionCallIndex)
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "id", itemResult.Get("call_id").String())

			name := itemResult.Get("name").String()
			rev := buildReverseMapFromOriginalOpenAI(originalRequestRawJSON)
			if orig, ok := rev[name]; ok {
				name = orig
			}
			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "function.name", name)

			functionCallItemTemplate, _ = sjson.Set(functionCallItemTemplate, "function.arguments", arguments)
			template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
			template, _ = sjson.SetRaw(template, "choices.0.delta.tool_calls.-1", functionCallItemTemplate)
		} else {
			return []string{}
		}
	} else {
		return []string{}
	}

	return []string{template}
}

// ConvertCodexResponseToOpenAINonStream converts a non-streaming Codex response to a non-streaming OpenAI response.
func ConvertCodexResponseToOpenAINonStream(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, _ *any) string {
	rootResult := gjson.ParseBytes(rawJSON)
	if rootResult.Get("type").String() != "response.completed" {
		return ""
	}

	unixTimestamp := time.Now().Unix()

	responseResult := rootResult.Get("response")
	template := `{"id":"","object":"chat.completion","created":123456,"model":"model","choices":[{"index":0,"message":{"role":"assistant","content":null,"reasoning_content":null,"tool_calls":null},"finish_reason":null,"native_finish_reason":null}]}`

	if modelResult := responseResult.Get("model"); modelResult.Exists() {
		template, _ = sjson.Set(template, "model", modelResult.String())
	}

	if createdAtResult := responseResult.Get("created_at"); createdAtResult.Exists() {
		template, _ = sjson.Set(template, "created", createdAtResult.Int())
	} else {
		template, _ = sjson.Set(template, "created", unixTimestamp)
	}

	if idResult := responseResult.Get("id"); idResult.Exists() {
		template, _ = sjson.Set(template, "id", idResult.String())
	}

	if usageResult := responseResult.Get("usage"); usageResult.Exists() {
		if outputTokensResult := usageResult.Get("output_tokens"); outputTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.completion_tokens", outputTokensResult.Int())
		}
		if totalTokensResult := usageResult.Get("total_tokens"); totalTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.total_tokens", totalTokensResult.Int())
		}
		if inputTokensResult := usageResult.Get("input_tokens"); inputTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.prompt_tokens", inputTokensResult.Int())
		}
		if cachedTokensResult := usageResult.Get("input_tokens_details.cached_tokens"); cachedTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.prompt_tokens_details.cached_tokens", cachedTokensResult.Int())
		}
		if reasoningTokensResult := usageResult.Get("output_tokens_details.reasoning_tokens"); reasoningTokensResult.Exists() {
			template, _ = sjson.Set(template, "usage.completion_tokens_details.reasoning_tokens", reasoningTokensResult.Int())
		}
	}

	outputResult := responseResult.Get("output")
	if outputResult.IsArray() {
		outputArray := outputResult.Array()
		var contentText string
		var reasoningText string
		var toolCalls []string

		for _, outputItem := range outputArray {
			outputType := outputItem.Get("type").String()

			switch outputType {
			case "reasoning":
				if summaryResult := outputItem.Get("summary"); summaryResult.IsArray() {
					summaryArray := summaryResult.Array()
					for _, summaryItem := range summaryArray {
						if summaryItem.Get("type").String() == "summary_text" {
							reasoningText = summaryItem.Get("text").String()
							break
						}
					}
				}
			case "message":
				if contentResult := outputItem.Get("content"); contentResult.IsArray() {
					contentArray := contentResult.Array()
					for _, contentItem := range contentArray {
						if contentItem.Get("type").String() == "output_text" {
							contentText = contentItem.Get("text").String()
							break
						}
					}
				}
			case "function_call":
				functionCallTemplate := `{"id":"","type":"function","function":{"name":"","arguments":""}}`

				if callIdResult := outputItem.Get("call_id"); callIdResult.Exists() {
					functionCallTemplate, _ = sjson.Set(functionCallTemplate, "id", callIdResult.String())
				}

				if nameResult := outputItem.Get("name"); nameResult.Exists() {
					n := nameResult.String()
					rev := buildReverseMapFromOriginalOpenAI(originalRequestRawJSON)
					if orig, ok := rev[n]; ok {
						n = orig
					}
					functionCallTemplate, _ = sjson.Set(functionCallTemplate, "function.name", n)
				}

				if argsResult := outputItem.Get("arguments"); argsResult.Exists() {
					functionCallTemplate, _ = sjson.Set(functionCallTemplate, "function.arguments", argsResult.String())
				}

				toolCalls = append(toolCalls, functionCallTemplate)
			}
		}

		if contentText != "" {
			template, _ = sjson.Set(template, "choices.0.message.content", contentText)
		}
		if reasoningText != "" {
			template, _ = sjson.Set(template, "choices.0.message.reasoning_content", reasoningText)
		}
		if len(toolCalls) > 0 {
			template, _ = sjson.Set(template, "choices.0.finish_reason", "tool_calls")
			template, _ = sjson.Set(template, "choices.0.native_finish_reason", "tool_calls")
			template, _ = sjson.SetRaw(template, "choices.0.message.tool_calls", "["+strings.Join(toolCalls, ",")+"]")
		}
	}

	if statusResult := responseResult.Get("status"); statusResult.Exists() {
		status := statusResult.String()
		if status == "completed" {
			template, _ = sjson.Set(template, "choices.0.finish_reason", "stop")
			template, _ = sjson.Set(template, "choices.0.native_finish_reason", "stop")
		}
	}

	return template
}

func buildReverseMapFromOriginalOpenAI(original []byte) map[string]string {
	tools := gjson.GetBytes(original, "tools")
	rev := map[string]string{}
	if tools.IsArray() && len(tools.Array()) > 0 {
		var names []string
		arr := tools.Array()
		for i := 0; i < len(arr); i++ {
			t := arr[i]
			if t.Get("type").String() != "function" {
				continue
			}
			fn := t.Get("function")
			if !fn.Exists() {
				continue
			}
			if v := fn.Get("name"); v.Exists() {
				names = append(names, v.String())
			}
		}
		if len(names) > 0 {
			m := buildShortNameMap(names)
			for orig, short := range m {
				rev[short] = orig
			}
		}
	}
	return rev
}
