package main

import "testing"

func TestParseAwsEventStreamBuffer(t *testing.T) {
	raw := "\x00\x01garbage{" +
		`"content":"hello "}` +
		"\x00tail{" +
		`"name":"tool_a","toolUseId":"tu_1"}` +
		"{" +
		`"input":"{\"x\":1"}` +
		"{" +
		`"input":"}"}` +
		"{" +
		`"stop":true}` +
		"{" +
		`"content":"world"}`
	events, remain := parseAwsEventStreamBuffer(raw)
	if len(events) < 4 {
		t.Fatalf("expected >=4 events, got %d", len(events))
	}
	if remain != "" {
		t.Fatalf("expected empty remain, got %q", remain)
	}
}

func TestParseKiroMixedResponse(t *testing.T) {
	raw := `{"content":"hello"}{"name":"tool","toolUseId":"t1"}{"input":"{\"a\":1}"}{"stop":true}`
	text, toolCalls, _ := parseKiroMixedResponse(raw)
	if text != "hello" {
		t.Fatalf("unexpected text: %q", text)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].Name != "tool" {
		t.Fatalf("unexpected tool name: %s", toolCalls[0].Name)
	}
}
