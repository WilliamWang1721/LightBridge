# Getting Started ①：准备工作（Prerequisites）

本篇说明在开始运行 LightBridge 之前，你需要准备哪些环境与信息。

---

## 1. 运行环境要求

### 1.1 操作系统

- macOS / Linux / Windows 均可运行（Go 单体 + SQLite）。

### 1.2 Go 版本

本仓库 `go.mod` 指定：

- Go `1.23.0`（建议 `>= 1.23`）

验证：

```bash
go version
```

---

## 2. 你需要提前准备的“上游密钥”

LightBridge 本质是网关：它会把请求路由到某个 Provider（上游/模块）。

因此在“客户端能成功调用”之前，你通常需要至少准备一个上游 Provider 的凭证：

### 2.1 OpenAI（forward / http_openai / codex）

- **forward/http_openai/http_rpc**：通常需要 OpenAI API Key（或你的 OpenAI 兼容上游的 Key）
- **codex**：通常需要 OpenAI API Key，或配合 `openai-codex-oauth` 模块走 OAuth（见后续文档）

### 2.2 Anthropic（anthropic）

- 需要 Anthropic API Key

> 说明：仓库初始化时会自动创建内置 Provider（例如 `forward`、`anthropic`），但**不会**自动填入你的 API Key。

---

## 3. 端口与访问方式

### 3.1 默认监听地址

LightBridge 默认监听：

- `127.0.0.1:3210`

这意味着：

- 仅本机可访问（最安全的默认）
- 若你需要局域网/服务器访问，需要调整监听地址（见下一篇运行文档），并同时处理鉴权与网络安全

---

## 4. “数据目录”与磁盘空间

LightBridge 会在数据目录写入：

- `lightbridge.db`（SQLite）
- `modules/`（已安装模块）
- `module_data/`（模块运行时配置与私有数据）
- `MODULES/`（本地 Marketplace 扫描目录）

数据目录默认位置由 Go 的 `os.UserConfigDir()` 决定（不同系统路径不同），也可通过 `LIGHTBRIDGE_DATA_DIR` 指定自定义路径（见文档 [环境变量一览](../reference/01-env-vars.md)）。

---

## 5. 推荐的阅读顺序

- 下一步：运行服务与环境变量说明：[02-run.md](./02-run.md)

