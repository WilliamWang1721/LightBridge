package mihomo

import (
	"testing"

	proxynode "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/node"
	proxyprofile "github.com/WilliamWang1721/LightBridge/internal/modules/proxy/internal/profile"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCompileGeneratesPinnedLocalRuntimeConfig(t *testing.T) {
	compiled, err := Compile(Profile{
		ID:              12,
		Strategy:        proxyprofile.StrategyURLTest,
		TestURL:         "https://example.com/generate_204",
		IntervalSeconds: 60,
	}, []proxynode.Node{{
		ID:     34,
		Name:   "user supplied name",
		Type:   proxynode.TypeHTTP,
		Config: map[string]any{"server": "proxy.example.com", "port": 8080},
		Secret: map[string]any{"username": "user", "password": "pass"},
	}}, RuntimeConfig{
		MixedPort:        17012,
		ControllerPort:   18012,
		ControllerSecret: "secret",
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(compiled, &doc))
	require.Equal(t, false, doc["allow-lan"])
	require.Equal(t, "127.0.0.1", doc["bind-address"])
	require.Equal(t, "127.0.0.1:18012", doc["external-controller"])
	require.NotContains(t, doc, "external-ui")
	require.NotContains(t, doc, "tun")
	require.NotContains(t, doc, "tproxy")
	require.NotContains(t, doc, "redir-port")
	require.NotContains(t, doc, "rule-providers")
	require.NotContains(t, doc, "script")

	proxies, ok := doc["proxies"].([]any)
	require.True(t, ok)
	firstProxy, ok := proxies[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "lb-node-34", firstProxy["name"])
	require.Equal(t, "proxy.example.com", firstProxy["server"])

	groups, ok := doc["proxy-groups"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, groups)
	group, ok := groups[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "LB-PROFILE-12", group["name"])
	require.Equal(t, "url-test", group["type"])
}

func TestCompileRejectsUnsupportedStrategy(t *testing.T) {
	_, err := Compile(Profile{ID: 1, Strategy: proxyprofile.Strategy("bad")}, []proxynode.Node{{
		ID:     1,
		Type:   proxynode.TypeHTTP,
		Config: map[string]any{"server": "proxy.example.com", "port": 8080},
	}}, RuntimeConfig{MixedPort: 17001, ControllerPort: 18001, ControllerSecret: "secret"})
	require.ErrorContains(t, err, "unsupported proxy profile strategy")
}

func TestCompileUsesInternalNodeNamesOnly(t *testing.T) {
	compiled, err := Compile(Profile{ID: 9, Strategy: proxyprofile.StrategySelect}, []proxynode.Node{{
		ID:     7,
		Name:   "User Clash Name",
		Type:   proxynode.TypeSOCKS5,
		Config: map[string]any{"server": "127.0.0.1", "port": 1080},
	}}, RuntimeConfig{MixedPort: 17009, ControllerPort: 18009, ControllerSecret: "secret"})
	require.NoError(t, err)
	require.Contains(t, string(compiled), "lb-node-7")
	require.NotContains(t, string(compiled), "User Clash Name")
}
