//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/ctxkey"
)

// TestQuotaPlatform 锁定配额计量口径：ForcePlatform 路由（如 /antigravity）按 ForcePlatform 计；
// 普通网关请求按入站协议计；分组不参与平台归因。
// preflight 与 post-billing 共用此口径，保证一致。
func TestQuotaPlatform(t *testing.T) {
	apiKey := &APIKey{Group: &Group{}}

	t.Run("no request protocol stays neutral", func(t *testing.T) {
		if got := QuotaPlatform(context.Background(), apiKey); got != "" {
			t.Errorf("QuotaPlatform without request protocol = %q, want empty", got)
		}
	})

	t.Run("force platform is authoritative", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAntigravity)
		if got := QuotaPlatform(ctx, apiKey); got != PlatformAntigravity {
			t.Errorf("QuotaPlatform with force = %q, want %q", got, PlatformAntigravity)
		}
	})

	t.Run("inbound openai responses determines quota platform", func(t *testing.T) {
		ctx := WithInboundProtocol(context.Background(), CustomProtocolOpenAIResponses)
		if got := QuotaPlatform(ctx, apiKey); got != PlatformOpenAI {
			t.Errorf("QuotaPlatform with OpenAI Responses inbound = %q, want %q", got, PlatformOpenAI)
		}
	})

	t.Run("inbound gemini determines quota platform", func(t *testing.T) {
		ctx := WithInboundProtocol(context.Background(), CustomProtocolGemini)
		if got := QuotaPlatform(ctx, apiKey); got != PlatformGemini {
			t.Errorf("QuotaPlatform with Gemini inbound = %q, want %q", got, PlatformGemini)
		}
	})

	t.Run("empty force platform stays neutral without inbound protocol", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, "")
		if got := QuotaPlatform(ctx, apiKey); got != "" {
			t.Errorf("QuotaPlatform with empty force = %q, want empty", got)
		}
	})

	t.Run("nil api key with force platform returns force platform", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAntigravity)
		if got := QuotaPlatform(ctx, nil); got != PlatformAntigravity {
			t.Errorf("QuotaPlatform(nil) with force = %q, want %q", got, PlatformAntigravity)
		}
	})
}
