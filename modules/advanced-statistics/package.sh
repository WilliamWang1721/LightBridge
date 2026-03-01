#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"

eval "$(python3 - <<'PY'
import json, pathlib, shlex
p = pathlib.Path("modules/advanced-statistics/manifest.json")
m = json.loads(p.read_text(encoding="utf-8"))
print("MODULE_ID=" + shlex.quote(m["id"]))
print("VERSION=" + shlex.quote(m["version"]))
print("NAME_JSON=" + shlex.quote(json.dumps(m.get("name", ""))))
print("DESC_JSON=" + shlex.quote(json.dumps(m.get("description", ""))))
print("TAGS_JSON=" + shlex.quote(json.dumps(m.get("tags", []))))
PY
)"

DIST="${HERE}/dist"
ZIP_NAME="${MODULE_ID}_${VERSION}_universal.zip"
ZIP_PATH="${DIST}/${ZIP_NAME}"

rm -rf "${DIST}"
mkdir -p "${DIST}"

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
  else
    shasum -a 256 "${path}" | awk '{print $1}'
  fi
}

targets=(
  "darwin arm64"
  "darwin amd64"
  "linux arm64"
  "linux amd64"
)

echo "Building universal module binaries..."
for t in "${targets[@]}"; do
  goos="${t%% *}"
  goarch="${t##* }"
  out_dir="${DIST}/bin/${goos}/${goarch}"
  mkdir -p "${out_dir}"
  echo "  - ${goos}/${goarch}"
  env CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags="-s -w" \
      -o "${out_dir}/advanced-statistics" \
      "${ROOT}/modules/advanced-statistics/cmd/advanced-statistics"
done

cp "${HERE}/manifest.json" "${DIST}/manifest.json"
cp "${HERE}/README.md" "${DIST}/README.md"

echo "Packaging zip..."
(cd "${DIST}" && zip -q -r "${ZIP_NAME}" manifest.json README.md bin/)

SHA256="$(sha256_file "${ZIP_PATH}")"

cat > "${DIST}/index.json" <<JSON
{
  "generated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "min_core_version": "0.1.0",
  "modules": [
    {
      "id": "${MODULE_ID}",
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
