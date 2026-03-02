package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"lightbridge/internal/app"
	"lightbridge/internal/db"
	"lightbridge/internal/modules"
	"lightbridge/internal/store"
	"lightbridge/internal/types"
)

var flagErrHelp = errors.New("help requested")

func runModuleCLI(ctx context.Context, cfg app.Config, args []string) error {
	if len(args) == 0 {
		printModuleHelp(os.Stdout)
		return flagErrHelp
	}
	switch args[0] {
	case "install":
		return runModuleInstall(ctx, cfg, args[1:])
	case "help", "-h", "--help":
		printModuleHelp(os.Stdout)
		return flagErrHelp
	default:
		return fmt.Errorf("unknown module command %q", args[0])
	}
}

func runModuleInstall(ctx context.Context, cfg app.Config, args []string) error {
	moduleID, indexURL, version, showHelp, err := parseModuleInstallArgs(args)
	if err != nil {
		printModuleInstallHelp(os.Stderr)
		return err
	}
	if showHelp {
		printModuleInstallHelp(os.Stdout)
		return flagErrHelp
	}

	if err := ensureDataDirs(cfg); err != nil {
		return err
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

	marketplace := modules.NewMarketplace(st, cfg.DataDir, nil)
	resolvedIndexURL := strings.TrimSpace(indexURL)
	if resolvedIndexURL == "" {
		resolvedIndexURL = cfg.ModuleIndexURL
	}
	index, err := marketplace.FetchIndex(ctx, resolvedIndexURL)
	if err != nil {
		return fmt.Errorf("fetch module index failed: %w", err)
	}
	selected, err := selectModuleEntry(index, moduleID, strings.TrimSpace(version))
	if err != nil {
		return err
	}

	installed, manifest, err := marketplace.Install(ctx, *selected)
	if err != nil {
		return fmt.Errorf("install module failed: %w", err)
	}

	fmt.Printf("module installed: %s@%s\n", installed.ID, installed.Version)
	fmt.Printf("install path: %s\n", installed.InstallPath)
	fmt.Printf("entrypoint service count: %d\n", len(manifest.Services))
	fmt.Printf("enabled: %t\n", installed.Enabled)
	return nil
}

func parseModuleInstallArgs(args []string) (moduleID, indexURL, version string, showHelp bool, err error) {
	args = append([]string(nil), args...)
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "help" || arg == "-h" || arg == "--help" {
			return "", "", "", true, nil
		}
		if strings.HasPrefix(arg, "--index=") {
			indexURL = strings.TrimSpace(strings.TrimPrefix(arg, "--index="))
			if indexURL == "" {
				return "", "", "", false, errors.New("--index value cannot be empty")
			}
			continue
		}
		if arg == "--index" {
			if i+1 >= len(args) {
				return "", "", "", false, errors.New("--index requires a value")
			}
			i++
			indexURL = strings.TrimSpace(args[i])
			if indexURL == "" {
				return "", "", "", false, errors.New("--index value cannot be empty")
			}
			continue
		}
		if strings.HasPrefix(arg, "--version=") {
			version = strings.TrimSpace(strings.TrimPrefix(arg, "--version="))
			if version == "" {
				return "", "", "", false, errors.New("--version value cannot be empty")
			}
			continue
		}
		if arg == "--version" {
			if i+1 >= len(args) {
				return "", "", "", false, errors.New("--version requires a value")
			}
			i++
			version = strings.TrimSpace(args[i])
			if version == "" {
				return "", "", "", false, errors.New("--version value cannot be empty")
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", "", "", false, fmt.Errorf("unknown flag %s", arg)
		}
		if moduleID != "" {
			return "", "", "", false, errors.New("only one module id can be provided")
		}
		moduleID = arg
	}
	if strings.TrimSpace(moduleID) == "" {
		return "", "", "", false, errors.New("module id is required")
	}
	return strings.TrimSpace(moduleID), indexURL, version, false, nil
}

func ensureDataDirs(cfg app.Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "modules"), 0o755); err != nil {
		return fmt.Errorf("create module dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "MODULES"), 0o755); err != nil {
		return fmt.Errorf("create local marketplace dir: %w", err)
	}
	return nil
}

func selectModuleEntry(index *types.ModuleIndex, moduleID, version string) (*types.ModuleEntry, error) {
	moduleID = strings.TrimSpace(moduleID)
	version = strings.TrimSpace(version)
	if index == nil {
		return nil, errors.New("module index is empty")
	}

	matches := make([]types.ModuleEntry, 0, 4)
	for _, entry := range index.Modules {
		if strings.TrimSpace(entry.ID) != moduleID {
			continue
		}
		if version != "" && strings.TrimSpace(entry.Version) != version {
			continue
		}
		matches = append(matches, entry)
	}
	if len(matches) == 0 {
		if version != "" {
			return nil, fmt.Errorf("module %s@%s not found in index", moduleID, version)
		}
		return nil, fmt.Errorf("module %s not found in index", moduleID)
	}
	sort.Slice(matches, func(i, j int) bool {
		return compareVersion(matches[i].Version, matches[j].Version) > 0
	})
	selected := matches[0]
	return &selected, nil
}

func compareVersion(a, b string) int {
	aTokens := tokenizeVersion(a)
	bTokens := tokenizeVersion(b)
	maxLen := len(aTokens)
	if len(bTokens) > maxLen {
		maxLen = len(bTokens)
	}
	for i := 0; i < maxLen; i++ {
		aMissing := i >= len(aTokens)
		bMissing := i >= len(bTokens)
		if aMissing || bMissing {
			if aMissing && bMissing {
				continue
			}
			if aMissing {
				return -compareTokenWithMissing(bTokens[i])
			}
			return compareTokenWithMissing(aTokens[i])
		}
		ta := aTokens[i]
		tb := bTokens[i]
		if ta.isNumber && tb.isNumber {
			if ta.num != tb.num {
				if ta.num > tb.num {
					return 1
				}
				return -1
			}
			continue
		}
		if ta.isNumber != tb.isNumber {
			if ta.isNumber {
				return 1
			}
			return -1
		}
		if ta.raw != tb.raw {
			if ta.raw > tb.raw {
				return 1
			}
			return -1
		}
	}
	return 0
}

type versionToken struct {
	isNumber bool
	num      int
	raw      string
}

func tokenizeVersion(v string) []versionToken {
	v = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(v), "v"))
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == '+'
	})
	out := make([]versionToken, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			out = append(out, versionToken{isNumber: true, num: n, raw: p})
			continue
		}
		out = append(out, versionToken{raw: p})
	}
	return out
}

func compareTokenWithMissing(token versionToken) int {
	if token.isNumber {
		if token.num == 0 {
			return 0
		}
		return 1
	}
	// Treat extra string tokens as pre-release suffixes (e.g. 1.0.0-rc1),
	// which are lower precedence than a plain release.
	return -1
}

func printModuleHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "LightBridge module commands")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  lightbridge module install <module-id> [--index <url|local>] [--version <version>]")
}

func printModuleInstallHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Install a module from marketplace index")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  lightbridge module install <module-id> [--index <url|local>] [--version <version>]")
}
