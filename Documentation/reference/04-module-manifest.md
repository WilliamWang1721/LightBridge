# Reference ④：模块规范（index.json / ZIP / manifest.json）

本篇给出“模块市场”真正依赖的格式要求，便于你：

- 自己制作模块包
- 在远程/本地 Marketplace 发布与安装
- 编写可被 Core 自动接入的 Provider 模块

---

## 1. Module Index（index.json）

Core 从索引获取模块列表（或在 local/GitHub 模式下即时生成）。

索引整体结构（概念）：

```json
{
  "generated_at": "2026-02-28T00:00:00Z",
  "min_core_version": "0.1.0",
  "modules": [
    {
      "id": "openai-codex-oauth",
      "name": "Codex OAuth",
      "version": "0.2.0",
      "description": "...",
      "license": "MIT",
      "tags": ["OpenAI","OAuth","provider"],
      "protocols": ["codex"],
      "download_url": "https://.../openai-codex-oauth.zip",
      "sha256": "....",
      "homepage": ""
    }
  ]
}
```

说明：

- `download_url` 可以是 HTTP(S) URL
- local 模式下会自动生成 `file://...` 的 download_url（允许）
- 但 **index 本身不支持** `file://`（本地请用 `local`）

---

## 2. 模块包（ZIP）要求

ZIP 内必须能找到一个可用的 `manifest.json`。

Core 的查找规则：

1. 递归扫描解压目录下所有 `manifest.json`（忽略 `__MACOSX`）
2. 若存在 `dist/manifest.json`，优先选择它
3. 否则选择“entrypoint command 文件存在”的 manifest
4. 仍不行则退化选择第一个（兼容旧包）

建议你把运行时文件放在：

- `dist/`（并在其中放 `manifest.json`）

---

## 3. manifest.json（核心字段）

Core 当前要求 manifest 至少包含：

- `id`（必填）
- `version`（必填）
- `entrypoints`（必填，至少一个）
- `services`（必填，至少一个 provider service）

并且 `services[].kind` 当前仅支持：

- `"provider"`

`services[].protocol` 当前仅支持：

- `"http_openai"`
- `"http_rpc"`
- `"grpc_chat"`
- `"codex"`

### 3.1 entrypoints

用于按 OS/ARCH 选择启动命令。

支持的 key（示例）：

- `"darwin/arm64"`
- `"darwin"`
- `"linux/amd64"`
- `"default"`

每个 entrypoint：

```json
{
  "command": "bin/my-module",
  "args": ["--flag"]
}
```

> 注意：如果 `command` 不是绝对路径，Core 会把它解析为“相对于 manifest 所在目录”的路径。

### 3.2 services（provider 声明）

示例：

```json
{
  "kind": "provider",
  "protocol": "http_openai",
  "health": { "type": "http", "path": "/health" },
  "expose_provider_aliases": ["myprovider"]
}
```

说明：

- `expose_provider_aliases`：模块启动后，Core 会为这些 alias 自动创建/更新 Provider 记录，并据此参与路由与 `model@provider`
- `health`：
  - `type: "http"`：Core 会请求 `http://127.0.0.1:<LIGHTBRIDGE_HTTP_PORT><path>`
  - `type: "grpc"`：Core 会对 `127.0.0.1:<LIGHTBRIDGE_GRPC_PORT>` 做 gRPC health check（预留）

---

## 4. 模块配置（config_schema / config_defaults）

manifest 可携带：

- `config_schema`：JSON Schema（用于管理后台生成表单/校验）
- `config_defaults`：默认配置（Core 会写入 config.json 初始内容）

Core 会：

- 把模块配置写到 `LIGHTBRIDGE_CONFIG_PATH` 指定的位置
- 管理后台可读取/更新配置并选择重启模块

模块实现应：

- 启动时读取 `LIGHTBRIDGE_CONFIG_PATH` 的 JSON
- 运行中按需热加载或依赖重启

---

## 5. 模块运行时环境变量（Core 注入）

Core 启动模块时注入（详见环境变量参考）：

- `LIGHTBRIDGE_MODULE_ID`
- `LIGHTBRIDGE_DATA_DIR`
- `LIGHTBRIDGE_CONFIG_PATH`
- `LIGHTBRIDGE_HTTP_PORT`
- `LIGHTBRIDGE_GRPC_PORT`
- `LIGHTBRIDGE_LOG_LEVEL`

---

## 6. “协议 = codex”的模块要注意什么？

Core 的 `codex` 协议适配器会把请求统一转成对上游的：

- `POST <endpoint>/responses`

因此当模块提供 `protocol=codex` 时，建议模块实现：

- `POST /responses`（在根路径）
- 可选：`GET /v1/models`（便于 Provider 拉取模型列表）

参考实现：

- `modules/openai-codex-oauth`

