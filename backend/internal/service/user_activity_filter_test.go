package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUserActivityFilter(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       string
		wantValid  bool
	}{
		{name: "empty", input: "", want: "", wantValid: true},
		{name: "any", input: "any", want: UserActivityFilterAny, wantValid: true},
		{name: "usage alias", input: "USED", want: UserActivityFilterUsage, wantValid: true},
		{name: "balance alias", input: "balance", want: UserActivityFilterBalanceChange, wantValid: true},
		{name: "none alias", input: "no_activity", want: UserActivityFilterNone, wantValid: true},
		{name: "invalid", input: "recent", want: "", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := NormalizeUserActivityFilter(tt.input)
			require.Equal(t, tt.wantValid, valid)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUserActivitySearchToken(t *testing.T) {
	require.Equal(t, "@activity:any", UserActivitySearchToken("active"))
	require.Equal(t, "@activity:balance_change", UserActivitySearchToken("balance"))
	require.Empty(t, UserActivitySearchToken("unknown"))
}
