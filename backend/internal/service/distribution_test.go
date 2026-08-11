package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDistributionImportLines(t *testing.T) {
	lines, err := ParseDistributionImportLines(`
# comment
1001
user@example.com
1002 | Personal title | Personal content
second@example.com | Notice
`)
	require.NoError(t, err)
	require.Equal(t, []DistributionImportLine{
		{UserID: 1001},
		{Email: "user@example.com"},
		{UserID: 1002, TitleOverride: "Personal title", ContentOverride: "Personal content"},
		{Email: "second@example.com", TitleOverride: "Notice"},
	}, lines)
}

func TestParseDistributionImportLinesRejectsInvalidRecipient(t *testing.T) {
	_, err := ParseDistributionImportLines("not-a-recipient")
	require.Error(t, err)
	require.ErrorContains(t, err, "recipient must be a user id or email")
}

func TestDistributionKindValidation(t *testing.T) {
	for _, kind := range []string{
		DistributionKindText,
		DistributionKindMessage,
		DistributionKindFile,
		DistributionKindAccountExport,
		DistributionKindPaid,
	} {
		require.True(t, isDistributionKind(kind), kind)
	}
	require.False(t, isDistributionKind("unknown"))
}

func TestAccountExportFileValidation(t *testing.T) {
	require.True(t, isAccountExportFileName("accounts.json"))
	require.True(t, isAccountExportFileName("bundle.ZIP"))
	require.False(t, isAccountExportFileName("accounts.txt"))
}

func TestSanitizeDistributionFileName(t *testing.T) {
	require.Equal(t, "folder_file.json", sanitizeDistributionFileName(`folder/file.json`))
	require.Equal(t, "folder_file.json", sanitizeDistributionFileName(`folder\file.json`))
}

func TestAllDistributionRecipientsHaveContent(t *testing.T) {
	require.True(t, allDistributionRecipientsHaveContent([]DistributionRecipient{
		{UserID: 1, ContentOverride: "CDK-A"},
		{UserID: 2, ContentOverride: "CDK-B"},
	}))
	require.False(t, allDistributionRecipientsHaveContent([]DistributionRecipient{
		{UserID: 1, ContentOverride: "CDK-A"},
		{UserID: 2},
	}))
}

func TestDistributionPriceValidation(t *testing.T) {
	require.True(t, validDistributionPrice(DistributionKindPaid, 1.25))
	require.False(t, validDistributionPrice(DistributionKindPaid, 0))
	require.True(t, validDistributionPrice(DistributionKindMessage, 0))
	require.False(t, validDistributionPrice(DistributionKindMessage, 1.25))
}
