package admin

import (
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/stretchr/testify/require"
)

func TestValidDistributionAudience(t *testing.T) {
	tests := []struct {
		name     string
		audience service.DistributionAudienceInput
		want     bool
	}{
		{name: "explicit user ids", audience: service.DistributionAudienceInput{UserIDs: []int64{1}}, want: true},
		{name: "explicit emails", audience: service.DistributionAudienceInput{Emails: []string{"user@example.com"}}, want: true},
		{name: "non-empty filters", audience: service.DistributionAudienceInput{Filters: &service.DistributionUserFilters{Activity: "none"}}, want: true},
		{name: "lines", audience: service.DistributionAudienceInput{Lines: "1 | title | content"}, want: true},
		{name: "all", audience: service.DistributionAudienceInput{All: true}, want: true},
		{name: "empty", audience: service.DistributionAudienceInput{}, want: false},
		{name: "empty filters", audience: service.DistributionAudienceInput{Filters: &service.DistributionUserFilters{}}, want: false},
		{name: "mixed explicit and lines", audience: service.DistributionAudienceInput{UserIDs: []int64{1}, Lines: "2"}, want: false},
		{name: "mixed all and filters", audience: service.DistributionAudienceInput{All: true, Filters: &service.DistributionUserFilters{Role: "user"}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, validDistributionAudience(tt.audience))
		})
	}
}
