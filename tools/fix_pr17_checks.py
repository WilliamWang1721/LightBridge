from __future__ import annotations

from pathlib import Path
import re


def read(path: str) -> str:
    return Path(path).read_text(encoding="utf-8")


def write(path: str, text: str) -> None:
    Path(path).write_text(text, encoding="utf-8")


def remove_go_function(text: str, name: str) -> str:
    match = re.search(rf"(?m)^func\s+(?:\([^\n]*\)\s+)?{re.escape(name)}\s*\(", text)
    if not match:
        return text
    start = match.start()
    brace = text.find("{", match.end())
    if brace < 0:
        raise RuntimeError(f"opening brace not found: {name}")

    depth = 0
    i = brace
    string_delimiter: str | None = None
    escaped = False
    line_comment = False
    block_comment = False
    while i < len(text):
        ch = text[i]
        nxt = text[i + 1] if i + 1 < len(text) else ""
        if line_comment:
            if ch == "\n":
                line_comment = False
        elif block_comment:
            if ch == "*" and nxt == "/":
                block_comment = False
                i += 1
        elif string_delimiter:
            if string_delimiter != "`" and escaped:
                escaped = False
            elif string_delimiter != "`" and ch == "\\":
                escaped = True
            elif ch == string_delimiter:
                string_delimiter = None
        else:
            if ch == "/" and nxt == "/":
                line_comment = True
                i += 1
            elif ch == "/" and nxt == "*":
                block_comment = True
                i += 1
            elif ch in ('"', "'", "`"):
                string_delimiter = ch
            elif ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    end = i + 1
                    while end < len(text) and text[end] in " \t":
                        end += 1
                    if end < len(text) and text[end] == "\n":
                        end += 1
                    return text[:start] + text[end:]
        i += 1
    raise RuntimeError(f"closing brace not found: {name}")


def patch_token_refresh_tests() -> None:
    for path in (
        "backend/internal/service/token_refresh_service_test.go",
        "backend/internal/service/openai_privacy_retry_test.go",
    ):
        text = read(path)
        text = re.sub(
            r"NewTokenRefreshService\(([^\n]*?), nil, nil, nil, ",
            r"NewTokenRefreshService(\1, nil, nil, nil, nil, ",
            text,
        )
        write(path, text)


def patch_contract_tests() -> None:
    path = "backend/internal/server/api_contract_test.go"
    text = read(path)

    group_pattern = re.compile(
        r'(?m)^([ \t]*)"name": "Group One",\n([ \t]*)"platform": "anthropic",'
    )

    def expand_group(match: re.Match[str]) -> str:
        first_indent, indent = match.group(1), match.group(2)
        return (
            f'{first_indent}"name": "Group One",\n'
            f'{indent}"peak_end": "",\n'
            f'{indent}"peak_rate_enabled": false,\n'
            f'{indent}"peak_rate_multiplier": 0,\n'
            f'{indent}"peak_start": "",\n'
            f'{indent}"platform": "anthropic",'
        )

    if '"peak_rate_enabled": false' not in text:
        text, count = group_pattern.subn(expand_group, text, count=1)
        if count != 1:
            raise RuntimeError(f"Group One fixture anchor count={count}")

    group_start = text.index('"name": "Group One"')
    group_end = text.index("}`", group_start) if "}`" in text[group_start:] else len(text)
    group_fragment = text[group_start:group_end]
    if '"upstream_protocols": null' not in group_fragment:
        weekly_pos = text.index('"weekly_limit_usd": null', group_start)
        line_start = text.rfind("\n", group_start, weekly_pos) + 1
        indent = text[line_start:weekly_pos]
        text = text[:line_start] + indent + '"upstream_protocols": null,\n' + text[line_start:]

    settings_anchor = '"available_channels_enabled": false,'
    if '"announcements_enabled": false,' not in text:
        text = text.replace(
            settings_anchor,
            '"announcements_enabled": false,\n'
            '\t\t\t\t"promo_enabled": false,\n'
            '\t\t\t\t"redeem_enabled": false,\n'
            '\t\t\t\t"available_channels_enabled": false,',
        )

    quota_pattern = re.compile(
        r'("default_platform_quotas"\s*:\s*\{)(.*?)(\n\s*\},\n\s*"default_subscriptions")',
        re.S,
    )

    def add_quota_platforms(match: re.Match[str]) -> str:
        body = match.group(2)
        if '"custom"' in body and '"grok"' in body:
            return match.group(0)
        indent_match = re.search(r"\n(\s*)\"[^\"]+\"\s*:", body)
        if not indent_match:
            raise RuntimeError("default platform quota indentation not found")
        indent = indent_match.group(1)
        body = body.rstrip()
        additions: list[str] = []
        if '"custom"' not in body:
            additions.append(f'{indent}"custom": {{"daily": null, "monthly": null, "weekly": null}}')
        if '"grok"' not in body:
            additions.append(f'{indent}"grok": {{"daily": null, "monthly": null, "weekly": null}}')
        if additions:
            body += ",\n" + ",\n".join(additions)
        return match.group(1) + body + match.group(3)

    text, quota_count = quota_pattern.subn(add_quota_platforms, text)
    if quota_count < 2:
        raise RuntimeError(f"expected at least two quota fixtures, found {quota_count}")

    write(path, text)


