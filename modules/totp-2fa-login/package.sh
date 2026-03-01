#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"

eval "$(python3 - <<'PY'
import json, pathlib, shlex
p = pathlib.Path("modules/totp-2fa-login/manifest.json")
m = json.loads(p.read_text(encoding="utf-8"))
print("VERSION=" + shlex.quote(m["version"]))
print("NAME_JSON=" + shlex.quote(json.dumps(m.get("name", ""))))
print("DESC_JSON=" + shlex.quote(json.dumps(m.get("description", ""))))
print("TAGS_JSON=" + shlex.quote(json.dumps(m.get("tags", []))))
PY
)"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

DIST="${HERE}/dist"
ZIP_NAME="totp-2fa-login_${VERSION}_${GOOS}_${GOARCH}.zip"
ZIP_PATH="${DIST}/${ZIP_NAME}"

rm -rf "${DIST}"
mkdir -p "${DIST}/bin"

echo "Building module binary..."
go build -o "${DIST}/bin/totp-2fa-login" "${ROOT}/modules/totp-2fa-login/cmd/totp-2fa-login"

cp "${HERE}/manifest.json" "${DIST}/manifest.json"
cp "${HERE}/README.md" "${DIST}/README.md"

echo "Packaging zip..."
(cd "${DIST}" && zip -q -r "${ZIP_NAME}" manifest.json README.md bin/)

SHA256="$(shasum -a 256 "${ZIP_PATH}" | awk '{print $1}')"

cat > "${DIST}/index.json" <<JSON
{
  "generated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "min_core_version": "0.1.0",
  "modules": [
    {
      "id": "totp-2fa-login",
      "name": ${NAME_JSON},
      "version": "${VERSION}",
      "description": ${DESC_JSON},
      "license": "UNLICENSED",
      "tags": ${TAGS_JSON},
      "protocols": ["http_rpc"],
      "download_url": "http://127.0.0.1:8000/${ZIP_NAME}",
      "sha256": "${SHA256}",
      "homepage": ""
    }
  ]
}
JSON

echo "OK"
echo "ZIP:    ${ZIP_PATH}"
echo "SHA256: ${SHA256}"
