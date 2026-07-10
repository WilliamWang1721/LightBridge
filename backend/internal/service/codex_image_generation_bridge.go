package service

import "strings"

const featureKeyCodexImageGenerationBridge = "codex_image_generation_bridge"

// CodexImageGenerationBridgeMode represents the four-state control for Codex
// image_generation bridge injection.
type CodexImageGenerationBridgeMode string

const (
	CodexImageGenerationBridgeFollow CodexImageGenerationBridgeMode = ""      // follow channel/global setting
	CodexImageGenerationBridgeForce  CodexImageGenerationBridgeMode = "force" // force inject image generation tool
	CodexImageGenerationBridgeOff    CodexImageGenerationBridgeMode = "off"   // don't inject
	CodexImageGenerationBridgeBlock  CodexImageGenerationBridgeMode = "block" // block and strip existing image generation tools
)

// parseBridgeModeFromValue converts a raw config value to a bridge mode.
// Legacy booleans are mapped: true→force, false→off, nil→follow.
// String values "force"/"off"/"block"/"follow" are accepted directly.
func parseBridgeModeFromValue(v any) CodexImageGenerationBridgeMode {
	switch val := v.(type) {
	case nil:
		return CodexImageGenerationBridgeFollow
	case bool:
		if val {
			return CodexImageGenerationBridgeForce
		}
		return CodexImageGenerationBridgeOff
	case string:
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "force":
			return CodexImageGenerationBridgeForce
		case "off":
			return CodexImageGenerationBridgeOff
		case "block":
			return CodexImageGenerationBridgeBlock
		default:
			return CodexImageGenerationBridgeFollow
		}
	}
	return CodexImageGenerationBridgeFollow
}

// bridgeModeFromMap extracts a bridge mode from a map, trying the given keys.
func bridgeModeFromMap(values map[string]any, keys ...string) CodexImageGenerationBridgeMode {
	if values == nil {
		return CodexImageGenerationBridgeFollow
	}
	for _, key := range keys {
		if v, ok := values[key]; ok {
			return parseBridgeModeFromValue(v)
		}
	}
	return CodexImageGenerationBridgeFollow
}

// platformBridgeMode extracts a bridge mode from a map, first trying a
// platform-nested map (e.g. {"openai": "block"}) then a plain value.
func platformBridgeMode(values map[string]any, key string, platform string) CodexImageGenerationBridgeMode {
	if values == nil {
		return CodexImageGenerationBridgeFollow
	}
	// Try string value first (new four-state).
	if v, ok := values[key].(string); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "force":
			return CodexImageGenerationBridgeForce
		case "off":
			return CodexImageGenerationBridgeOff
		case "block":
			return CodexImageGenerationBridgeBlock
		case "follow":
			return CodexImageGenerationBridgeFollow
		}
	}
	// Legacy boolean value.
	if v, ok := values[key].(bool); ok {
		return parseBridgeModeFromValue(v)
	}
	// Platform-nested map: {"openai": "block"} or {"openai": true}.
	raw, ok := values[key].(map[string]any)
	if !ok {
		return CodexImageGenerationBridgeFollow
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return CodexImageGenerationBridgeFollow
	}
	if v, ok := raw[platform]; ok {
		return parseBridgeModeFromValue(v)
	}
	return CodexImageGenerationBridgeFollow
}

// CodexImageGenerationBridgeOverride returns the channel-level override for Codex
// image_generation bridge injection. Empty string means follow the global/account policy.
func (c *Channel) CodexImageGenerationBridgeOverride(platform string) CodexImageGenerationBridgeMode {
	if c == nil {
		return CodexImageGenerationBridgeFollow
	}
	return platformBridgeMode(c.FeaturesConfig, featureKeyCodexImageGenerationBridge, platform)
}

// CodexImageGenerationBridgeOverride returns the account-level override for Codex
// image_generation bridge injection. Empty string means follow the channel/global policy.
func (a *Account) CodexImageGenerationBridgeOverride() CodexImageGenerationBridgeMode {
	if a == nil || a.Platform != PlatformOpenAI || a.Extra == nil {
		return CodexImageGenerationBridgeFollow
	}
	if mode := bridgeModeFromMap(a.Extra, featureKeyCodexImageGenerationBridge, "codex_image_generation_bridge_enabled"); mode != CodexImageGenerationBridgeFollow {
		return mode
	}
	openaiConfig, _ := a.Extra[PlatformOpenAI].(map[string]any)
	return bridgeModeFromMap(openaiConfig, featureKeyCodexImageGenerationBridge, "codex_image_generation_bridge_enabled")
}
