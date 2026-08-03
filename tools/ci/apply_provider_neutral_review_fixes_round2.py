#!/usr/bin/env python3
import runpy
from pathlib import Path

runpy.run_path("tools/ci/apply_provider_neutral_review_fixes.py", run_name="__main__")

path = Path("backend/internal/service/channel_service_test.go")
text = path.read_text(encoding="utf-8")


def replace_once(old: str, new: str) -> None:
    global text
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one occurrence, found {count}: {old[:120]!r}")
    text = text.replace(old, new, 1)


def replace_in_section(start: str, end: str, old: str, new: str) -> None:
    global text
    start_index = text.index(start)
    end_index = text.index(end, start_index)
    section = text[start_index:end_index]
    count = section.count(old)
    if count != 1:
        raise SystemExit(f"expected one occurrence in section {start!r}, found {count}: {old!r}")
    section = section.replace(old, new, 1)
    text = text[:start_index] + section + text[end_index:]


old_matching = '''func TestMatchingPlatforms(t *testing.T) {
\ttests := []struct {
\t\tname          string
\t\tgroupPlatform string
\t\twant          []string
\t}{
\t\t{"antigravity returns itself only", PlatformAntigravity, []string{PlatformAntigravity}},
\t\t{"anthropic returns itself", PlatformAnthropic, []string{PlatformAnthropic}},
\t\t{"gemini returns itself", PlatformGemini, []string{PlatformGemini}},
\t\t{"openai returns itself", PlatformOpenAI, []string{PlatformOpenAI}},
\t}

\tfor _, tt := range tests {
\t\tt.Run(tt.name, func(t *testing.T) {
\t\t\tresult := matchingPlatforms(tt.groupPlatform)
\t\t\trequire.Equal(t, tt.want, result)
\t\t})
\t}
}
'''
new_matching = '''func TestMatchingPlatforms(t *testing.T) {
\tcache := newEmptyChannelCache()
\tcache.platformsByGroupID[10] = map[string]struct{}{PlatformAnthropic: {}}
\tcache.platformsByGroupID[20] = map[string]struct{}{
\t\tPlatformAnthropic: {},
\t\tPlatformOpenAI:    {},
\t}

\trequire.Equal(t, []string{PlatformAntigravity}, matchingPlatforms(cache, 10, PlatformAntigravity))
\trequire.Equal(t, []string{PlatformAnthropic}, matchingPlatforms(cache, 10, ""))
\trequire.Nil(t, matchingPlatforms(cache, 20, ""))
\trequire.Nil(t, matchingPlatforms(cache, 999, ""))
}
'''
replace_once(old_matching, new_matching)

replacements = [
    (
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeCrossPlatformPricing",
        "func TestGetChannelModelPricing_AnthropicCannotSeeAntigravityPricing",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "claude-opus-4-6")',
        'result := svc.GetChannelModelPricing(channelContextForPlatform(PlatformAntigravity), 10, "claude-opus-4-6")',
    ),
    (
        "func TestGetChannelModelPricing_AnthropicCannotSeeAntigravityPricing",
        "// ===========================================================================\n// 10. Antigravity platform isolation",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "claude-opus-4-6")',
        'result := svc.GetChannelModelPricing(channelContextForPlatform(PlatformAnthropic), 10, "claude-opus-4-6")',
    ),
    (
        "func TestResolveChannelMapping_AntigravityDoesNotSeeCrossPlatformMapping",
        "// ===========================================================================\n// 11. Antigravity platform isolation",
        'result := svc.ResolveChannelMapping(context.Background(), 10, "claude-opus-4-5")',
        'result := svc.ResolveChannelMapping(channelContextForPlatform(PlatformAntigravity), 10, "claude-opus-4-5")',
    ),
    (
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeSameModelFromOtherPlatforms",
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeGeminiOnlyPricing",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "shared-model")',
        'result := svc.GetChannelModelPricing(channelContextForPlatform(PlatformAntigravity), 10, "shared-model")',
    ),
    (
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeGeminiOnlyPricing",
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeWildcardFromOtherPlatforms",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "gemini-model")',
        'result := svc.GetChannelModelPricing(channelContextForPlatform(PlatformAntigravity), 10, "gemini-model")',
    ),
    (
        "func TestGetChannelModelPricing_AntigravityDoesNotSeeWildcardFromOtherPlatforms",
        "func TestResolveChannelMapping_AntigravityDoesNotSeeMappingFromOtherPlatforms",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "shared-model")',
        'result := svc.GetChannelModelPricing(channelContextForPlatform(PlatformAntigravity), 10, "shared-model")',
    ),
    (
        "func TestResolveChannelMapping_AntigravityDoesNotSeeMappingFromOtherPlatforms",
        "func TestCheckRestricted_AntigravityDoesNotSeeModelsFromOtherPlatforms",
        'result := svc.ResolveChannelMapping(context.Background(), 10, "alias")',
        'result := svc.ResolveChannelMapping(channelContextForPlatform(PlatformAntigravity), 10, "alias")',
    ),
    (
        "func TestCheckRestricted_AntigravityDoesNotSeeModelsFromOtherPlatforms",
        "func TestGetChannelModelPricing_AntigravityOwnPricingWorks",
        'restricted := svc.IsModelRestricted(context.Background(), 10, "shared-model")',
        'antigravityCtx := channelContextForPlatform(PlatformAntigravity)\n\trestricted := svc.IsModelRestricted(antigravityCtx, 10, "shared-model")',
    ),
    (
        "func TestCheckRestricted_AntigravityDoesNotSeeModelsFromOtherPlatforms",
        "func TestGetChannelModelPricing_AntigravityOwnPricingWorks",
        'restricted = svc.IsModelRestricted(context.Background(), 10, "unknown-model")',
        'restricted = svc.IsModelRestricted(antigravityCtx, 10, "unknown-model")',
    ),
    (
        "func TestGetChannelModelPricing_AntigravityOwnPricingWorks",
        "func TestGetChannelModelPricing_NonAntigravityUnaffected",
        'result := svc.GetChannelModelPricing(context.Background(), 10, "claude-sonnet-4")',
        'antigravityCtx := channelContextForPlatform(PlatformAntigravity)\n\tresult := svc.GetChannelModelPricing(antigravityCtx, 10, "claude-sonnet-4")',
    ),
    (
        "func TestGetChannelModelPricing_AntigravityOwnPricingWorks",
        "func TestGetChannelModelPricing_NonAntigravityUnaffected",
        'result = svc.GetChannelModelPricing(context.Background(), 10, "gemini-2.5-flash")',
        'result = svc.GetChannelModelPricing(antigravityCtx, 10, "gemini-2.5-flash")',
    ),
]

for start, end, old, new in replacements:
    replace_in_section(start, end, old, new)

path.write_text(text, encoding="utf-8")
