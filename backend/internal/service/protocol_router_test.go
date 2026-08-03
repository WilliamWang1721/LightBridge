package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolRouteDecision_MessageProtocolMatrix(t *testing.T) {
	inbounds := []string{
		CustomProtocolOpenAIResponses,
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
	}
	targets := []string{
		CustomProtocolOpenAIResponses,
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
	}

	for _, inbound := range inbounds {
		for _, target := range targets {
			t.Run(inbound+"->"+target, func(t *testing.T) {
				account := &Account{
					ID:       1,
					Platform: PlatformCustom,
					Type:     AccountTypeAPIKey,
					Extra: map[string]any{
						"protocol": target,
					},
				}

				decision, ok := ProtocolRouteDecisionForAccountProtocols(inbound, account)
				require.True(t, ok)
				require.Equal(t, inbound, decision.InboundProtocol)
				require.Equal(t, target, decision.TargetProtocol)
				require.Equal(t, RelayModeRouter, decision.RelayMode)
				require.Equal(t, target, decision.FinalRelayFormat)
				if inbound == target {
					require.Equal(t, []string{inbound}, decision.ConversionChain)
				} else if inbound == CustomProtocolOpenAIResponses || target == CustomProtocolOpenAIResponses {
					require.Equal(t, []string{inbound, target}, decision.ConversionChain)
				} else {
					require.Equal(t, []string{inbound, CustomProtocolOpenAIResponses, target}, decision.ConversionChain)
				}
			})
		}
	}
}

func TestProtocolRouteDecision_RelayModes(t *testing.T) {
	sameProtocol := &Account{
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModePassthrough,
		},
	}
	decision, ok := ProtocolRouteDecisionForAccountProtocols(CustomProtocolOpenAIResponses, sameProtocol)
	require.True(t, ok)
	require.Equal(t, RelayModePassthrough, decision.RelayMode)
	require.Equal(t, CustomProtocolOpenAIResponses, decision.TargetProtocol)
	require.Equal(t, []string{CustomProtocolOpenAIResponses}, decision.ConversionChain)

	_, ok = ProtocolRouteDecisionForAccountProtocols(CustomProtocolAnthropicMessages, sameProtocol)
	require.False(t, ok)

	fullPassthrough := &Account{
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"protocol":   CustomProtocolGemini,
			"relay_mode": RelayModeFullPassthrough,
		},
	}
	_, ok = ProtocolRouteDecisionForAccountProtocols(CustomProtocolAnthropicMessages, fullPassthrough)
	require.False(t, ok)

	decision, ok = ProtocolRouteDecisionForAccountProtocols(CustomProtocolGemini, fullPassthrough)
	require.True(t, ok)
	require.Equal(t, RelayModeFullPassthrough, decision.RelayMode)
	require.Equal(t, CustomProtocolGemini, decision.TargetProtocol)
	require.Equal(t, CustomProtocolGemini, decision.FinalRelayFormat)
	require.Equal(t, []string{CustomProtocolGemini}, decision.ConversionChain)
}

func TestAccountRelayMode_LegacyFieldsMapToFullPassthrough(t *testing.T) {
	require.Equal(t, RelayModeFullPassthrough, (&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{"openai_passthrough": true},
	}).RelayMode())

	require.Equal(t, RelayModeFullPassthrough, (&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{"openai_oauth_passthrough": true},
	}).RelayMode())

	require.Equal(t, RelayModeFullPassthrough, (&Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{"anthropic_passthrough": true},
	}).RelayMode())

	require.Equal(t, RelayModePassthrough, (&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"relay_mode":         RelayModePassthrough,
			"openai_passthrough": true,
		},
	}).RelayMode(), "explicit relay_mode should take precedence over legacy fields")
}

func TestFilterAccountsByRequestProtocol_MixedGroupAllowsRouterButRestrictsPassthroughModes(t *testing.T) {
	ctx := WithInboundProtocol(context.Background(), CustomProtocolOpenAIResponses)
	accounts := []Account{
		{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey},
		{ID: 2, Platform: PlatformGemini, Type: AccountTypeAPIKey},
		{ID: 3, Platform: PlatformCustom, Type: AccountTypeAPIKey, Extra: map[string]any{
			"protocol":   CustomProtocolAnthropicMessages,
			"relay_mode": RelayModePassthrough,
		}},
		{ID: 4, Platform: PlatformCustom, Type: AccountTypeAPIKey, Extra: map[string]any{
			"protocol":   CustomProtocolGemini,
			"relay_mode": RelayModeFullPassthrough,
		}},
		{ID: 5, Platform: PlatformCustom, Type: AccountTypeAPIKey, Extra: map[string]any{
			"protocol": CustomProtocolAnthropicMessages,
		}},
		{ID: 6, Platform: PlatformCustom, Type: AccountTypeAPIKey, Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModePassthrough,
		}},
	}

	filtered := filterAccountsByRequestProtocol(ctx, accounts)
	var ids []int64
	for _, account := range filtered {
		ids = append(ids, account.ID)
	}
	require.ElementsMatch(t, []int64{1, 2, 5, 6}, ids)
}

