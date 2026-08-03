#!/usr/bin/env python3
"""Validate the checked-in runtime and dependency contract using stdlib only."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ERRORS: list[str] = []
INFO: list[str] = []


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        ERRORS.append(message)


def version_tuple(value: str) -> tuple[int, ...]:
    match = re.search(r"(\d+(?:\.\d+)+)", value)
    if not match:
        return ()
    return tuple(int(part) for part in match.group(1).split("."))


def at_least(actual: str, minimum: str) -> bool:
    actual_parts = version_tuple(actual)
    minimum_parts = version_tuple(minimum)
    width = max(len(actual_parts), len(minimum_parts))
    return actual_parts + (0,) * (width - len(actual_parts)) >= minimum_parts + (0,) * (
        width - len(minimum_parts)
    )


go_mod = read("backend/go.mod")
go_match = re.search(r"^go\s+([^\s]+)$", go_mod, re.MULTILINE)
require(go_match is not None, "backend/go.mod must declare an exact Go version")
go_version = go_match.group(1) if go_match else "unknown"
INFO.append(f"Go toolchain contract: {go_version}")

module_versions = dict(
    re.findall(r"^\s*([\w./-]+)\s+(v[^\s]+)", go_mod, re.MULTILINE)
)
for module, minimum in {
    "google.golang.org/grpc": "v1.82.1",
    "golang.org/x/text": "v0.39.0",
}.items():
    actual = module_versions.get(module, "")
    require(bool(actual), f"backend/go.mod must require {module}")
    require(
        bool(actual) and at_least(actual, minimum),
        f"{module} must be at least {minimum}; found {actual or 'missing'}",
    )
    if actual:
        INFO.append(f"{module}: {actual}")

package_json = json.loads(read("frontend/package.json"))
package_manager = str(package_json.get("packageManager", ""))
require(
    package_manager == "pnpm@9.15.9",
    f"frontend packageManager must be pnpm@9.15.9; found {package_manager or 'missing'}",
)
postcss = str(package_json.get("devDependencies", {}).get("postcss", ""))
require(
    at_least(postcss, "8.5.25"),
    f"PostCSS must be at least 8.5.25; found {postcss or 'missing'}",
)
INFO.append(f"Frontend package manager: {package_manager}")
INFO.append(f"PostCSS: {postcss}")

lockfile = read("frontend/pnpm-lock.yaml")
require("lockfileVersion: '9.0'" in lockfile, "pnpm lockfile must use lockfile version 9.0")
require((ROOT / "backend/go.sum").is_file(), "backend/go.sum is required")

root_dockerfile = read("Dockerfile")
require(
    f"ARG GOLANG_IMAGE=golang:{go_version}-alpine" in root_dockerfile,
    "Dockerfile Go builder must match backend/go.mod exactly",
)
require(
    "corepack prepare pnpm@9.15.9 --activate" in root_dockerfile,
    "Dockerfile must pin pnpm to frontend/package.json packageManager",
)
require("COPY --from=pg-client /usr/local/bin/pg_dump" in root_dockerfile, "runtime image must include pg_dump")
require("COPY --from=pg-client /usr/local/bin/psql" in root_dockerfile, "runtime image must include psql")
require("HEALTHCHECK" in root_dockerfile and "/health" in root_dockerfile, "runtime image must define /health check")

workflow_expectations = {
    ".github/workflows/backend-ci.yml": ("go-version-file: backend/go.mod", "version: 9.15.9", "node-version: '22.13'"),
    ".github/workflows/security-scan.yml": ("go-version-file: backend/go.mod", "version: 9.15.9", "node-version: '22.13'"),
    ".github/workflows/runtime-environment-check.yml": ("go-version-file: backend/go.mod", "version: 9.15.9", "node-version: '22.13'"),
}
for workflow, needles in workflow_expectations.items():
    path = ROOT / workflow
    require(path.is_file(), f"missing workflow: {workflow}")
    if not path.is_file():
        continue
    content = path.read_text(encoding="utf-8")
    for needle in needles:
        require(needle in content, f"{workflow} must contain {needle!r}")

compose_files = [
    "deploy/docker-compose.yml",
    "deploy/docker-compose.local.yml",
    "deploy/docker-compose.dev.yml",
    "deploy/docker-compose.standalone.yml",
]
for compose_file in compose_files:
    content = read(compose_file)
    require("LIGHTBRIDGE_DEPLOYMENT_TYPE=container" in content, f"{compose_file} must identify container deployment")
    require("AUTO_SETUP=true" in content, f"{compose_file} must enable automatic setup")
    require("/health" in content, f"{compose_file} must define an application health check")
    require("SERVER_PORT=8080" in content, f"{compose_file} must keep the internal application port at 8080")

for compose_file in compose_files[:3]:
    content = read(compose_file)
    require("postgres:18-alpine" in content, f"{compose_file} must pin PostgreSQL 18")
    require("redis:8-alpine" in content, f"{compose_file} must pin Redis 8")
    require("PGDATA=/var/lib/postgresql/data" in content, f"{compose_file} must persist PostgreSQL 18 data in the mounted path")
    require(content.count("condition: service_healthy") >= 2, f"{compose_file} must wait for PostgreSQL and Redis health")

entrypoint = ROOT / "deploy/docker-entrypoint.sh"
require(entrypoint.is_file(), "deploy/docker-entrypoint.sh is required")
if entrypoint.is_file():
    require(entrypoint.read_text(encoding="utf-8").startswith("#!/"), "deploy/docker-entrypoint.sh must have a shell shebang")


installer = ROOT / "deploy/install-datamanagementd.sh"
require(installer.is_file(), "deploy/install-datamanagementd.sh is required")
if installer.is_file():
    installer_text = installer.read_text(encoding="utf-8")
    require(installer_text.startswith("#!/"), "datamanagement installer must have a shell shebang")
    require("--binary" in installer_text, "datamanagement installer must accept a release binary")
    require("--source" not in installer_text, "datamanagement installer must not advertise missing source builds")

service_unit = ROOT / "deploy/LightBridge-datamanagementd.service"
require(service_unit.is_file(), "datamanagement systemd unit is required")
if service_unit.is_file():
    unit_text = service_unit.read_text(encoding="utf-8")
    require("ExecStart=/opt/LightBridge/datamanagementd" in unit_text, "datamanagement service must run the installed binary")
    require("NoNewPrivileges=true" in unit_text, "datamanagement service must enable NoNewPrivileges")

makefile = read("Makefile")
require("build-datamanagementd" not in makefile, "Makefile must not build a source module that is not present")
require("test-datamanagementd" not in makefile, "Makefile must not test a source module that is not present")
datamanagement_docs = read("deploy/DATAMANAGEMENTD_CN.md")
require("--source" not in datamanagement_docs, "datamanagement documentation must describe binary-only installation")

if ERRORS:
    print("Runtime/environment contract failed:", file=sys.stderr)
    for error in ERRORS:
        print(f"- {error}", file=sys.stderr)
    sys.exit(1)

print("Runtime/environment contract passed:")
for item in INFO:
    print(f"- {item}")
print(f"- Compose variants checked: {len(compose_files)}")
print("- Docker build, database tools, deployment markers, and health checks are aligned")
