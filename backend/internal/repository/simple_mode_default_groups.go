package repository

import (
	"context"
	"fmt"

	dbent "github.com/WilliamWang1721/LightBridge/ent"
	"github.com/WilliamWang1721/LightBridge/ent/group"
	"github.com/WilliamWang1721/LightBridge/internal/service"
)

func ensureSimpleModeDefaultGroups(ctx context.Context, client *dbent.Client) error {
	if client == nil {
		return fmt.Errorf("nil ent client")
	}

	return createGroupIfNotExists(ctx, client, "default")
}

func createGroupIfNotExists(ctx context.Context, client *dbent.Client, name string) error {
	exists, err := client.Group.Query().
		Where(group.NameEQ(name), group.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check group exists %s: %w", name, err)
	}
	if exists {
		return nil
	}

	_, err = client.Group.Create().
		SetName(name).
		SetDescription("Auto-created default group").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		SetRateMultiplier(1.0).
		SetIsExclusive(false).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			// Concurrent server startups may race on creation; treat as success.
			return nil
		}
		return fmt.Errorf("create default group %s: %w", name, err)
	}
	return nil
}
