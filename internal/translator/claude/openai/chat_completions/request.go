// Package chat_completions provides request translation from OpenAI Chat Completions
// format to Claude Messages API format.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package chat_completions

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	user    = ""
	account = ""
	session = ""
)

// ConvertOpenAIRequestToClaude transforms an OpenAI Chat Completions request
// into a Claude Messages API request.
func ConvertOpenAIRequestToClaude(modelName string, inputRawJSON []byte, stream bool) []byte {
	rawJSON := inputRawJSON

	if account == "" {
		u, _ := uuid.NewRandom()
		account = u.String()
	}
	if session == "" {
		u, _ := uuid.NewRandom()
		session = u.String()
	}
	if user == "" {
		sum := sha256.Sum256([]byte(account + session))
		user = hex.EncodeToString(sum[:])
	}
	userID := fmt.Sprintf("user_%s_account_%s_session_%s", user, account, session)

	out := fmt.Sprintf(`{"model":"","max_tokens":32000,"messages":[],"metadata":{"user_id":"%s"}}`, userID)

	root := gjson.ParseBytes(rawJSON)

	// reasoning_effort → thinking config
	if v := root.Get("reasoning_effort"); v.Exists() {
		effort := strings.ToLower(strings.TrimSpace(v.String()))
		budget := convertEffortToBudget(effort)
		if budget == 0 {
			out, _ = sjson.Set(out, "thinking.type", "disabled")
		} else if budget > 0 {
			out, _ = sjson.Set(out, "thinking.type", "enabled")
			out, _ = sjson.Set(out, "thinking.budget_tokens", budget)
		}
	}

	genToolCallID := func() string {
		const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		var b strings.Builder
		for i := 0; i < 24; i++ {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
			b.WriteByte(letters[n.Int64()])
		}
		return "toolu_" + b.String()
	}

	out, _ = sjson.Set(out, "model", modelName)

	if maxTokens := root.Get("max_tokens"); maxTokens.Exists() {
		out, _ = sjson.Set(out, "max_tokens", maxTokens.Int())
	}
	if temp := root.Get("temperature"); temp.Exists() {
		out, _ = sjson.Set(out, "temperature", temp.Float())
	} else if topP := root.Get("top_p"); topP.Exists() {
		out, _ = sjson.Set(out, "top_p", topP.Float())
	}
	if stop := root.Get("stop"); stop.Exists() {
		if stop.IsArray() {
			var seqs []string
			stop.ForEach(func(_, v gjson.Result) bool { seqs = append(seqs, v.String()); return true })
			if len(seqs) > 0 {
				out, _ = sjson.Set(out, "stop_sequences", seqs)
			}
		} else {
			out, _ = sjson.Set(out, "stop_sequences", []string{stop.String()})
		}
	}
	out, _ = sjson.Set(out, "stream", stream)

	if messages := root.Get("messages"); messages.Exists() && messages.IsArray() {
		var systemParts []string
		messages.ForEach(func(_, message gjson.Result) bool {
			role := message.Get("role").String()
			contentResult := message.Get("content")

			switch role {
			case "system":
				if contentResult.Exists() && contentResult.Type == gjson.String && contentResult.String() != "" {
					systemParts = append(systemParts, contentResult.String())
				} else if contentResult.Exists() && contentResult.IsArray() {
					contentResult.ForEach(func(_, p gjson.Result) bool {
						if p.Get("type").String() == "text" {
							systemParts = append(systemParts, p.Get("text").String())
						}
						return true
					})
				}

			case "user", "assistant":
				msg := `{"role":"","content":[]}`
				msg, _ = sjson.Set(msg, "role", role)

				if contentResult.Exists() && contentResult.Type == gjson.String && contentResult.String() != "" {
					part := `{"type":"text","text":""}`
					part, _ = sjson.Set(part, "text", contentResult.String())
					msg, _ = sjson.SetRaw(msg, "content.-1", part)
				} else if contentResult.Exists() && contentResult.IsArray() {
					contentResult.ForEach(func(_, part gjson.Result) bool {
						switch part.Get("type").String() {
						case "text":
							tp := `{"type":"text","text":""}`
							tp, _ = sjson.Set(tp, "text", part.Get("text").String())
							msg, _ = sjson.SetRaw(msg, "content.-1", tp)
						case "image_url":
							imageURL := part.Get("image_url.url").String()
							if strings.HasPrefix(imageURL, "data:") {
								parts := strings.Split(imageURL, ",")
								if len(parts) == 2 {
									mediaTypePart := strings.Split(parts[0], ";")[0]
									mediaType := strings.TrimPrefix(mediaTypePart, "data:")
									data := parts[1]
									ip := `{"type":"image","source":{"type":"base64","media_type":"","data":""}}`
									ip, _ = sjson.Set(ip, "source.media_type", mediaType)
									ip, _ = sjson.Set(ip, "source.data", data)
									msg, _ = sjson.SetRaw(msg, "content.-1", ip)
								}
							} else if imageURL != "" {
								ip := `{"type":"image","source":{"type":"url","url":""}}`
								ip, _ = sjson.Set(ip, "source.url", imageURL)
								msg, _ = sjson.SetRaw(msg, "content.-1", ip)
							}
						case "file":
							fileData := part.Get("file.file_data").String()
							if strings.HasPrefix(fileData, "data:") {
								semicolonIdx := strings.Index(fileData, ";")
								commaIdx := strings.Index(fileData, ",")
								if semicolonIdx != -1 && commaIdx != -1 && commaIdx > semicolonIdx {
									mediaType := strings.TrimPrefix(fileData[:semicolonIdx], "data:")
									data := fileData[commaIdx+1:]
									dp := `{"type":"document","source":{"type":"base64","media_type":"","data":""}}`
									dp, _ = sjson.Set(dp, "source.media_type", mediaType)
									dp, _ = sjson.Set(dp, "source.data", data)
									msg, _ = sjson.SetRaw(msg, "content.-1", dp)
								}
							}
						}
						return true
					})
				}

				// tool_calls for assistant messages
				if toolCalls := message.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() && role == "assistant" {
					toolCalls.ForEach(func(_, tc gjson.Result) bool {
						if tc.Get("type").String() == "function" {
							toolCallID := tc.Get("id").String()
							if toolCallID == "" {
								toolCallID = genToolCallID()
							}
							fn := tc.Get("function")
							tu := `{"type":"tool_use","id":"","name":"","input":{}}`
							tu, _ = sjson.Set(tu, "id", toolCallID)
							tu, _ = sjson.Set(tu, "name", fn.Get("name").String())
							if args := fn.Get("arguments"); args.Exists() {
								argsStr := args.String()
								if argsStr != "" && gjson.Valid(argsStr) {
									argsJSON := gjson.Parse(argsStr)
									if argsJSON.IsObject() {
										tu, _ = sjson.SetRaw(tu, "input", argsJSON.Raw)
									}
								}
							}
							msg, _ = sjson.SetRaw(msg, "content.-1", tu)
						}
						return true
					})
				}

				out, _ = sjson.SetRaw(out, "messages.-1", msg)

			case "tool":
				toolCallID := message.Get("tool_call_id").String()
				content := message.Get("content").String()
				msg := `{"role":"user","content":[{"type":"tool_result","tool_use_id":"","content":""}]}`
				msg, _ = sjson.Set(msg, "content.0.tool_use_id", toolCallID)
				msg, _ = sjson.Set(msg, "content.0.content", content)
				out, _ = sjson.SetRaw(out, "messages.-1", msg)
			}
			return true
		})
		if len(systemParts) > 0 {
			out, _ = sjson.Set(out, "system", strings.Join(systemParts, "\n"))
		}
	}

	// Tools mapping
	if tools := root.Get("tools"); tools.Exists() && tools.IsArray() && len(tools.Array()) > 0 {
		hasTools := false
		tools.ForEach(func(_, tool gjson.Result) bool {
			if tool.Get("type").String() == "function" {
				fn := tool.Get("function")
				at := `{"name":"","description":""}`
				at, _ = sjson.Set(at, "name", fn.Get("name").String())
				at, _ = sjson.Set(at, "description", fn.Get("description").String())
				if params := fn.Get("parameters"); params.Exists() {
					at, _ = sjson.SetRaw(at, "input_schema", params.Raw)
				}
				out, _ = sjson.SetRaw(out, "tools.-1", at)
				hasTools = true
			}
			return true
		})
		if !hasTools {
			out, _ = sjson.Delete(out, "tools")
		}
	}

	// tool_choice mapping
	if tc := root.Get("tool_choice"); tc.Exists() {
		switch tc.Type {
		case gjson.String:
			switch tc.String() {
			case "auto":
				out, _ = sjson.SetRaw(out, "tool_choice", `{"type":"auto"}`)
			case "required":
				out, _ = sjson.SetRaw(out, "tool_choice", `{"type":"any"}`)
			}
		case gjson.JSON:
			if tc.Get("type").String() == "function" {
				fn := tc.Get("function.name").String()
				tcj := `{"type":"tool","name":""}`
				tcj, _ = sjson.Set(tcj, "name", fn)
				out, _ = sjson.SetRaw(out, "tool_choice", tcj)
			}
		}
	}

	return []byte(out)
}

// convertEffortToBudget maps OpenAI reasoning_effort to Claude thinking budget_tokens.
func convertEffortToBudget(effort string) int {
	switch effort {
	case "none", "off":
		return 0
	case "low":
		return 1024
	case "medium":
		return 4096
	case "high":
		return 16384
	default:
		return 8192
	}
}
