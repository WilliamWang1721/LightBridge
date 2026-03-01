# Reference ②：对外 HTTP API（多协议入口）

本篇仅描述“对外服务端口”暴露的接口（下游客户端使用），并强调鉴权与边界。

---

## 0. 鉴权（必须）

除 `/healthz` 外，对外协议入口均需要 Client API Key，支持以下任一形式：

- Header：`Authorization: Bearer <CLIENT_KEY>`
- Header：`x-api-key: <CLIENT_KEY>`
- Header：`x-goog-api-key: <CLIENT_KEY>`
- Query：`?key=<CLIENT_KEY>`（主要用于 Gemini SDK 兼容）

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

### 其它协议前缀（原生入站）

- `GET /openai-responses/v1/models`
- `GET /anthropic/v1/models`
- `GET /azure/openai/v1/models`
- `GET /gemini/v1beta/models`

说明：其中 `/gemini/v1beta/models` 走 Gemini 原生语义；其余会按入口协议 + 路由结果决定返回语义。

---

## 3. Chat Completions

### `POST /v1/chat/completions`

行为取决于路由选中的 Provider：

- `forward/http_openai/http_rpc`：HTTP 透传上游
- `openai`：HTTP 透传上游（规范协议名）
- `openai_responses`：优先走 `/responses`，并在必要时完成 chat↔responses 转换
- `gemini`：OpenAI ↔ Gemini 转换
- `anthropic`：OpenAI → Claude 协议转换
- `azure_openai`：Azure OpenAI `v1` / legacy deployments 双栈分发
- `codex`：OpenAI Chat → Codex Responses 转换（上游调用 `POST <endpoint>/responses`）

注意：

- Core 会读取 JSON body 提取 `model` 做路由；body 非空且非 JSON 会返回 `invalid_json`

### `POST /openai/v1/chat/completions`

等价别名。

### `POST /openai/<app>/v1/chat/completions`

等价别名 + app 绑定 key + app 模型映射。

### 其它入口别名（OpenAI 风格）

- `POST /openai-responses/v1/chat/completions`
- `POST /azure/openai/v1/chat/completions`

---

## 4. Responses

### `POST /v1/responses`

同样会根据 Provider 协议决定：

- `forward`：透传上游 `/v1/responses`
- `openai/openai_responses`：透传或转换到目标 `/responses`
- `gemini`：OpenAI Responses ↔ Gemini 转换
- `anthropic`：OpenAI Responses → Claude 转换（上游走 messages）
- `azure_openai`：在 `v1` 栈下透传 `/openai/v1/responses`（legacy 栈不支持）
- `codex`：转发到 `POST <endpoint>/responses`

---

## 5. 原生协议入口（新增）

### Anthropic Messages

- `POST /anthropic/v1/messages`
- `POST /anthropic/<app>/v1/messages`

### Gemini Native

- `POST /gemini/v1beta/models/{model}:generateContent`
- `POST /gemini/v1beta/models/{model}:streamGenerateContent`
- `POST /gemini/v1beta/models/{model}:countTokens`
- 以及对应 `/<app>/v1beta/...` 变体

### Azure OpenAI

- `POST /azure/openai/v1/*`
- `POST /azure/openai/<app>/v1/*`
- `POST /azure/openai/deployments/{deployment}/*?api-version=...`
- `POST /azure/openai/<app>/deployments/{deployment}/*?api-version=...`

对于协议天然不支持的组合，网关返回 `not_supported`，并在 `error` 对象里返回结构化字段：

- `source_protocol`
- `target_protocol`
- `endpoint_kind`

示例：

```json
{
  "error": {
    "message": "route is not supported for this protocol combination (source=gemini target=openai kind=generate_content)",
    "type": "not_supported",
    "code": "not_supported",
    "source_protocol": "gemini",
    "target_protocol": "openai",
    "endpoint_kind": "generate_content"
  }
}
```

---

## 6. 其它 `/v1/*` 透传路径

### `GET/POST /v1/*`

Core 会把 `/v1/*` 交给选中的 Provider 处理：

- 如果 body 里存在 `model`，会用于路由解析
- 如果 body 为空或没有 model，会走默认 provider（通常是 `forward`）

边界提示：

- Core 当前对“非空且非 JSON body”会返回 `invalid_json`（因为需要解析 model）

---

## 7. 错误格式（OpenAI 风格）

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
- `not_supported`

---

## 8. 限流

默认对以下入口做每 Key 每分钟 120 次限流：

- `/v1/*`
- `/openai/*`
- `/openai-responses/*`
- `/gemini/*`
- `/anthropic/*`
- `/claude/*`
- `/azure/openai/*`

- Header：`Retry-After: 60`
- code：`rate_limit_exceeded`

详见：[日志与限流](../guide/06-logs-and-rate-limit.md)
