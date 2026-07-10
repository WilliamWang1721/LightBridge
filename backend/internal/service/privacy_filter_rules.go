package service

import (
	"regexp"
	"strings"
)

// 隐私过滤内置规则 ID。前端用这些 ID 做 i18n 文案映射。
const (
	PrivacyFilterBuiltinEmail      = "email"
	PrivacyFilterBuiltinCNPhone    = "cn_phone"
	PrivacyFilterBuiltinIDCard     = "id_card"
	PrivacyFilterBuiltinBankCard   = "bank_card"
	PrivacyFilterBuiltinIPv4       = "ipv4"
	PrivacyFilterBuiltinIPv6       = "ipv6"
	PrivacyFilterBuiltinSecret     = "secret"
	PrivacyFilterBuiltinJWT        = "jwt"
	PrivacyFilterBuiltinPrivateKey = "private_key"
	PrivacyFilterBuiltinAWSKey     = "aws_key"
	PrivacyFilterBuiltinGitHubPAT  = "github_pat"
	PrivacyFilterBuiltinSlackToken = "slack_token"
	PrivacyFilterBuiltinCreditCard = "credit_card"
	PrivacyFilterBuiltinCNLicense  = "cn_license"
	PrivacyFilterBuiltinURLQuery   = "url_query"
)

const (
	maxPrivacyFilterCustomRules    = 200
	maxPrivacyFilterRuleNameRunes  = 80
	maxPrivacyFilterPatternRunes   = 500
	maxPrivacyFilterReplaceRunes   = 80
	maxPrivacyFilterRedactRunes    = 200000 // 单段文本脱敏上限，超出截断处理避免极端正则开销
	defaultPrivacyFilterReplaceFmt = "[REDACTED]"
)

// privacyFilterBuiltin 描述一条内置 PII 规则。
type privacyFilterBuiltin struct {
	ID          string
	Pattern     string
	Replacement string
}

// privacyFilterBuiltinRules 内置规则的固定应用顺序。
// 注意顺序：id_card（18 位）需先于 bank_card（16-19 位）应用，
// 否则 18 位身份证会被当作银行卡处理（两者都脱敏，仅占位符不同）。
var privacyFilterBuiltinRules = []privacyFilterBuiltin{
	{ID: PrivacyFilterBuiltinEmail, Pattern: `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, Replacement: "[EMAIL]"},
	{ID: PrivacyFilterBuiltinSecret, Pattern: `\b(?:sk|pk|rk)-[A-Za-z0-9_\-]{16,}\b`, Replacement: "[SECRET]"},
	{ID: PrivacyFilterBuiltinIDCard, Pattern: `\b\d{17}[\dXx]\b`, Replacement: "[ID_CARD]"},
	{ID: PrivacyFilterBuiltinBankCard, Pattern: `\b\d{16,19}\b`, Replacement: "[BANK_CARD]"},
	{ID: PrivacyFilterBuiltinCNPhone, Pattern: `\b1[3-9]\d{9}\b`, Replacement: "[PHONE]"},
	{ID: PrivacyFilterBuiltinIPv4, Pattern: `\b(?:\d{1,3}\.){3}\d{1,3}\b`, Replacement: "[IP]"},
	// IPv6（含完整与压缩形式）
	{ID: PrivacyFilterBuiltinIPv6, Pattern: `\b(?:[0-9a-fA-F]{1,4}:){2,7}[0-9a-fA-F]{1,4}\b`, Replacement: "[IPV6]"},
	// JWT（三段式，每段 base64url，以点分隔）
	{ID: PrivacyFilterBuiltinJWT, Pattern: `\beyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\b`, Replacement: "[JWT]"},
	// 私钥（PEM 头：RSA / EC / OPENSSH / PRIVATE KEY）
	{ID: PrivacyFilterBuiltinPrivateKey, Pattern: `-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----`, Replacement: "[PRIVATE_KEY]"},
	// AWS Access Key ID（AKIA 开头 + 16 位）
	{ID: PrivacyFilterBuiltinAWSKey, Pattern: `\bAKIA[0-9A-Z]{16}\b`, Replacement: "[AWS_KEY]"},
	// GitHub Personal Access Token（ghp_/gho_/ghu_/ghs_/ghr_ 前缀）
	{ID: PrivacyFilterBuiltinGitHubPAT, Pattern: `\bgh[posur]_[A-Za-z0-9]{36,}\b`, Replacement: "[GITHUB_PAT]"},
	// Slack Token（xox[baprs]- 前缀）
	{ID: PrivacyFilterBuiltinSlackToken, Pattern: `\bxox[baprs]-[A-Za-z0-9\-]{10,}\b`, Replacement: "[SLACK_TOKEN]"},
	// 信用卡号（Visa/MasterCard/Amex 常见长度）
	{ID: PrivacyFilterBuiltinCreditCard, Pattern: `\b(?:\d[ -]*?){13,16}\b`, Replacement: "[CREDIT_CARD]"},
	// 中国车牌号
	{ID: PrivacyFilterBuiltinCNLicense, Pattern: `\b[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤川青藏琼宁][A-Z][A-HJ-NP-Z0-9]{4,5}[A-HJ-NP-Z0-9]\b`, Replacement: "[LICENSE_PLATE]"},
	// URL 查询参数中的敏感键（token/key/secret/password/passwd/pwd 等）
	{ID: PrivacyFilterBuiltinURLQuery, Pattern: `(?i)(?:token|secret|api[_-]?key|access[_-]?token|refresh[_-]?token|password|passwd|pwd|authorization)=[^&\s\"'<>]+`, Replacement: "[REDACTED_QUERY]"},
}

// privacyFilterBuiltinCompiled 进程级别只编译一次的内置正则。
var privacyFilterBuiltinCompiled = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(privacyFilterBuiltinRules))
	for _, r := range privacyFilterBuiltinRules {
		out[r.ID] = regexp.MustCompile(r.Pattern)
	}
	return out
}()

