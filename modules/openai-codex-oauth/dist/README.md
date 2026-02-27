# OpenAI Codex OAuth Module (LightBridge)

This module provides an `http_openai` provider that proxies OpenAI-compatible requests to **OpenAI Codex (OAuth)**.

## Endpoints

- `GET /health` — module health check (always `200`)
- `POST /auth/device/start` — start device login flow (returns `verification_url` + `user_code`)
- `GET /auth/status` — current auth status / token expiry
- `POST /v1/chat/completions` — OpenAI Chat Completions compatible proxy (stream + non-stream)
- `GET /v1/models` — optional model list (from module config)

## Config

LightBridge writes module config to the file path in `LIGHTBRIDGE_CONFIG_PATH`.
See `manifest.json` for supported fields and defaults.

## Notes

- Tokens are stored in `LIGHTBRIDGE_DATA_DIR/credentials.json` (mode `0600`).
- If upstream returns `401`, the module will refresh the token once and retry.

## Packaging & install (local)

1. Build a zip + `index.json`:

```bash
bash modules/openai-codex-oauth/package.sh
```

2. Host the `dist/` folder:

```bash
cd modules/openai-codex-oauth/dist && python3 -m http.server 8000
```

3. In LightBridge Admin:

- Install module `openai-codex-oauth` from `http://127.0.0.1:8000/index.json`
- Start the module

## OAuth (device flow)

After the module is running, call the module directly (use `/admin/api/modules` to find `http_port`):

```bash
curl -s -X POST "http://127.0.0.1:<http_port>/auth/device/start"
curl -s "http://127.0.0.1:<http_port>/auth/status"
```

Open `verification_url` in a browser and enter `user_code`. When `status` becomes `authorized`, the provider is ready.
