#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"

eval "$(python3 - <<'PY'
import json, pathlib, shlex
p = pathlib.Path("modules/openai-codex-oauth/manifest.json")
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
ZIP_NAME="openai-codex-oauth_${VERSION}_${GOOS}_${GOARCH}.zip"
ZIP_PATH="${DIST}/${ZIP_NAME}"

rm -rf "${DIST}"
mkdir -p "${DIST}/bin"

echo "Building module binary..."
go build -o "${DIST}/bin/openai-codex-oauth" "${ROOT}/modules/openai-codex-oauth/cmd/openai-codex-oauth"

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
      "id": "openai-codex-oauth",
      "name": ${NAME_JSON},
      "version": "${VERSION}",
      "description": ${DESC_JSON},
      "license": "UNLICENSED",
      "tags": ${TAGS_JSON},
      "protocols": ["http_openai"],
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
echo
echo "Local marketplace (default):"
echo "  mkdir -p \"${ROOT}/MODULES\" && cp \"${ZIP_PATH}\" \"${ROOT}/MODULES/\""
echo "  Open: http://127.0.0.1:3210/admin/marketplace (source = local)"
echo
echo "Remote marketplace (optional):"
echo "  cd \"${DIST}\" && python3 -m http.server 8000"
echo "  In LightBridge admin, install module_id=openai-codex-oauth from index_url=http://127.0.0.1:8000/index.json"
