package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/claude"
	"github.com/gin-gonic/gin"
)

// ClaudeAuthenticityResult 是一次 Claude 模型真伪检测的结论。
//
// 原理：真 Anthropic 服务端会校验 assistant 历史 thinking block 的 signature。
// 我们构造一个带伪造 signature 的多轮请求发给该账号：
//   - 真 Claude → 校验失败，返回 "Invalid signature in thinking block" 类 400 → genuine
//   - 套壳/中转假冒 → 不认识 Anthropic 签名，不会报签名错 → counterfeit
//
// 伪造签名通常在计费前被拒，且 max_tokens=1 兜底，单次探针成本趋近于零。
type ClaudeAuthenticityResult struct {
	Verdict    string    `json:"verdict"`               // genuine / counterfeit / unknown
	Method     string    `json:"method"`                // probe
	CheckedAt  time.Time `json:"checked_at"`            // 检测时间
	Detail     string    `json:"detail,omitempty"`      // 人类可读说明（错误原因/状态码等）
	HTTPStatus int       `json:"http_status,omitempty"` // 上游返回的状态码（便于排障）
}

// ExtraMap 返回需要增量合并进 Account.Extra 的键值（key 级覆盖，不影响其它运行态键）。
func (r *ClaudeAuthenticityResult) ExtraMap() map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		AccountExtraKeyAuthenticityVerdict:   r.Verdict,
		AccountExtraKeyAuthenticityCheckedAt: r.CheckedAt.UTC().Format(time.RFC3339),
		AccountExtraKeyAuthenticityMethod:    r.Method,
		AccountExtraKeyAuthenticityDetail:    r.Detail,
	}
}

// fakeSignatureForProbe 是一个故意非法的 base64，用于触发真 Anthropic 的签名校验失败。
// 真 Claude 会拒绝它；不识别 signature 的假冒上游会忽略它并正常返回。
const fakeSignatureForProbe = "!!!not-a-valid-signature!!!"

// probeThinkingBudget 探针请求的 thinking budget_tokens。真模型会用它来开启 thinking。
const probeThinkingBudget = 1024

// ProbeClaudeAuthenticity 对一个 Claude/Anthropic 账号执行主动真伪探针。
//
// 它复用 testClaudeAccountConnection 的取凭证/构造请求/DoWithTLS 逻辑，
// 但 payload 是带伪造签名 thinking 块的多轮请求，且 stream=false 以便直接读取完整 JSON 错误体。
// 探针不会向 c 写 SSE，仅返回结论；调用方负责持久化到 Account.Extra。
//
// 适用账号类型：Anthropic OAuth / SetupToken / APIKey（含自定义 baseURL）。
// Bedrock / Vertex / 非 Anthropic 平台直接判 unknown（探针语义不适用，避免误伤）。
func (s *AccountTestService) ProbeClaudeAuthenticity(c *gin.Context, accountID int64) (*ClaudeAuthenticityResult, error) {
	ctx := c.Request.Context()

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// 非 Claude/Anthropic 平台不做真伪判定。
	if !account.IsOAuth() && account.Type != AccountTypeAPIKey && account.Type != AccountTypeServiceAccount {
		return &ClaudeAuthenticityResult{
			Verdict:   AuthenticityVerdictUnknown,
			Method:    AuthenticityMethodProbe,
			CheckedAt: time.Now(),
			Detail:    "authenticity probe only applies to Claude/Anthropic accounts",
		}, nil
	}
	// Bedrock / Vertex 走不同的请求签名体系，伪造 thinking 签名的探针语义不适用。
	if account.IsBedrock() {
		return &ClaudeAuthenticityResult{
			Verdict:   AuthenticityVerdictUnknown,
			Method:    AuthenticityMethodProbe,
			CheckedAt: time.Now(),
			Detail:    "bedrock accounts are not supported by the authenticity probe",
		}, nil
	}
	if account.Type == AccountTypeServiceAccount {
		return &ClaudeAuthenticityResult{
			Verdict:   AuthenticityVerdictUnknown,
			Method:    AuthenticityMethodProbe,
			CheckedAt: time.Now(),
			Detail:    "vertex service accounts are not supported by the authenticity probe",
		}, nil
	}

	return s.probeClaudeAuthenticity(ctx, c, account)
}

