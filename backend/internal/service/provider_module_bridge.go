package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/modules"
	"github.com/gin-gonic/gin"
)

const moduleAccountPlatform = "module"

func (s *GatewayService) SetProviderRegistry(registry *modules.ProviderRegistry) {
	if s == nil {
		return
	}
	s.providerRegistry = registry
}

func (s *AccountTestService) SetProviderRegistry(registry *modules.ProviderRegistry) {
	if s == nil {
		return
	}
	s.providerRegistry = registry
}

func (s *adminServiceImpl) SetProviderRegistry(registry *modules.ProviderRegistry) {
	if s == nil {
		return
	}
	s.providerRegistry = registry
}

func (s *GatewayService) forwardModuleProvider(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest, startTime any) (*ForwardResult, bool, error) {
	adapter, providerID, ok, err := s.resolveModuleProviderAdapter(account)
	if !ok || err != nil {
		return nil, ok, err
	}
	if parsed == nil {
		return nil, true, fmt.Errorf("parse request: empty request")
	}

	req := modules.GatewayRequest{
		DownstreamProtocol: "anthropic",
		Method:             http.MethodPost,
		Headers:            cloneGinRequestHeader(c),
		Body:               json.RawMessage(parsed.Body),
		Stream:             parsed.Stream,
		Account:            providerAccountFromService(account, providerID),
		Metadata: map[string]any{
			"model": parsed.Model,
		},
	}
	if c != nil && c.Request != nil {
		req.Method = c.Request.Method
		req.Endpoint = c.Request.URL.Path
	}

	events, err := adapter.Forward(ctx, req)
	if err != nil {
		return nil, true, err
	}

	start := time.Now()
	if t, ok := startTime.(time.Time); ok {
		start = t
	}
	result, err := s.writeModuleProviderEvents(ctx, c, events, parsed, start)
	return result, true, err
}

func (s *GatewayService) resolveModuleProviderAdapter(account *Account) (modules.ProviderAdapter, string, bool, error) {
	if s == nil {
		return nil, "", false, nil
	}
	return resolveModuleProviderAdapter(s.providerRegistry, account)
}

func resolveModuleProviderAdapter(registry *modules.ProviderRegistry, account *Account) (modules.ProviderAdapter, string, bool, error) {
	if !accountUsesModuleProvider(account) {
		return nil, "", false, nil
	}
	providerID := effectiveServiceProviderID(account)
	if providerID == "" {
		return nil, "", true, errors.New("module provider id is empty")
	}
	if registry == nil {
		return nil, providerID, true, errors.New("module provider registry is not configured")
	}
	adapter, err := registry.Resolve(providerID)
	if err != nil {
		return nil, providerID, true, err
	}
	return adapter, providerID, true, nil
}

func (s *GatewayService) writeModuleProviderEvents(ctx context.Context, c *gin.Context, events <-chan modules.GatewayEvent, parsed *ParsedRequest, start time.Time) (*ForwardResult, error) {
	result := &ForwardResult{
		Model:    parsed.Model,
		Stream:   parsed.Stream,
		Duration: time.Since(start),
	}
	var usage ClaudeUsage
	statusCode := http.StatusOK
	var wroteHeaders bool
	var sawData bool
	var firstTokenMs *int
	var buffered bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			result.ClientDisconnect = true
			result.Duration = time.Since(start)
			result.FirstTokenMs = firstTokenMs
			result.Usage = usage
			return result, ctx.Err()
		case ev, ok := <-events:
			if !ok {
				result.Duration = time.Since(start)
				result.FirstTokenMs = firstTokenMs
				result.Usage = usage
				if !parsed.Stream && buffered.Len() > 0 && c != nil {
					writeModuleHeaders(c, nil, statusCode, "application/json")
					c.Data(statusCode, "application/json", buffered.Bytes())
				}
				return result, nil
			}
			switch strings.ToLower(strings.TrimSpace(ev.Type)) {
			case "headers":
				if ev.StatusCode > 0 {
					statusCode = ev.StatusCode
				}
				writeModuleHeaders(c, ev.Headers, statusCode, "")
				wroteHeaders = true
			case "data":
				if ev.Usage != nil {
					mergeModuleUsage(&usage, ev.Usage)
				}
				if len(ev.Data) == 0 {
					continue
				}
				sawData = true
				if firstTokenMs == nil {
					v := int(time.Since(start).Milliseconds())
					firstTokenMs = &v
				}
				if parsed.Stream {
					if !wroteHeaders {
						writeModuleHeaders(c, nil, statusCode, "text/event-stream")
						wroteHeaders = true
					}
					writeModuleSSEData(c, ev.Data)
				} else {
					_, _ = buffered.Write(ev.Data)
				}
			case "usage":
				if ev.Usage != nil {
					mergeModuleUsage(&usage, ev.Usage)
				}
			case "error":
				moduleErr := moduleProviderError(ev.Error)
				if c == nil || c.Writer.Size() < 0 {
					return result, &UpstreamFailoverError{
						StatusCode:             moduleErr.StatusCode,
						ResponseBody:           []byte(moduleErr.Message),
						RetryableOnSameAccount: moduleErr.Retryable,
					}
				}
				if parsed.Stream && ev.Error != nil {
					writeModuleSSEData(c, mustJSON(map[string]any{
						"type":  "error",
						"error": ev.Error,
					}))
				}
				result.Duration = time.Since(start)
				result.FirstTokenMs = firstTokenMs
				result.Usage = usage
				return result, fmt.Errorf("%s", moduleErr.Message)
			case "done":
				result.Duration = time.Since(start)
				result.FirstTokenMs = firstTokenMs
				result.Usage = usage
				if parsed.Stream {
					if !wroteHeaders {
						writeModuleHeaders(c, nil, statusCode, "text/event-stream")
					}
					if c != nil {
						_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
						c.Writer.Flush()
					}
				} else if sawData && c != nil {
					writeModuleHeaders(c, nil, statusCode, "application/json")
					c.Data(statusCode, "application/json", buffered.Bytes())
				}
				return result, nil
			}
		}
	}
}

