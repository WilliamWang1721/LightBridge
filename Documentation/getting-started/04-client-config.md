# Getting Started ④：客户端接入（Base URL / API Key）

本篇把 LightBridge 当作“OpenAI 兼容网关”来使用：你需要在第三方客户端里配置 Base URL 与 API Key，并理解几种常用 URL 形式。

---

## 1. 你需要填的两项

在任何 OpenAI 兼容客户端里，通常只需要两项：

1. **Base URL**
2. **API Key**（这里填 LightBridge 的 **Client API Key**，不是上游 OpenAI/Anthropic 的 Key）

---

## 2. Base URL 的三种常用形式

### 2.1 标准形式（推荐）

- Base URL：`http://127.0.0.1:3210/v1`

此时客户端会请求：

- `GET  /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- 以及其它 `/v1/*`

### 2.2 OpenAI 别名形式（兼容某些客户端）

LightBridge 额外提供一个别名前缀 `/openai`：

- Base URL：`http://127.0.0.1:3210/openai/v1`

其行为等价于 `/v1`（只是多了一层路径前缀，方便某些“固定路径”客户端）。

### 2.3 App 形式（应用适配 / 单独 Key / 单独模型映射）

LightBridge 支持：

- `http://127.0.0.1:3210/openai/<app>/v1`

例如：

- `http://127.0.0.1:3210/openai/codex/v1`
- `http://127.0.0.1:3210/openai/cherry-studio/v1`

App 的作用：

- 你可以为某个 app 绑定“专用 Client Key”（只有这个 key 才能访问该 app URL）
- 你可以为某个 app 配置“模型映射”（例如把客户端请求的 `gpt-4o-mini` 映射到你实际想走的模型/路由）

配置入口见：[OpenAI 别名与应用适配（Apps）](../guide/05-openai-alias-and-apps.md)

---

## 3. Client API Key 从哪里来？

三种方式获取：

1. Setup Wizard 完成时自动生成的 `default_client_key`
2. Router 页面：`/admin/router` → “新建密钥”
3. AUTH 页面：`/admin/auth` → “新建密钥”（若你直接访问该页面）

> 注意：Client API Key 存在 SQLite 中，可启用/禁用/删除；不要与上游 Provider 的 API Key 混淆。

---

## 4. 最小可用的调用验证（强烈建议先用 curl 验证）

### 4.1 Models

```bash
curl -s "http://127.0.0.1:3210/v1/models" \
  -H "Authorization: Bearer <CLIENT_KEY>"
```

### 4.2 Chat Completions（非流式）

```bash
curl -s "http://127.0.0.1:3210/v1/chat/completions" \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"gpt-4o-mini",
    "messages":[{"role":"user","content":"Say hi in one sentence."}]
  }'
```

### 4.3 Chat Completions（流式 SSE）

```bash
curl -N "http://127.0.0.1:3210/v1/chat/completions" \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"gpt-4o-mini",
    "stream": true,
    "messages":[{"role":"user","content":"Count 1 to 5."}]
  }'
```

---

## 5. 常见误区（非常重要）

### 5.1 “我配置了 Base URL 和 Client Key，但还是 401/502”

你可能还没配置任何可用的上游 Provider：

- 内置 `forward` 默认 endpoint 指向 `https://api.openai.com/v1`，但 **没有** 你的 OpenAI API Key
- 内置 `anthropic` 默认 endpoint 指向 `https://api.anthropic.com`，但 **没有** 你的 Anthropic Key

解决：去 `/admin/providers` 为对应 Provider 填写 `API Key / Token` 并保存。

### 5.2 “为什么我传了某些非 JSON body 的接口会报 invalid_json？”

当前网关在进入 `/v1/*` 时会尝试解析请求 body 为 JSON 以提取 `model` 做路由；如果 body 非 JSON 且非空，会返回 `invalid_json`。

因此本 MVP 更适合：

- Chat/Responses 等 JSON 请求

如果你需要兼容 multipart/form-data 等非 JSON 请求，请在使用前先用 curl 验证该端点是否符合预期（并结合后续版本演进）。

---

## 下一步（按你要做的事）

- 配置/管理 Provider：[Provider 管理](../guide/01-providers.md)
- 配置模型路由/负载均衡/故障转移：[模型路由与故障转移](../guide/02-routing.md)

