#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${RUNTIME_ARTIFACT_DIR:-$ROOT_DIR/artifacts/runtime-check}"
mkdir -p "$ARTIFACT_DIR"
LOGIN_RESPONSE_FILE="$(mktemp)"
LOGIN_AFTER_RESTART_FILE="$(mktemp)"

export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-lightbridge-runtime-check}"
export BIND_HOST="${BIND_HOST:-127.0.0.1}"
export SERVER_PORT="${SERVER_PORT:-18080}"
export POSTGRES_USER="${POSTGRES_USER:-LightBridge}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-LightBridge-ci-postgres-2026}"
export POSTGRES_DB="${POSTGRES_DB:-LightBridge}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-LightBridge-ci-redis-2026}"
export ADMIN_EMAIL="${ADMIN_EMAIL:-runtime-check@lightbridge.local}"
export ADMIN_PASSWORD="${ADMIN_PASSWORD:-LightBridge-CI-Admin-2026!}"
export JWT_SECRET="${JWT_SECRET:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
export TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY:-abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789}"
export TZ="${TZ:-UTC}"

COMPOSE=(docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.ci.yml)
BASE_URL="http://${BIND_HOST}:${SERVER_PORT}"

cleanup() {
  status=$?
  trap - EXIT
  rm -f "$LOGIN_RESPONSE_FILE" "$LOGIN_AFTER_RESTART_FILE" || true
  "${COMPOSE[@]}" ps -a >"$ARTIFACT_DIR/compose-ps.txt" 2>&1 || true
  "${COMPOSE[@]}" logs --no-color >"$ARTIFACT_DIR/compose.log" 2>&1 || true
  docker image inspect lightbridge:ci >"$ARTIFACT_DIR/image-inspect.json" 2>&1 || true
  if (( status != 0 )); then
    echo "Runtime check failed. Recent Compose logs:" >&2
    tail -n 250 "$ARTIFACT_DIR/compose.log" >&2 || true
  fi
  "${COMPOSE[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

wait_for_url() {
  local url="$1"
  local attempts="${2:-60}"
  local delay="${3:-2}"
  local i
  for ((i = 1; i <= attempts; i++)); do
    if curl --fail --silent --show-error --max-time 5 "$url" >/dev/null; then
      return 0
    fi
    sleep "$delay"
  done
  echo "Timed out waiting for $url" >&2
  return 1
}

validate_json_url() {
  local url="$1"
  local output="$2"
  curl --fail --silent --show-error --max-time 10 "$url" -o "$output"
  python3 - "$output" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if not isinstance(payload, dict):
    raise SystemExit("expected a JSON object")
PY
}

"${COMPOSE[@]}" config --quiet
"${COMPOSE[@]}" config --no-interpolate >"$ARTIFACT_DIR/compose-template.yml"
"${COMPOSE[@]}" up -d --wait --wait-timeout 240

wait_for_url "$BASE_URL/health"
validate_json_url "$BASE_URL/health" "$ARTIFACT_DIR/health.json"
python3 - "$ARTIFACT_DIR/health.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if payload.get("status") != "ok":
    raise SystemExit(f"unexpected health response: {payload!r}")
PY

validate_json_url "$BASE_URL/setup/status" "$ARTIFACT_DIR/setup-status.json"
python3 - "$ARTIFACT_DIR/setup-status.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
data = payload.get("data", {})
if payload.get("code") != 0 or data.get("needs_setup") is not False:
    raise SystemExit(f"unexpected setup status: {payload!r}")
PY

curl --fail --silent --show-error --max-time 10 "$BASE_URL/" -o "$ARTIFACT_DIR/index.html"
grep -Eqi '<!doctype html|<html' "$ARTIFACT_DIR/index.html"
validate_json_url "$BASE_URL/api/v1/settings/public" "$ARTIFACT_DIR/public-settings.json"

LOGIN_PAYLOAD="$(python3 - <<'PY'
import json
import os
print(json.dumps({"email": os.environ["ADMIN_EMAIL"], "password": os.environ["ADMIN_PASSWORD"]}))
PY
)"
curl --fail --silent --show-error --max-time 15 \
  -H 'Content-Type: application/json' \
  --data "$LOGIN_PAYLOAD" \
  "$BASE_URL/api/v1/auth/login" \
  -o "$LOGIN_RESPONSE_FILE"

ACCESS_TOKEN="$(python3 - "$LOGIN_RESPONSE_FILE" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if payload.get("code") != 0:
    raise SystemExit(f"admin login failed: {payload!r}")
token = payload.get("data", {}).get("access_token")
if not token:
    raise SystemExit("admin login response did not contain access_token")
print(token)
PY
)"

curl --fail --silent --show-error --max-time 15 \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$BASE_URL/api/v1/admin/system/version" \
  -o "$ARTIFACT_DIR/system-version.json"
python3 - "$ARTIFACT_DIR/system-version.json" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if payload.get("code") != 0:
    raise SystemExit(f"version endpoint failed: {payload!r}")
if not payload.get("data", {}).get("current_version"):
    raise SystemExit(f"version endpoint did not expose current_version: {payload!r}")
PY

"${COMPOSE[@]}" exec -T LightBridge sh -ec '
  test "$LIGHTBRIDGE_DEPLOYMENT_TYPE" = container
  test "$(id -u)" = 1000
  pg_dump --version
  psql --version
' >"$ARTIFACT_DIR/runtime-tools.txt"

"${COMPOSE[@]}" exec -T redis redis-cli --no-auth-warning ping \
  | tee "$ARTIFACT_DIR/redis-ping.txt" \
  | grep -qx PONG

USER_COUNT_BEFORE="$("${COMPOSE[@]}" exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atqc 'SELECT COUNT(*) FROM users;')"
case "$USER_COUNT_BEFORE" in
  ''|*[!0-9]*) echo "invalid users count: $USER_COUNT_BEFORE" >&2; exit 1 ;;
esac
if (( USER_COUNT_BEFORE < 1 )); then
  echo "automatic setup did not create an administrator" >&2
  exit 1
fi
printf '%s\n' "$USER_COUNT_BEFORE" >"$ARTIFACT_DIR/user-count-before-restart.txt"

"${COMPOSE[@]}" restart LightBridge
wait_for_url "$BASE_URL/health" 90 2

"${COMPOSE[@]}" restart postgres redis
"${COMPOSE[@]}" up -d --wait --wait-timeout 240
wait_for_url "$BASE_URL/health" 90 2

curl --fail --silent --show-error --max-time 15 \
  -H 'Content-Type: application/json' \
  --data "$LOGIN_PAYLOAD" \
  "$BASE_URL/api/v1/auth/login" \
  -o "$LOGIN_AFTER_RESTART_FILE"
python3 - "$LOGIN_AFTER_RESTART_FILE" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if payload.get("code") != 0 or not payload.get("data", {}).get("access_token"):
    raise SystemExit(f"login failed after dependency restart: {payload!r}")
PY

USER_COUNT_AFTER="$("${COMPOSE[@]}" exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atqc 'SELECT COUNT(*) FROM users;')"
printf '%s\n' "$USER_COUNT_AFTER" >"$ARTIFACT_DIR/user-count-after-restart.txt"
if [[ "$USER_COUNT_AFTER" != "$USER_COUNT_BEFORE" ]]; then
  echo "database persistence check failed: before=$USER_COUNT_BEFORE after=$USER_COUNT_AFTER" >&2
  exit 1
fi

"${COMPOSE[@]}" ps
printf 'Full-stack runtime check passed for %s\n' "$BASE_URL"
