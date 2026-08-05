package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/modules"
	"github.com/gin-gonic/gin"
)

type fakeProviderAdapter struct {
	forwardReq modules.GatewayRequest
	testReq    modules.TestAccountRequest
	events     []modules.GatewayEvent
	testResult *modules.TestAccountResult
}

func (a *fakeProviderAdapter) ValidateAccount(_ context.Context, req modules.ProviderAccount) (*modules.AccountValidationResult, error) {
	return &modules.AccountValidationResult{Valid: true}, nil
}

func (a *fakeProviderAdapter) RefreshAccount(_ context.Context, req modules.ProviderAccount) (*modules.ProviderAccount, error) {
	return &req, nil
}

func (a *fakeProviderAdapter) Forward(_ context.Context, req modules.GatewayRequest) (<-chan modules.GatewayEvent, error) {
	a.forwardReq = req
	ch := make(chan modules.GatewayEvent, len(a.events))
	for _, ev := range a.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (a *fakeProviderAdapter) TestAccount(_ context.Context, req modules.TestAccountRequest) (*modules.TestAccountResult, error) {
	a.testReq = req
	return a.testResult, nil
}

func (a *fakeProviderAdapter) Close() error { return nil }

func TestGatewayServiceForwardModuleProviderUsesRegisteredAdapter(t *testing.T) {
	adapter := &fakeProviderAdapter{
		events: []modules.GatewayEvent{
			{Type: "headers", StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}}},
			{Type: "data", Data: json.RawMessage(`{"ok":true}`), Usage: &modules.Usage{InputTokens: 3, OutputTokens: 5}},
			{Type: "done"},
		},
	}
	registry := modules.NewProviderRegistry()
	registry.Register("openai", adapter)

	svc := &GatewayService{}
	svc.SetProviderRegistry(registry)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5","stream":false}`))
	c.Request.Header.Set("X-Test", "yes")
	account := &Account{
		ID:          42,
		Name:        "module-openai",
		Platform:    moduleAccountPlatform,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"provider_id": "openai", "base_url": "https://api.openai.com/v1"},
	}
	parsed := &ParsedRequest{Body: []byte(`{"model":"gpt-5","stream":false}`), Model: "gpt-5"}

	result, handled, err := svc.forwardModuleProvider(context.Background(), c, account, parsed, nil)
	if err != nil {
		t.Fatalf("forwardModuleProvider returned error: %v", err)
	}
	if !handled {
		t.Fatal("forwardModuleProvider did not handle module account")
	}
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("status = %d, want 200", got)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"ok":true}` {
		t.Fatalf("body = %q", got)
	}
	if result == nil || result.Usage.InputTokens != 3 || result.Usage.OutputTokens != 5 {
		t.Fatalf("usage = %+v, want input=3 output=5", result)
	}
	if adapter.forwardReq.Account.ProviderID != "openai" {
		t.Fatalf("provider id = %q, want openai", adapter.forwardReq.Account.ProviderID)
	}
	if adapter.forwardReq.Account.Secrets["api_key"] != "sk-test" {
		t.Fatalf("api key was not passed through provider account secrets")
	}
	if got := http.Header(adapter.forwardReq.Headers).Get("X-Test"); got != "yes" {
		t.Fatalf("request headers were not passed to adapter")
	}
}

func TestAccountTestServiceTestModuleProviderAccountUsesRegisteredAdapter(t *testing.T) {
	adapter := &fakeProviderAdapter{
		testResult: &modules.TestAccountResult{OK: true, Message: "usable", Metadata: map[string]any{"provider_id": "openai"}},
	}
	registry := modules.NewProviderRegistry()
	registry.Register("openai", adapter)
	svc := &AccountTestService{}
	svc.SetProviderRegistry(registry)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/1/test", nil)
	account := &Account{ID: 7, Platform: moduleAccountPlatform, Type: AccountTypeAPIKey, Extra: map[string]any{"provider_id": "openai"}}

	handled, err := svc.testModuleProviderAccount(c, account, "gpt-5", "hi", "default")
	if err != nil {
		t.Fatalf("testModuleProviderAccount returned error: %v", err)
	}
	if !handled {
		t.Fatal("testModuleProviderAccount did not handle module account")
	}
	if adapter.testReq.Account.ProviderID != "openai" {
		t.Fatalf("provider id = %q, want openai", adapter.testReq.Account.ProviderID)
	}
	if adapter.testReq.Account.Metadata["model"] != "gpt-5" {
		t.Fatalf("model metadata = %v, want gpt-5", adapter.testReq.Account.Metadata["model"])
	}
	if !strings.Contains(rec.Body.String(), `"type":"test_complete"`) {
		t.Fatalf("SSE output did not include completion event: %s", rec.Body.String())
	}
}

func TestResolveModuleProviderAdapterSkipsNonModuleAccounts(t *testing.T) {
	_, _, handled, err := resolveModuleProviderAdapter(modules.NewProviderRegistry(), &Account{Platform: PlatformAnthropic})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Fatal("non-module account should not be handled")
	}
}

func TestResolveModuleProviderAdapterErrorsWhenRegistryMissingProvider(t *testing.T) {
	_, _, handled, err := resolveModuleProviderAdapter(modules.NewProviderRegistry(), &Account{Platform: moduleAccountPlatform, Extra: map[string]any{"provider_id": "openai"}})
	if !handled {
		t.Fatal("module account should be handled")
	}
	if err == nil || !strings.Contains(err.Error(), `provider "openai" is not registered`) {
		t.Fatalf("expected missing provider error, got %v", err)
	}
}

func TestCanonicalOpenAIAccountStillUsesModuleProvider(t *testing.T) {
	adapter := &fakeProviderAdapter{}
	registry := modules.NewProviderRegistry()
	registry.Register("openai", adapter)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"provider_id": "openai",
			"module_migration": map[string]any{
				"provider_id": "openai",
			},
		},
	}

	got, providerID, handled, err := resolveModuleProviderAdapter(registry, account)
	if err != nil {
		t.Fatalf("resolveModuleProviderAdapter returned error: %v", err)
	}
	if !handled || got != adapter || providerID != "openai" {
		t.Fatalf("adapter=%T providerID=%q handled=%v, want registered OpenAI adapter", got, providerID, handled)
	}
	if account.EffectivePlatform() != PlatformOpenAI {
		t.Fatalf("EffectivePlatform() = %q, want openai", account.EffectivePlatform())
	}
}

func TestEffectivePlatformRepairsLegacyModuleOpenAIIdentity(t *testing.T) {
	account := &Account{
		Platform: moduleAccountPlatform,
		Extra:    map[string]any{"provider_id": "openai"},
	}
	if got := account.EffectivePlatform(); got != PlatformOpenAI {
		t.Fatalf("EffectivePlatform() = %q, want openai", got)
	}
}
