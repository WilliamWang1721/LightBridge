package service

import "testing"

func TestCustomProtocolFallsBackToLegacyCredentials(t *testing.T) {
	account := &Account{Platform: PlatformCustom, Credentials: map[string]any{"protocol": CustomProtocolOpenAIResponses}}
	if got := account.CustomProtocol(); got != CustomProtocolOpenAIResponses {
		t.Fatalf("CustomProtocol() = %q, want %q", got, CustomProtocolOpenAIResponses)
	}
	if _, ok := ProtocolRouteDecisionForAccountProtocols(CustomProtocolAnthropicMessages, account); !ok {
		t.Fatal("legacy Custom OpenAI Responses account should be routable from Anthropic Messages")
	}
}

func TestCustomProtocolPrefersAuthoritativeExtraField(t *testing.T) {
	account := &Account{Platform: PlatformCustom, Extra: map[string]any{"protocol": CustomProtocolAnthropicMessages}, Credentials: map[string]any{"protocol": CustomProtocolOpenAIResponses}}
	if got := account.CustomProtocol(); got != CustomProtocolAnthropicMessages {
		t.Fatalf("CustomProtocol() = %q, want authoritative extra value %q", got, CustomProtocolAnthropicMessages)
	}
}
