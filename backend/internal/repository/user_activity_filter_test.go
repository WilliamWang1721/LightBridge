package repository

import (
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSplitUserAdvancedSearch(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantSearch   string
		wantActivity string
	}{
		{
			name:         "activity only",
			input:        "@activity:any",
			wantActivity: service.UserActivityFilterAny,
		},
		{
			name:         "text and activity",
			input:        "customer@example.com @activity:none",
			wantSearch:   "customer@example.com",
			wantActivity: service.UserActivityFilterNone,
		},
		{
			name:         "alias is normalized",
			input:        "@activity:balance",
			wantActivity: service.UserActivityFilterBalanceChange,
		},
		{
			name:         "last valid token wins",
			input:        "@activity:usage alice @activity:any",
			wantSearch:   "alice",
			wantActivity: service.UserActivityFilterAny,
		},
		{
			name:       "unknown token stays searchable",
			input:      "@activity:recent",
			wantSearch: "@activity:recent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			search, activity := splitUserAdvancedSearch(tt.input)
			require.Equal(t, tt.wantSearch, search)
			require.Equal(t, tt.wantActivity, activity)
		})
	}
}
