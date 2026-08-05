package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeDistributionRecipientPreservesPersonalization(t *testing.T) {
	personalized := mergeDistributionRecipient(
		DistributionRecipient{},
		42,
		"Personal title",
		"Personal content",
	)

	mergedFromBroadAudience := mergeDistributionRecipient(personalized, 42, "", "")
	require.Equal(t, personalized, mergedFromBroadAudience)

	updatedTitle := mergeDistributionRecipient(mergedFromBroadAudience, 42, "Updated title", "")
	require.Equal(t, "Updated title", updatedTitle.TitleOverride)
	require.Equal(t, "Personal content", updatedTitle.ContentOverride)
}
