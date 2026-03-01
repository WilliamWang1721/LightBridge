package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lightbridge/internal/db"
	"lightbridge/internal/types"
)

func TestEnsureBuiltinProviders_SkipsRemovedBuiltinProviders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	st := New(database)

	if err := st.SetSetting(ctx, "builtin_provider_removed:forward", "1"); err != nil {
		t.Fatalf("set removed setting: %v", err)
	}
	if err := st.EnsureBuiltinProviders(ctx); err != nil {
		t.Fatalf("ensure builtin providers: %v", err)
	}

	forward, err := st.GetProvider(ctx, "forward")
	if err != nil {
		t.Fatalf("get forward provider: %v", err)
	}
	if forward != nil {
		t.Fatalf("expected forward provider to be skipped, got %+v", *forward)
	}

	anthropic, err := st.GetProvider(ctx, "anthropic")
	if err != nil {
		t.Fatalf("get anthropic provider: %v", err)
	}
	if anthropic == nil {
		t.Fatalf("expected anthropic provider to exist")
	}
}

func TestPathModelUsageSince(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	st := New(database)
	now := time.Now().UTC()

	mustInsert := func(meta types.RequestLogMeta) {
		t.Helper()
		if err := st.InsertRequestLog(ctx, meta); err != nil {
			t.Fatalf("insert request log: %v", err)
		}
	}

	mustInsert(types.RequestLogMeta{
		Timestamp:    now.Add(-2 * time.Hour),
		RequestID:    "r1",
		ProviderID:   "forward",
		ModelID:      "gpt-4o-mini",
		Path:         "/v1/chat/completions",
		Status:       200,
		InputTokens:  10,
		OutputTokens: 3,
	})
	mustInsert(types.RequestLogMeta{
		Timestamp:    now.Add(-90 * time.Minute),
		RequestID:    "r2",
		ProviderID:   "forward",
		ModelID:      "gpt-4o-mini",
		Path:         "/v1/chat/completions",
		Status:       200,
		InputTokens:  5,
		OutputTokens: 2,
	})
	mustInsert(types.RequestLogMeta{
		Timestamp:    now.Add(-30 * time.Minute),
		RequestID:    "r3",
		ProviderID:   "anthropic",
		ModelID:      "claude-3-5-sonnet",
		Path:         "/v1/responses",
		Status:       200,
		InputTokens:  4,
		OutputTokens: 6,
	})

	rows, err := st.PathModelUsageSince(ctx, now.Add(-24*time.Hour), 20)
	if err != nil {
		t.Fatalf("path model usage: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 grouped rows, got %d", len(rows))
	}

	if rows[0].Path != "/v1/chat/completions" || rows[0].ModelID != "gpt-4o-mini" {
		t.Fatalf("unexpected top row identity: %+v", rows[0])
	}
	if rows[0].Requests != 2 || rows[0].InputTokens != 15 || rows[0].OutputTokens != 5 {
		t.Fatalf("unexpected top row aggregate: %+v", rows[0])
	}
}

func TestChatConversationPersistence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	st := New(database)
	conversationID := "chat_test_001"

	if err := st.CreateChatConversation(ctx, types.ChatConversation{
		ID:           conversationID,
		Title:        "新对话",
		ModelID:      "gpt-4o-mini",
		SystemPrompt: "你是测试助手",
	}); err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	if err := st.AppendChatExchange(
		ctx,
		conversationID,
		"gpt-4o-mini",
		"测试标题",
		"请给我一段 Markdown 列表",
		"- 第一项\n- 第二项",
		"思维链样例",
		"forward",
		"gpt-4o-mini",
	); err != nil {
		t.Fatalf("append exchange: %v", err)
	}

	conv, err := st.GetChatConversation(ctx, conversationID)
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	if conv == nil {
		t.Fatalf("conversation should exist")
	}
	if conv.Title != "测试标题" {
		t.Fatalf("unexpected title: %q", conv.Title)
	}
	if conv.MessageCount != 2 {
		t.Fatalf("expected 2 messages, got %d", conv.MessageCount)
	}
	if !strings.Contains(conv.LastMessagePreview, "第二项") {
		t.Fatalf("unexpected last message preview: %q", conv.LastMessagePreview)
	}

	msgs, err := st.ListChatMessages(ctx, conversationID, 20)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("unexpected role order: %#v", []string{msgs[0].Role, msgs[1].Role})
	}
	if msgs[1].ReasoningText != "思维链样例" {
		t.Fatalf("unexpected reasoning text: %q", msgs[1].ReasoningText)
	}
}
