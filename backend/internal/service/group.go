package service

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/domain"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/timezone"
)

type OpenAIMessagesDispatchModelConfig = domain.OpenAIMessagesDispatchModelConfig
type GroupModelsListConfig = domain.GroupModelsListConfig

type Group struct {
	ID                 int64
	Name               string
	Description        string
	Icon               string
	Color              string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakStart          string
	PeakEnd            string
	PeakRateMultiplier float64
	IsExclusive        bool
	Status             string
	Hydrated           bool // indicates the group was loaded from a trusted repository source

	SubscriptionType    string
	DailyLimitUSD       *float64
	WeeklyLimitUSD      *float64
	MonthlyLimitUSD     *float64
	DefaultValidityDays int

	// 图片生成计费配置（antigravity 和 gemini 平台使用）
	AllowImageGeneration bool
	ImageRateIndependent bool
	ImageRateMultiplier  float64
	ImagePrice1K         *float64
	ImagePrice2K         *float64
	ImagePrice4K         *float64

	// Claude Code 客户端限制
	ClaudeCodeOnly  bool
	FallbackGroupID *int64
	// 无效请求兜底分组（仅 anthropic 平台使用）
	FallbackGroupIDOnInvalidRequest *int64

	// 模型路由配置
	// key: 模型匹配模式（支持 * 通配符，如 "claude-opus-*"）
	// value: 优先账号 ID 列表
	ModelRouting        map[string][]int64
	ModelRoutingEnabled bool

	// MCP XML 协议注入开关
	MCPXMLInject bool

	// 分组排序
	SortOrder int

	// OpenAI Messages 调度配置（仅 openai 平台使用）
	AllowMessagesDispatch       bool
	RequireOAuthOnly            bool // 仅允许非 apikey 类型账号关联（OpenAI/Antigravity/Anthropic/Gemini）
	RequirePrivacySet           bool // 调度时仅允许 privacy 已成功设置的账号（OpenAI/Antigravity/Anthropic/Gemini）
	DefaultMappedModel          string
	MessagesDispatchModelConfig OpenAIMessagesDispatchModelConfig
	ModelsListConfig            GroupModelsListConfig

	// RPMLimit 分组级每分钟请求数上限（0 = 不限制）。
	// 一旦设置即接管该分组用户的限流（覆盖用户级 rpm_limit），可被 user-group rpm_override 进一步覆盖。
	RPMLimit int

	CreatedAt time.Time
	UpdatedAt time.Time

	AccountGroups           []AccountGroup
	AccountCount            int64
	ActiveAccountCount      int64
	RateLimitedAccountCount int64
	// UpstreamProtocols is derived from accounts currently bound to the group.
	// It describes the request protocols accepted by at least one bound account.
	UpstreamProtocols []string
	// UpstreamPlatforms is derived from the effective platforms of accounts
	// currently bound to the group. Unlike protocol, this distinguishes providers
	// that share a wire format, such as Grok and OpenAI Responses.
	UpstreamPlatforms []string
	// AvailableIngressProtocols is derived from each bound account's upstream
	// protocol and relay mode. Router accounts contribute all implemented message
	// ingress protocols; passthrough accounts contribute only their actual protocol.
	AvailableIngressProtocols []string
}

func (g *Group) IsActive() bool {
	return g.Status == StatusActive
}

func (g *Group) IsSubscriptionType() bool {
	return g.SubscriptionType == SubscriptionTypeSubscription
}

func AccountUpstreamProtocols(account *Account) []string {
	if account == nil {
		return nil
	}
	supported := account.SupportedTargetProtocols()
	protocols := make([]string, 0, len(supported))
	seen := make(map[string]struct{}, len(supported))
	for _, proto := range supported {
		proto = strings.TrimSpace(proto)
		if proto == "" {
			continue
		}
		if _, ok := seen[proto]; ok {
			continue
		}
		seen[proto] = struct{}{}
		protocols = append(protocols, proto)
	}
	sort.Strings(protocols)
	return protocols
}

var groupMessageIngressProtocols = []string{
	CustomProtocolAnthropicMessages,
	CustomProtocolOpenAIResponses,
	CustomProtocolOpenAIChatCompletions,
	CustomProtocolGemini,
}

