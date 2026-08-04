package service

import (
	"encoding/json"
	"regexp"
	"strings"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

var uiThemeHexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

var reservedUIThemeConfigKeys = map[string]struct{}{
	"style":           {},
	"ui_style":        {},
	"ui-style":        {},
	"componentstyle":  {},
	"component_style": {},
	"component-style": {},
}

// ValidateUIThemeConfig validates a submitted package configuration against the
// package manifest. It intentionally does not permit component style overrides:
// the LightBridge modern component style is fixed to Luma.
func ValidateUIThemeConfig(theme *UITheme, raw json.RawMessage) (json.RawMessage, error) {
	if theme == nil {
		return nil, infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}

	var submitted map[string]any
	if err := json.Unmarshal(raw, &submitted); err != nil {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_CONFIG", "config must be a JSON object")
	}
	if submitted == nil {
		submitted = map[string]any{}
	}

	var manifest UIThemeManifest
	if err := json.Unmarshal(theme.Manifest, &manifest); err != nil {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_MANIFEST", "stored theme manifest is invalid")
	}

	fields := make(map[string]UIThemeConfigField, len(manifest.Config))
	for _, field := range manifest.Config {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		if _, reserved := reservedUIThemeConfigKeys[strings.ToLower(key)]; reserved {
			return nil, infraerrors.BadRequest("UI_THEME_RESERVED_CONFIG_FIELD", "component style is fixed to Luma and cannot be configured by a theme package")
		}
		fields[key] = field
	}

	normalized := make(map[string]any, len(submitted))
	for key, value := range submitted {
		field, ok := fields[key]
		if !ok {
			return nil, infraerrors.BadRequest("UI_THEME_UNKNOWN_CONFIG_FIELD", "theme config contains an undeclared field: "+key)
		}
		validated, err := validateUIThemeConfigValue(field, value)
		if err != nil {
			return nil, err
		}
		normalized[key] = validated
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_CONFIG", "theme config could not be encoded")
	}
	return encoded, nil
}

func validateUIThemeConfigValue(field UIThemeConfigField, value any) (any, error) {
	key := strings.TrimSpace(field.Key)
	typeName := strings.TrimSpace(field.Type)

	switch typeName {
	case "color":
		rendered, ok := value.(string)
		if !ok || !uiThemeHexColorPattern.MatchString(strings.TrimSpace(rendered)) {
			return nil, invalidUIThemeConfigValue(key, "must be a hexadecimal color")
		}
		return strings.TrimSpace(rendered), nil
	case "text":
		rendered, ok := value.(string)
		if !ok {
			return nil, invalidUIThemeConfigValue(key, "must be text")
		}
		if len(rendered) > 2048 {
			return nil, invalidUIThemeConfigValue(key, "must be 2048 characters or fewer")
		}
		return rendered, nil
	case "select":
		rendered, ok := value.(string)
		if !ok {
			return nil, invalidUIThemeConfigValue(key, "must be one of the declared options")
		}
		for _, option := range field.Options {
			if rendered == option {
				return rendered, nil
			}
		}
		return nil, invalidUIThemeConfigValue(key, "must be one of the declared options")
	case "number":
		number, ok := value.(float64)
		if !ok {
			return nil, invalidUIThemeConfigValue(key, "must be a number")
		}
		return number, nil
	case "boolean":
		boolean, ok := value.(bool)
		if !ok {
			return nil, invalidUIThemeConfigValue(key, "must be true or false")
		}
		return boolean, nil
	default:
		return nil, infraerrors.BadRequest("UI_THEME_INVALID_CONFIG_FIELD", "theme config type is invalid")
	}
}

func invalidUIThemeConfigValue(key, requirement string) error {
	return infraerrors.BadRequest("UI_THEME_INVALID_CONFIG_VALUE", "theme config field "+key+" "+requirement)
}