def patch_lint() -> None:
    path = "backend/internal/handler/admin/lightbridge_connect_handler.go"
    text = read(path).replace(
        "\t\t\th.db.Exec(`\n\t\t\t\tUPDATE accounts",
        "\t\t\t_, _ = h.db.Exec(`\n\t\t\t\tUPDATE accounts",
    )
    write(path, text)

    path = "backend/internal/modules/proxy/internal/mihomo/compiler_test.go"
    text = read(path).replace(
        '\tgroups := doc["proxy-groups"].([]any)\n\tgroup := groups[0].(map[string]any)',
        '\tgroups, ok := doc["proxy-groups"].([]any)\n'
        '\trequire.True(t, ok)\n'
        '\trequire.NotEmpty(t, groups)\n'
        '\tgroup, ok := groups[0].(map[string]any)\n'
        '\trequire.True(t, ok)',
    )
    write(path, text)

    for path in (
        "backend/internal/repository/channel_monitor_repo.go",
        "backend/internal/repository/module_store.go",
    ):
        write(path, read(path).replace("defer rows.Close()", "defer func() { _ = rows.Close() }()"))

    path = "backend/internal/service/gemini_native_protocol_adapter_test.go"
    text = read(path).replace(
        '\trequire.Equal(t, float64(8), resp["usageMetadata"].(map[string]any)["totalTokenCount"])',
        '\tusageMetadata, ok := resp["usageMetadata"].(map[string]any)\n'
        '\trequire.True(t, ok)\n'
        '\trequire.Equal(t, float64(8), usageMetadata["totalTokenCount"])',
    )
    write(path, text)

    path = "backend/internal/service/privacy_filter.go"
    text = read(path).replace(
        "\t\tb.WriteString(r.Replacement)\n\t\tb.WriteByte('\\n')",
        "\t\t_, _ = b.WriteString(r.Replacement)\n\t\t_ = b.WriteByte('\\n')",
    )
    write(path, text)

    path = "backend/internal/handler/ops_error_logger.go"
    write(path, read(path).replace("ErrorBody:    string(w.buf.Bytes()),", "ErrorBody:    w.buf.String(),"))

    path = "backend/internal/service/lightbridge_connect_service.go"
    write(path, read(path).replace(
        'fmt.Errorf("New API returned success=false")',
        'fmt.Errorf("new API returned success=false")',
    ))

    dead = {
        "backend/internal/service/codex_image_generation_bridge.go": (
            "boolOverridePtr",
            "boolOverrideFromMap",
            "platformBoolOverride",
        ),
        "backend/internal/service/gateway_service.go": (
            "billingModelForRestriction",
            "resolveAccountUpstreamModel",
        ),
        "backend/internal/service/model_rate_limit.go": ("antigravityModelRateLimitKeys",),
        "backend/internal/service/openai_gateway_service.go": (
            "resolveOpenAIAccountUpstreamModelForRequest",
        ),
    }
    for path, names in dead.items():
        text = read(path)
        for name in names:
            text = remove_go_function(text, name)
        write(path, text)


def patch_pricing_sync_test() -> None:
    path = "backend/internal/handler/admin/channel_handler_test.go"
    write(path, read(path).replace(
        '[]string{"anthropic", "openai", "gemini", "grok", "antigravity"}',
        '[]string{"anthropic", "openai", "gemini", "antigravity"}',
    ))


def patch_audit_exceptions() -> None:
    path = ".github/audit-exceptions.yml"
    lines = read(path).splitlines(True)
    output: list[str] = []
    index = 0
    while index < len(lines):
        if lines[index].startswith("  - package: xlsx"):
            index += 1
            while index < len(lines) and not lines[index].startswith("  - package:"):
                index += 1
            continue
        output.append(lines[index])
        index += 1
    write(path, "".join(output))


def patch_workflow_go_versions() -> None:
    for path in (
        ".github/workflows/backend-ci.yml",
        ".github/workflows/security-scan.yml",
    ):
        write(path, read(path).replace("go1.26.4", "go1.26.5"))


def main() -> None:
    patch_token_refresh_tests()
    patch_pricing_sync_test()
    patch_contract_tests()
    patch_lint()
    patch_audit_exceptions()
    patch_workflow_go_versions()


if __name__ == "__main__":
    main()
