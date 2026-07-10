package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// defaultAuthenticityPassiveThreshold 是被动检测连续可疑次数的默认阈值。
// 真值由 AuthenticitySettings.PassiveThreshold 提供；设置读取失败时回退到该默认值。
const defaultAuthenticityPassiveThreshold = 3

// thinkingSignatureState 跟踪一次流式响应中 thinking block 与其 signature 的出现情况。
// 真 Anthropic 在开启 thinking 时，每个 thinking content block 都会带 signature_delta。
type thinkingSignatureState struct {
	enabled          bool // 本次请求是否明确开启了 thinking
	sawThinkingBlock bool // 是否出现 content_block_start.type == thinking
	sawSignature     bool // 是否出现 signature_delta 且 signature 非空
}

// isThinkingEnabledPayload 判断请求体是否明确开启了 thinking。
// 复用 gateway_service.go 中既有的判定口径：thinking.type 为 enabled 或 adaptive。
func isThinkingEnabledPayload(body []byte) bool {
	t := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String()))
	return t == "enabled" || t == "adaptive"
}

// parseAuthenticityPassthrough 在 SSE 旁路解析里检测 thinking block 与 signature_delta。
// 仅做状态累加，不持久化；持久化在流结束后由 evaluateAuthenticityPassive 统一处理。
func (s *GatewayService) parseAuthenticityPassthrough(data string, st *thinkingSignatureState) {
	if st == nil || !st.enabled || data == "" || data == "[DONE]" {
		return
	}
	parsed := gjson.Parse(data)
	switch parsed.Get("type").String() {
	case "content_block_start":
		// content_block.type == "thinking" 或 "redacted_thinking"
		if bt := strings.ToLower(parsed.Get("content_block.type").String()); bt == "thinking" || bt == "redacted_thinking" {
			st.sawThinkingBlock = true
		}
	case "content_block_delta":
		// delta.type == "signature_delta" 且 signature 非空 → 合法签名
		if strings.ToLower(parsed.Get("delta.type").String()) == "signature_delta" {
			if sig := parsed.Get("delta.signature").String(); strings.TrimSpace(sig) != "" {
				st.sawSignature = true
			}
		}
	}
}

// evaluateAuthenticityPassive 在一次完整流结束后评估真伪并持久化到 Account.Extra。
//
// 判定逻辑（仅对明确开启 thinking 的请求）：
//   - 出现合法 signature → 真（verdict=genuine），并清零可疑计数。
//   - 开了 thinking 却始终无合法 signature → 可疑计数 +1；累计达到阈值则标记 counterfeit。
//
// 阈值（默认 3）可在 AuthenticitySettings.PassiveThreshold 配置；连续多次可疑才标记，
// 避免上游临时降级/不支持 thinking 的模型造成误判。
func (s *GatewayService) evaluateAuthenticityPassive(ctx context.Context, account *Account, st *thinkingSignatureState) {
	if account == nil || st == nil || !st.enabled {
		return
	}

	// 优先读取配置阈值；设置不可用时回退到默认值。
	threshold := defaultAuthenticityPassiveThreshold
	if settings, err := s.settingService.GetAuthenticitySettings(ctx); err == nil && settings.Enabled && settings.PassiveThreshold > 0 {
		threshold = settings.PassiveThreshold
	} else if err == nil && settings != nil && !settings.Enabled {
		// 总开关关闭：不做被动判定。
		return
	}

	now := time.Now().UTC()

	if st.sawSignature {
		// 检测到合法 signature → 确认真，清零可疑计数。
		updates := map[string]any{
			AccountExtraKeyAuthenticityVerdict:    AuthenticityVerdictGenuine,
			AccountExtraKeyAuthenticityCheckedAt:  now.Format(time.RFC3339),
			AccountExtraKeyAuthenticityMethod:     AuthenticityMethodPassive,
			AccountExtraKeyAuthenticityDetail:     "valid thinking signature observed in stream",
			AccountExtraKeyAuthenticitySuspicious: 0,
		}
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
			slog.Warn("authenticity_passive_persist_failed", "account_id", account.ID, "verdict", AuthenticityVerdictGenuine, "error", err)
		}
		return
	}

	// 开了 thinking 却无 signature_delta：可疑。累计计数。
	count := readAuthenticitySuspiciousCount(account) + 1
	if count < threshold {
		// 还没到阈值：只更新计数与最近一次可疑时间，不改变已有 verdict。
		updates := map[string]any{
			AccountExtraKeyAuthenticitySuspicious: count,
		}
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
			slog.Warn("authenticity_passive_count_failed", "account_id", account.ID, "error", err)
		}
		return
	}

	// 达到阈值：标记假冒（被动）。
	updates := map[string]any{
		AccountExtraKeyAuthenticityVerdict:    AuthenticityVerdictCounterfeit,
		AccountExtraKeyAuthenticityCheckedAt:  now.Format(time.RFC3339),
		AccountExtraKeyAuthenticityMethod:     AuthenticityMethodPassive,
		AccountExtraKeyAuthenticityDetail:     "thinking enabled but no valid signature across consecutive streams",
		AccountExtraKeyAuthenticitySuspicious: count,
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		slog.Warn("authenticity_passive_persist_failed", "account_id", account.ID, "verdict", AuthenticityVerdictCounterfeit, "error", err)
	}
}

// readAuthenticitySuspiciousCount 从 Account.Extra 读取当前可疑计数（容忍数字/字符串类型）。
func readAuthenticitySuspiciousCount(account *Account) int {
	if account == nil || account.Extra == nil {
		return 0
	}
	v, ok := account.Extra[AccountExtraKeyAuthenticitySuspicious]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
