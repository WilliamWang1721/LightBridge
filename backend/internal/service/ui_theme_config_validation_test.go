package service

import (
	"encoding/json"
	"testing"
)

func testUIThemeWithConfig(t *testing.T, fields []UIThemeConfigField) *UITheme {
	t.Helper()
	manifest, err := json.Marshal(UIThemeManifest{
		ID:       "test-theme",
		Name:     "Test Theme",
		Version:  "1.0.0",
		EntryCSS: "style.css",
		Config:   fields,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return &UITheme{ID: "test-theme", Manifest: manifest}
}

func TestValidateUIThemeConfigAcceptsDeclaredValues(t *testing.T) {
	theme := testUIThemeWithConfig(t, []UIThemeConfigField{
		{Key: "primary_color", Type: "color"},
		{Key: "radius", Type: "number"},
		{Key: "menu", Type: "select", Options: []string{"default", "compact"}},
		{Key: "glass", Type: "boolean"},
	})

	raw := json.RawMessage(`{"primary_color":"#2563eb","radius":8,"menu":"compact","glass":true}`)
	normalized, err := ValidateUIThemeConfig(theme, raw)
	if err != nil {
		t.Fatalf("ValidateUIThemeConfig returned error: %v", err)
	}

	var value map[string]any
	if err := json.Unmarshal(normalized, &value); err != nil {
		t.Fatalf("unmarshal normalized config: %v", err)
	}
	if value["primary_color"] != "#2563eb" || value["menu"] != "compact" {
		t.Fatalf("unexpected normalized config: %#v", value)
	}
}

func TestValidateUIThemeConfigRejectsUndeclaredField(t *testing.T) {
	theme := testUIThemeWithConfig(t, []UIThemeConfigField{{Key: "radius", Type: "number"}})
	if _, err := ValidateUIThemeConfig(theme, json.RawMessage(`{"unknown":1}`)); err == nil {
		t.Fatal("expected undeclared field to be rejected")
	}
}

func TestValidateUIThemeConfigRejectsStyleOverride(t *testing.T) {
	theme := testUIThemeWithConfig(t, []UIThemeConfigField{{Key: "style", Type: "select", Options: []string{"luma", "other"}}})
	if _, err := ValidateUIThemeConfig(theme, json.RawMessage(`{"style":"other"}`)); err == nil {
		t.Fatal("expected component style override to be rejected")
	}
}

func TestValidateUIThemeConfigRejectsInvalidType(t *testing.T) {
	theme := testUIThemeWithConfig(t, []UIThemeConfigField{{Key: "glass", Type: "boolean"}})
	if _, err := ValidateUIThemeConfig(theme, json.RawMessage(`{"glass":"yes"}`)); err == nil {
		t.Fatal("expected invalid boolean value to be rejected")
	}
}
