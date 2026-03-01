package gateway

import "testing"

func TestNormalizeVoucherConfigCompatibilityFields(t *testing.T) {
	in := voucherConfig{
		BaseURL: "https://gateway.example.com/openai/v1/",
	}
	out := normalizeVoucherConfig(in)

	if out.BaseOrigin != "https://gateway.example.com/openai/v1" {
		t.Fatalf("base_origin mismatch: %q", out.BaseOrigin)
	}
	if out.BaseURL != out.BaseOrigin {
		t.Fatalf("base_url should mirror base_origin for compatibility: base_url=%q base_origin=%q", out.BaseURL, out.BaseOrigin)
	}
	if out.DefaultInterface == "" {
		t.Fatalf("default_interface should be set")
	}
	if len(out.Apps) < len(defaultVoucherConfig().Apps) {
		t.Fatalf("default apps should be present, got=%d", len(out.Apps))
	}
}

func TestNormalizeVoucherConfigDeduplicatesMappings(t *testing.T) {
	in := voucherConfig{
		BaseOrigin:       "https://gateway.example.com/openai",
		DefaultInterface: "gemini",
		Apps: map[string]voucherAppConfig{
			"Codex": {
				KeyID: "  key_1  ",
				ModelMappings: []voucherModelMapping{
					{From: "gpt-4o", To: "gpt-4o-mini"},
					{From: " gpt-4o ", To: " gpt-4o-mini "},
					{From: "", To: "invalid"},
				},
			},
		},
	}
	out := normalizeVoucherConfig(in)
	app := out.Apps["codex"]

	if app.KeyID != "key_1" {
		t.Fatalf("key_id should be trimmed, got=%q", app.KeyID)
	}
	if len(app.ModelMappings) != 1 {
		t.Fatalf("expected deduplicated mappings length 1, got %d", len(app.ModelMappings))
	}
	if app.ModelMappings[0].From != "gpt-4o" || app.ModelMappings[0].To != "gpt-4o-mini" {
		t.Fatalf("unexpected normalized mapping: %+v", app.ModelMappings[0])
	}
	if out.DefaultInterface != "gemini" {
		t.Fatalf("default interface should be preserved, got=%q", out.DefaultInterface)
	}
}
