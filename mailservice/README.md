# LightBridge Mail Service

LightBridge Mail Service (LBMS) is an optional sidecar for LightBridge. It manages mailbox entities, OAuth account bindings, verification-code retrieval, and driver integration without modifying the LightBridge core service.

## Current scope

This implementation is still a Phase 1 scaffold, but it now has durable local state:

- Exposes `/mail/v1/*` APIs.
- Uses the external name **LightBridge Mail Service** only.
- Keeps LightBridge account `extra` minimal: `lbms_link` should be the only persisted reference in the main LightBridge database.
- Maintains a bidirectional mailbox-to-OAuth binding model inside the sidecar.
- Allows one mailbox to bind multiple OAuth accounts.
- Allows one OAuth account to have only one active mailbox binding.
- Persists mailbox and OAuth binding metadata to a LBMS-owned JSON store through `LBMS_DATA_PATH`.
- Uses a short-lived in-memory verification-code cache only for repeated reads.

The JSON store is intended as the first durable step for single-node deployments. A future production phase can replace it with SQLite or PostgreSQL without changing LightBridge account `extra`, because the main database only stores `lbms_link`.

## API

### Health

```bash
curl http://127.0.0.1:8091/mail/v1/health
```

The health response includes the service name, driver status, version, and configured store path.

### List mailboxes

```bash
curl http://127.0.0.1:8091/mail/v1/mailboxes \
  -H "Authorization: Bearer $LBMS_API_KEY"
```

This endpoint powers the mailbox pool UI. It returns mailbox summaries with binding counts and timestamps.

### Link or create mailbox

```bash
curl -X POST http://127.0.0.1:8091/mail/v1/mailboxes/link-or-create \
  -H "Authorization: Bearer $LBMS_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "email_address": "aa@qq.com",
    "lightbridge_account_id": 101,
    "lightbridge_platform": "openai",
    "lightbridge_account_type": "oauth",
    "lightbridge_account_name": "OpenAI OAuth A"
  }'
```

Response contains the only value that should be written back to LightBridge account `extra`:

```json
{
  "lbms_link": "lbms://mailbox/mbx_xxx"
}
```

### Get mailbox link by OAuth account

```bash
curl http://127.0.0.1:8091/mail/v1/accounts/101/mailbox-link \
  -H "Authorization: Bearer $LBMS_API_KEY"
```

### Get verification code by OAuth account

```bash
curl "http://127.0.0.1:8091/mail/v1/accounts/101/verification-code?since_minutes=10&code_length=6" \
  -H "Authorization: Bearer $LBMS_API_KEY"
```

### List OAuth bindings for a mailbox

```bash
curl http://127.0.0.1:8091/mail/v1/mailboxes/mbx_xxx/bindings \
  -H "Authorization: Bearer $LBMS_API_KEY"
```

### Unlink account

```bash
curl -X DELETE http://127.0.0.1:8091/mail/v1/accounts/101/mailbox-link \
  -H "Authorization: Bearer $LBMS_API_KEY"
```

## Configuration

| Variable | Default | Description |
|---|---:|---|
| `LBMS_HOST` | `0.0.0.0` | Bind host. |
| `LBMS_PORT` | `8091` | Bind port. |
| `LBMS_API_KEY` | empty | Required for all non-health APIs. |
| `LBMS_DATA_PATH` | `data/lbms-store.json` | LBMS-owned persistent JSON store path. |
| `LBMS_DRIVER` | `outlook_email_plus` | Internal driver identifier. Do not expose this in UI. |
| `LBMS_DRIVER_BASE_URL` | empty | Internal driver base URL. |
| `LBMS_DRIVER_API_KEY` | empty | Internal driver API key. |
| `LBMS_REQUEST_TIMEOUT_SECONDS` | `10` | Outbound driver timeout. |
| `LBMS_VERIFICATION_CACHE_SECONDS` | `30` | Short-lived verification result cache. |

## Run locally

```bash
cd mailservice
export LBMS_API_KEY=dev-lbms-token
export LBMS_DATA_PATH=/tmp/lbms-store.json
export LBMS_DRIVER_BASE_URL=http://127.0.0.1:5000
export LBMS_DRIVER_API_KEY=driver-token
go run .
```

## Store behavior

The sidecar writes mailbox and binding metadata to `LBMS_DATA_PATH` after each link or unlink operation. Writes use a temporary file, fsync, and atomic rename so a process crash is less likely to leave a partially written store file.

The verification-code cache is intentionally not persisted. Codes are short lived and should be fetched from the configured mail driver again after restart.

## Production TODO

Before production rollout, continue with:

1. SQLite or PostgreSQL store adapter for higher write concurrency.
2. Migration files for `lbms_mailboxes`, `lbms_oauth_bindings`, `lbms_mail_events`, and `lbms_driver_accounts`.
3. Audit logging and retention jobs.
4. LightBridge API Key verification adapter.
5. Request rate limits and idempotency-key storage.
6. UI wiring that writes `extra.lbms_link` after OAuth account save succeeds.