// PrivacyFilterBuiltinIDs 返回内置规则 ID（按应用顺序），供配置默认值与前端展示使用。
func PrivacyFilterBuiltinIDs() []string {
	out := make([]string, 0, len(privacyFilterBuiltinRules))
	for _, r := range privacyFilterBuiltinRules {
		out = append(out, r.ID)
	}
	return out
}

// privacyCompiledRule 是一条已编译、可直接应用的脱敏规则。
type privacyCompiledRule struct {
	re          *regexp.Regexp
	replacement string
}

// compilePrivacyPattern 编译单条自定义正则，供配置校验复用。
func compilePrivacyPattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}

// applyPrivacyRules 依次对 text 应用所有规则，返回脱敏后的文本与是否发生改写。
func applyPrivacyRules(rules []privacyCompiledRule, text string) (string, bool) {
	if text == "" || len(rules) == 0 {
		return text, false
	}
	if len([]rune(text)) > maxPrivacyFilterRedactRunes {
		// 超长文本仅对前缀脱敏，剩余原样拼接，避免极端正则回溯开销。
		runes := []rune(text)
		head := string(runes[:maxPrivacyFilterRedactRunes])
		tail := string(runes[maxPrivacyFilterRedactRunes:])
		out, changed := applyPrivacyRules(rules, head)
		return out + tail, changed
	}
	out := text
	for _, rule := range rules {
		if rule.re == nil {
			continue
		}
		out = rule.re.ReplaceAllString(out, rule.replacement)
	}
	return out, out != text
}

// compilePrivacyRules 根据配置编译出有序的规则列表：先内置（按 enabled），后自定义。
func compilePrivacyRules(builtinEnabled map[string]bool, custom []PrivacyFilterRule) []privacyCompiledRule {
	rules := make([]privacyCompiledRule, 0, len(privacyFilterBuiltinRules)+len(custom))
	for _, b := range privacyFilterBuiltinRules {
		if enabled, ok := builtinEnabled[b.ID]; ok && !enabled {
			continue
		}
		if re := privacyFilterBuiltinCompiled[b.ID]; re != nil {
			rules = append(rules, privacyCompiledRule{re: re, replacement: b.Replacement})
		}
	}
	for _, r := range custom {
		if !r.Enabled {
			continue
		}
		pattern := strings.TrimSpace(r.Pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		replacement := r.Replacement
		if strings.TrimSpace(replacement) == "" {
			replacement = defaultPrivacyFilterReplaceFmt
		}
		rules = append(rules, privacyCompiledRule{re: re, replacement: replacement})
	}
	return rules
}
