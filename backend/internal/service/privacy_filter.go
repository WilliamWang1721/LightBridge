package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

// PrivacyFilterModelFilter 复用与内容审计相同的 {type, models} 形态。
type PrivacyFilterModelFilter struct {
	Type   string   `json:"type"`
	Models []string `json:"models"`
}

// 隐私过滤应用对象（针对谁过滤）。
const (
	PrivacyFilterTargetAllUsers  = "all_users"     // 全部用户
	PrivacyFilterTargetPartial   = "partial_users" // 部分用户
	PrivacyFilterTargetAdminOnly = "admin_only"    // 仅管理员
)

// 隐私过滤渠道维度（在哪些渠道生效）。
const (
	PrivacyFilterChannelAll     = "all"     // 全部渠道
	PrivacyFilterChannelGroup   = "group"   // 按分组
	PrivacyFilterChannelChannel = "channel" // 按渠道
	PrivacyFilterChannelAccount = "account" // 按账号
)

// PrivacyFilterRule 一条管理员自定义正则脱敏规则。
type PrivacyFilterRule struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Enabled     bool   `json:"enabled"`
}

// PrivacyFilterConfig 隐私过滤配置（存于 Setting JSON）。
type PrivacyFilterConfig struct {
	Enabled        bool                     `json:"enabled"`
	FilterRequest  bool                     `json:"filter_request"`
	FilterResponse bool                     `json:"filter_response"`
	BuiltinRules   map[string]bool          `json:"builtin_rules"`
	CustomRules    []PrivacyFilterRule      `json:"custom_rules"`
	AllGroups      bool                     `json:"all_groups"`
	GroupIDs       []int64                  `json:"group_ids"`
	ModelFilter    PrivacyFilterModelFilter `json:"model_filter"`
	// 应用对象（针对谁过滤）
	TargetScope   string  `json:"target_scope"`
	TargetUserIDs []int64 `json:"target_user_ids"`
	// 渠道维度
	ChannelScope string  `json:"channel_scope"`
	ChannelIDs   []int64 `json:"channel_ids"`
	AccountIDs   []int64 `json:"account_ids"`
}

// PrivacyFilterConfigView 返回给前端的配置视图（含内置规则 ID 列表，便于渲染）。
type PrivacyFilterConfigView struct {
	Enabled        bool                     `json:"enabled"`
	FilterRequest  bool                     `json:"filter_request"`
	FilterResponse bool                     `json:"filter_response"`
	BuiltinRules   map[string]bool          `json:"builtin_rules"`
	BuiltinRuleIDs []string                 `json:"builtin_rule_ids"`
	CustomRules    []PrivacyFilterRule      `json:"custom_rules"`
	AllGroups      bool                     `json:"all_groups"`
	GroupIDs       []int64                  `json:"group_ids"`
	ModelFilter    PrivacyFilterModelFilter `json:"model_filter"`
	TargetScope    string                   `json:"target_scope"`
	TargetUserIDs  []int64                  `json:"target_user_ids"`
	ChannelScope   string                   `json:"channel_scope"`
	ChannelIDs     []int64                  `json:"channel_ids"`
	AccountIDs     []int64                  `json:"account_ids"`
}

// UpdatePrivacyFilterConfigInput 部分更新输入（指针表示"未提供则不变"）。
type UpdatePrivacyFilterConfigInput struct {
	Enabled        *bool                     `json:"enabled"`
	FilterRequest  *bool                     `json:"filter_request"`
	FilterResponse *bool                     `json:"filter_response"`
	BuiltinRules   *map[string]bool          `json:"builtin_rules"`
	CustomRules    *[]PrivacyFilterRule      `json:"custom_rules"`
	AllGroups      *bool                     `json:"all_groups"`
	GroupIDs       *[]int64                  `json:"group_ids"`
	ModelFilter    *PrivacyFilterModelFilter `json:"model_filter"`
	TargetScope    *string                   `json:"target_scope"`
	TargetUserIDs  *[]int64                  `json:"target_user_ids"`
	ChannelScope   *string                   `json:"channel_scope"`
	ChannelIDs     *[]int64                  `json:"channel_ids"`
	AccountIDs     *[]int64                  `json:"account_ids"`
}

// PrivacyRedactor 是一次请求/响应内复用的脱敏器快照（持有已编译规则）。
type PrivacyRedactor struct {
	rules []privacyCompiledRule
}

// Redact 对单段文本脱敏，返回（脱敏后文本, 是否改写）。
func (r *PrivacyRedactor) Redact(text string) (string, bool) {
	if r == nil {
		return text, false
	}
	return applyPrivacyRules(r.rules, text)
}

// HasRules 报告该脱敏器是否含有任何启用规则。
func (r *PrivacyRedactor) HasRules() bool {
	return r != nil && len(r.rules) > 0
}