func (s *AccountTestService) testModuleProviderAccount(c *gin.Context, account *Account, modelID string, prompt string, mode string) (bool, error) {
	adapter, providerID, ok, err := resolveModuleProviderAdapter(s.providerRegistry, account)
	if !ok || err != nil {
		return ok, err
	}
	req := modules.TestAccountRequest{
		Account: providerAccountFromService(account, providerID),
		Mode:    strings.TrimSpace(mode),
	}
	if req.Mode == "" {
		req.Mode = AccountTestModeDefault
	}
	req.Account.Metadata["model"] = modelID
	req.Account.Metadata["prompt"] = prompt

	result, err := adapter.TestAccount(c.Request.Context(), req)
	if err != nil {
		return true, s.sendErrorAndEnd(c, err.Error())
	}
	if result == nil {
		return true, s.sendErrorAndEnd(c, "Module provider returned empty test result")
	}
	if result.Message != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: result.Message})
	}
	if result.OK {
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true, Data: result.Metadata})
		return true, nil
	}
	msg := strings.TrimSpace(result.Message)
	if msg == "" {
		msg = "Module provider account test failed"
	}
	return true, s.sendErrorAndEnd(c, msg)
}

func (s *adminServiceImpl) RefreshModuleProviderAccount(ctx context.Context, account *Account) (*Account, bool, error) {
	if s == nil || account == nil {
		return nil, false, nil
	}
	adapter, providerID, ok, err := resolveModuleProviderAdapter(s.providerRegistry, account)
	if !ok || err != nil {
		return nil, ok, err
	}
	providerAccount, err := s.providerAccountForModuleRefresh(ctx, account, providerID)
	if err != nil {
		return nil, true, err
	}
	refreshed, err := adapter.RefreshAccount(ctx, providerAccount)
	if err != nil {
		return nil, true, err
	}
	if refreshed == nil {
		return nil, true, errors.New("module provider returned empty refreshed account")
	}
	input := &UpdateAccountInput{}
	if refreshed.Secrets != nil {
		input.Credentials = clearModuleTransientCredentials(refreshed.Secrets)
	}
	if refreshed.Config != nil {
		input.Extra = mergeModuleProviderExtra(account.Extra, refreshed.Config)
	}
	if refreshed.Metadata != nil {
		input.Extra = mergeModuleProviderExtra(input.Extra, map[string]any{"provider_metadata": refreshed.Metadata})
	}
	if input.Credentials == nil && input.Extra == nil {
		return account, true, nil
	}
	updated, err := s.UpdateAccount(ctx, account.ID, input)
	if err != nil {
		return nil, true, err
	}
	return updated, true, nil
}

