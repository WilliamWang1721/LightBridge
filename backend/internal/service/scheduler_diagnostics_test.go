package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestProtocolFilterReturnsOpsOnlyRequestTimeDiagnostics(t *testing.T) {
	ctx := WithInboundProtocol(context.Background(), CustomProtocolAnthropicMessages)
	account := Account{ID: 7, Name: "responses passthrough", Platform: PlatformCustom, Status: StatusActive, Schedulable: true, Extra: map[string]any{"protocol": CustomProtocolOpenAIResponses, "relay_mode": RelayModeFullPassthrough}}
	_, err := filterAccountsByRequestProtocolForScheduling(ctx, nil, PlatformAnthropic, []Account{account})
	if err == nil {
		t.Fatal("expected protocol rejection error")
	}
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("expected ErrNoAvailableAccounts, got %v", err)
	}
	if strings.Contains(err.Error(), schedulerDiagnosticsMarker) || strings.Contains(err.Error(), account.Name) {
		t.Fatalf("client-facing error leaked scheduler diagnostics: %q", err.Error())
	}

	detail := SchedulerDiagnosticsFromError(err)
	if !strings.HasPrefix(detail, schedulerDiagnosticsMarker) {
		t.Fatalf("ops diagnostics marker missing from %q", detail)
	}
	encoded := strings.TrimPrefix(detail, schedulerDiagnosticsMarker)
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
