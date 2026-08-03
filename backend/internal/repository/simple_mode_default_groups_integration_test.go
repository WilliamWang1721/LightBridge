//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/WilliamWang1721/LightBridge/ent/group"
	"github.com/stretchr/testify/require"
)

func TestEnsureSimpleModeDefaultGroups_CreatesMissingDefaults(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()

	seedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	require.NoError(t, ensureSimpleModeDefaultGroups(seedCtx, client))

	assertGroupExists := func(name string) {
		exists, err := client.Group.Query().Where(group.NameEQ(name), group.DeletedAtIsNil()).Exist(seedCtx)
		require.NoError(t, err)
		require.True(t, exists, "expected group %s to exist", name)
	}

	assertGroupExists("default")
}

func TestEnsureSimpleModeDefaultGroups_IgnoresSoftDeletedGroups(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()

	seedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Migration 008 seeds the neutral default group. Soft-delete that row so
	// this test verifies recreation instead of colliding with the seed data.
	g, err := client.Group.Query().
		Where(group.NameEQ("default"), group.DeletedAtIsNil()).
		Only(seedCtx)
	require.NoError(t, err)

	_, err = client.Group.Delete().Where(group.IDEQ(g.ID)).Exec(seedCtx)
	require.NoError(t, err)

	require.NoError(t, ensureSimpleModeDefaultGroups(seedCtx, client))

	// New active one should exist.
	count, err := client.Group.Query().Where(group.NameEQ("default"), group.DeletedAtIsNil()).Count(seedCtx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestEnsureSimpleModeDefaultGroups_IsIdempotentAndProviderNeutral(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()

	seedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	require.NoError(t, ensureSimpleModeDefaultGroups(seedCtx, client))
	require.NoError(t, ensureSimpleModeDefaultGroups(seedCtx, client))

	count, err := client.Group.Query().Where(group.NameEQ("default"), group.DeletedAtIsNil()).Count(seedCtx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
