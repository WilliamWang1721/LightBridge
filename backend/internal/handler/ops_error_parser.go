package handler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

// isCountTokensRequest checks if the request is a count_tokens request
func isCountTokensRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	return strings.Contains(c.Request.URL.Path, "/count_tokens")
}

func applyOpsLatencyFieldsFromContext(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || entry == nil {
		return
	}
	entry.AuthLatencyMs = getContextLatencyMs(c, service.OpsAuthLatencyMsKey)
	entry.RoutingLatencyMs = getContextLatencyMs(c, service.OpsRoutingLatencyMsKey)
	entry.UpstreamLatencyMs = getContextLatencyMs(c, service.OpsUpstreamLatencyMsKey)
	entry.ResponseLatencyMs = getContextLatencyMs(c, service.OpsResponseLatencyMsKey)
	entry.TimeToFirstTokenMs = getContextLatencyMs(c, service.OpsTimeToFirstTokenMsKey)
}

func getContextLatencyMs(c *gin.Context, key string) *int64 {
	if c == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	var ms int64
	switch t := v.(type) {
	case int:
		ms = int64(t)
	case int32:
		ms = int64(t)
	case int64:
		ms = t
	case float64:
		ms = int64(t)
	default:
		return nil
	}
	if ms < 0 {
		return nil
	}
	return &ms
}

type parsedOpsError struct {
	ErrorType string
	Message   string
	Code      string
}

func parseOpsErrorResponse(body []byte) parsedOpsError {
	if len(body) == 0 {
		return parsedOpsError{}
	}

	// Fast path: attempt to decode into a generic map.
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return parsedOpsError{Message: truncateString(string(body), 1024)}
	}

	// Claude/OpenAI-style gateway error: { type:"error", error:{ type, message } }
	if errObj, ok := m["error"].(map[string]any); ok {
		t, _ := errObj["type"].(string)
		msg, _ := errObj["message"].(string)
		// Gemini googleError also uses "error": { code, message, status }
		if msg == "" {
			if v, ok := errObj["message"]; ok {
				msg, _ = v.(string)
			}
		}
		if t == "" {
			// Gemini error does not have "type" field.
			t = "api_error"
		}
		// For gemini error, capture numeric code as string for business-limited mapping if needed.
		var code string
		if v, ok := errObj["code"]; ok {
			switch n := v.(type) {
			case string:
				code = strings.TrimSpace(n)
			case float64:
				code = strconvItoa(int(n))
			case int:
				code = strconvItoa(n)
			}
		}
		return parsedOpsError{ErrorType: t, Message: msg, Code: code}
	}

	// APIKeyAuth-style: { code:"INSUFFICIENT_BALANCE", message:"..." }
	code := stringValue(m["reason"])
	if code == "" {
		code = stringValue(m["code"])
	}
	msg, _ := m["message"].(string)
	errorType := stringValue(m["type"])
	if errorType == "" {
		errorType = "api_error"
	}
	if code != "" || msg != "" {
		return parsedOpsError{ErrorType: errorType, Message: msg, Code: code}
	}

	return parsedOpsError{Message: truncateString(string(body), 1024)}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconvItoa(int(typed))
	case int:
		return strconvItoa(typed)
	default:
		return ""
	}
}

func resolveOpsPlatform(ctx context.Context, apiKey *service.APIKey, fallback string) string {
	_ = apiKey
	return service.PlatformForRequest(ctx, fallback)
}

func guessPlatformFromPath(path string) string {
	p := strings.ToLower(path)
	switch {
	case strings.HasPrefix(p, "/antigravity/"):
		return service.PlatformAntigravity
	case strings.HasPrefix(p, "/v1beta/"):
		return service.PlatformGemini
	case strings.Contains(p, "/responses"), strings.Contains(p, "/images/"):
		return service.PlatformOpenAI
	default:
		return ""
	}
}
