# Codex OAuth Module (LightBridge)

This module provides an `http_openai` provider that proxies OpenAI-compatible requests to **OpenAI Codex (OAuth)**.

## Endpoints

- `GET /health` — module health check (always `200`)
- `GET /auth/status` — current auth status / token expiry
- `POST /auth/oauth/start` — start OAuth auth-code flow (expects `redirect_uri`, returns `auth_url` + `state`)
- `POST /auth/oauth/exchange` — exchange `{code,state}` (or `{callback_url}`) for tokens
- `POST /auth/device/start` — start device login flow (returns `verification_url` + `user_code`)
- `POST /auth/import` — import `refresh_token` / `access_token` / Codex `Auth.json`
- `POST /v1/chat/completions` — OpenAI Chat Completions compatible proxy (stream + non-stream)
- `GET /v1/models` — optional model list (from module config)

## Config

LightBridge writes module config to the file path in `LIGHTBRIDGE_CONFIG_PATH`.
See `manifest.json` for supported fields and defaults.

## Notes

- Tokens are stored in `LIGHTBRIDGE_DATA_DIR/credentials.json` (mode `0600`).
- If upstream returns `401`, the module will refresh the token once and retry.

## Recommended setup (LightBridge Admin)

1. Install + start module `openai-codex-oauth` in Admin Marketplace.
2. Go to **Providers** → **添加** → **Codex（OpenAI）**.
3. Click **生成 OAuth 链接** and finish login in the browser.
4. The callback page `/admin/codex/oauth/callback` exchanges the code for tokens automatically.

## Packaging & install (local)

1. Build a zip + `index.json`:

```bash
bash modules/openai-codex-oauth/package.sh
```

2. Copy the generated `.zip` into LightBridge local marketplace folder:

```bash
mkdir -p MODULES
cp modules/openai-codex-oauth/dist/*.zip MODULES/
```

3. In LightBridge Admin Marketplace (default source = `local`):

- Install module `openai-codex-oauth`
- Start the module

Optional (remote marketplace): host the `dist/` folder and install from the generated `index.json`.

```bash
cd modules/openai-codex-oauth/dist && python3 -m http.server 8000
```

## OAuth (device flow)

After the module is running, call the module directly (use `/admin/api/modules` to find `runtime.http_port`):

```bash
curl -s -X POST "http://127.0.0.1:<http_port>/auth/device/start"
curl -s "http://127.0.0.1:<http_port>/auth/status"
```

Open `verification_url` in a browser and enter `user_code`. When `status` becomes `authorized`, the provider is ready.
