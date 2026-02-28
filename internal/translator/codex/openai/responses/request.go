// Package responses provides utilities to translate OpenAI Responses requests
// into Codex-compatible OpenAI Responses requests.
//
// Ported from the reference project CLIProxyAPI (MIT License).
package responses

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ConvertOpenAIResponsesRequestToCodex(modelName string, inputRawJSON []byte, _ bool) []byte {
	rawJSON := inputRawJSON

	// Codex upstream requires an "instructions" field (empty string is OK).
	if !gjson.GetBytes(rawJSON, "instructions").Exists() {
		rawJSON, _ = sjson.SetBytes(rawJSON, "instructions", "")
	}

	inputResult := gjson.GetBytes(rawJSON, "input")
	if inputResult.Type == gjson.String {
		input, _ := sjson.Set(`[{"type":"message","role":"user","content":[{"type":"input_text","text":""}]}]`, "0.content.0.text", inputResult.String())
		rawJSON, _ = sjson.SetRawBytes(rawJSON, "input", []byte(input))
	}

	rawJSON, _ = sjson.SetBytes(rawJSON, "stream", true)
	rawJSON, _ = sjson.SetBytes(rawJSON, "store", false)
	rawJSON, _ = sjson.SetBytes(rawJSON, "parallel_tool_calls", true)
	rawJSON, _ = sjson.SetBytes(rawJSON, "include", []string{"reasoning.encrypted_content"})
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "max_output_tokens")
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "max_completion_tokens")
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "temperature")
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "top_p")
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "service_tier")
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "truncation")
	rawJSON = applyResponsesCompactionCompatibility(rawJSON)

	rawJSON, _ = sjson.DeleteBytes(rawJSON, "user")

	rawJSON = convertSystemRoleToDeveloper(rawJSON)
	return rawJSON
}

func applyResponsesCompactionCompatibility(rawJSON []byte) []byte {
	if !gjson.GetBytes(rawJSON, "context_management").Exists() {
		return rawJSON
	}
	rawJSON, _ = sjson.DeleteBytes(rawJSON, "context_management")
	return rawJSON
}

func convertSystemRoleToDeveloper(rawJSON []byte) []byte {
	inputResult := gjson.GetBytes(rawJSON, "input")
	if !inputResult.IsArray() {
		return rawJSON
	}

	inputArray := inputResult.Array()
	result := rawJSON

	for i := 0; i < len(inputArray); i++ {
		rolePath := fmt.Sprintf("input.%d.role", i)
		if gjson.GetBytes(result, rolePath).String() == "system" {
			result, _ = sjson.SetBytes(result, rolePath, "developer")
		}
	}

	return result
}