// PrivacyFilterService 管理隐私过滤配置并提供脱敏能力。
type PrivacyFilterService struct {
	settingRepo SettingRepository
	groupRepo   GroupRepository

	cacheMu   sync.Mutex
	cacheSig  string
	cacheRule []privacyCompiledRule
}

// NewPrivacyFilterService 构造隐私过滤服务。
func NewPrivacyFilterService(settingRepo SettingRepository, groupRepo GroupRepository) *PrivacyFilterService {
	return &PrivacyFilterService{settingRepo: settingRepo, groupRepo: groupRepo}
}

func defaultPrivacyFilterConfig() *PrivacyFilterConfig {
	builtins := make(map[string]bool, len(privacyFilterBuiltinRules))
	for _, id := range PrivacyFilterBuiltinIDs() {
		builtins[id] = true
	}
	return &PrivacyFilterConfig{
		Enabled:        false,
		FilterRequest:  true,
		FilterResponse: true,
		BuiltinRules:   builtins,
		CustomRules:    []PrivacyFilterRule{},
		AllGroups:      true,
		GroupIDs:       []int64{},
		ModelFilter:    PrivacyFilterModelFilter{Type: ContentModerationModelFilterAll, Models: []string{}},
		TargetScope:    PrivacyFilterTargetAllUsers,
		TargetUserIDs:  []int64{},
		ChannelScope:   PrivacyFilterChannelAll,
		ChannelIDs:     []int64{},
		AccountIDs:     []int64{},
	}
}

// GetConfig 返回当前配置视图。
func (s *PrivacyFilterService) GetConfig(ctx context.Context) (*PrivacyFilterConfigView, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s.configView(cfg), nil
}

// UpdateConfig 部分更新并持久化配置。
func (s *PrivacyFilterService) UpdateConfig(ctx context.Context, input UpdatePrivacyFilterConfigInput) (*PrivacyFilterConfigView, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.FilterRequest != nil {
		cfg.FilterRequest = *input.FilterRequest
	}
	if input.FilterResponse != nil {
		cfg.FilterResponse = *input.FilterResponse
	}
	if input.BuiltinRules != nil {
		cfg.BuiltinRules = *input.BuiltinRules
	}
	if input.CustomRules != nil {
		cfg.CustomRules = *input.CustomRules
	}
	if input.AllGroups != nil {
		cfg.AllGroups = *input.AllGroups
	}
	if input.GroupIDs != nil {
		cfg.GroupIDs = normalizeInt64IDs(*input.GroupIDs)
	}
	if input.ModelFilter != nil {
		cfg.ModelFilter = *input.ModelFilter
	}
	if input.TargetScope != nil {
		cfg.TargetScope = *input.TargetScope
	}
	if input.TargetUserIDs != nil {
		cfg.TargetUserIDs = normalizeInt64IDs(*input.TargetUserIDs)
	}
	if input.ChannelScope != nil {
		cfg.ChannelScope = *input.ChannelScope
	}
	if input.ChannelIDs != nil {
		cfg.ChannelIDs = normalizeInt64IDs(*input.ChannelIDs)
	}
	if input.AccountIDs != nil {
		cfg.AccountIDs = normalizeInt64IDs(*input.AccountIDs)
	}
	if err := s.validateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	cfg.normalize()
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal privacy filter config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyPrivacyFilterConfig, string(raw)); err != nil {
		return nil, fmt.Errorf("save privacy filter config: %w", err)
	}
	return s.configView(cfg), nil
}

func (s *PrivacyFilterService) loadConfig(ctx context.Context) (*PrivacyFilterConfig, error) {
	cfg := defaultPrivacyFilterConfig()
	if s == nil || s.settingRepo == nil {
		cfg.normalize()
		return cfg, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyPrivacyFilterConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			cfg.normalize()
			return cfg, nil
		}
		return nil, fmt.Errorf("get privacy filter config: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		cfg.normalize()
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, infraerrors.BadRequest("INVALID_PRIVACY_FILTER_CONFIG", "隐私过滤配置不是有效 JSON")
	}
	cfg.normalize()
	return cfg, nil
}

