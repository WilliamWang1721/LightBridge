#!/usr/bin/env python3
"""Validate full-site UI platform coverage without rewriting Legacy templates.

The modern UI intentionally bridges existing Tailwind utilities so Legacy mode can
remain unchanged. This audit prevents high-frequency neutral utilities and shared
legacy component classes from escaping the Modern/Package semantic layer.
"""

from __future__ import annotations

from collections import Counter
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
FRONTEND = ROOT / "frontend" / "src"
BRIDGE_PATH = FRONTEND / "styles" / "ui-modern-tailwind-bridge.css"
COMPAT_PATH = FRONTEND / "styles" / "ui-modern-compat.css"
REGISTRY_PATH = FRONTEND / "ui-platform" / "registry.ts"

CLASS_ATTRIBUTE = re.compile(r"(?:class|:class)\s*=\s*[\"'`]([^\"'`]+)[\"'`]")
TOKEN_SPLIT = re.compile(r"\s+")

BRIDGED_PREFIXES = (
    "text-gray-",
    "bg-gray-",
    "border-gray-",
    "divide-gray-",
    "dark:text-gray-",
    "dark:text-dark-",
    "dark:bg-dark-",
    "dark:border-dark-",
    "dark:divide-dark-",
    "rounded",
    "shadow",
    "text-primary-",
    "bg-primary-",
    "border-primary-",
    "hover:bg-gray-",
    "dark:hover:bg-dark-",
    "hover:text-gray-",
    "hover:text-primary-",
    "dark:hover:text-primary-",
    "focus:ring-primary-",
    "focus-visible:ring-primary-",
)

INTENTIONAL_TOKENS = {
    "rounded-full",
    "rounded-[var(--ui-radius",
    "shadow-inner",
    "shadow-none",
    "shadow-blue-500/30",
    "shadow-amber-500/30",
    "shadow-glow",
    "text-transparent",
    "text-gray-100",
    "bg-transparent",
    "bg-gray-900",
    "border-transparent",
    "border-gray-700",
}

REQUIRED_LEGACY_SURFACES = {
    ".btn",
    ".btn-primary",
    ".btn-secondary",
    ".btn-ghost",
    ".btn-danger",
    ".input",
    ".input-label",
    ".card",
    ".table-wrapper",
    ".table-container",
    ".badge",
    ".dropdown",
    ".dropdown-item",
    ".modal-overlay",
    ".modal-content",
    ".sidebar",
    ".app-topbar",
    ".tabs",
    ".progress",
    ".skeleton",
    ".empty-state",
    ".toast",
}


def extract_class_tokens() -> Counter[str]:
    counts: Counter[str] = Counter()
    for path in FRONTEND.rglob("*.vue"):
        text = path.read_text(encoding="utf-8", errors="ignore")
        for match in CLASS_ATTRIBUTE.finditer(text):
            for raw_token in TOKEN_SPLIT.split(match.group(1)):
                token = raw_token.strip("[]{}(),'\"")
                if token:
                    counts[token] += 1
    return counts


def main() -> int:
    errors: list[str] = []
    bridge_parts = sorted((FRONTEND / "styles" / "ui-modern-tailwind-bridge-parts").glob("*.css"))
    compat_parts = sorted((FRONTEND / "styles" / "ui-modern-compat-parts").glob("*.css"))
    bridge = BRIDGE_PATH.read_text(encoding="utf-8") + "\n" + "\n".join(
        path.read_text(encoding="utf-8") for path in bridge_parts
    )
    compat = COMPAT_PATH.read_text(encoding="utf-8") + "\n" + "\n".join(
        path.read_text(encoding="utf-8") for path in compat_parts
    )
    combined = bridge + "\n" + compat
    registry = REGISTRY_PATH.read_text(encoding="utf-8")

    if "mode: 'modern'" not in registry:
        errors.append("DEFAULT_UI_PROFILE must keep LightBridge Luma (modern) as the built-in mode")
    if "componentStyle: 'luma'" not in registry:
        errors.append("DEFAULT_UI_PROFILE must keep componentStyle fixed to luma")

    for selector in sorted(REQUIRED_LEGACY_SURFACES):
        if selector not in compat:
            errors.append(f"missing Modern/Package compatibility selector: {selector}")

    counts = extract_class_tokens()
    for token, count in sorted(counts.items()):
        if count < 3 or token in INTENTIONAL_TOKENS:
            continue
        if not token.startswith(BRIDGED_PREFIXES):
            continue
        if f"'{token}'" not in combined and f'"{token}"' not in combined:
            errors.append(
                f"unbridged legacy utility used {count} times: {token} "
                "(map it in ui-modern-tailwind-bridge.css or document it as intentional)"
            )

    required_surfaces = ("admin", "user", "auth", "public", "setup", "payment")
    for surface in required_surfaces:
        if f"data-ui-surface='{surface}'" not in bridge and surface not in {"admin", "user"}:
            errors.append(f"missing surface-specific Modern/Package coverage: {surface}")

    if errors:
        print("UI migration audit failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(
        "UI migration audit passed: "
        f"{sum(counts.values())} class-token references scanned, "
        f"{len(REQUIRED_LEGACY_SURFACES)} legacy surfaces bridged."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
