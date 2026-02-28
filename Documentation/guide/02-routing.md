# Guide ②：模型路由与故障转移（Routing & Failover）

本篇解释 LightBridge 如何从一次 `/v1/*` 请求中选择 Provider，并给出可操作的配置方式（包括：变体语法、权重、优先级、5xx 故障转移、以及 App 级模型映射）。

---

## 0. 先看一条请求会发生什么（总体流程）

以 `POST /v1/chat/completions` 为例，核心流程是：

1. 校验 Client API Key（`Authorization: Bearer ...`）
2. 读取 JSON body，提取 `model` 字段（若 body 非空且非 JSON，会返回 `invalid_json`）
3. 根据 `model` 做路由解析（Resolver）
4. 从 DB 读取目标 Provider（endpoint/config/protocol/enabled/health）
5. 交给对应协议适配器（Adapter）执行上游调用/转换/透传
6. 将响应写回给下游

---

## 1. 变体语法：`model@providerAlias`（强制指定 Provider）

当下游请求的 `model` 形如：

- `something@someProvider`

Core 会认为你在使用“变体语法”，行为特点：

- **直接锁定** `providerAlias`（即 Provider ID）
- 不走权重/优先级选择
- 不走 5xx 故障转移（显式指定，尊重你的选择）

同时，上游模型名的决定规则：

- 默认上游模型为 `something`（`@` 左侧）
- 如果你为该 `base model` 配置过“该 provider 的 UpstreamModel”，则会用那条路由的 `UpstreamModel`

示例：

```json
{ "model": "gpt-4o-mini@forward", "messages": [...] }
```

> 适用场景：压测某个上游、临时绕过路由、对比不同 Provider 的效果。

---

## 2. 普通路由：models + model_routes（全局模型路由表）

当 `model` 不含 `@` 时，Core 会查询：

- `model_routes`：该 model 是否配置了路由目标

若存在路由，将按以下规则选择：

### 2.1 过滤条件（必须满足）

候选 route 必须满足：

- route 自身 `enabled = true`
- 对应 Provider：
  - `enabled = true`
  - `health_status` 不为 `down/unhealthy/disabled`（空值也视为健康）

### 2.2 优先级（priority）

在通过过滤的 routes 中：

- 选择 **最小 priority** 的那一组（数值越小优先级越高）

### 2.3 权重（weight）

在同一 priority 组内：

- 按 `weight` 做加权随机
- `weight <= 0` 会按 `1` 处理

### 2.4 UpstreamModel

被选中的 route 若配置了 `UpstreamModel`：

- 实际发往上游的 `model` 会被改写为该 `UpstreamModel`

否则：

- 上游模型名默认为下游请求的 `model`

---

## 3. 兜底路由（当某个 model 没有配置 routes）

当 `model_routes` 里找不到路由时，Core 会按 model 前缀猜一个 fallback provider ID：

- `claude-*` → `anthropic`
- `gemini-*` → `gemini`
- `gpt-*` / `o1-*` / `o3-*` / `o4-*` / `chatgpt-*` → `codex`
- 其它 → `forward`

然后：

1. 如果该 fallback Provider 存在且健康，则使用它
2. 否则从 DB 中找“任意一个启用且健康”的 Provider 作为最后兜底

> 注意：当前 Core 默认只内置 `forward` 与 `anthropic` 两个 Provider 记录；`codex`/`gemini` 需要你自己创建（或由模块自动注册）。因此在未显式创建 `codex` 时，`gpt-*` 最终可能还是会落到 `forward`（取决于你有哪些健康 Provider）。

---

## 4. 5xx 故障转移（Failover）

当你使用“普通路由”（不含 `@`）时，Core 还提供一次简单的故障转移：

- 如果上游返回 **5xx**
- 且当前请求不是变体（`model@provider`）

则 Core 会：

- 最多重试 **2 次**
- 每次重试会把“刚失败的 Provider”加入排除列表，然后重新 Resolve 选择下一个可用 Provider

这样可以在多上游时提高可用性。

---

## 5. 如何配置“全局模型路由”（可直接操作）

目前管理后台 UI 主要用于 Provider/模块/应用适配；**全局模型路由**建议直接用 Admin API 配置。

### 5.1 查看当前 models 与 routes

```bash
curl -s "http://127.0.0.1:3210/admin/api/models" \
  -H "Cookie: lightbridge_admin=<your_cookie_here>"
```

> 说明：Admin API 使用 Cookie 会话认证；更适合在浏览器内调用或用脚本复用 Cookie。更完整的 Admin API 见参考文档。

### 5.2 新增/更新一个 model，并配置 routes

向 `POST /admin/api/models` 提交：

- `model`: `{id, displayName, enabled}`
- `routes`: `[{providerID, upstreamModel, priority, weight, enabled}]`

示例：把 `gpt-4o-mini` 主要走 `forward`，备用走 `codex`：

```bash
curl -s "http://127.0.0.1:3210/admin/api/models" \
  -H "Content-Type: application/json" \
  -H "Cookie: lightbridge_admin=<your_cookie_here>" \
  -d '{
    "model": { "id": "gpt-4o-mini", "displayName": "GPT-4o Mini", "enabled": true },
    "routes": [
      { "providerID": "forward", "upstreamModel": "gpt-4o-mini", "priority": 0, "weight": 9, "enabled": true },
      { "providerID": "codex",  "upstreamModel": "gpt-4o-mini", "priority": 0, "weight": 1, "enabled": true }
    ]
  }'
```

> 提示：`priority` 用于分层（例如主用=0，备用=10）；同层用 `weight` 做负载均衡。

### 5.3 删除一个 model（以及其 routes）

```bash
curl -s "http://127.0.0.1:3210/admin/api/models/delete" \
  -H "Content-Type: application/json" \
  -H "Cookie: lightbridge_admin=<your_cookie_here>" \
  -d '{ "id": "gpt-4o-mini" }'
```

---

## 6. App 级模型映射（应用适配：先映射再路由）

除了全局 routes，LightBridge 还支持“App 级模型映射”：

- 请求路径使用 `/openai/<app>/v1/*`
- Core 会在路由前，把该 app 的 `model_mappings` 应用到请求的 model 上（相当于“重写 model”）

这非常适合：

- 让某个客户端固定请求 `gpt-4o-mini`，但你希望它实际走 `gpt-4.1-mini` 或 `claude-...`
- 为不同客户端设置不同的“模型别名策略”

App 映射的配置入口：

- `/admin/router` 页面（应用适配面板）

详见：[OpenAI 别名与应用适配（Apps）](./05-openai-alias-and-apps.md)

---

## 7. `/v1/models` 返回的内容与你看到的路由关系

`GET /v1/models` 会返回一个“虚拟模型列表”，包含：

1. `models` 表中启用的基础模型（例如 `gpt-4o-mini`）
2. 若某些 routes 的 Provider 健康可用，会暴露变体模型：
   - `upstreamModel@providerID` 或 `modelID@providerID`
3. 同时也会为每个基础模型暴露“内置 alias 变体”（便于你显式指定 Provider）

因此你可能会在列表中看到：

- `gpt-4o-mini`
- `gpt-4o-mini@forward`
- `gpt-4o-mini@codex`
- …

---

## 下一步

- 若你需要安装模块以提供更多 Provider：见 [模块 Marketplace](./03-modules-marketplace.md)
- 若你需要给 Codex 配置 OAuth：见 [Codex OAuth](./04-codex-oauth.md)