func (s *PrivacyFilterService) validateConfig(ctx context.Context, cfg *PrivacyFilterConfig) error {
	if cfg == nil {
		return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_CONFIG", "隐私过滤配置不能为空")
	}
	if len(cfg.CustomRules) > maxPrivacyFilterCustomRules {
		return infraerrors.BadRequest("TOO_MANY_PRIVACY_FILTER_RULES", fmt.Sprintf("自定义规则不能超过 %d 条", maxPrivacyFilterCustomRules))
	}
	for i, rule := range cfg.CustomRules {
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_RULE", fmt.Sprintf("第 %d 条自定义规则的正则不能为空", i+1))
		}
		if len([]rune(pattern)) > maxPrivacyFilterPatternRunes {
			return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_RULE", fmt.Sprintf("第 %d 条自定义规则的正则过长", i+1))
		}
		if _, err := compilePrivacyPattern(pattern); err != nil {
			return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_RULE", fmt.Sprintf("第 %d 条自定义规则的正则无效: %s", i+1, err.Error()))
		}
	}
	if cfg.ModelFilter.Type != ContentModerationModelFilterAll && len(cfg.ModelFilter.Models) == 0 {
		return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_MODEL_FILTER", "指定或排除模型时至少需要配置 1 个模型")
	}
	if !cfg.AllGroups && len(cfg.GroupIDs) > 0 && s.groupRepo != nil {
		for _, groupID := range cfg.GroupIDs {
			if _, err := s.groupRepo.GetByIDLite(ctx, groupID); err != nil {
				return infraerrors.BadRequest("INVALID_PRIVACY_FILTER_GROUP", fmt.Sprintf("分组不存在: %d", groupID))
			}
		}
	}
	return nil
}

func (cfg *PrivacyFilterConfig) normalize() {
	if cfg.BuiltinRules == nil {
		cfg.BuiltinRules = map[string]bool{}
	}
	for _, id := range PrivacyFilterBuiltinIDs() {
		if _, ok := cfg.BuiltinRules[id]; !ok {
			cfg.BuiltinRules[id] = true
		}
	}
	// 丢弃未知的内置 ID。
	for id := range cfg.BuiltinRules {
		if privacyFilterBuiltinCompiled[id] == nil {
			delete(cfg.BuiltinRules, id)
		}
	}
	cfg.CustomRules = normalizePrivacyCustomRules(cfg.CustomRules)
	cfg.GroupIDs = normalizeInt64IDs(cfg.GroupIDs)
	cfg.ModelFilter = PrivacyFilterModelFilter{
		Type:   normalizeContentModerationModelFilterType(cfg.ModelFilter.Type),
		Models: normalizeContentModerationModelNames(cfg.ModelFilter.Models),
	}
	if cfg.ModelFilter.Type == ContentModerationModelFilterAll {
		cfg.ModelFilter.Models = []string{}
	}
	cfg.TargetScope = normalizePrivacyTargetScope(cfg.TargetScope)
	cfg.ChannelScope = normalizePrivacyChannelScope(cfg.ChannelScope)
	cfg.TargetUserIDs = normalizeInt64IDs(cfg.TargetUserIDs)
	cfg.ChannelIDs = normalizeInt64IDs(cfg.ChannelIDs)
	cfg.AccountIDs = normalizeInt64IDs(cfg.AccountIDs)
}

func normalizePrivacyTargetScope(s string) string {
	switch s {
	case PrivacyFilterTargetPartial, PrivacyFilterTargetAdminOnly:
		return s
	default:
		return PrivacyFilterTargetAllUsers
	}
}

func normalizePrivacyChannelScope(s string) string {
	switch s {
	case PrivacyFilterChannelGroup, PrivacyFilterChannelChannel, PrivacyFilterChannelAccount:
		return s
	default:
		return PrivacyFilterChannelAll
	}
}

func normalizePrivacyCustomRules(rules []PrivacyFilterRule) []PrivacyFilterRule {
	if len(rules) == 0 {
		return []PrivacyFilterRule{}
	}
	out := make([]PrivacyFilterRule, 0, len(rules))
	for _, rule := range rules {
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		out = append(out, PrivacyFilterRule{
			Name:        trimRunes(strings.TrimSpace(rule.Name), maxPrivacyFilterRuleNameRunes),
			Pattern:     trimRunes(pattern, maxPrivacyFilterPatternRunes),
			Replacement: trimRunes(rule.Replacement, maxPrivacyFilterReplaceRunes),
			Enabled:     rule.Enabled,
		})
		if len(out) >= maxPrivacyFilterCustomRules {
			break
		}
	}
	return out
}

func (s *PrivacyFilterService) configView(cfg *PrivacyFilterConfig) *PrivacyFilterConfigView {
	builtins := make(map[string]bool, len(cfg.BuiltinRules))
	for k, v := range cfg.BuiltinRules {
		builtins[k] = v
	}
	return &PrivacyFilterConfigView{
		Enabled:        cfg.Enabled,
		FilterRequest:  cfg.FilterRequest,
		FilterResponse: cfg.FilterResponse,
		BuiltinRules:   builtins,
		BuiltinRuleIDs: PrivacyFilterBuiltinIDs(),
		CustomRules:    append([]PrivacyFilterRule(nil), cfg.CustomRules...),
		AllGroups:      cfg.AllGroups,
		GroupIDs:       append([]int64(nil), cfg.GroupIDs...),
		ModelFilter:    PrivacyFilterModelFilter{Type: cfg.ModelFilter.Type, Models: append([]string(nil), cfg.ModelFilter.Models...)},
		TargetScope:    cfg.TargetScope,
		TargetUserIDs:  append([]int64(nil), cfg.TargetUserIDs...),
		ChannelScope:   cfg.ChannelScope,
		ChannelIDs:     append([]int64(nil), cfg.ChannelIDs...),
		AccountIDs:     append([]int64(nil), cfg.AccountIDs...),
	}
}

