package main

import "testing"

func TestMergeCodexModels(t *testing.T) {
	t.Parallel()

	oldDefaults := []string{
		"gpt-5-codex",
		"gpt-5-codex-mini",
		"gpt-5.1-codex",
		"gpt-5.1-codex-mini",
		"gpt-5.2-codex",
	}
	merged := mergeCodexModels(oldDefaults)

	wantSet := map[string]bool{
		"gpt-5-codex":        true,
		"gpt-5-codex-mini":   true,
		"gpt-5.1-codex":      true,
		"gpt-5.1-codex-mini": true,
		"gpt-5.1-codex-max":  true,
		"gpt-5.2-codex":      true,
		"gpt-5.2":            true,
		"gpt-5.3-codex":      true,
	}
	gotSet := make(map[string]bool, len(merged))
	for _, id := range merged {
		gotSet[id] = true
	}
	for id := range wantSet {
		if !gotSet[id] {
			t.Fatalf("expected model %q to exist in merged list, got=%v", id, merged)
		}
	}
}

func TestMergeCodexModelsKeepsCustom(t *testing.T) {
	t.Parallel()

	custom := []string{"custom-codex-model", "GPT-5.3-CODEX"}
	merged := mergeCodexModels(custom)

	var hasCustom bool
	var codex53Count int
	for _, id := range merged {
		if id == "custom-codex-model" {
			hasCustom = true
		}
		if id == "gpt-5.3-codex" || id == "GPT-5.3-CODEX" {
			codex53Count++
		}
	}
	if !hasCustom {
		t.Fatalf("expected custom model to be preserved, got=%v", merged)
	}
	if codex53Count != 1 {
		t.Fatalf("expected gpt-5.3-codex to be deduplicated, got count=%d list=%v", codex53Count, merged)
	}
}
