package app

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"lightbridge/internal/db"
	"lightbridge/internal/gateway"
	"lightbridge/internal/modules"
	"lightbridge/internal/providers"
	"lightbridge/internal/routing"
	"lightbridge/internal/store"
)

func Run(ctx context.Context, cfg Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "modules"), 0o755); err != nil {
		return fmt.Errorf("create module dir: %w", err)
	}

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer database.Close()

	st := store.New(database)
	if err := st.EnsureBuiltinProviders(ctx); err != nil {
		return err
	}
	if err := st.EnsureDefaultModels(ctx); err != nil {
		return err
	}

	resolver := routing.NewResolver(st, rand.New(rand.NewSource(42)))
	providerRegistry := providers.NewRegistry(
		providers.NewHTTPForwardAdapter("forward", nil),
		providers.NewHTTPForwardAdapter("http_openai", nil),
		providers.NewHTTPForwardAdapter("http_rpc", nil),
		providers.NewAnthropicAdapter(nil),
		providers.NewGRPCChatAdapter(),
	)
	moduleMgr := modules.NewManager(st, cfg.DataDir)
	marketplace := modules.NewMarketplace(st, cfg.DataDir, nil)

	server, err := gateway.New(gateway.Config{
		ListenAddr:     cfg.ListenAddr,
		ModuleIndexURL: cfg.ModuleIndexURL,
	}, st, resolver, providerRegistry, marketplace, moduleMgr, cfg.CookieSecretKey)
	if err != nil {
		return err
	}

	_ = moduleMgr.StartEnabledModules(ctx)
	return server.ListenAndServe(ctx)
}
