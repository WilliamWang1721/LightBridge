package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"lightbridge/internal/app"
)

func main() {
	cfg, err := app.DefaultConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("lightbridge exited with error: %v", err)
	}
}
