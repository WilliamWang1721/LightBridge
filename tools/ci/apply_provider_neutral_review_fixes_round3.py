#!/usr/bin/env python3
import runpy
from pathlib import Path

runpy.run_path("tools/ci/apply_provider_neutral_review_fixes_round2.py", run_name="__main__")

service_path = Path("backend/internal/service/channel_service.go")
service = service_path.read_text(encoding="utf-8")
old_function = '''// matchingPlatforms 返回请求平台对应的可匹配平台列表。
// 显式平台始终严格隔离；缺少协议上下文时，仅在渠道配置只有一个平台时
// 使用该唯一平台。多平台配置保持 fail-closed，避免跨平台同名模型串价。
func matchingPlatforms(cache *channelCache, groupID int64, requestPlatform string) []string {
\trequestPlatform = strings.TrimSpace(requestPlatform)
\tif requestPlatform != "" {
\t\treturn []string{requestPlatform}
\t}
\tplatforms := cache.platformsByGroupID[groupID]
\tif len(platforms) != 1 {
\t\treturn nil
\t}
\tfor platform := range platforms {
\t\treturn []string{platform}
\t}
\treturn nil
}
'''
new_function = '''// matchingPlatforms 返回请求平台对应的可匹配平台列表。
// 显式平台查询先查本平台，再查平台中立（Platform 为空）的定价或映射。
// 缺少协议上下文时，仅在渠道配置只有一个具体平台时允许回退到该平台；
// 多平台配置只允许平台中立项，避免同名模型跨平台串价。
func matchingPlatforms(cache *channelCache, groupID int64, requestPlatform string) []string {
\trequestPlatform = strings.TrimSpace(requestPlatform)
\tif requestPlatform != "" {
\t\treturn []string{requestPlatform, ""}
\t}
\tplatforms := cache.platformsByGroupID[groupID]
\tif len(platforms) == 1 {
\t\tfor platform := range platforms {
\t\t\treturn []string{platform, ""}
\t\t}
\t}
\treturn []string{""}
}
'''
if service.count(old_function) != 1:
    raise SystemExit("matchingPlatforms implementation was not found exactly once")
service_path.write_text(service.replace(old_function, new_function, 1), encoding="utf-8")

test_path = Path("backend/internal/service/channel_service_test.go")
test = test_path.read_text(encoding="utf-8")
old_test = '''func TestMatchingPlatforms(t *testing.T) {
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
new_test = '''func TestMatchingPlatforms(t *testing.T) {
\tcache := newEmptyChannelCache()
\tcache.platformsByGroupID[10] = map[string]struct{}{PlatformAnthropic: {}}
\tcache.platformsByGroupID[20] = map[string]struct{}{
\t\tPlatformAnthropic: {},
\t\tPlatformOpenAI:    {},
\t}

\trequire.Equal(t, []string{PlatformAntigravity, ""}, matchingPlatforms(cache, 10, PlatformAntigravity))
\trequire.Equal(t, []string{PlatformAnthropic, ""}, matchingPlatforms(cache, 10, ""))
\trequire.Equal(t, []string{""}, matchingPlatforms(cache, 20, ""))
\trequire.Equal(t, []string{""}, matchingPlatforms(cache, 999, ""))
}
'''
if test.count(old_test) != 1:
    raise SystemExit("TestMatchingPlatforms was not found exactly once")
test_path.write_text(test.replace(old_test, new_test, 1), encoding="utf-8")
