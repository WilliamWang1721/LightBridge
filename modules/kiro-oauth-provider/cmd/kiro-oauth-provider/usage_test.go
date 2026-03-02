package main

import "testing"

func TestNormalizeUsageLimits(t *testing.T) {
	raw := map[string]any{
		"nextDateReset": float64(1767225600),
		"usageBreakdownList": []any{
			map[string]any{
				"resourceType":              "AGENTIC_REQUEST",
				"displayName":               "Requests",
				"currentUsageWithPrecision": 42.0,
				"usageLimitWithPrecision":   100.0,
				"nextDateReset":             float64(1767225600),
			},
		},
	}
	q := normalizeUsageLimits(raw)
	if len(q.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(q.Items))
	}
	if q.UsedPercent != 42 {
		t.Fatalf("expected used 42, got %v", q.UsedPercent)
	}
	if q.RemainingPercent != 58 {
		t.Fatalf("expected remaining 58, got %v", q.RemainingPercent)
	}
	if q.ResetAt == "" {
		t.Fatalf("expected reset_at")
	}
}

func TestParseUnitNumber(t *testing.T) {
	if got := parseUnitNumber("1.5k"); got != 1500 {
		t.Fatalf("expected 1500, got %v", got)
	}
	if got := parseUnitNumber("2 mega"); got != 2e6 {
		t.Fatalf("expected 2e6, got %v", got)
	}
}