// AccountAvailableIngressProtocols returns protocols that callers may use at
// LightBridge's public endpoints for this account. It deliberately differs from
// AccountUpstreamProtocols: Router can convert between implemented message
// protocols, while passthrough modes require an exact wire protocol match.
func AccountAvailableIngressProtocols(account *Account) []string {
	actual := AccountUpstreamProtocols(account)
	if account == nil || account.RelayMode() != RelayModeRouter {
		return actual
	}

	seen := make(map[string]struct{}, len(actual)+len(groupMessageIngressProtocols))
	out := make([]string, 0, len(actual)+len(groupMessageIngressProtocols))
	hasMessageUpstream := false
	for _, protocol := range actual {
		if IsMessageProtocol(protocol) {
			hasMessageUpstream = true
		}
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		out = append(out, protocol)
	}
	if hasMessageUpstream {
		for _, protocol := range groupMessageIngressProtocols {
			if _, ok := seen[protocol]; ok {
				continue
			}
			seen[protocol] = struct{}{}
			out = append(out, protocol)
		}
	}
	sort.Strings(out)
	return out
}

var groupColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

var allowedGroupIcons = map[string]struct{}{
	"":         {},
	"folder":   {},
	"server":   {},
	"cloud":    {},
	"bolt":     {},
	"shield":   {},
	"cube":     {},
	"terminal": {},
	"sparkles": {},
	"users":    {},
}

// NormalizeGroupAppearance validates and canonicalizes administrator-supplied
// presentation metadata. Empty values deliberately mean neutral defaults.
func NormalizeGroupAppearance(icon, color string) (string, string, error) {
	icon = strings.ToLower(strings.TrimSpace(icon))
	color = strings.TrimSpace(color)
	if _, ok := allowedGroupIcons[icon]; !ok {
		return "", "", fmt.Errorf("unsupported group icon %q", icon)
	}
	if color != "" {
		if !groupColorPattern.MatchString(color) {
			return "", "", errors.New("group color must be empty or use #RRGGBB")
		}
		color = strings.ToUpper(color)
	}
	return icon, color, nil
}

func NormalizeGroupUpstreamProtocolFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case CustomProtocolOpenAIResponses:
		return CustomProtocolOpenAIResponses
	case CustomProtocolOpenAIChatCompletions:
		return CustomProtocolOpenAIChatCompletions
	case CustomProtocolOpenAIEmbeddings:
		return CustomProtocolOpenAIEmbeddings
	case CustomProtocolAnthropicMessages:
		return CustomProtocolAnthropicMessages
	case CustomProtocolGemini:
		return CustomProtocolGemini
	default:
		return ""
	}
}

func (g *Group) HasDailyLimit() bool {
	return g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0
}

func (g *Group) HasWeeklyLimit() bool {
	return g.WeeklyLimitUSD != nil && *g.WeeklyLimitUSD > 0
}

func (g *Group) HasMonthlyLimit() bool {
	return g.MonthlyLimitUSD != nil && *g.MonthlyLimitUSD > 0
}

// GetImagePrice 根据 image_size 返回对应的图片生成价格
// 如果分组未配置价格，返回 nil（调用方应使用默认值）
func (g *Group) GetImagePrice(imageSize string) *float64 {
	switch imageSize {
	case "1K":
		return g.ImagePrice1K
	case "2K":
		return g.ImagePrice2K
	case "4K":
		return g.ImagePrice4K
	default:
		// 未知尺寸默认按 2K 计费
		return g.ImagePrice2K
	}
}

// IsGroupContextValid reports whether a group from context has the fields required for routing decisions.
func IsGroupContextValid(group *Group) bool {
	if group == nil {
		return false
	}
	if group.ID <= 0 {
		return false
	}
	if !group.Hydrated {
		return false
	}
	if group.Status == "" {
		return false
	}
	return true
}

