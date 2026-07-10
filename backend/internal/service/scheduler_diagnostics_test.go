package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestProtocolFilterReturnsRequestTimeDiagnostics(t *testing.T) {
	ctx := WithInboundProtocol(context.Background(), CustomProtocolAnthropicMessages)
	account := Account{ID: 7, Name: "responses passthrough", Platform: PlatformCustom, Status: StatusActive, Schedulable: true, Extra: map[string]any{"protocol": CustomProtocolOpenAIResponses, "relay_mode": RelayModeFullPassthrough}}
	_, err := filterAccountsByRequestProtocolForScheduling(ctx, nil, PlatformAnthropic, []Account{account})
	if err == nil {
		t.Fatal("expected protocol rejection error")
	}
	message := err.Error()
	idx := strings.Index(message, schedulerDiagnosticsMarker)
	if idx < 0 {
		t.Fatalf("diagnostics marker missing from %q", message)
	}
	encoded := strings.TrimRight(strings.Fields(message[idx+len(schedulerDiagnosticsMarker):])[0], ")")
	payload, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
	if decodeErr != nil {
		t.Fatalf("decode diagnostics: %v", decodeErr)
	}
	var envelope schedulerDiagnosticsEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("unmarshal diagnostics: %v", err)
	}
	if len(envelope.Accounts) != 1 || envelope.Accounts[0].Reason != "relay_mode_protocol_mismatch" {
		t.Fatalf("unexpected diagnostics: %+v", envelope.Accounts)
	}
}