// isFeatureEnabled 读取总开关 SettingKeyPrivacyFilterEnabled。
func (s *PrivacyFilterService) isFeatureEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyPrivacyFilterEnabled)
	if err != nil {
		return false
	}
	return raw == "true"
}

func (cfg *PrivacyFilterConfig) includesGroup(groupID *int64) bool {
	if cfg.AllGroups {
		return true
	}
	if groupID == nil {
		return false
	}
	for _, id := range cfg.GroupIDs {
		if id == *groupID {
			return true
		}
	}
	return false
}

func (cfg *PrivacyFilterConfig) includesModel(model string) bool {
	switch cfg.ModelFilter.Type {
	case ContentModerationModelFilterInclude:
		return contentModerationModelListContains(cfg.ModelFilter.Models, model)
	case ContentModerationModelFilterExclude:
		return !contentModerationModelListContains(cfg.ModelFilter.Models, model)
	default:
		return true
	}
}

// redactorFromConfig 编译（带缓存）配置对应的脱敏器。
func (s *PrivacyFilterService) redactorFromConfig(cfg *PrivacyFilterConfig) *PrivacyRedactor {
	sig := privacyRulesSignature(cfg)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if sig != s.cacheSig || s.cacheRule == nil {
		s.cacheRule = compilePrivacyRules(cfg.BuiltinRules, cfg.CustomRules)
		s.cacheSig = sig
	}
	return &PrivacyRedactor{rules: s.cacheRule}
}

// RequestRedactor 在请求侧脱敏开启且作用域命中时返回脱敏器，否则返回 nil。
func (s *PrivacyFilterService) RequestRedactor(ctx context.Context, groupID *int64, model string) *PrivacyRedactor {
	if s == nil || !s.isFeatureEnabled(ctx) {
		return nil
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil || !cfg.Enabled || !cfg.FilterRequest {
		return nil
	}
	if !cfg.includesGroup(groupID) || !cfg.includesModel(model) {
		return nil
	}
	r := s.redactorFromConfig(cfg)
	if !r.HasRules() {
		return nil
	}
	return r
}

// ResponseRedactor 在响应侧脱敏开启且分组命中时返回脱敏器，否则返回 nil。
// 响应侧仅按分组作用域过滤（模型级过滤仅作用于请求侧）。
func (s *PrivacyFilterService) ResponseRedactor(ctx context.Context, groupID *int64) *PrivacyRedactor {
	if s == nil || !s.isFeatureEnabled(ctx) {
		return nil
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil || !cfg.Enabled || !cfg.FilterResponse {
		return nil
	}
	if !cfg.includesGroup(groupID) {
		return nil
	}
	r := s.redactorFromConfig(cfg)
	if !r.HasRules() {
		return nil
	}
	return r
}

// ResponseFilterEnabled 报告响应侧脱敏总体是否开启（用于中间件快速短路）。
func (s *PrivacyFilterService) ResponseFilterEnabled(ctx context.Context) bool {
	if s == nil || !s.isFeatureEnabled(ctx) {
		return false
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return false
	}
	return cfg.Enabled && cfg.FilterResponse
}

// RedactRequestBody 按协议对请求体脱敏。redactor 为 nil 时原样返回。
func (s *PrivacyFilterService) RedactRequestBody(protocol string, body []byte, redactor *PrivacyRedactor) []byte {
	if redactor == nil || !redactor.HasRules() {
		return body
	}
	return RewritePrivacyFilterBody(protocol, body, redactor.Redact)
}

func privacyRulesSignature(cfg *PrivacyFilterConfig) string {
	var b strings.Builder
	ids := make([]string, 0, len(cfg.BuiltinRules))
	for id := range cfg.BuiltinRules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		_, _ = b.WriteString(id)
		_ = b.WriteByte('=')
		_, _ = b.WriteString(strconv.FormatBool(cfg.BuiltinRules[id]))
		_ = b.WriteByte(';')
	}
	_, _ = b.WriteString("|custom|")
	for _, r := range cfg.CustomRules {
		if !r.Enabled {
			continue
		}
		_, _ = b.WriteString(r.Pattern)
		_ = b.WriteByte('\x00')
		_, _ = b.WriteString(r.Replacement)
		_ = b.WriteByte('\n')
	}
	return b.String()
}