// GetRoutingAccountIDs 根据请求模型获取路由账号 ID 列表
// 返回匹配的优先账号 ID 列表，如果没有匹配规则则返回 nil
func (g *Group) GetRoutingAccountIDs(requestedModel string) []int64 {
	if !g.ModelRoutingEnabled || len(g.ModelRouting) == 0 || requestedModel == "" {
		return nil
	}

	// 1. 精确匹配优先
	if accountIDs, ok := g.ModelRouting[requestedModel]; ok && len(accountIDs) > 0 {
		return accountIDs
	}

	// 2. 通配符匹配（前缀匹配）
	for pattern, accountIDs := range g.ModelRouting {
		if matchModelPattern(pattern, requestedModel) && len(accountIDs) > 0 {
			return accountIDs
		}
	}

	return nil
}

// matchModelPattern 检查模型是否匹配模式
// 支持 * 通配符，如 "claude-opus-*" 匹配 "claude-opus-4-20250514"
func matchModelPattern(pattern, model string) bool {
	if pattern == model {
		return true
	}

	// 处理 * 通配符（仅支持末尾通配符）
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}

	return false
}

// parseMinutes 把 "HH:MM" 解析为当日分钟数（0..1439），格式非法返回 (0,false)。
func parseMinutes(hhmm string) (int, bool) {
	colon := strings.IndexByte(hhmm, ':')
	if (colon != 1 && colon != 2) || len(hhmm)-colon-1 != 2 {
		return 0, false
	}
	h := 0
	for i := 0; i < colon; i++ {
		d := hhmm[i] - '0'
		if d > 9 {
			return 0, false
		}
		h = h*10 + int(d)
	}
	m1, m2 := hhmm[colon+1]-'0', hhmm[colon+2]-'0'
	if m1 > 9 || m2 > 9 {
		return 0, false
	}
	m := int(m1)*10 + int(m2)
	if h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// PeakMultiplierAt 返回指定时刻 now 的高峰因子。
// 未启用、非订阅分组、配置非法或非高峰时段时返回 1.0。
func (g *Group) PeakMultiplierAt(now time.Time) float64 {
	if g == nil || !g.IsSubscriptionType() || !g.PeakRateEnabled || g.PeakStart == "" || g.PeakEnd == "" {
		return 1.0
	}
	start, ok1 := parseMinutes(g.PeakStart)
	end, ok2 := parseMinutes(g.PeakEnd)
	if !ok1 || !ok2 || start >= end {
		return 1.0
	}
	t := now.In(timezone.Location())
	cur := t.Hour()*60 + t.Minute()
	if cur >= start && cur < end {
		return g.PeakRateMultiplier
	}
	return 1.0
}

func ValidatePeakRateConfig(subscriptionType string, enabled bool, start, end string, multiplier float64) error {
	if !enabled {
		return nil
	}
	if subscriptionType != SubscriptionTypeSubscription {
		return errors.New("高峰时段倍率仅支持订阅类型分组")
	}
	if start == "" || end == "" {
		return errors.New("peak_rate_enabled 为 true 时 peak_start 与 peak_end 必填")
	}
	st, okStart := parseMinutes(start)
	if !okStart {
		return fmt.Errorf("peak_start 格式应为 HH:MM，got %q", start)
	}
	en, okEnd := parseMinutes(end)
	if !okEnd {
		return fmt.Errorf("peak_end 格式应为 HH:MM，got %q", end)
	}
	if st >= en {
		return errors.New("peak_end 必须大于 peak_start（不支持跨天区间，如 22:00-02:00）")
	}
	if multiplier < 0 {
		return errors.New("peak_rate_multiplier 不能为负")
	}
	return nil
}

func NormalizePeakRateConfig(subscriptionType string, enabled bool, start, end string, multiplier float64) (bool, string, string, float64) {
	if subscriptionType != SubscriptionTypeSubscription {
		return false, "", "", 1.0
	}
	if !enabled {
		if _, ok := parseMinutes(start); !ok {
			start = ""
		}
		if _, ok := parseMinutes(end); !ok {
			end = ""
		}
		if multiplier < 0 {
			multiplier = 1.0
		}
	}
	return enabled, start, end, multiplier
}

func computePeakAwareMultipliers(apiKey *APIKey, base float64, now time.Time) (text, image float64) {
	image = resolveImageRateMultiplier(apiKey, base)
	peak := 1.0
	if apiKey != nil && apiKey.Group != nil {
		peak = apiKey.Group.PeakMultiplierAt(now)
	}
	text = base * peak
	return
}
