package handler

import (
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSanitizeDistributionForUser(t *testing.T) {
	createdBy := int64(42)
	item := &service.Distribution{
		Audience:  map[string]any{"preview": []any{"other@example.com"}},
		Metadata:  map[string]any{"internal": true},
		CreatedBy: &createdBy,
		Title:     "visible",
		Content:   "visible content",
	}

	sanitizeDistributionForUser(item)

	require.Nil(t, item.Audience)
	require.Nil(t, item.Metadata)
	require.Nil(t, item.CreatedBy)
	require.Equal(t, "visible", item.Title)
	require.Equal(t, "visible content", item.Content)
}
