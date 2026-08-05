package repository

import (
	"context"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/pagination"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryListWithActivityFilters(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	_, err := repo.sql.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS user_affiliate_ledger (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL
)`)
	require.NoError(t, err)

	createUser := func(email string) service.User {
		user := service.User{
			Email:        email,
			Username:     email,
			PasswordHash: "hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
		}
		require.NoError(t, repo.Create(ctx, &user))
		return user
	}

	balanceUser := createUser("balance@example.com")
	affiliateUser := createUser("affiliate@example.com")
	unusedUser := createUser("unused@example.com")

	_, err = client.RedeemCode.Create().
		SetCode("BALANCE-ACTIVITY-TEST").
		SetType(service.RedeemTypeBalance).
		SetStatus(service.StatusUsed).
		SetUsedBy(balanceUser.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = repo.sql.ExecContext(ctx,
		"INSERT INTO user_affiliate_ledger (user_id, action) VALUES ($1, $2)",
		affiliateUser.ID,
		"transfer",
	)
	require.NoError(t, err)

	params := pagination.PaginationParams{Page: 1, PageSize: 50}
	listIDs := func(activity string) []int64 {
		users, _, listErr := repo.ListWithFilters(ctx, params, service.UserListFilters{
			Search: service.UserActivitySearchToken(activity),
		})
		require.NoError(t, listErr)
		ids := make([]int64, 0, len(users))
		for _, user := range users {
			ids = append(ids, user.ID)
		}
		return ids
	}

	require.ElementsMatch(t,
		[]int64{balanceUser.ID, affiliateUser.ID},
		listIDs(service.UserActivityFilterAny),
	)
	require.ElementsMatch(t,
		[]int64{balanceUser.ID, affiliateUser.ID},
		listIDs(service.UserActivityFilterBalanceChange),
	)
	require.Empty(t, listIDs(service.UserActivityFilterUsage))
	require.Equal(t,
		[]int64{unusedUser.ID},
		listIDs(service.UserActivityFilterNone),
	)
}
