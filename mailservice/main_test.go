package main

import (
	"path/filepath"
	"testing"
)

func TestStorePersistsMailboxBindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lbms-store.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	mailboxA, bindingA, err := store.LinkOrCreate(LinkOrCreateRequest{
		EmailAddress:           "AA@qq.com",
		LightBridgeAccountID:   101,
		LightBridgePlatform:    "openai",
		LightBridgeAccountType: "oauth",
		LightBridgeAccountName: "OpenAI A",
	})
	if err != nil {
		t.Fatalf("LinkOrCreate first account: %v", err)
	}
	if mailboxA.NormalizedEmail != "aa@qq.com" {
		t.Fatalf("normalized email = %q", mailboxA.NormalizedEmail)
	}
	if bindingA.MailboxID != mailboxA.ID {
		t.Fatalf("binding mailbox id = %q, want %q", bindingA.MailboxID, mailboxA.ID)
	}

	mailboxB, _, err := store.LinkOrCreate(LinkOrCreateRequest{
		EmailAddress:           "aa@qq.com",
		LightBridgeAccountID:   102,
		LightBridgePlatform:    "gemini",
		LightBridgeAccountType: "oauth",
		LightBridgeAccountName: "Gemini B",
	})
	if err != nil {
		t.Fatalf("LinkOrCreate second account: %v", err)
	}
	if mailboxB.ID != mailboxA.ID {
		t.Fatalf("same email created different mailbox: %q != %q", mailboxB.ID, mailboxA.ID)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore reload: %v", err)
	}
	mailbox, bindings, ok := reloaded.BindingsByMailbox(mailboxA.ID)
	if !ok {
		t.Fatalf("mailbox %q was not reloaded", mailboxA.ID)
	}
	if mailbox.EmailAddress != "AA@qq.com" {
		t.Fatalf("email address = %q", mailbox.EmailAddress)
	}
	if len(bindings) != 2 {
		t.Fatalf("binding count = %d, want 2", len(bindings))
	}
	if _, _, ok := reloaded.BindingByAccount(101); !ok {
		t.Fatalf("account 101 binding was not reloaded")
	}
	if _, _, ok := reloaded.BindingByAccount(102); !ok {
		t.Fatalf("account 102 binding was not reloaded")
	}
}

func TestStoreMovesAccountBetweenMailboxes(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "lbms-store.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	first, _, err := store.LinkOrCreate(LinkOrCreateRequest{
		EmailAddress:           "first@example.com",
		LightBridgeAccountID:   201,
		LightBridgePlatform:    "openai",
		LightBridgeAccountType: "oauth",
	})
	if err != nil {
		t.Fatalf("link first: %v", err)
	}
	second, _, err := store.LinkOrCreate(LinkOrCreateRequest{
		EmailAddress:           "second@example.com",
		LightBridgeAccountID:   201,
		LightBridgePlatform:    "openai",
		LightBridgeAccountType: "oauth",
	})
	if err != nil {
		t.Fatalf("move account: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected account to move to a different mailbox")
	}

	_, oldBindings, ok := store.BindingsByMailbox(first.ID)
	if !ok {
		t.Fatalf("first mailbox missing")
	}
	if len(oldBindings) != 0 {
		t.Fatalf("old mailbox still has %d bindings", len(oldBindings))
	}
	_, newBindings, ok := store.BindingsByMailbox(second.ID)
	if !ok {
		t.Fatalf("second mailbox missing")
	}
	if len(newBindings) != 1 || newBindings[0].LightBridgeAccountID != 201 {
		t.Fatalf("new mailbox bindings = %#v", newBindings)
	}
}