// probeClaudeAuthenticity 执行实际的探针请求。拆分出来便于单元测试。
func (s *AccountTestService) probeClaudeAuthenticity(ctx context.Context, c *gin.Context, account *Account) (*ClaudeAuthenticityResult, error) {
	// 取凭证与 API URL（与 testClaudeAccountConnection 完全一致）。
	var authToken string
	var useBearer bool
	var apiURL string

	if account.IsOAuth() {
		useBearer = true
		apiURL = testClaudeAPIURL
		authToken = account.GetCredential("access_token")
		if authToken == "" {
			return &ClaudeAuthenticityResult{
				Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
				CheckedAt: time.Now(), Detail: "no access token available",
			}, nil
		}
	} else if account.Type == AccountTypeAPIKey {
		useBearer = false
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return &ClaudeAuthenticityResult{
				Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
				CheckedAt: time.Now(), Detail: "no api key available",
			}, nil
		}
		baseURL := account.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return &ClaudeAuthenticityResult{
				Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
				CheckedAt: time.Now(), Detail: fmt.Sprintf("invalid base url: %s", err.Error()),
			}, nil
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages?beta=true"
	} else {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), Detail: fmt.Sprintf("unsupported account type: %s", account.Type),
		}, nil
	}

	// 测试模型（沿用连接测试的默认模型，API Key 账号同样应用模型映射）。
	testModelID := claude.DefaultTestModel
	if account.Type == AccountTypeAPIKey {
		testModelID = account.GetMappedModel(testModelID)
	}

	// 构造带伪造签名 thinking 块的多轮 payload。
	// 注意：stream=false，以便直接读取完整 JSON 错误体（探针不关心流式内容）。
	payload := map[string]any{
		"model": testModelID,
		"messages": []map[string]any{
			{"role": "user", "content": "hi"},
			{
				"role": "assistant",
				"content": []map[string]any{
					{
						"type":      "thinking",
						"thinking":  "probe",
						"signature": fakeSignatureForProbe,
					},
				},
			},
			{"role": "user", "content": "go on"},
		},
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": probeThinkingBudget,
		},
		"max_tokens": 1,
		"stream":     false,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), Detail: fmt.Sprintf("failed to create request: %s", err.Error()),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	for key, value := range claude.DefaultHeaders {
		req.Header.Set(key, value)
	}
	if useBearer {
		req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
		req.Header.Set("x-api-key", authToken)
	}

	// 解析代理（与连接测试一致；探针没有 group 上下文，传空 group）。
	proxyURL, err := s.resolveAccountProxyURL(ctx, account, account.Platform, nil)
	if err != nil {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), Detail: fmt.Sprintf("failed to resolve proxy: %s", err.Error()),
		}, nil
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), Detail: fmt.Sprintf("request failed: %s", err.Error()),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	// 403 表示账号被上游封禁：保持与连接测试一致的封禁标记，并判 unknown。
	if resp.StatusCode == http.StatusForbidden {
		_ = s.accountRepo.SetError(ctx, account.ID, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), HTTPStatus: resp.StatusCode,
			Detail: fmt.Sprintf("upstream forbidden (%d); account flagged error", resp.StatusCode),
		}, nil
	}

	// 核心判定：真 Claude 会对伪造签名返回 400 + signature 相关错误。
	// 用与整流器完全相同的检测函数，保证判定口径一致。
	if resp.StatusCode >= 400 && detectThinkingSignatureError(body) {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictGenuine, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), HTTPStatus: resp.StatusCode,
			Detail: fmt.Sprintf("upstream rejected forged thinking signature (%d): %s",
				resp.StatusCode, strings.TrimSpace(extractUpstreamErrorMessage(body))),
		}, nil
	}

	// 其它 4xx/5xx（鉴权失败、限流、超载、内部错误等）无法区分真假 → unknown。
	if resp.StatusCode >= 400 {
		return &ClaudeAuthenticityResult{
			Verdict: AuthenticityVerdictUnknown, Method: AuthenticityMethodProbe,
			CheckedAt: time.Now(), HTTPStatus: resp.StatusCode,
			Detail: fmt.Sprintf("upstream returned %d: %s",
				resp.StatusCode, strings.TrimSpace(extractUpstreamErrorMessage(body))),
		}, nil
	}

	// 2xx：上游接受了伪造签名的 thinking 块 → 不校验签名 → 判假冒。
	return &ClaudeAuthenticityResult{
		Verdict: AuthenticityVerdictCounterfeit, Method: AuthenticityMethodProbe,
		CheckedAt: time.Now(), HTTPStatus: resp.StatusCode,
		Detail: "upstream accepted a forged thinking signature; likely non-genuine Claude",
	}, nil
}
