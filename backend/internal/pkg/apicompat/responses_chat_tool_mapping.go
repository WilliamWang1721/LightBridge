package apicompat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const maxChatBridgeToolAliasLength = 64

// ResponsesChatToolIdentity is the client-visible identity of a Responses
// tool after its declaration has been flattened for a Chat Completions
// upstream. Namespace and custom-tool identity must be restored on the way
// back or strict clients such as Grok Build classify the call as "Other".
type ResponsesChatToolIdentity struct {
	Type      string
	Namespace string
	Name      string
}

func (i ResponsesChatToolIdentity) key() string {
	return i.Type + "\x00" + i.Namespace + "\x00" + i.Name
}

// ResponsesChatToolMapping is request-scoped. It records the aliases sent to
// the Chat Completions upstream and is reused while converting that request's
// non-streaming or streaming response.
type ResponsesChatToolMapping struct {
	aliases         map[string]ResponsesChatToolIdentity
	identityAliases map[string]string
}

func newResponsesChatToolMapping() *ResponsesChatToolMapping {
	return &ResponsesChatToolMapping{
		aliases:         make(map[string]ResponsesChatToolIdentity),
		identityAliases: make(map[string]string),
	}
}

func (m *ResponsesChatToolMapping) alias(identity ResponsesChatToolIdentity) string {
	if m == nil {
		return responsesChatToolAliasBase(identity)
	}
	key := identity.key()
	if alias, ok := m.identityAliases[key]; ok {
		return alias
	}

	base := sanitizeChatBridgeToolAlias(responsesChatToolAliasBase(identity))
	if base == "" {
		base = "tool"
	}
	alias := truncateChatBridgeToolAlias(base, key)
	if existing, collision := m.aliases[alias]; collision && existing.key() != key {
		alias = hashedChatBridgeToolAlias(base, key)
	}
	m.aliases[alias] = identity
	m.identityAliases[key] = alias
	return alias
}

// resolve accepts both the exact alias and a unique original short name. The
// latter is a compatibility fallback for routers that echo the child name even
// after receiving a qualified namespace alias.
func (m *ResponsesChatToolMapping) resolve(name string) (ResponsesChatToolIdentity, bool) {
	name = strings.TrimSpace(name)
	if m == nil || name == "" {
		return ResponsesChatToolIdentity{}, false
	}
	if identity, ok := m.aliases[name]; ok {
		return identity, true
	}

	var match ResponsesChatToolIdentity
	found := false
	for _, identity := range m.aliases {
		if identity.Name != name && !strings.EqualFold(identity.Name, name) {
			continue
		}
		if found && match.key() != identity.key() {
			return ResponsesChatToolIdentity{}, false
		}
		match = identity
		found = true
	}
	return match, found
}

func responsesChatToolAliasBase(identity ResponsesChatToolIdentity) string {
	name := strings.TrimSpace(identity.Name)
	namespace := strings.TrimSpace(identity.Namespace)
	if namespace == "" || name == "" {
		return name
	}
	prefix := namespace
	if !strings.HasSuffix(prefix, "__") {
		prefix += "__"
	}
	if strings.HasPrefix(name, prefix) {
		return name
	}
	return prefix + name
}

func sanitizeChatBridgeToolAlias(value string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			_, _ = builder.WriteRune(r)
			continue
		}
		_, _ = builder.WriteRune('_')
	}
	return builder.String()
}

func truncateChatBridgeToolAlias(base, key string) string {
	if len(base) <= maxChatBridgeToolAliasLength {
		return base
	}
	return hashedChatBridgeToolAlias(base, key)
}

func hashedChatBridgeToolAlias(base, key string) string {
	suffix := "__" + shortChatBridgeToolHash(key)
	limit := maxChatBridgeToolAliasLength - len(suffix)
	if limit < 1 {
		limit = 1
	}
	if len(base) > limit {
		base = base[:limit]
	}
	return base + suffix
}

func shortChatBridgeToolHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:9]
}

