# Guide ⑤：OpenAI 别名与“应用适配”(Apps)

LightBridge 除了标准 `/v1/*` 入口，还提供 `/openai/*` 别名路由，用于：

- 兼容某些客户端对路径的固定假设
- 为不同客户端提供“独立 URL”（App URL），从而实现：
  - 绑定专用 Client Key（隔离不同客户端）
  - App 级模型映射（同一个客户端请求不同模型名时自动改写）

---

## 1. `/openai/*` 路由规则（你需要记住的）

Core 支持两种形式：

1. Base alias：
   - `http://<ADDR>/openai/v1/*` → 等价于 `http://<ADDR>/v1/*`
2. App alias：
   - `http://<ADDR>/openai/<app>/v1/*` → 等价于 `http://<ADDR>/v1/*`，但会启用该 app 的“绑定 Key + 模型映射”

注意：

- `/openai/*` **只是路径前缀路由**，底层鉴权仍使用同一套 Client API Key 机制
- App alias 额外增加“可选的 Key 限制”，并在路由前对 `model` 做映射

---

## 2. Router 页面（/admin/router）能配置什么？

入口：

- `http://<ADDR>/admin/router`

该页面包含三块能力：

1. Port URL（你的对外访问地址配置，用于生成 Base URL）
2. Client API Keys 管理（新建/启用/禁用/复制）
3. Apps 配置（绑定 Key + 模型映射）

---

## 3. Port URL：生成你要给客户端的 Base URL

Router 顶部的 Port URL 卡片会显示：

- 形如：`http(s)://<host>/openai`

并在内部派生：

- OpenAI Base V1：`.../openai/v1`
- App Base V1：`.../openai/<app>/v1`

你可以：

- 选择协议（HTTP/HTTPS）
- 输入域名或 IP（也可以点“自动检测 IP”）
- 点击“编辑配置/保存”写入配置

该配置会存入系统 setting（`voucher_config_v1.base_url`），用于后续 App URL 展示与复制。

---

## 4. Apps：为特定客户端生成“独立 URL”

默认提供的 app 列表（可在设置中扩展）：

- `codex`
- `claude-code`
- `opencode`
- `gemini-cli`
- `cherry-studio`

每个 app 都有两项核心配置：

### 4.1 绑定密钥（key_id）

你可以为某个 app 绑定一个特定的 Client Key ID。

效果：

- 访问 `http://<ADDR>/openai/<app>/v1/*` 时，若请求携带的 Client Key 不匹配该 `key_id`，会返回 `invalid_api_key`

适用：

- 一台机器上接多个客户端/团队时，隔离不同调用方

### 4.2 模型映射（model_mappings）

你可以配置多条 `from -> to`：

- 当 app 的请求里 `model == from` 时，Core 会在路由前把 model 改写为 `to`

示例：

- `from: gpt-4o-mini`
- `to: gpt-4o-mini@codex`

这样某个客户端即使不支持 `model@provider` 语法，你也能让它强制走某个 Provider。

> 提示：App 映射属于“请求预处理”，之后仍会进入全局路由（models/routes/fallback）。

---

## 5. `/v1/models` 在 App URL 下会有什么不同？

当你访问：

- `GET /openai/<app>/v1/models`

返回的 model list 会额外包含“映射补充项”：

- 如果你映射了 `from -> to`，而 `from` 本身不在全局 models 列表里，Core 会把 `from` 追加到列表中，并在 `provider_hint` 里标注 `mapped->to`

这样可以提升一些客户端的“模型下拉列表”体验。

---

## 6. 推荐用法（可直接照抄）

目标：让某个客户端固定调用 `gpt-4o-mini`，但实际上走 Codex Provider。

1. 确保你已有可用 Provider `codex`（见 Codex OAuth 文档）
2. Router → 选择 app（例如 `cherry-studio`）
3. 添加模型映射：
   - `from = gpt-4o-mini`
   - `to   = gpt-4o-mini@codex`
4. 保存
5. 客户端配置：
   - Base URL：`http://<ADDR>/openai/cherry-studio/v1`
   - API Key：绑定的 Client Key 值

---

## 下一步

- 了解全局路由（priority/weight/failover）：[模型路由与故障转移](./02-routing.md)

