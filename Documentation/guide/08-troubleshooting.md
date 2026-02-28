# Guide ⑧：故障排查（Troubleshooting）

本篇按“你看到的现象 → 可能原因 → 解决步骤”的方式整理常见问题。

---

## 1) 访问 `/admin` 一直跳转 / 进不去

### 现象 A：跳到 `/admin/setup`

原因：

- 数据库里还没有管理员账号（未完成 Setup）

解决：

- 打开 `/admin/setup` 完成初始化

### 现象 B：跳到 `/admin/login`，登录后又被踢出

可能原因：

- 你未设置 `LIGHTBRIDGE_COOKIE_SECRET`，每次重启都会随机生成 secret，导致旧 Cookie 失效

解决：

- 这是预期行为：重新登录即可
- 若你希望跨重启保持：设置 `LIGHTBRIDGE_COOKIE_SECRET` 为固定值

---

## 2) 调用 `/v1/*` 返回 401

### 现象：`missing_api_key`

返回类似：

```json
{"error":{"message":"missing bearer token","type":"authentication_error","code":"missing_api_key"}}
```

原因：

- 请求没带 `Authorization: Bearer <CLIENT_KEY>`

解决：

- 在客户端/脚本中配置 Client API Key

### 现象：`invalid_api_key`

原因：

- Client Key 填错
- 该 key 在管理后台被禁用/删除
- 你访问的是 App URL（`/openai/<app>/v1`），但该 app 绑定了其它 key_id

解决：

- 去 `/admin/router` 检查 key 是否启用
- 若使用 App URL：检查该 app 的绑定 key_id

---

## 3) 调用 `/v1/chat/completions` 返回 400：`invalid_json`

原因：

- 请求 body 非 JSON（或 JSON 格式错误）

解决：

- 确保 `Content-Type: application/json`
- 确保 body 是合法 JSON

补充说明：

- Core 需要解析 JSON 才能提取 `model` 做路由；因此对“非 JSON 且非空 body”的请求会直接报错。

---

## 4) 返回 502/501：Provider 相关

### 现象：`provider_not_found`

原因：

- 路由解析得到的 provider ID 在 DB 中不存在
- 例如你请求了 `model@codex` 但并没有 `id=codex` 的 Provider

解决：

- 去 `/admin/providers` 确认该 Provider 是否存在/启用
- 若是 Codex：按 [Codex OAuth](./04-codex-oauth.md) 安装模块并保存 Provider

### 现象：`provider_protocol_not_supported`

原因：

- Provider 的 `protocol` 在 Core 中没有对应适配器
- 常见：手动把 protocol 填成了 `gemini`（目前未实现）或拼写错误

解决：

- 去 `/admin/providers` 修正协议字段

### 现象：`provider_misconfigured`

原因：

- Provider endpoint 为空
- 或 anthropic/codex 等协议缺少必需的 `api_key`（取决于你指向的上游）

解决：

- 去 `/admin/providers` 完整填写 Endpoint 与 API Key/Token

### 现象：`501_not_supported`

原因：

- 你请求了某个 Provider 协议不支持的端点
- 例如 `anthropic` / `codex` 目前只支持 `/v1/chat/completions` 与 `/v1/responses`

解决：

- 对其它 `/v1/*` 端点使用 `forward` 或 `http_openai` 类 Provider

---

## 5) 返回 429：rate_limit_exceeded

原因：

- 同一个 Client API Key 在 1 分钟内请求次数超过 120（默认）

解决：

- 客户端做退避重试（根据 `Retry-After: 60`）
- 拆分多个 Client Key 分流

---

## 6) 模块安装/启动失败

### 现象：Marketplace 安装时报 sha256 错误

原因：

- index.json 的 `sha256` 与实际 zip 不一致

解决：

- 重新生成模块包与 index（以模块 `package.sh` 输出为准）

### 现象：模块启动失败 / health check 超时

可能原因：

- manifest 里 health path 配错
- entrypoint 不存在或不可执行
- 模块监听的端口与 Core 注入的端口不一致（模块必须绑定 `LIGHTBRIDGE_HTTP_PORT` 或 `LIGHTBRIDGE_GRPC_PORT`）

解决：

- 通过模块 stdout/stderr 查看错误
- 用 `/admin/api/modules` 找到端口后，直接 curl 模块的 `/health`

---

## 7) Codex OAuth 弹窗“生成 OAuth 链接”失败

可能原因：

- `openai-codex-oauth` 模块未安装/未启动
- 本机 `1455` 端口被占用，Core 无法启动回调监听

解决：

1. 确认模块已运行（`/admin/api/modules` 里有 http_port）
2. 若 `1455` 端口冲突：
   - 释放该端口，或
   - 使用备用方案：粘贴回调 URL → 从回调 URL 获取 Token

详见：[Codex OAuth](./04-codex-oauth.md)

---

## 8) 我不知道数据库/数据目录在哪

优先看：

- [数据目录结构](../reference/05-data-dir-layout.md)

快速结论：

- 默认在系统配置目录下（macOS 常见为 `~/Library/Application Support/LightBridge`）
- 也可通过 `LIGHTBRIDGE_DATA_DIR` 覆盖

