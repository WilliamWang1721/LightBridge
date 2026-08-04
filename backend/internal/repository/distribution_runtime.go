package repository

import (
	"database/sql"
	"errors"
	"sync"

	"github.com/WilliamWang1721/LightBridge/ent"
	"github.com/WilliamWang1721/LightBridge/internal/service"
)

var distributionRuntime struct {
	sync.RWMutex
	client    *ent.Client
	db        *sql.DB
	encryptor service.SecretEncryptor
}

func registerDistributionEntClient(client *ent.Client) {
	distributionRuntime.Lock()
	distributionRuntime.client = client
	distributionRuntime.Unlock()
}

func registerDistributionSQLDB(db *sql.DB) {
	distributionRuntime.Lock()
	distributionRuntime.db = db
	distributionRuntime.Unlock()
}

func registerDistributionEncryptor(encryptor service.SecretEncryptor) {
	distributionRuntime.Lock()
	distributionRuntime.encryptor = encryptor
	distributionRuntime.Unlock()
}

// BuildRuntimeDistributionService builds the distribution service from the same
// Ent client, SQL pool, and encryptor already initialized by the application.
// This keeps the feature isolated from the generated Wire graph while still
// sharing the application's lifecycle-managed dependencies.
func BuildRuntimeDistributionService() (*service.DistributionService, error) {
	distributionRuntime.RLock()
	client := distributionRuntime.client
	db := distributionRuntime.db
	encryptor := distributionRuntime.encryptor
	distributionRuntime.RUnlock()

	if client == nil || db == nil || encryptor == nil {
		return nil, errors.New("distribution runtime dependencies are not initialized")
	}

	userRepo := newUserRepositoryWithSQL(client, db)
	distributionRepo := NewDistributionRepository(db)
	return service.NewDistributionService(distributionRepo, userRepo, encryptor), nil
}