func (s *adminServiceImpl) providerAccountForModuleRefresh(ctx context.Context, account *Account, providerID string) (modules.ProviderAccount, error) {
	out := providerAccountFromService(account, providerID)
	if account == nil || account.ProxyID == nil || stringFromMapAny(out.Metadata, "proxy_url") != "" {
		return out, nil
	}
	if s == nil || s.proxyRepo == nil {
		return out, errors.New("module provider account has proxy_id but proxy repository is not configured")
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil {
		return out, err
	}
	if proxy == nil {
		return out, errors.New("module provider account proxy not found")
	}
	out.Metadata["proxy_url"] = proxy.URL()
	return out, nil
}

func clearModuleTransientCredentials(secrets map[string]any) map[string]any {
	out := copyAnyMap(secrets)
	if out == nil {
		out = map[string]any{}
	}
	for _, key := range []string{"authorization_code", "code_verifier", "oauth_state", "session_key"} {
		out[key] = ""
	}
	return out
}

func providerAccountFromService(account *Account, providerID string) modules.ProviderAccount {
	out := modules.ProviderAccount{
		ProviderID: providerID,
		Metadata:   map[string]any{},
	}
	if account != nil {
		out.ID = fmt.Sprintf("%d", account.ID)
		out.DisplayName = account.Name
		out.Config = copyAnyMap(account.Extra)
		out.Secrets = copyAnyMap(account.Credentials)
		out.Metadata["platform"] = account.Platform
		out.Metadata["type"] = account.Type
		if account.ProxyID != nil && account.Proxy != nil {
			out.Metadata["proxy_url"] = account.Proxy.URL()
		}
	}
	if out.Config == nil {
		out.Config = map[string]any{}
	}
	if out.Secrets == nil {
		out.Secrets = map[string]any{}
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out
}

func effectiveServiceProviderID(account *Account) string {
	if account == nil {
		return ""
	}
	if providerID := stringFromMapAny(account.Extra, "provider_id"); providerID != "" {
		return strings.ToLower(providerID)
	}
	if strings.EqualFold(account.Platform, moduleAccountPlatform) {
		if providerID := stringFromMapAny(account.Credentials, "provider_id"); providerID != "" {
			return strings.ToLower(providerID)
		}
		if providerID := moduleMigrationProviderID(account.Extra); providerID != "" {
			return strings.ToLower(providerID)
		}
	}
	return strings.ToLower(strings.TrimSpace(account.Platform))
}

func accountUsesModuleProvider(account *Account) bool {
	if account == nil {
		return false
	}
	if strings.EqualFold(account.Platform, moduleAccountPlatform) {
		return true
	}
	return stringFromMapAny(account.Extra, "provider_id") != ""
}

func moduleMigrationProviderID(extra map[string]any) string {
	migration, ok := extra["module_migration"].(map[string]any)
	if !ok {
		return ""
	}
	return stringFromMapAny(migration, "provider_id")
}

func cloneGinRequestHeader(c *gin.Context) map[string][]string {
	if c == nil || c.Request == nil {
		return nil
	}
	return c.Request.Header.Clone()
}

func writeModuleHeaders(c *gin.Context, headers map[string][]string, statusCode int, contentType string) {
	if c == nil {
		return
	}
	for key, values := range headers {
		if strings.EqualFold(key, "Content-Length") || strings.EqualFold(key, "Transfer-Encoding") {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	if contentType != "" && c.Writer.Header().Get("Content-Type") == "" {
		c.Header("Content-Type", contentType)
	}
	if strings.Contains(strings.ToLower(c.Writer.Header().Get("Content-Type")), "text/event-stream") {
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	if c.Writer.Size() < 0 {
		c.Status(statusCode)
	}
}

func writeModuleSSEData(c *gin.Context, data []byte) {
	if c == nil {
		return
	}
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}

func mergeModuleUsage(target *ClaudeUsage, usage *modules.Usage) {
	if target == nil || usage == nil {
		return
	}
	target.InputTokens = int(usage.InputTokens)
	target.OutputTokens = int(usage.OutputTokens)
}

func moduleProviderError(err *modules.GatewayError) modules.GatewayError {
	if err == nil {
		return modules.GatewayError{
			StatusCode: http.StatusBadGateway,
			Code:       "module_provider_error",
			Message:    "Module provider returned an error",
		}
	}
	out := *err
	if out.StatusCode <= 0 {
		out.StatusCode = http.StatusBadGateway
	}
	if strings.TrimSpace(out.Message) == "" {
		out.Message = "Module provider returned an error"
	}
	return out
}

func copyAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mergeModuleProviderExtra(existing map[string]any, updates map[string]any) map[string]any {
	if existing == nil && updates == nil {
		return nil
	}
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range updates {
		out[key] = value
	}
	return out
}

func stringFromMapAny(m map[string]any, key string) string {
	if len(m) == 0 {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error"}`)
	}
	return data
}
