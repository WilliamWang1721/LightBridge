# Reference ②：对外 HTTP API（OpenAI 兼容入口）

本篇仅描述“对外服务端口”暴露的接口（下游客户端使用），并强调鉴权与边界。

---

## 0. 鉴权（必须）

除 `/healthz` 外，对外 `/v1/*` 与 `/openai/*` 均需要 Client API Key：

- Header：`Authorization: Bearer <CLIENT_KEY>`

无 token：

- 401 + `missing_api_key`

token 无效/被禁用：

- 401 + `invalid_api_key`

---

## 1. Health

### `GET /healthz`

无需鉴权。

响应：

```json
{"status":"ok","name":"lightbridge"}
```

---

## 2. Models

### `GET /v1/models`

返回虚拟模型列表（包含基础模型与变体 `model@provider`）。

- 需要鉴权
- 仅支持 `GET`，其它方法返回 405（OpenAI 风格错误）

### `GET /openai/v1/models`

与 `/v1/models` 等价（路径别名）。

### `GET /openai/<app>/v1/models`

与 `/v1/models` 等价，但：

- 可能受 app 绑定 key 限制
- 会额外补充 app 的 `from -> to` 映射项（`provider_hint: mapped->...`）

---

## 3. Chat Completions

### `POST /v1/chat/completions`

行为取决于路由选中的 Provider：

- `forward/http_openai/http_rpc`：HTTP 透传上游
- `anthropic`：OpenAI → Claude 协议转换
- `codex`：OpenAI Chat → Codex Responses 转换（上游调用 `POST <endpoint>/responses`）

注意：

- Core 会读取 JSON body 提取 `model` 做路由；body 非空且非 JSON 会返回 `invalid_json`

### `POST /openai/v1/chat/completions`

等价别名。

### `POST /openai/<app>/v1/chat/completions`

等价别名 + app 绑定 key + app 模型映射。

---

## 4. Responses

### `POST /v1/responses`

同样会根据 Provider 协议决定：

- `forward`：透传上游 `/v1/responses`
- `anthropic`：OpenAI Responses → Claude 转换（上游走 messages）
- `codex`：转发到 `POST <endpoint>/responses`

---

## 5. 其它 `/v1/*` 透传路径

### `GET/POST /v1/*`

Core 会把 `/v1/*` 交给选中的 Provider 处理：

- 如果 body 里存在 `model`，会用于路由解析
- 如果 body 为空或没有 model，会走默认 provider（通常是 `forward`）

边界提示：

- Core 当前对“非空且非 JSON body”会返回 `invalid_json`（因为需要解析 model）

---

## 6. 错误格式（OpenAI 风格）

Core 返回的错误通常符合：

```json
{
  "error": {
    "message": "...",
    "type": "...",
    "code": "..."
  }
}
```

常见 code：

- `missing_api_key`
- `invalid_api_key`
- `invalid_json`
- `routing_failed`
- `provider_not_found`
- `provider_protocol_not_supported`
- `provider_misconfigured`
- `rate_limit_exceeded`
- `501_not_supported`

---

## 7. 限流

默认对 `/v1/*` 与 `/openai/*` 做每 Key 每分钟 120 次限流，超限返回 429：

- Header：`Retry-After: 60`
- code：`rate_limit_exceeded`

详见：[日志与限流](../guide/06-logs-and-rate-limit.md)

