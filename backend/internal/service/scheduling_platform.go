package service

import (
	"context"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/ctxkey"
)

// 本文件集中封装 Gemini / Antigravity 合并、以及 Custom Provider 引入后的「调度平台」语义。
//
// 背景一（Antigravity）：Antigravity 账号以 platform="gemini" + sub_platform="antigravity"
// 存储；调度别名仍可能是 "antigravity"，见 Account.IsAntigravity / EffectivePlatform。
//
// 背景二（Custom）：Custom 账号以 platform="custom" + extra["protocol"] 存储，可加入
// **任意（非 antigravity）分组**。router 模式下，调度层不再把账号 protocol 与入站协议
// 做强匹配；只要求账号拥有可路由的消息协议能力，最终转换链由 ProtocolRouter 在转发前决定。
// 只有 passthrough / full_passthrough 模式保留请求级协议匹配。因此 Custom 的调度分两级：
//   - 候选级（与 endpoint 无关）：Custom 账号进入其所属分组的候选集（见 accountServesSchedulingPlatform）。
//   - 请求级（与 endpoint 相关）：仅对非消息协议、passthrough、full_passthrough 执行协议过滤
//     （见 filterAccountsByRequestProtocol）。请求级过滤必须在取得候选列表之后做，因为调度快照
//     按 (group, platform, mode) 缓存、跨入站 endpoint 共享。
//
// 调度语义总表（候选级）：
//   - platform=="antigravity"            → 仅 Antigravity 账号（Custom 一律排除）。
//   - platform=="gemini"  且 useMixed    → 纯 Gemini + 启用 mixed 的 Antigravity + 全部 Custom。
//   - platform=="gemini"  且 !useMixed   → 纯 Gemini + 全部 Custom（Antigravity 排除）。
//   - platform=="anthropic" 且 useMixed  → 纯 Anthropic + 启用 mixed 的 Antigravity + 全部 Custom。
//   - platform=="anthropic" 且 !useMixed → 纯 Anthropic + 全部 Custom。
//   - platform=="openai"                 → 纯 OpenAI + 全部 Custom。
//   - platform=="grok"                   → 仅 Grok 账号（Custom 一律排除）。
//   - platform=="custom"                 → 仅 Custom 账号。
//   - 其他                                → account.Platform == platform + 全部 Custom。

// schedulingQueryPlatforms 返回为「服务于给定调度目标平台（别名）」需要从数据库查询的
// platform 列表。Antigravity 账号位于 gemini 平台下；Custom 账号位于 custom 平台下，
// 凡可能涉及 Custom 的查询都需追加 PlatformCustom（antigravity 专用查询除外）。
func schedulingQueryPlatforms(platform string, useMixed bool) []string {
	switch platform {
	case PlatformAntigravity:
		// Antigravity 专用：账号在 gemini 平台下，且不混入 Custom。
		return []string{PlatformGemini}
	case PlatformGrok:
		return []string{PlatformGrok}
	case PlatformAnthropic:
		if useMixed {
			return []string{PlatformAnthropic, PlatformGemini, PlatformCustom}
		}
		return []string{PlatformAnthropic, PlatformCustom}
	case PlatformGemini:
		return []string{PlatformGemini, PlatformCustom}
	case PlatformCustom:
		return []string{PlatformCustom}
	default:
		return []string{platform, PlatformCustom}
	}
}

func normalizeOpenAICompatiblePlatform(platform string) string {
	if strings.TrimSpace(platform) == PlatformGrok {
		return PlatformGrok
	}
	return PlatformOpenAI
}

func schedulingQueryPlatformsForRequest(ctx context.Context, platform string, useMixed bool) []string {
	inbound := InboundProtocolFromContext(ctx)
	if inbound == "" || !IsMessageProtocol(inbound) {
		if useMixed {
			return schedulingQueryPlatforms(platform, useMixed)
		}
		if platform == PlatformAntigravity {
			return []string{PlatformGemini}
		}
		return []string{platform}
	}
	if platform == PlatformAntigravity {
		return []string{PlatformGemini}
	}
	if platform == PlatformGrok {
		return []string{PlatformGrok}
	}
	return []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformGrok, PlatformCustom}
}

