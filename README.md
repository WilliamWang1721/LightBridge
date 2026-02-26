# LightBridge (Go MVP v0.1)

LightBridge is a microkernel-style AI gateway with an OpenAI-compatible downstream API (`/v1/*`), multi-provider routing (`model` / `model@provider`), and an installable module marketplace.

This repository now contains a runnable Go implementation for the v0.1 scope in your plan:

- Single-port service (default `127.0.0.1:3210`)
- OpenAI-compatible gateway endpoints:
  - `GET /v1/models`
  - `POST /v1/chat/completions`
  - `POST/GET /v1/*` passthrough for forward/http_openai providers
- Provider system:
  - Built-in `forward` provider (`/v1/*` passthrough)
  - Built-in `anthropic` provider (`/v1/chat/completions` conversion, stream/non-stream)
  - Module providers (`http_openai`, `http_rpc`, `grpc_chat` placeholder)
- Routing:
  - Virtual model routing table (`models` + `model_routes`)
  - Variant routing with `model@providerAlias`
  - Priority + weight + health filtering
  - Fallback: `claude-* -> anthropic`, others -> `forward`
- Module marketplace/runtime:
  - Fetch `index.json`
  - ZIP download + SHA256 verification
  - `manifest.json` validation
  - Install + enable + process launch + health check
  - Module provider alias registration
- Admin web pages and admin APIs (`/admin/*`, `/admin/api/*`)
- SQLite schema + migrations (idempotent)
- Metadata-only request logs

## Project layout

- `cmd/lightbridge/main.go` - entrypoint
- `internal/app` - service wiring / startup
- `internal/db` - SQLite open + migration
- `internal/store` - persistence operations
- `internal/routing` - model routing and model list assembly
- `internal/providers` - provider adapters (`forward`, `anthropic`, `grpc_chat`)
- `internal/modules` - marketplace + module runtime manager
- `internal/gateway` - HTTP gateway + admin endpoints + templates
- `tests/testdata/module-sample` - sample provider module source used by integration tests

## Run

```bash
go run ./cmd/lightbridge
```

Default bind: `127.0.0.1:3210`

Default data directory:
- macOS/Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/LightBridge`
- Override with env: `LIGHTBRIDGE_DATA_DIR=/path/to/data`

Optional env vars:

- `LIGHTBRIDGE_ADDR=127.0.0.1:3210`
- `LIGHTBRIDGE_MODULE_INDEX=https://.../index.json`
- `LIGHTBRIDGE_COOKIE_SECRET=...`

## First-time setup

1. Open `http://127.0.0.1:3210/admin/setup`
2. Create admin username/password
3. Copy generated default client API key
4. Use that key for downstream `Authorization: Bearer <key>`

## Admin APIs (MVP)

- `POST /admin/api/setup`
- `POST /admin/api/login`
- `GET/POST /admin/api/providers`
- `GET/POST /admin/api/models`
- `GET /admin/api/dashboard`
- `GET /admin/api/logs`
- `GET /admin/api/marketplace/index`
- `POST /admin/api/marketplace/install`
- `POST /admin/api/modules/start`
- `POST /admin/api/modules/stop`

## Module manifest (implemented fields)

Required:

- `id`, `name`, `version`, `license`, `min_core_version`
- `entrypoints` (`<os>/<arch>`, `<os>`, or `default`)
- `services[]` with:
  - `kind: "provider"`
  - `protocol: "http_openai" | "http_rpc" | "grpc_chat"`
  - `health`
  - `expose_provider_aliases[]`
- `config_schema`
- `config_defaults`

Core injects env vars on module start:

- `LIGHTBRIDGE_MODULE_ID`
- `LIGHTBRIDGE_DATA_DIR`
- `LIGHTBRIDGE_CONFIG_PATH`
- `LIGHTBRIDGE_HTTP_PORT`
- `LIGHTBRIDGE_GRPC_PORT`
- `LIGHTBRIDGE_LOG_LEVEL`

## Tests

```bash
go test ./...
```

Current tests cover:

- Routing parse/fallback/priority/weight
- Model list assembly (base + `model@provider` variants)
- SQLite migration idempotency
- Forward passthrough (stream/non-stream)
- Anthropic conversion (stream/non-stream)
- Marketplace SHA mismatch rejection
- Module install/start and gateway call-through

## Current v0.1 boundaries

- `grpc_chat` adapter is a placeholder (returns `501_not_supported`)
- Admin pages are functional MVP pages, not yet a fully rich HTMX workflow
- API keys/provider secrets are stored in plaintext SQLite (as planned)
- Logs store metadata only, not prompt/response body
