//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGroup_GetImagePrice_1K 测试 1K 尺寸返回正确价格
func TestGroup_GetImagePrice_1K(t *testing.T) {
	price := 0.10
	group := &Group{
		ImagePrice1K: &price,
	}

	result := group.GetImagePrice("1K")
	require.NotNil(t, result)
	require.InDelta(t, 0.10, *result, 0.0001)
}

// TestGroup_GetImagePrice_2K 测试 2K 尺寸返回正确价格
func TestGroup_GetImagePrice_2K(t *testing.T) {
	price := 0.15
	group := &Group{
		ImagePrice2K: &price,
	}

	result := group.GetImagePrice("2K")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)
}

// TestGroup_GetImagePrice_4K 测试 4K 尺寸返回正确价格
func TestGroup_GetImagePrice_4K(t *testing.T) {
	price := 0.30
	group := &Group{
		ImagePrice4K: &price,
	}

	result := group.GetImagePrice("4K")
	require.NotNil(t, result)
	require.InDelta(t, 0.30, *result, 0.0001)
}

// TestGroup_GetImagePrice_UnknownSize 测试未知尺寸回退 2K
func TestGroup_GetImagePrice_UnknownSize(t *testing.T) {
	price2K := 0.15
	group := &Group{
		ImagePrice2K: &price2K,
	}

	// 未知尺寸 "3K" 应该回退到 2K
	result := group.GetImagePrice("3K")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)

	// 空字符串也回退到 2K
	result = group.GetImagePrice("")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)
}

// TestGroup_GetImagePrice_NilValues 测试未配置时返回 nil
func TestGroup_GetImagePrice_NilValues(t *testing.T) {
	group := &Group{
		// 所有 ImagePrice 字段都是 nil
	}

	require.Nil(t, group.GetImagePrice("1K"))
	require.Nil(t, group.GetImagePrice("2K"))
	require.Nil(t, group.GetImagePrice("4K"))
	require.Nil(t, group.GetImagePrice("unknown"))
}

// TestGroup_GetImagePrice_PartialConfig 测试部分配置
func TestGroup_GetImagePrice_PartialConfig(t *testing.T) {
	price1K := 0.10
	group := &Group{
		ImagePrice1K: &price1K,
		// ImagePrice2K 和 ImagePrice4K 未配置
	}

	result := group.GetImagePrice("1K")
	require.NotNil(t, result)
	require.InDelta(t, 0.10, *result, 0.0001)

	// 2K 和 4K 返回 nil
	require.Nil(t, group.GetImagePrice("2K"))
	require.Nil(t, group.GetImagePrice("4K"))
}

func TestAccountUpstreamProtocols_ReportsActualTargetProtocolsOnly(t *testing.T) {
	openAI := &Account{Platform: PlatformOpenAI}
	require.Equal(t, []string{
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolOpenAIEmbeddings,
		CustomProtocolOpenAIResponses,
	}, AccountUpstreamProtocols(openAI))

	antigravity := &Account{Platform: PlatformGemini, SubPlatform: SubPlatformAntigravity}
	require.Equal(t, []string{
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
	}, AccountUpstreamProtocols(antigravity))

	customRouter := &Account{
		Platform: PlatformCustom,
		Extra: map[string]any{
			"protocol": CustomProtocolOpenAIResponses,
		},
	}
	require.Equal(t, []string{
		CustomProtocolOpenAIResponses,
	}, AccountUpstreamProtocols(customRouter))

	customPassthrough := &Account{
		Platform: PlatformCustom,
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModePassthrough,
		},
	}
	require.Equal(t, []string{
		CustomProtocolOpenAIResponses,
	}, AccountUpstreamProtocols(customPassthrough))

	customEmbedding := &Account{
		Platform: PlatformCustom,
		Extra: map[string]any{
			"protocol": CustomProtocolOpenAIEmbeddings,
		},
	}
	require.Equal(t, []string{CustomProtocolOpenAIEmbeddings}, AccountUpstreamProtocols(customEmbedding))
}

func TestAccountAvailableIngressProtocols_RespectsRelayMode(t *testing.T) {
	router := &Account{
		Platform: PlatformCustom,
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModeRouter,
		},
	}
	require.Equal(t, []string{
		CustomProtocolAnthropicMessages,
		CustomProtocolGemini,
		CustomProtocolOpenAIChatCompletions,
		CustomProtocolOpenAIResponses,
	}, AccountAvailableIngressProtocols(router))

	for _, mode := range []string{RelayModePassthrough, RelayModeFullPassthrough} {
		account := &Account{
			Platform: PlatformCustom,
			Extra: map[string]any{
				"protocol":   CustomProtocolOpenAIResponses,
				"relay_mode": mode,
			},
		}
		require.Equal(t, []string{CustomProtocolOpenAIResponses}, AccountAvailableIngressProtocols(account))
	}

	embeddings := &Account{
		Platform: PlatformCustom,
		Extra: map[string]any{
			"protocol": CustomProtocolOpenAIEmbeddings,
		},
	}
	require.Equal(t, []string{CustomProtocolOpenAIEmbeddings}, AccountAvailableIngressProtocols(embeddings))
}

func TestNormalizeGroupUpstreamProtocolFilter_RejectsPlatformAliases(t *testing.T) {
	require.Empty(t, NormalizeGroupUpstreamProtocolFilter(PlatformOpenAI))
	require.Empty(t, NormalizeGroupUpstreamProtocolFilter("claude"))
	require.Empty(t, NormalizeGroupUpstreamProtocolFilter(PlatformAntigravity))
	require.Equal(t, CustomProtocolOpenAIChatCompletions, NormalizeGroupUpstreamProtocolFilter(CustomProtocolOpenAIChatCompletions))
	require.Equal(t, CustomProtocolOpenAIEmbeddings, NormalizeGroupUpstreamProtocolFilter(CustomProtocolOpenAIEmbeddings))
}
