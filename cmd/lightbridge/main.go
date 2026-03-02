package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lightbridge/internal/app"
)

func main() {
	cfg, err := app.DefaultConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if len(os.Args) > 1 {
		ctx := context.Background()
		if err := runCLI(ctx, cfg, os.Args[1:]); err != nil {
			if errors.Is(err, flagErrHelp) {
				os.Exit(0)
			}
			log.Fatalf("cli error: %v", err)
		}
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("lightbridge exited with error: %v", err)
	}
}

func runCLI(ctx context.Context, cfg app.Config, args []string) error {
	switch args[0] {
	case "module":
		return runModuleCLI(ctx, cfg, args[1:])
	case "help", "-h", "--help":
		printRootHelp(os.Stdout)
		return flagErrHelp
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printRootHelp(w *os.File) {
	_, _ = fmt.Fprintln(w, "LightBridge CLI")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  lightbridge                 Start gateway server")
	_, _ = fmt.Fprintln(w, "  lightbridge module install <module-id> [--index <url|local>] [--version <version>]")
}
