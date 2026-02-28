package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func writeOpenAIError(w http.ResponseWriter, status int, message, errType, code string) {
	if errType == "" {
		errType = "invalid_request_error"
	}
	if code == "" {
		code = "error"
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"code":    code,
		},
	})
}

func parseUpstreamError(body io.Reader) string {
	buf, _ := io.ReadAll(io.LimitReader(body, 1<<20))
	if len(buf) == 0 {
		return "Upstream provider returned an error"
	}
	var obj map[string]any
	if err := json.Unmarshal(buf, &obj); err == nil {
		if errObj, ok := obj["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return msg
			}
		}
		if msg, ok := obj["message"].(string); ok && msg != "" {
			return msg
		}
	}
	msg := strings.TrimSpace(string(buf))
	if msg == "" {
		return "Upstream provider returned an error"
	}
	return msg
}
