# Guide ⑥：日志与限流（Logs & Rate Limit）

本篇讲清楚 LightBridge 当前“记录了什么”“没记录什么”，以及默认的限流行为与返回码，方便你上线后排障。

---

## 1. 进程日志（stdout）

Core 在 stdout 会打印最简单的访问日志（方法 + 路径 + 耗时），例如：

```text
POST /v1/chat/completions 120.3ms
GET /admin/providers 3.1ms
```

适用：

- 开发调试（直接看终端）
- 生产部署（由 systemd/docker/日志采集接管）

---

## 2. 请求元数据日志（Request Logs Meta）

Core 会把“请求元数据”写入 SQLite（不包含完整 prompt/response）。

可在管理后台查看：

- `/admin/logs`

底层 API：

- `GET /admin/api/logs`（最多返回 200 条）

典型字段包括：

- `ts`：时间戳（UTC）
- `request_id`：Core 生成的请求 ID
- `client_key_id`：使用的 Client Key 的 ID（不是 key 明文）
- `provider_id`：最终选中的 Provider
- `model_id`：请求的 model（或映射后的 model）
- `path`：请求路径（含 `/openai/...` 的原始路径会尽量保留用于排障）
- `status`：最终状态码
- `latency_ms`：耗时
- `error_code`：错误码（如 `invalid_json` / `provider_not_found` 等）
- `input_tokens/output_tokens`：当前版本可能为 0（若未实现 usage 统计或上游未返回）

> 重要：该日志用于“运维排障与统计趋势”，不是审计级别的全量日志。

---

## 3. 日志清理（Prune）

提供手动清理接口：

- `POST /admin/api/logs/prune`

当前默认策略（服务端写死）：

- 删除 30 天前的日志
- 同时限制最多保留 50000 行

返回：

```json
{ "ok": true, "deleted": 123 }
```

---

## 4. 限流（Rate Limit）

Core 默认对以下路径启用“按 Key 的 token-bucket 限流”：

- `/v1/*`
- `/openai/*`

默认限制：

- **120 次请求 / 分钟 / Key**
- burst 也是 120

触发限流时：

- HTTP 429
- Header：`Retry-After: 60`
- Body（OpenAI 风格错误）：

```json
{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}
```

说明：

- 限流 key 默认取 `Authorization: Bearer <token>` 的 token 值
- 若请求未携带 token，会退化为按 `RemoteAddr` 限流

---

## 5. 你应该如何用这些信息排查问题？

建议顺序：

1. `/admin/logs` 看最近请求的 `status / error_code / provider_id / path`
2. 确认对应 Provider 是否：
   - enabled
   - health 不是 down/disabled
   - endpoint 与 api_key 配置正确
3. 若是模块 Provider：
   - `/admin/api/modules` 看模块 runtime 端口是否存在
   - 模块 stdout 是否有报错
4. 若是限流：
   - 客户端侧做退避重试
   - 或拆分不同 Client Key 分流

---

## 下一步

- 数据与密钥安全建议：[数据目录与安全注意事项](./07-security.md)
- 常见错误码排查：[故障排查](./08-troubleshooting.md)

