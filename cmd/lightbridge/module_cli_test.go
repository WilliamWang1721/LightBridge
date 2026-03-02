package main

import (
	"testing"

	"lightbridge/internal/types"
)

func TestSelectModuleEntryByLatestVersion(t *testing.T) {
	index := &types.ModuleIndex{
		Modules: []types.ModuleEntry{
			{ID: "openai-codex-oauth", Version: "0.1.0"},
			{ID: "openai-codex-oauth", Version: "0.2.0"},
		},
	}

	got, err := selectModuleEntry(index, "openai-codex-oauth", "")
	if err != nil {
		t.Fatalf("selectModuleEntry returned error: %v", err)
	}
	if got.Version != "0.2.0" {
		t.Fatalf("expected latest version 0.2.0, got %s", got.Version)
	}
}

func TestSelectModuleEntryByExactVersion(t *testing.T) {
	index := &types.ModuleIndex{
		Modules: []types.ModuleEntry{
			{ID: "passkey-login", Version: "0.1.0"},
			{ID: "passkey-login", Version: "0.2.0"},
		},
	}

	got, err := selectModuleEntry(index, "passkey-login", "0.1.0")
	if err != nil {
		t.Fatalf("selectModuleEntry returned error: %v", err)
	}
	if got.Version != "0.1.0" {
		t.Fatalf("expected version 0.1.0, got %s", got.Version)
	}
}

func TestParseModuleInstallArgsSupportsMixedOrder(t *testing.T) {
	moduleID, indexURL, version, showHelp, err := parseModuleInstallArgs([]string{
		"openai-codex-oauth",
		"--index",
		"local",
		"--version",
		"0.2.0",
	})
	if err != nil {
		t.Fatalf("parseModuleInstallArgs returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("expected showHelp false")
	}
	if moduleID != "openai-codex-oauth" {
		t.Fatalf("unexpected moduleID: %s", moduleID)
	}
	if indexURL != "local" {
		t.Fatalf("unexpected indexURL: %s", indexURL)
	}
	if version != "0.2.0" {
		t.Fatalf("unexpected version: %s", version)
	}
}

func TestParseModuleInstallArgsFlagThenModuleID(t *testing.T) {
	moduleID, indexURL, version, showHelp, err := parseModuleInstallArgs([]string{
		"--index=local",
		"passkey-login",
	})
	if err != nil {
		t.Fatalf("parseModuleInstallArgs returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("expected showHelp false")
	}
	if moduleID != "passkey-login" {
		t.Fatalf("unexpected moduleID: %s", moduleID)
	}
	if indexURL != "local" {
		t.Fatalf("unexpected indexURL: %s", indexURL)
	}
	if version != "" {
		t.Fatalf("expected empty version, got %s", version)
	}
}
