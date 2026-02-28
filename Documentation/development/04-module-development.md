# Development ④：开发模块（Provider Module）

本篇面向“我要写一个模块，让 LightBridge 通过 Marketplace 安装并作为 Provider 使用”的开发场景。

---

## 1. 模块是什么？（Core 视角）

在 Core 中，模块是：

- 一个可执行程序（子进程）
- 由 Core 启动并注入端口与配置路径
- 通过 manifest 声明自己提供的 Provider 协议与健康检查

模块可以暴露一个或多个 `providerAlias`：

- 这些 alias 会出现在 Providers 列表里
- 下游可用 `model@providerAlias` 强制路由

---

## 2. 模块运行时约定（必须遵守）

当 Core 启动模块时，会注入以下环境变量（详见参考）：

- `LIGHTBRIDGE_HTTP_PORT`：模块 HTTP 端口
- `LIGHTBRIDGE_GRPC_PORT`：模块 gRPC 端口
- `LIGHTBRIDGE_CONFIG_PATH`：模块配置文件路径
- `LIGHTBRIDGE_DATA_DIR`：模块数据目录

你需要做到：

1. **绑定回环地址**
   - HTTP：监听 `127.0.0.1:<LIGHTBRIDGE_HTTP_PORT>`
   - gRPC：监听 `127.0.0.1:<LIGHTBRIDGE_GRPC_PORT>`
2. **实现健康检查**
   - `health.type=http` 时，实现 `GET <health.path>` 返回 200
3. **读取配置**
   - 从 `LIGHTBRIDGE_CONFIG_PATH` 读取 JSON（Core 会在不存在时写入 defaults）

---

## 3. 支持的模块协议（manifest.services[].protocol）

当前 Core 允许模块声明的协议：

- `http_openai`
- `http_rpc`
- `grpc_chat`
- `codex`

### 3.1 http_openai / http_rpc（最常用）

你需要实现：

- OpenAI 兼容 HTTP 路径（通常是 `/v1/*`）

当下游请求：

- `/v1/chat/completions`

若路由选中你的模块 Provider，Core 会把请求转发到：

- `http://127.0.0.1:<module_http_port>/v1/chat/completions`

> 提示：如果你的模块要兼容更多端点（如 embeddings、models、images），只要实现对应 `/v1/*` 即可。

### 3.2 codex

Core 的 `codex` 适配器对上游调用固定为：

- `POST <endpoint>/responses`

因此模块建议实现：

- `POST /responses`（透传或实现 Responses SSE 协议）
- 可选：`GET /v1/models`（便于拉取模型）

参考实现：

- `modules/openai-codex-oauth`（模块侧负责 OAuth Token 与上游透传，Core 侧负责转换）

### 3.3 grpc_chat

当前 Core 的 `grpc_chat` 适配器为占位实现（会返回 501）。

你仍可以在 manifest 声明该协议并实现 gRPC health，但端到端调用需要后续版本完善。

---

## 4. manifest.json 编写要点

最小 manifest 示例（概念）：

```json
{
  "id": "my-module",
  "name": "My Module",
  "version": "0.1.0",
  "license": "MIT",
  "min_core_version": "0.1.0",
  "entrypoints": {
    "darwin/arm64": { "command": "bin/my-module", "args": [] },
    "default": { "command": "bin/my-module", "args": [] }
  },
  "services": [
    {
      "kind": "provider",
      "protocol": "http_openai",
      "health": { "type": "http", "path": "/health" },
      "expose_provider_aliases": ["myprovider"]
    }
  ],
  "config_schema": {},
  "config_defaults": {}
}
```

关键点：

- `entrypoints.command` 若是相对路径，会以 manifest 所在目录为基准
- 建议把运行时文件放在 `dist/` 并使用 `dist/manifest.json`（Core 优先选择）

---

## 5. 打包与发布（local Marketplace）

最简单的发布方式：

1. 把你的模块打成 zip（包含可执行文件 + manifest.json）
2. 放到 `./MODULES/` 或 `<DATA_DIR>/MODULES/`
3. 在管理后台 Marketplace 选择 `local` 安装

> 你也可以生成 `index.json` 并用 HTTP 服务暴露远程安装；或用 GitHub 目录扫描。

---

## 6. 调试技巧

### 6.1 看模块 stdout/stderr

Core 启动模块时会把模块 stdout/stderr 直接输出到 Core 的 stdout/stderr。

### 6.2 获取模块端口

通过：

- `GET /admin/api/modules`

拿到 `runtime.http_port`，即可直接 `curl http://127.0.0.1:<port>/health`。

