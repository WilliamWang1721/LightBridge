package modules

import "testing"

func TestSkipAutoProviderAliasRegistration(t *testing.T) {
	cases := []struct {
		moduleID string
		want     bool
	}{
		{moduleID: "kiro-oauth-provider", want: true},
		{moduleID: "KIRO-OAUTH-PROVIDER", want: true},
		{moduleID: " openai-codex-oauth ", want: false},
		{moduleID: "", want: false},
	}

	for _, tc := range cases {
		got := skipAutoProviderAliasRegistration(tc.moduleID)
		if got != tc.want {
			t.Fatalf("moduleID=%q: want %v, got %v", tc.moduleID, tc.want, got)
		}
	}
}
