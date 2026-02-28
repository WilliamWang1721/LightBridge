# Getting Started ③：首次初始化（Setup Wizard）

本篇覆盖你第一次启动 LightBridge 后，需要做的初始化操作：创建管理员账号、获取默认 Client API Key，并完成最小可用的“可调用”状态。

---

## 0. 先理解两个概念：管理员登录 vs 客户端调用

LightBridge 有两套“认证/权限”：

1. **管理后台（Admin）**
   - 入口：`/admin/*`
   - 认证方式：浏览器 Cookie（登录后写入 `lightbridge_admin`）
2. **对外 OpenAI 兼容 API（Client）**
   - 入口：`/v1/*` 或 `/openai/*`
   - 认证方式：HTTP Header `Authorization: Bearer <CLIENT_KEY>`

你在 Setup Wizard 创建的是：

- 1 个管理员账号（用于登录 `/admin`）
- 1 个默认 Client API Key（用于调用 `/v1/*`）

---

## 1. 打开 Setup Wizard

确保服务已启动（见上一篇）后，打开：

- `http://127.0.0.1:3210/admin/setup`

按页面提示输入：

- 管理员用户名
- 管理员密码

提交后系统会：

- 写入管理员账号到 SQLite
- **生成默认 Client API Key**
- 自动创建 Admin 会话（你会直接进入 Dashboard）

---

## 2. 获取默认 Client API Key

Setup 完成后，页面会展示/返回默认 Client Key（只保证“创建当下”能完整看到，建议立刻复制保存）。

如果你忘了保存：

- 登录管理后台后，在 `Router` 或 `AUTH` 页面新建一个 Client Key 即可（无需重置系统）。

---

## 3. 让“第一次调用”成功：配置至少一个 Provider 的上游 Key

初始化完成并不代表马上能成功调用模型，因为内置 Provider（如 `forward` / `anthropic`）默认只有 endpoint，没有你的上游密钥。

### 3.1 方案 A：配置 `forward`（最通用）

适合：

- 你希望把 LightBridge 当“OpenAI 兼容反代/路由层”
- 你有一个 OpenAI 兼容上游（例如 OpenAI 官方、第三方兼容服务、你自己部署的兼容网关）

操作：

1. 打开 `/admin/providers`
2. 找到内置 Provider `forward`
3. 填入：
   - `Endpoint`：上游 Base URL（通常形如 `https://api.openai.com/v1`）
   - `API Key / Token`：你的上游 Key（保存后会写入 `configJSON.api_key`）
4. 保存

### 3.2 方案 B：配置 `anthropic`（用于 Claude）

操作：

1. 打开 `/admin/providers`
2. 找到内置 Provider `anthropic`
3. 填入 `API Key / Token`
4. 保存

注意：

- `anthropic` 协议当前只实现 `/v1/chat/completions` 与 `/v1/responses` 的协议转换；其它 `/v1/*` 会返回 `501_not_supported`（详见后续 Provider 文档）。

### 3.3 方案 C：安装 `openai-codex-oauth` 模块（用于 Codex OAuth）

适合：

- 你希望通过 OAuth 登录方式接入 Codex（无需手动保存 OpenAI API Key）

详见：[Codex OAuth（openai-codex-oauth）](../guide/04-codex-oauth.md)

---

## 4. 验证：发起一次最小调用

### 4.1 先确认 Models 列表可访问

```bash
curl -s http://127.0.0.1:3210/v1/models \
  -H "Authorization: Bearer <CLIENT_KEY>"
```

### 4.2 再调用 Chat Completions（示例）

> 说明：具体能否成功取决于你配置的 Provider 与路由（以及你使用的 `model`）。

```bash
curl -s http://127.0.0.1:3210/v1/chat/completions \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role":"user","content":"hello"}]
  }'
```

如果你希望“强制指定某个 Provider”，可以使用变体语法：

- `model@providerAlias`

例如（假设你有一个 Provider ID 叫 `forward` 或 `codex`）：

```json
{ "model": "gpt-4o-mini@forward", "messages": [...] }
```

路由与变体语法详见：[模型路由与故障转移](../guide/02-routing.md)

---

## 5. 登录入口与常用页面

- Admin 首页：`/admin`
- Dashboard：`/admin/dashboard`
- Providers：`/admin/providers`
- Router（密钥 / 应用适配 / 映射）：`/admin/router`
- Marketplace：`/admin/marketplace`
- Logs：`/admin/logs`

---

## 下一步

- 客户端如何配置 Base URL / API Key / App URL：[04-client-config.md](./04-client-config.md)