func responsesToolsToChatTools(tools []ResponsesTool, mapping *ResponsesChatToolMapping) []ChatTool {
	out := make([]ChatTool, 0, len(tools))
	seen := make(map[string]struct{})
	var appendTools func([]ResponsesTool, string)
	appendTools = func(items []ResponsesTool, namespace string) {
		for _, tool := range items {
			kind := strings.ToLower(strings.TrimSpace(tool.Type))
			if kind == "namespace" {
				appendTools(tool.Tools, strings.TrimSpace(tool.Name))
				continue
			}
			if kind != "function" && kind != "custom" {
				continue
			}
			name := strings.TrimSpace(tool.Name)
			if name == "" {
				continue
			}
			identity := ResponsesChatToolIdentity{Type: kind, Namespace: namespace, Name: name}
			alias := mapping.alias(identity)
			if _, duplicate := seen[alias]; duplicate {
				continue
			}
			seen[alias] = struct{}{}

			description := strings.TrimSpace(tool.Description)
			parameters := normalizeChatBridgeFunctionParameters(tool.Parameters)
			strict := tool.Strict
			if kind == "custom" {
				if description != "" {
					description += "\n"
				}
				description += "Provide the custom tool input in the input string field."
				parameters = json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}},"required":["input"],"additionalProperties":false}`)
				strictValue := true
				strict = &strictValue
			}
			// xAI/Grok Build is known to hang on the large Codex Desktop
			// codex_app.automation_update schema. Keep this workaround exact.
			if kind == "function" && strings.EqualFold(namespace, "codex_app") && strings.EqualFold(name, "automation_update") {
				parameters = json.RawMessage(`{"type":"object","additionalProperties":true}`)
				strictValue := false
				strict = &strictValue
			}
			if description == "" {
				description = "Invoke " + name
			}
			out = append(out, ChatTool{
				Type: "function",
				Function: &ChatFunction{
					Name:        alias,
					Description: description,
					Parameters:  parameters,
					Strict:      strict,
				},
			})
		}
	}
	appendTools(tools, "")
	return out
}

func normalizeChatBridgeFunctionParameters(raw json.RawMessage) json.RawMessage {
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`{"type":"object","additionalProperties":true}`)
	}
	var schema map[string]json.RawMessage
	if json.Unmarshal(raw, &schema) != nil {
		return json.RawMessage(`{"type":"object","additionalProperties":true}`)
	}
	var rootType string
	_ = json.Unmarshal(schema["type"], &rootType)
	if rootType == "" {
		if schema["properties"] == nil && schema["additionalProperties"] == nil && schema["oneOf"] == nil && schema["anyOf"] == nil {
			return json.RawMessage(`{"type":"object","additionalProperties":true}`)
		}
		schema["type"] = json.RawMessage(`"object"`)
		if normalized, err := json.Marshal(schema); err == nil {
			return normalized
		}
	}
	if rootType != "" && rootType != "object" {
		return json.RawMessage(`{"type":"object","additionalProperties":true}`)
	}
	return raw
}

func responsesAdditionalTools(inputRaw json.RawMessage) []ResponsesTool {
	var rawItems []json.RawMessage
	if json.Unmarshal(inputRaw, &rawItems) != nil {
		return nil
	}
	var tools []ResponsesTool
	for _, raw := range rawItems {
		var item struct {
			Type  string          `json:"type"`
			Tools []ResponsesTool `json:"tools"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if item.Type == "additional_tools" || item.Type == "tool_search_output" {
			tools = append(tools, item.Tools...)
		}
	}
	return tools
}

func customToolInputToArguments(raw json.RawMessage) string {
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return `{"input":""}`
	}
	var input string
	if json.Unmarshal(raw, &input) != nil {
		input = string(raw)
	}
	encoded, err := json.Marshal(map[string]string{"input": input})
	if err != nil {
		return `{"input":""}`
	}
	return string(encoded)
}

func customToolArgumentsToInput(arguments string) string {
	var wrapper map[string]json.RawMessage
	if json.Unmarshal([]byte(arguments), &wrapper) == nil {
		if raw, ok := wrapper["input"]; ok {
			var input string
			if json.Unmarshal(raw, &input) == nil {
				return input
			}
		}
	}
	return arguments
}
