#!/usr/bin/env python3
"""Verify LightBridge release assets and GHCR manifests at content level."""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
from typing import Any, Callable, TypeVar
from urllib.error import HTTPError
from urllib.parse import quote, urlencode
from urllib.request import Request, urlopen
import zipfile

SEMVER_RE = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)$")
MANIFEST_ACCEPT = ", ".join(
    (
        "application/vnd.oci.image.index.v1+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.docker.distribution.manifest.v2+json",
    )
)
T = TypeVar("T")


def fail(message: str) -> None:
    raise RuntimeError(message)


def github_token() -> str:
    return os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN") or ""


def http_bytes(url: str, *, token: str = "", headers: dict[str, str] | None = None) -> tuple[bytes, Any]:
    request_headers = {
        "User-Agent": "LightBridge-release-integrity",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    if headers:
        request_headers.update(headers)
    if token:
        request_headers["Authorization"] = f"Bearer {token}"
    request = Request(url, headers=request_headers)
    with urlopen(request, timeout=120) as response:
        return response.read(), response.headers


def github_json(url: str) -> Any:
    body, _ = http_bytes(
        url,
        token=github_token(),
        headers={"Accept": "application/vnd.github+json"},
    )
    return json.loads(body)


def retry(description: str, operation: Callable[[], T], attempts: int, delay: float) -> T:
    last_error: Exception | None = None
    for attempt in range(1, attempts + 1):
        try:
            return operation()
        except (HTTPError, OSError, RuntimeError, ValueError, json.JSONDecodeError) as exc:
            last_error = exc
            if attempt == attempts:
                break
            print(f"{description} not ready ({attempt}/{attempts}): {exc}", file=sys.stderr)
            time.sleep(delay)
    assert last_error is not None
    raise last_error


def parse_stable_version(tag: str) -> tuple[int, int, int] | None:
    match = SEMVER_RE.fullmatch(tag.strip())
    if not match:
        return None
    return tuple(int(part) for part in match.groups())


def guard_release_order(repository: str, tag: str) -> None:
    requested = parse_stable_version(tag)
    if requested is None:
        print(f"{tag} is not a stable semantic version; stable rollback guard skipped")
        return

    stable_releases: list[tuple[tuple[int, int, int], str]] = []
    for page in range(1, 11):
        releases = github_json(
            f"https://api.github.com/repos/{repository}/releases?per_page=100&page={page}"
        )
        if not releases:
            break
        for release in releases:
            if release.get("draft") or release.get("prerelease"):
                continue
            parsed = parse_stable_version(str(release.get("tag_name", "")))
            if parsed is not None:
                stable_releases.append((parsed, str(release["tag_name"])))
        if len(releases) < 100:
            break

    if not stable_releases:
        print(f"No prior stable release; allowing {tag}")
        return

    latest_version, latest_tag = max(stable_releases)
    if requested < latest_version:
        fail(
            f"refusing stable release rollback: requested {tag}, latest published stable release is {latest_tag}"
        )
    print(f"Stable release order accepted: requested={tag}, latest={latest_tag}")


def expected_assets(version: str) -> set[str]:
    return {
        "checksums.txt",
        f"LightBridge_{version}_linux_amd64.tar.gz",
        f"LightBridge_{version}_linux_arm64.tar.gz",
        f"LightBridge_{version}_darwin_amd64.tar.gz",
        f"LightBridge_{version}_darwin_arm64.tar.gz",
        f"LightBridge_{version}_windows_amd64.zip",
    }


def safe_archive_path(name: str) -> None:
    path = PurePosixPath(name.replace("\\", "/"))
    if path.is_absolute() or ".." in path.parts:
        fail(f"unsafe archive path: {name}")


def extract_binary(archive_path: Path, destination: Path) -> Path:
    candidates: list[tuple[str, bytes]] = []
    if archive_path.name.endswith(".zip"):
        with zipfile.ZipFile(archive_path) as archive:
            for info in archive.infolist():
                safe_archive_path(info.filename)
                if not info.is_dir() and PurePosixPath(info.filename).name in {"LightBridge", "LightBridge.exe"}:
                    candidates.append((PurePosixPath(info.filename).name, archive.read(info)))
    else:
        with tarfile.open(archive_path, mode="r:gz") as archive:
            for member in archive.getmembers():
                safe_archive_path(member.name)
                if member.isfile() and PurePosixPath(member.name).name in {"LightBridge", "LightBridge.exe"}:
                    stream = archive.extractfile(member)
                    if stream is None:
                        fail(f"unable to read binary from {archive_path.name}")
                    candidates.append((PurePosixPath(member.name).name, stream.read()))
    if len(candidates) != 1:
        fail(f"{archive_path.name} contains {len(candidates)} LightBridge binaries")
    name, data = candidates[0]
    output = destination / f"{archive_path.name}-{name}"
    output.write_bytes(data)
    output.chmod(output.stat().st_mode | stat.S_IXUSR)
    return output


def parse_archive_platform(name: str, version: str) -> tuple[str, str]:
    match = re.fullmatch(
        rf"LightBridge_{re.escape(version)}_(linux|darwin|windows)_(amd64|arm64)\.(?:tar\.gz|zip)",
        name,
    )
    if not match:
        fail(f"unexpected archive name: {name}")
    return match.group(1), match.group(2)


def verify_go_binary(binary: Path, *, version: str, commit: str, goos: str, goarch: str) -> None:
    data = binary.read_bytes()
    if version.encode() not in data:
        fail(f"{binary.name} does not contain version {version}")
    if commit.encode() not in data:
        fail(f"{binary.name} does not contain commit {commit}")

    result = subprocess.run(
        ["go", "version", "-m", str(binary)],
        check=True,
        capture_output=True,
        text=True,
    )
    metadata = result.stdout + result.stderr
    expected_settings = {
        f"GOOS={goos}",
        f"GOARCH={goarch}",
        f"vcs.revision={commit}",
    }
    for expected in expected_settings:
        if expected not in metadata:
            fail(f"{binary.name} build metadata is missing {expected}:\n{metadata}")

    if goos == "linux" and goarch == "amd64":
        version_result = subprocess.run(
            [str(binary), "-version"],
            check=True,
            capture_output=True,
            text=True,
            timeout=30,
        )
        version_output = version_result.stdout + version_result.stderr
        if f"LightBridge {version}" not in version_output or f"commit: {commit}" not in version_output:
            fail(f"native binary version output is incorrect:\n{version_output}")


def parse_checksums(content: str) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for line in content.splitlines():
        if not line.strip():
            continue
        parts = line.split(None, 1)
        if len(parts) != 2 or not re.fullmatch(r"[0-9a-fA-F]{64}", parts[0]):
            fail(f"invalid checksum line: {line}")
        name = parts[1].lstrip("* ")
        if name in checksums:
            fail(f"duplicate checksum entry: {name}")
        checksums[name] = parts[0].lower()
    return checksums


def verify_release_assets(repository: str, tag: str, commit: str, workdir: Path) -> None:
    version = tag.removeprefix("v")
    release = github_json(f"https://api.github.com/repos/{repository}/releases/tags/{quote(tag, safe='')}")
    if release.get("draft"):
        fail(f"release {tag} is still a draft")
    if parse_stable_version(tag) is not None and release.get("prerelease"):
        fail(f"stable release {tag} is marked prerelease")

    assets = {str(asset["name"]): asset for asset in release.get("assets", [])}
    expected = expected_assets(version)
    if set(assets) != expected:
        fail(f"release asset set mismatch: expected={sorted(expected)} actual={sorted(assets)}")

    downloaded: dict[str, Path] = {}
    for name, asset in assets.items():
        body, _ = http_bytes(
            str(asset["url"]),
            token=github_token(),
            headers={"Accept": "application/octet-stream"},
        )
        path = workdir / name
        path.write_bytes(body)
        downloaded[name] = path
        digest = str(asset.get("digest") or "")
        if digest.startswith("sha256:"):
            actual = hashlib.sha256(body).hexdigest()
            if actual != digest.removeprefix("sha256:"):
                fail(f"GitHub asset digest mismatch for {name}")

    checksums = parse_checksums(downloaded["checksums.txt"].read_text(encoding="utf-8"))
    archive_names = expected - {"checksums.txt"}
    if set(checksums) != archive_names:
        fail(f"checksums.txt entries mismatch: expected={sorted(archive_names)} actual={sorted(checksums)}")

    binary_dir = workdir / "binaries"
    binary_dir.mkdir()
    for name in sorted(archive_names):
        actual = hashlib.sha256(downloaded[name].read_bytes()).hexdigest()
        if actual != checksums[name]:
            fail(f"checksum mismatch for {name}")
        goos, goarch = parse_archive_platform(name, version)
        binary = extract_binary(downloaded[name], binary_dir)
        verify_go_binary(binary, version=version, commit=commit, goos=goos, goarch=goarch)


def registry_pull_token(repository_path: str) -> str:
    query = urlencode({"service": "ghcr.io", "scope": f"repository:{repository_path}:pull"})
    headers = {"Accept": "application/json"}
    token = github_token()
    actor = os.environ.get("GITHUB_ACTOR", "github-actions")
    if token:
        basic = base64.b64encode(f"{actor}:{token}".encode()).decode()
        headers["Authorization"] = f"Basic {basic}"
    body, _ = http_bytes(f"https://ghcr.io/token?{query}", headers=headers)
    payload = json.loads(body)
    access_token = payload.get("token") or payload.get("access_token")
    if not access_token:
        fail("GHCR token response did not include a token")
    return str(access_token)


def registry_get(repository_path: str, suffix: str, token: str, accept: str) -> tuple[bytes, Any]:
    encoded_suffix = quote(suffix, safe=":@/.-_")
    return http_bytes(
        f"https://ghcr.io/v2/{repository_path}/{encoded_suffix}",
        headers={"Accept": accept, "Authorization": f"Bearer {token}"},
    )


def manifest_digest(body: bytes, headers: Any) -> str:
    digest = headers.get("Docker-Content-Digest")
    return str(digest) if digest else f"sha256:{hashlib.sha256(body).hexdigest()}"


def verify_container(repository_path: str, tag: str, commit: str) -> None:
    version = tag.removeprefix("v")
    parsed = parse_stable_version(tag)
    if parsed is None:
        floating_tags = [version]
    else:
        major, minor, _ = parsed
        floating_tags = [version, "latest", f"{major}.{minor}", str(major)]

    token = registry_pull_token(repository_path)
    manifests: dict[str, tuple[bytes, Any]] = {}
    for image_tag in floating_tags:
        manifests[image_tag] = registry_get(
            repository_path,
            f"manifests/{image_tag}",
            token,
            MANIFEST_ACCEPT,
        )

    immutable_digest = manifest_digest(*manifests[version])
    for image_tag, (body, headers) in manifests.items():
        actual_digest = manifest_digest(body, headers)
        if actual_digest != immutable_digest:
            fail(
                f"floating GHCR tag {image_tag} points to {actual_digest}, expected immutable {version} digest {immutable_digest}"
            )

    index = json.loads(manifests[version][0])
    descriptors = index.get("manifests")
    if not isinstance(descriptors, list):
        fail(f"ghcr.io/{repository_path}:{version} is not a multi-platform image index")

    platform_descriptors: dict[tuple[str, str], dict[str, Any]] = {}
    for descriptor in descriptors:
        platform = descriptor.get("platform") or {}
        key = (str(platform.get("os", "")), str(platform.get("architecture", "")))
        if key in platform_descriptors:
            fail(f"duplicate image platform descriptor: {key}")
        platform_descriptors[key] = descriptor
    expected_platforms = {("linux", "amd64"), ("linux", "arm64")}
    if set(platform_descriptors) != expected_platforms:
        fail(f"image platform mismatch: {sorted(platform_descriptors)}")

    for platform, descriptor in sorted(platform_descriptors.items()):
        descriptor_digest = str(descriptor.get("digest", ""))
        manifest_body, manifest_headers = registry_get(
            repository_path,
            f"manifests/{descriptor_digest}",
            token,
            MANIFEST_ACCEPT,
        )
        if manifest_digest(manifest_body, manifest_headers) != descriptor_digest:
            fail(f"manifest digest mismatch for {platform}")
        manifest = json.loads(manifest_body)
        config_digest = str((manifest.get("config") or {}).get("digest", ""))
        if not config_digest:
            fail(f"image config digest missing for {platform}")
        config_body, _ = registry_get(
            repository_path,
            f"blobs/{config_digest}",
            token,
            "application/octet-stream",
        )
        config = json.loads(config_body)
        labels = ((config.get("config") or {}).get("Labels") or {})
        if labels.get("org.opencontainers.image.revision") != commit:
            fail(f"OCI revision mismatch for {platform}: {labels.get('org.opencontainers.image.revision')}")
        if labels.get("org.opencontainers.image.version") != version:
            fail(f"OCI version mismatch for {platform}: {labels.get('org.opencontainers.image.version')}")


def verify_release(repository: str, tag: str, commit: str, image_repository: str, attempts: int, delay: float) -> None:
    with tempfile.TemporaryDirectory(prefix="lightbridge-release-") as temporary_directory:
        workdir = Path(temporary_directory)
        retry(
            "GitHub release assets",
            lambda: verify_release_assets(repository, tag, commit, workdir),
            attempts,
            delay,
        )
        retry(
            "GHCR manifests",
            lambda: verify_container(image_repository, tag, commit),
            attempts,
            delay,
        )
    print(f"Release integrity verified: {tag} -> {commit}")


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    guard_parser = subparsers.add_parser("guard", help="refuse stable release rollback")
    guard_parser.add_argument("--repository", required=True)
    guard_parser.add_argument("--tag", required=True)

    verify_parser = subparsers.add_parser("verify", help="verify published assets and GHCR images")
    verify_parser.add_argument("--repository", required=True)
    verify_parser.add_argument("--tag", required=True)
    verify_parser.add_argument("--commit", required=True)
    verify_parser.add_argument("--image-repository", required=True)
    verify_parser.add_argument("--attempts", type=int, default=18)
    verify_parser.add_argument("--delay", type=float, default=10.0)

    args = parser.parse_args()
    try:
        if args.command == "guard":
            guard_release_order(args.repository, args.tag)
        else:
            verify_release(
                args.repository,
                args.tag,
                args.commit,
                args.image_repository,
                args.attempts,
                args.delay,
            )
    except Exception as exc:  # noqa: BLE001 - CLI boundary
        print(f"release integrity verification failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
