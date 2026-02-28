# Guide ①：Provider 管理（上游配置与协议）

Provider 是 LightBridge 连接“上游模型服务”的抽象。你可以把它理解为：

- 一个可用的上游入口（endpoint + 协议 + 认证信息）
- 一个可被路由/变体语法选中的目标（`model@providerAlias`）

本篇讲清楚 Provider 的字段、协议、以及如何在 Admin 中正确配置。

---

## 1. Provider 的关键字段（你需要关心的）

在数据库与 Admin API 中，Provider 主要字段如下（概念层面）：

- `id`：Provider ID（也是 providerAlias，路由与 `model@provider` 使用它）
- `display_name`：账户名称（仅用于展示/分组）
- `group_name`：分组（仅用于展示/筛选）
- `type`：`builtin`（内置）或 `module`（模块提供/自动注册）
- `protocol`：协议（决定适配器与支持的端点）
- `endpoint`：上游地址（不同协议解释不同）
- `config_json`：JSON 配置（通常放 `api_key`、`base_url`、`extra_headers`、`model_remap` 等）
- `enabled`：是否启用
- `health_status`：健康状态（`healthy`/`down`/`disabled` 等，影响是否可被选中）

> 约定：本项目很多表单会把 “API Key / Token” 单独输入，保存时会自动写入 `config_json.api_key`，避免你手写 JSON。

---

## 2. 支持的 Provider 协议（protocol）与能力边界

LightBridge Core 当前内置以下协议适配器（见 `internal/providers/*`）：

### 2.1 `forward`（HTTP 透传到 OpenAI 兼容上游）

- 适用：你有一个 OpenAI 兼容上游（OpenAI 官方/第三方/自建网关）
- 行为：把请求转发到 `endpoint/base_url` 对应的上游路径
- 常用配置：
  - `endpoint`: `https://api.openai.com/v1`
  - `config_json.api_key`: `sk-...`
  - `config_json.extra_headers`: 额外 Header
  - `config_json.model_remap`: 额外的 model 重写（高级）

注意：

- 若 `endpoint` 以 `/v1` 结尾，且请求路径也是 `/v1/...`，Core 会避免拼出 `/v1/v1/...` 的重复路径。

### 2.2 `http_openai` / `http_rpc`（HTTP 透传到“模块 HTTP 上游”）

这两个协议在 Core 内部复用同一个 HTTP 转发适配器：

- 适用：你安装了某个模块，它在本机端口提供 OpenAI 兼容 HTTP 服务
- 常见 endpoint 形式：`http://127.0.0.1:<module_http_port>`
- 行为：同 `forward`，但语义上更偏“模块 Provider”

> 备注：模块 `manifest.json` 中的 `services[].protocol` 只允许 `http_openai` / `http_rpc` / `grpc_chat` / `codex`。

### 2.3 `anthropic`（OpenAI → Claude 协议转换）

- 适用：你希望下游仍用 OpenAI API，但上游是 Anthropic
- 已实现端点（MVP）：
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
- 其它 `/v1/*`：返回 `501_not_supported`
- 配置要求：
  - `config_json.api_key` 必填（否则 `provider_misconfigured`）
  - `endpoint` 默认 `https://api.anthropic.com`

### 2.4 `codex`（OpenAI Chat/Responses ↔ Codex Responses 转换）

`codex` 协议适配器的核心特点：

- 对下游暴露：`/v1/chat/completions` 与 `/v1/responses`
- 对上游调用：统一调用 `POST <endpoint>/responses`（注意：这里是把 `/responses` 直接拼在 endpoint 后面）

因此 endpoint 的“正确写法”取决于你把 codex 指向哪里：

1. 指向 OpenAI 官方（Responses API 在 `/v1/responses`）  
   - endpoint 应写为：`https://api.openai.com/v1`  
   - Core 拼接后为：`https://api.openai.com/v1/responses`
2. 指向 `openai-codex-oauth` 模块（模块提供 `/responses` 在根路径）  
   - endpoint 应写为：`http://127.0.0.1:<module_http_port>`  
   - Core 拼接后为：`http://127.0.0.1:<port>/responses`

配置：

- 若指向 OpenAI 官方：需要 `config_json.api_key`
- 若指向 OAuth 模块：通常无需 `api_key`（模块会用 OAuth Token 处理上游鉴权）

### 2.5 `grpc_chat`（占位/保留）

当前 `grpc_chat` 适配器会直接返回 `501_not_supported`，用于后续模块化 gRPC 集成预留。

### 2.6 `gemini`（常量已定义，但 Core 未实现适配器）

`types.ProtocolGemini` 在类型常量中存在，但 Core 未注册对应 adapter：

- 若你创建 `protocol=gemini` 的 Provider：调用时会得到 `provider_protocol_not_supported`

---

## 3. 在管理后台配置 Provider（推荐流程）

入口：

- `http://<ADDR>/admin/providers`

### 3.1 配置内置 Provider：`forward` / `anthropic`

常见最小操作：

1. 打开 Providers 页面
2. 找到 `forward`（或 `anthropic`）
3. 填入 `Endpoint` 与 `API Key / Token`
4. 保存

保存后，网关才可能成功把请求转发到上游。

### 3.2 通过模块“自动读取”添加 Provider（推荐）

适用：你已在 Marketplace 安装并启动模块，模块声明了 `expose_provider_aliases`。

流程（概念上）：

1. Marketplace 安装并启动模块
2. Providers → 添加 → “自动读取”
3. 选择模块导出的 provider alias
4. 系统会自动推导 endpoint（例如 `http://127.0.0.1:<http_port>`）并写入 Provider

### 3.3 高级添加（手动填写）

适用：你需要自定义 `id/protocol/endpoint/config_json`，或你接入的是第三方 OpenAI 兼容服务。

关键注意事项：

- `endpoint` 必须带 scheme（如 `http://` / `https://`），否则部分功能（如拉取模型列表）会失败
- 若你不熟悉 `config_json`，优先使用表单里的 `API Key / Token` 字段

---

## 4. 拉取上游模型列表（/v1/models）

Providers 页面支持从部分协议的 Provider 拉取 `/v1/models` 并写入本地 `models` 表（仅新增缺失项）。

支持的 protocol：

- `forward` / `http_openai` / `http_rpc` / `codex`

不支持：

- `anthropic` / `grpc_chat` / `gemini`

底层接口：

- `POST /admin/api/providers/pull_models`

---

## 5. 常见配置示例（Config JSON）

### 5.1 forward + OpenAI 官方

```json
{
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-xxx",
  "extra_headers": {
    "OpenAI-Organization": "org_xxx"
  }
}
```

> 说明：多数情况下你只需在表单里填 `Endpoint` 与 `API Key / Token`，无需手写 JSON。

### 5.2 forward：模型重写（高级）

当你希望把下游请求的模型名重写为上游实际模型名：

```json
{
  "model_remap": {
    "gpt-4o-mini": "gpt-4.1-mini"
  }
}
```

> 备注：路由层也有 `UpstreamModel` 的概念；请按你的团队约定选择一种方式，避免“多处重写”难以排查。

---

## 下一步

- 了解模型如何选中 Provider、以及 `model@provider` 的行为：[模型路由与故障转移](./02-routing.md)

