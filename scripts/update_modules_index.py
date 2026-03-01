#!/usr/bin/env python3
import argparse
import datetime as dt
import json
import os
from typing import Any, Dict, List


def utc_now_rfc3339() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def load_json(path: str) -> Any:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def dump_json(path: str, obj: Any) -> None:
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=2, ensure_ascii=True)
        f.write("\n")


def protocols_from_manifest(manifest: Dict[str, Any]) -> List[str]:
    seen = set()
    out: List[str] = []
    for svc in manifest.get("services", []) or []:
        if str(svc.get("kind", "")).strip() != "provider":
            continue
        p = str(svc.get("protocol", "")).strip()
        if not p:
            continue
        key = p.lower()
        if key in seen:
            continue
        seen.add(key)
        out.append(p)
    out.sort(key=lambda s: s.lower())
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description="Update MODULES/index.json with a module entry.")
    ap.add_argument("--index", default="market/MODULES/index.json", help="Path to ModuleIndex JSON")
    ap.add_argument("--manifest", required=True, help="Path to module manifest.json")
    ap.add_argument("--download-url", required=True, help="Module zip download URL")
    ap.add_argument("--sha256", required=True, help="SHA256 hex of the module zip")
    ap.add_argument("--homepage", default="", help="Homepage URL (optional)")
    args = ap.parse_args()

    manifest = load_json(args.manifest)
    module_id = str(manifest.get("id", "")).strip()
    if not module_id:
        raise SystemExit("manifest.id is required")
    version = str(manifest.get("version", "")).strip()
    if not version:
        raise SystemExit("manifest.version is required")

    name = str(manifest.get("name", "")).strip() or module_id
    desc = str(manifest.get("description", "")).strip()
    license_ = str(manifest.get("license", "")).strip()
    tags = manifest.get("tags", [])
    if not isinstance(tags, list):
        tags = []
    tags = [str(t) for t in tags if str(t).strip()]

    entry = {
        "id": module_id,
        "name": name,
        "version": version,
        "description": desc,
        "license": license_,
        "tags": tags,
        "protocols": protocols_from_manifest(manifest),
        "download_url": str(args.download_url).strip(),
        "sha256": str(args.sha256).strip(),
        "homepage": str(args.homepage).strip(),
    }

    # Load existing index if present; otherwise create a new one.
    if os.path.exists(args.index):
        idx = load_json(args.index)
        if not isinstance(idx, dict):
            idx = {}
    else:
        idx = {}

    modules = idx.get("modules", [])
    if not isinstance(modules, list):
        modules = []

    modules = [m for m in modules if not (isinstance(m, dict) and str(m.get("id", "")).strip() == module_id)]
    modules.append(entry)
    modules.sort(key=lambda m: str(m.get("id", "")).lower() if isinstance(m, dict) else "")

    idx["generated_at"] = utc_now_rfc3339()
    if not str(idx.get("min_core_version", "")).strip():
        idx["min_core_version"] = str(manifest.get("min_core_version", "")).strip() or "0.1.0"
    idx["modules"] = modules

    dump_json(args.index, idx)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