func PlatformForInboundProtocol(protocol string) string {
	switch protocol {
	case CustomProtocolAnthropicMessages:
		return PlatformAnthropic
	case CustomProtocolOpenAIResponses, CustomProtocolOpenAIChatCompletions, CustomProtocolOpenAIEmbeddings:
		return PlatformOpenAI
	case CustomProtocolGemini:
		return PlatformGemini
	default:
		return ""
	}
}

func PlatformForRequest(ctx context.Context, fallback string) string {
	if ctx == nil {
		return fallback
	}
	if forcePlatform, ok := ctx.Value(ctxkey.ForcePlatform).(string); ok && forcePlatform != "" {
		return forcePlatform
	}
	if platform := PlatformForInboundProtocol(InboundProtocolFromContext(ctx)); platform != "" {
		return platform
	}
	return fallback
}

// accountServesSchedulingPlatform 判断账号是否可进入给定调度目标平台（别名）的**候选集**。
//
// 这是「候选级」判定（与入站 endpoint 无关）。Custom 账号的 protocol 匹配在请求级由
// filterAccountsByRequestProtocol 完成。platform 为调度别名；useMixed 表示混合调度
// （仅 anthropic / gemini 且非强制平台时为 true）。
func accountServesSchedulingPlatform(a *Account, platform string, useMixed bool) bool {
	if a == nil {
		return false
	}
	// Antigravity 专用调度：只接受 Antigravity 账号（Custom 等一律排除）。
	if platform == PlatformAntigravity {
		return a.IsAntigravity()
	}
	if platform == PlatformGrok {
		return a.IsGrok()
	}
	// Custom 账号不受分组类型限制：可进入任意（非 antigravity）平台的候选集。
	if a.IsCustom() {
		return true
	}
	if a.IsAntigravity() {
		return useMixed &&
			(platform == PlatformAnthropic || platform == PlatformGemini) &&
			a.IsMixedSchedulingEnabled()
	}
	return a.Platform == platform
}

func accountServesRequestPlatform(ctx context.Context, a *Account, platform string, useMixed bool) bool {
	if a == nil {
		return false
	}
	if forcePlatform, ok := ctx.Value(ctxkey.ForcePlatform).(string); ok && forcePlatform != "" {
		return accountServesSchedulingPlatform(a, platform, useMixed)
	}
	if InboundProtocolFromContext(ctx) != "" {
		return true
	}
	return accountServesSchedulingPlatform(a, platform, useMixed)
}

// requiredProtocolFromContext 读取 handler 依入站 endpoint 注入的请求级入站协议。
// 为空表示当前上下文无协议约束（如非网关请求/测试），此时不做协议过滤。
func requiredProtocolFromContext(ctx context.Context) string {
	return InboundProtocolFromContext(ctx)
}

// filterAccountsByRequestProtocol 按请求级入站协议和账号 relay_mode 过滤候选账号。
// router 模式的消息协议请求不做入站/上游协议强匹配；passthrough / full_passthrough
// 以及非消息协议请求仍使用 ProtocolRouter 的严格判定。requiredProtocol 为空时原样返回。
// 返回新切片，不修改入参底层数组（避免污染快照缓存）。
func filterAccountsByRequestProtocol(ctx context.Context, accounts []Account) []Account {
	required := requiredProtocolFromContext(ctx)
	if required == "" {
		return accounts
	}
	out := make([]Account, 0, len(accounts))
	for i := range accounts {
		if !accountMatchesRequestProtocol(ctx, &accounts[i]) {
			continue
		}
		out = append(out, accounts[i])
	}
	return out
}

// accountMatchesRequestProtocol 判断单个账号是否满足请求级协议约束（用于 sticky session
// 等按 ID 取账号、绕过候选列表过滤的路径）。无协议约束时恒为真。
func accountMatchesRequestProtocol(ctx context.Context, a *Account) bool {
	required := requiredProtocolFromContext(ctx)
	if required == "" || a == nil {
		return true
	}
	if IsMessageProtocol(required) && a.RelayMode() == RelayModeRouter {
		return accountSupportsAnyMessageProtocol(a)
	}
	_, ok := ProtocolRouteDecisionForAccountProtocols(required, a)
	return ok
}

func accountSupportsAnyMessageProtocol(a *Account) bool {
	if a == nil {
		return false
	}
	for _, proto := range a.SupportedTargetProtocols() {
		if IsMessageProtocol(proto) {
			return true
		}
	}
	return false
}
