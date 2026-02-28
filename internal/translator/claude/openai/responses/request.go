// Package responses provides request translation from OpenAI Responses API
// format to Claude Messages API format.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package responses

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

// ConvertOpenAIResponsesRequestToClaude transforms an OpenAI Responses API request
// into a Claude Messages API request.
func ConvertOpenAIResponsesRequestToClaude(modelName string, inputRawJSON []byte, stream bool) []byte {
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

	// reasoning.effort → thinking config
	if v := root.Get("reasoning.effort"); v.Exists() {
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
	if mot := root.Get("max_output_tokens"); mot.Exists() {
		out, _ = sjson.Set(out, "max_tokens", mot.Int())
	}
	out, _ = sjson.Set(out, "stream", stream)

	// instructions → system message
	instructionsText := ""
	if instr := root.Get("instructions"); instr.Exists() && instr.Type == gjson.String && instr.String() != "" {
		instructionsText = instr.String()
		sysMsg := `{"role":"user","content":""}`
		sysMsg, _ = sjson.Set(sysMsg, "content", instructionsText)
		out, _ = sjson.SetRaw(out, "messages.-1", sysMsg)
	}

	// input array processing
	if input := root.Get("input"); input.Exists() && input.IsArray() {
		input.ForEach(func(_, item gjson.Result) bool {
			typ := item.Get("type").String()
			if typ == "" && item.Get("role").String() != "" {
				typ = "message"
			}
			switch typ {
			case "message":
				role, partsJSON := parseMessageItem(item)
				if len(partsJSON) > 0 {
					msg := `{"role":"","content":[]}`
					msg, _ = sjson.Set(msg, "role", role)
					for _, p := range partsJSON {
						msg, _ = sjson.SetRaw(msg, "content.-1", p)
					}
					out, _ = sjson.SetRaw(out, "messages.-1", msg)
				}
			case "function_call":
				callID := item.Get("call_id").String()
				if callID == "" {
					callID = genToolCallID()
				}
				name := item.Get("name").String()
				argsStr := item.Get("arguments").String()
				tu := `{"type":"tool_use","id":"","name":"","input":{}}`
				tu, _ = sjson.Set(tu, "id", callID)
				tu, _ = sjson.Set(tu, "name", name)
				if argsStr != "" && gjson.Valid(argsStr) {
					aj := gjson.Parse(argsStr)
					if aj.IsObject() {
						tu, _ = sjson.SetRaw(tu, "input", aj.Raw)
					}
				}
				asst := `{"role":"assistant","content":[]}`
				asst, _ = sjson.SetRaw(asst, "content.-1", tu)
				out, _ = sjson.SetRaw(out, "messages.-1", asst)
			case "function_call_output":
				callID := item.Get("call_id").String()
				outputStr := item.Get("output").String()
				tr := `{"type":"tool_result","tool_use_id":"","content":""}`
				tr, _ = sjson.Set(tr, "tool_use_id", callID)
				tr, _ = sjson.Set(tr, "content", outputStr)
				usr := `{"role":"user","content":[]}`
				usr, _ = sjson.SetRaw(usr, "content.-1", tr)
				out, _ = sjson.SetRaw(out, "messages.-1", usr)
			}
			return true
		})
	}

	// tools mapping: parameters → input_schema
	if tools := root.Get("tools"); tools.Exists() && tools.IsArray() {
		toolsJSON := "[]"
		tools.ForEach(func(_, tool gjson.Result) bool {
			tj := `{"name":"","description":"","input_schema":{}}`
			if n := tool.Get("name"); n.Exists() {
				tj, _ = sjson.Set(tj, "name", n.String())
			}
			if d := tool.Get("description"); d.Exists() {
				tj, _ = sjson.Set(tj, "description", d.String())
			}
			if params := tool.Get("parameters"); params.Exists() {
				tj, _ = sjson.SetRaw(tj, "input_schema", params.Raw)
			}
			toolsJSON, _ = sjson.SetRaw(toolsJSON, "-1", tj)
			return true
		})
		if gjson.Parse(toolsJSON).IsArray() && len(gjson.Parse(toolsJSON).Array()) > 0 {
			out, _ = sjson.SetRaw(out, "tools", toolsJSON)
		}
	}

	// tool_choice
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
				tcj := `{"name":"","type":"tool"}`
				tcj, _ = sjson.Set(tcj, "name", fn)
				out, _ = sjson.SetRaw(out, "tool_choice", tcj)
			}
		}
	}

	return []byte(out)
}

func parseMessageItem(item gjson.Result) (string, []string) {
	var role string
	var partsJSON []string

	if parts := item.Get("content"); parts.Exists() && parts.IsArray() {
		parts.ForEach(func(_, part gjson.Result) bool {
			ptype := part.Get("type").String()
			switch ptype {
			case "input_text":
				role = "user"
				tp := `{"type":"text","text":""}`
				tp, _ = sjson.Set(tp, "text", part.Get("text").String())
				partsJSON = append(partsJSON, tp)
			case "output_text":
				role = "assistant"
				tp := `{"type":"text","text":""}`
				tp, _ = sjson.Set(tp, "text", part.Get("text").String())
				partsJSON = append(partsJSON, tp)
			case "input_image":
				if role == "" {
					role = "user"
				}
				url := part.Get("image_url").String()
				if url == "" {
					url = part.Get("url").String()
				}
				if url != "" {
					if strings.HasPrefix(url, "data:") {
						trimmed := strings.TrimPrefix(url, "data:")
						mediaAndData := strings.SplitN(trimmed, ";base64,", 2)
						if len(mediaAndData) == 2 && mediaAndData[1] != "" {
							mediaType := mediaAndData[0]
							if mediaType == "" {
								mediaType = "application/octet-stream"
							}
							ip := `{"type":"image","source":{"type":"base64","media_type":"","data":""}}`
							ip, _ = sjson.Set(ip, "source.media_type", mediaType)
							ip, _ = sjson.Set(ip, "source.data", mediaAndData[1])
							partsJSON = append(partsJSON, ip)
						}
					} else {
						ip := `{"type":"image","source":{"type":"url","url":""}}`
						ip, _ = sjson.Set(ip, "source.url", url)
						partsJSON = append(partsJSON, ip)
					}
				}
			}
			return true
		})
	}

	if role == "" {
		r := item.Get("role").String()
		switch r {
		case "user", "assistant":
			role = r
		default:
			role = "user"
		}
	}
	return role, partsJSON
}

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
