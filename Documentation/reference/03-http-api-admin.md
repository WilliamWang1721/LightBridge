# Reference ③：管理后台 Admin（页面路由 + Admin API）

本篇列出管理后台的页面路由与 JSON API（用于自动化/脚本/二次开发）。

---

## 0. Admin 认证模型

管理后台分两类：

1. **页面（HTML）**：`/admin/*`
   - 使用 Cookie 会话（`lightbridge_admin`）
2. **API（JSON）**：`/admin/api/*`
   - 同样使用 Cookie 会话

### 0.1 常见返回

- 未初始化（没有 admin 用户）：
  - 页面：重定向到 `/admin/setup`
  - API：`403`，`{"error":"setup required"}`
- 未登录：
  - 页面：重定向到 `/admin/login`
  - API：`401`，`{"error":"login required"}`

---

## 1. 页面路由（/admin/*）

以下页面在当前 Core 中注册（`internal/gateway/server.go`）：

- `/admin`：入口（按初始化/登录状态跳转）
- `/admin/setup`：初始化向导
- `/admin/login`：管理员登录
- `/admin/dashboard`：概览
- `/admin/providers`：Provider 管理
- `/admin/router`：应用适配（Port URL / Client Keys / App 映射）
- `/admin/marketplace`：模块市场
- `/admin/logs`：请求元数据日志
- `/admin/docs`：内置简要说明页（静态）
- `/admin/auth`：Client Keys + Base URL（部分 UI 会直接使用 router 替代）
- `/admin/codex/oauth/callback`：Codex OAuth 回调落地页（用于浏览器回跳）

静态资源：

- `/admin/static/*`

---

## 2. 初始化与登录

### `POST /admin/api/setup`

用途：首次初始化（创建管理员 + 创建默认 Client API Key + 建立会话）。

请求：

```json
{ "username": "admin", "password": "pass", "remember": false }
```

响应（示例）：

```json
{
  "ok": true,
  "default_client_key": "sk-lb-...",
  "next": "/admin/dashboard",
  "message": "setup complete"
}
```

### `POST /admin/api/login`

用途：管理员登录（建立会话 Cookie）。

请求：

```json
{ "username": "admin", "password": "pass", "remember": true }
```

响应：

```json
{ "ok": true, "next": "/admin/dashboard" }
```

---

## 3. Providers

### `GET /admin/api/providers`

返回 Provider 列表（含禁用项）。

响应：

```json
{ "data": [ { "id":"forward", "protocol":"forward", ... } ] }
```

### `POST /admin/api/providers`

新增或更新 Provider（upsert）。

常用字段（请求 JSON）：

- `id`（必填）
- `displayName`（可选）
- `groupName`（可选）
- `type`（可选，默认 `builtin`）
- `protocol`（可选，默认 `forward`）
- `endpoint`（可选，但多数协议需要）
- `apiKey` / `token`（可选：会写入 `configJSON.api_key`）
- `configJSON`（可选：默认 `{}`）
- `enabled`（可选：不传则沿用旧值/默认 true）

### `POST /admin/api/providers/pull_models`

从指定 Provider 的 `/v1/models` 拉取模型列表并写入本地 `models` 表（仅新增缺失项）。

请求：

```json
{ "provider_id": "forward" }
```

响应：

```json
{ "ok": true, "provider_id":"forward", "total": 123, "inserted": 12, "source_url": "..." }
```

### `POST /admin/api/providers/delete`

删除 Provider（支持单个或批量）。

请求：

```json
{ "id": "forward" }
```

或：

```json
{ "ids": ["p1","p2"] }
```

---

## 4. Models & Routes（全局路由表）

### `GET /admin/api/models`

返回：

- `models`：基础模型列表
- `routes`：所有 model_routes

### `POST /admin/api/models`

新增/更新 model，并替换其 routes。

请求：

```json
{
  "model": { "id":"gpt-4o-mini", "displayName":"GPT-4o Mini", "enabled": true },
  "routes": [
    { "providerID":"forward", "upstreamModel":"gpt-4o-mini", "priority":0, "weight":1, "enabled": true }
  ]
}
```