func TestFilterAccountsByRequestProtocol_RouterMessageProtocolMatrix(t *testing.T) {
	inbounds := []string{
		CustomProtocolOpenAIResponses,
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
	}
	targets := []string{
		CustomProtocolOpenAIResponses,
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
	}

	for _, inbound := range inbounds {
		for _, target := range targets {
			t.Run(inbound+"->"+target, func(t *testing.T) {
				ctx := WithInboundProtocol(context.Background(), inbound)
				router := Account{
					ID:       1,
					Platform: PlatformCustom,
					Type:     AccountTypeAPIKey,
					Extra: map[string]any{
						"protocol": target,
					},
				}
				passthrough := Account{
					ID:       2,
					Platform: PlatformCustom,
					Type:     AccountTypeAPIKey,
					Extra: map[string]any{
						"protocol":   target,
						"relay_mode": RelayModePassthrough,
					},
				}
				fullPassthrough := Account{
					ID:       3,
					Platform: PlatformCustom,
					Type:     AccountTypeAPIKey,
					Extra: map[string]any{
						"protocol":   target,
						"relay_mode": RelayModeFullPassthrough,
					},
				}

				filtered := filterAccountsByRequestProtocol(ctx, []Account{router, passthrough, fullPassthrough})
				require.NotEmpty(t, filtered)
				require.Equal(t, int64(1), filtered[0].ID)
				if inbound == target {
					require.Len(t, filtered, 3)
				} else {
					require.Len(t, filtered, 1)
				}
			})
		}
	}
}

func TestAccountMatchesRequestProtocol_RouterIgnoresMessageProtocolMismatch(t *testing.T) {
	ctx := WithInboundProtocol(context.Background(), CustomProtocolOpenAIChatCompletions)
	router := &Account{
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"protocol": CustomProtocolOpenAIResponses,
		},
	}
	passthrough := &Account{
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModePassthrough,
		},
	}
	fullPassthrough := &Account{
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModeFullPassthrough,
		},
	}

	require.True(t, accountMatchesRequestProtocol(ctx, router))
	require.False(t, accountMatchesRequestProtocol(ctx, passthrough))
	require.False(t, accountMatchesRequestProtocol(ctx, fullPassthrough))
}

func TestSchedulingQueryPlatformsForMessageRequestIgnoresGroupPlatform(t *testing.T) {
	ctx := WithInboundProtocol(context.Background(), CustomProtocolGemini)

	require.ElementsMatch(t,
		[]string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformGrok, PlatformCustom},
		schedulingQueryPlatformsForRequest(ctx, PlatformAnthropic, false),
	)
	require.ElementsMatch(t,
		[]string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformGrok, PlatformCustom},
		schedulingQueryPlatformsForRequest(ctx, PlatformOpenAI, false),
	)
}

func TestPlatformForRequest_UsesInboundBeforeGroupFallback(t *testing.T) {
	require.Equal(t,
		PlatformOpenAI,
		PlatformForRequest(WithInboundProtocol(context.Background(), CustomProtocolOpenAIResponses), PlatformAnthropic),
	)
	require.Equal(t,
		PlatformOpenAI,
		PlatformForRequest(WithInboundProtocol(context.Background(), CustomProtocolOpenAIChatCompletions), PlatformGemini),
	)
	require.Equal(t,
		PlatformGemini,
		PlatformForRequest(WithInboundProtocol(context.Background(), CustomProtocolGemini), PlatformAnthropic),
	)
	require.Equal(t,
		PlatformAnthropic,
		PlatformForRequest(context.Background(), PlatformAnthropic),
	)
}

func TestQuotaPlatform_UsesIngressWithoutGroupTypeFallback(t *testing.T) {
	apiKey := &APIKey{Group: &Group{}}

	require.Equal(t,
		PlatformOpenAI,
		QuotaPlatform(WithInboundProtocol(context.Background(), CustomProtocolOpenAIResponses), apiKey),
	)
	require.Equal(t,
		PlatformGemini,
		QuotaPlatform(WithInboundProtocol(context.Background(), CustomProtocolGemini), apiKey),
	)
	require.Equal(t,
		"",
		QuotaPlatform(context.Background(), apiKey),
	)
}

func TestGatewayResolvePlatform_UsesInboundBeforeGroupFallback(t *testing.T) {
	svc := &GatewayService{}
	ctx := WithInboundProtocol(context.Background(), CustomProtocolOpenAIResponses)

	platform, hasForcePlatform, err := svc.resolvePlatform(ctx, nil, &Group{})

	require.NoError(t, err)
	require.False(t, hasForcePlatform)
	require.Equal(t, PlatformOpenAI, platform)
	require.Equal(t, SchedulerModeSingle, (&SchedulerSnapshotService{}).resolveMode(platform, hasForcePlatform))
}
