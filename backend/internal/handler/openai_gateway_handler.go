package handler

import (
	"context"
	"strings"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/ctxkey"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

// OpenAIGatewayHandler handles OpenAI API gateway requests
type OpenAIGatewayHandler struct {
	gatewayService           *service.OpenAIGatewayService
	billingCacheService      *service.BillingCacheService
	apiKeyService            *service.APIKeyService
	usageRecordWorkerPool    *service.UsageRecordWorkerPool
	errorPassthroughService  *service.ErrorPassthroughService
	contentModerationService *service.ContentModerationService
	privacyFilterService     *service.PrivacyFilterService
	concurrencyHelper        *ConcurrencyHelper
	imageLimiter             *imageConcurrencyLimiter
	maxAccountSwitches       int
	cfg                      *config.Config
}

func resolveOpenAIMessagesDispatchMappedModel(apiKey *service.APIKey, requestedModel string) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return strings.TrimSpace(apiKey.Group.ResolveMessagesDispatchModel(requestedModel))
}

func usageRecordContext(parent context.Context, base context.Context) context.Context {
	if base == nil {
		base = context.Background()
	}
	if parent == nil {
		return base
	}
	base = copyTrimmedStringContextValue(parent, base, ctxkey.ClientRequestID)
	base = copyTrimmedStringContextValue(parent, base, ctxkey.RequestID)
	base = copyTrimmedStringContextValue(parent, base, ctxkey.ForcePlatform)
	base = copyTrimmedStringContextValue(parent, base, ctxkey.InboundProtocol)
	base = copyTrimmedStringContextValue(parent, base, ctxkey.RequiredProtocol)
	return base
}

func copyTrimmedStringContextValue(parent context.Context, base context.Context, key ctxkey.Key) context.Context {
	if value, _ := parent.Value(key).(string); strings.TrimSpace(value) != "" {
		return context.WithValue(base, key, strings.TrimSpace(value))
	}
	return base
}

func wrapUsageRecordTaskContext(parent context.Context, task service.UsageRecordTask) service.UsageRecordTask {
	if task == nil {
		return nil
	}
	return func(ctx context.Context) {
		task(usageRecordContext(parent, ctx))
	}
}

// allowOpenAIResponsesHTTPContinuation is retained for compatibility with
// older response-stream tests. Grok is the only upstream whose HTTP response
// continuation semantics require this path.
func allowOpenAIResponsesHTTPContinuation(platform string) bool {
	return strings.EqualFold(strings.TrimSpace(platform), service.PlatformGrok)
}

func setCustomRequiredProtocol(c *gin.Context, protocol string) {
	if c == nil || c.Request == nil || strings.TrimSpace(protocol) == "" {
		return
	}
	c.Request = c.Request.WithContext(service.WithInboundProtocol(c.Request.Context(), strings.TrimSpace(protocol)))
}

// NewOpenAIGatewayHandler creates a new OpenAIGatewayHandler
func NewOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	privacyFilterService *service.PrivacyFilterService,
	cfg *config.Config,
) *OpenAIGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &OpenAIGatewayHandler{
		gatewayService:           gatewayService,
		billingCacheService:      billingCacheService,
		apiKeyService:            apiKeyService,
		usageRecordWorkerPool:    usageRecordWorkerPool,
		errorPassthroughService:  errorPassthroughService,
		contentModerationService: contentModerationService,
		privacyFilterService:     privacyFilterService,
		concurrencyHelper:        NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		imageLimiter:             &imageConcurrencyLimiter{},
		maxAccountSwitches:       maxAccountSwitches,
		cfg:                      cfg,
	}
}
