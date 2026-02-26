package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"lightbridge/internal/types"
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

func readBodyAndModel(req *http.Request) ([]byte, string, error) {
	if req.Body == nil {
		return nil, "", nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, "", err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return body, "", nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, "", err
	}
	model := ""
	if raw, ok := payload["model"].(string); ok {
		model = strings.TrimSpace(raw)
	}
	return body, model, nil
}

func ioNopCloser(body []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(body))
}

func statusOrDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func providerHealth(providers []types.Provider) map[string]string {
	out := map[string]string{}
	for _, p := range providers {
		out[p.ID] = p.Health
	}
	return out
}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}