说明：

- routes 中若 `weight=0` 会被自动改为 `1`
- 该接口会“替换”该 model 的 routes（不是增量追加）

### `POST /admin/api/models/delete`

删除 model（支持 `id` 或 `ids`）。

---

## 5. Dashboard / Logs / Password

### `GET /admin/api/dashboard`

返回 Provider/Model/Module 统计与最近日志、近 7 天 token 汇总等。

### `GET /admin/api/logs`

返回最近 200 条请求元数据日志。

### `POST /admin/api/logs/prune`

清理旧日志（服务端默认：删 30 天前，最多保留 50000 行）。

### `POST /admin/api/change_password`

修改当前登录管理员的密码。

请求：

```json
{ "old_password":"...", "new_password":"..." }
```

---

## 6. Client Keys（对外 API Key）

### `GET /admin/api/client_keys`

返回所有 Client Keys。

### `POST /admin/api/client_keys`

新建 Client Key（可自定义 name/key，也可留空自动生成）。

请求：

```json
{ "name":"Production", "key":"sk-lb-..." }
```

### `POST /admin/api/client_keys/enable`

启用/禁用：

```json
{ "id":"key_xxx", "enabled": true }
```

### `POST /admin/api/client_keys/delete`

删除：

```json
{ "id":"key_xxx" }
```

---

## 7. Router / Apps（voucher config）

### `GET /admin/api/voucher/config`

读取 App 配置（base_url + apps 映射）。

### `POST /admin/api/voucher/config`

写入 App 配置。

配置结构（概念）：

- `base_url`: `https://host/openai`
- `apps`: map
  - `<appID>.key_id`: 绑定的 client key id
  - `<appID>.model_mappings[]`: `{from,to}`

---

## 8. Server Addresses（辅助）

### `GET /admin/api/server_addrs`

返回服务器可用 IPv4 地址列表、当前请求的 scheme/host/port（用于 Router 页面“自动检测 IP”）。

---

## 9. Marketplace / Modules

### `GET /admin/api/marketplace/index`

获取模块索引：

- 默认使用 Core 启动配置的 `ModuleIndexURL`
- 也可通过 query `?url=` 覆盖

### `POST /admin/api/marketplace/install`

安装模块：

```json
{ "module_id":"openai-codex-oauth", "index_url":"local" }
```

### `GET /admin/api/modules`

列出已安装模块及其 runtime（端口、PID）。

### `POST /admin/api/modules/start`

```json
{ "module_id":"openai-codex-oauth" }
```

### `POST /admin/api/modules/stop`

```json
{ "module_id":"openai-codex-oauth" }
```

### `POST /admin/api/modules/enable`

```json
{ "module_id":"openai-codex-oauth", "enabled": true }
```

### `GET /admin/api/modules/manifest?module_id=...`

返回已安装模块的 manifest。

### `GET /admin/api/modules/config?module_id=...`

返回模块 config/schema/defaults/config_path。

### `POST /admin/api/modules/config`

写入模块 config：

```json
{ "module_id":"openai-codex-oauth", "config": { "x": 1 }, "restart": true }
```

### `POST /admin/api/modules/uninstall`

```json
{ "module_id":"openai-codex-oauth", "purge_data": true }
```

### `POST /admin/api/modules/upgrade`

```json
{ "module_id":"openai-codex-oauth", "index_url":"local" }
```

---

## 10. Codex OAuth（Core 代理模块接口）

### `GET /admin/api/codex/oauth/status`

读取模块的 auth 状态。

### `POST /admin/api/codex/oauth/start`

启动 OAuth（Core 会先启动本地回调服务，并强制注入 redirect_uri=`http://localhost:1455/auth/callback`）。

### `POST /admin/api/codex/oauth/exchange`

把 callback 的 `code/state` 交给模块换 token（支持 `{callback_url: "..."} `）。

### `POST /admin/api/codex/oauth/import`

导入 refresh/access token 或 Auth.json 内容。

### `POST /admin/api/codex/device/start`

启动 device flow。

