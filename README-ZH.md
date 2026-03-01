<div align="center">

# LightBridge 🚀

**一个微内核式 AI 网关：统一 OpenAI 兼容下游接口、支持多提供商路由，并可通过模块市场扩展能力。**

</div>

<div align="center">

[![Go](https://img.shields.io/badge/Go-%E2%89%A51.23-00ADD8?logo=go)](https://go.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57?logo=sqlite)](https://www.sqlite.org/)
[![Gateway](https://img.shields.io/badge/API-OpenAI%20Compatible-10a37f)](#-核心功能)
[![Status](https://img.shields.io/badge/Status-MVP%20v0.1-orange)](#-当前-v01-边界)

[**📚 Documentation（拆分版）**](./Documentation/README.md) | [README](./README.md)

</div>

`LightBridge` 是一个面向本地部署和可扩展代理场景的 AI Gateway。它提供 OpenAI 兼容的 `/v1/*` 接口，支持 `model` 与 `model@provider` 路由表达式，内置 `forward` 与 `anthropic` 提供商，并支持通过模块市场安装第三方 provider（`http_openai` / `http_rpc` / `grpc_chat` 协议）。

> [!NOTE]
> **📌 当前版本定位：Go MVP v0.1**
>
> - 默认单端口运行：`127.0.0.1:3210`
> - 已实现：网关转发、路由调度、模块安装与启动、管理后台基础能力
> - 适合：本地测试、开发联调、最小可用生产原型

---

## 💡 核心优势

### 🎯 统一接口，平滑接入
- **OpenAI 兼容入口**：统一接入 `/v1/models`、`/v1/chat/completions` 及 `/v1/*` 转发路径。
- **模型路由能力**：支持 `model` 与 `model@provider`，可做别名、优先级、权重与健康筛选。
- **默认回退策略**：按模型前缀推断 fallback（如 `claude-* → anthropic`、`gpt-*/o* → codex`、其它 → `forward`）；若 fallback 不可用则退到任意健康 Provider。

### 🚀 可扩展架构
- **微内核 + 模块化**：核心网关保持轻量，provider 能力通过模块动态扩展。
- **模块市场支持**：支持 `index.json` 拉取、ZIP 下载、SHA256 校验、`manifest.json` 校验、安装启停。
- **多协议 provider**：支持 `http_openai`、`http_rpc`、`grpc_chat`（当前为占位实现）。

### 🛡️ 可控与可运维
- **管理后台**：提供 `/admin/*` 页面与 `/admin/api/*` 管理接口。
- **SQLite 持久化**：内置迁移，支持幂等初始化。
- **元数据日志**：记录请求元信息（不落盘提示词/响应正文）。

---

## 📑 快速导航

- [🚀 快速启动](#-快速启动)
- [⚙️ 首次初始化](#️-首次初始化)
- [📚 Documentation（拆分版）](./Documentation/README.md)
- [📋 核心功能](#-核心功能)
- [📌 当前 v0.1 边界](#-当前-v01-边界)

---

## 🚀 快速启动

### 1. 启动服务

```bash
go run ./cmd/lightbridge
```

默认监听地址：`127.0.0.1:3210`

### 2. 可选环境变量

```bash
LIGHTBRIDGE_ADDR=127.0.0.1:3210
LIGHTBRIDGE_DATA_DIR=/path/to/data
LIGHTBRIDGE_MODULE_INDEX=https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json # optional (default)
# LIGHTBRIDGE_MODULE_INDEX=local # dev/offline fallback
LIGHTBRIDGE_MODULES_DIR=/path/to/MODULES # optional
LIGHTBRIDGE_COOKIE_SECRET=your-secret
```

默认数据目录：
- macOS: `$HOME/Library/Application Support/LightBridge`
- Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/LightBridge`
- Windows: `%AppData%\\LightBridge`

Marketplace 默认源：
- 默认（Phase 2）：静态 `index.json`：`LIGHTBRIDGE_MODULE_INDEX=https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json`
- GitHub 目录扫描（Phase 1，开发/救援路径）：`LIGHTBRIDGE_MODULE_INDEX=github:WilliamWang1721/LightBridge/market/MODULES@main`
- `local`（开发/离线兜底）：扫描 `./MODULES`（优先）或 `${LIGHTBRIDGE_DATA_DIR}/MODULES` 里的 `*.zip` 模块包

---

## ⚙️ 首次初始化

1. 打开 `http://127.0.0.1:3210/admin/setup`
2. 创建管理员账号和密码
3. 复制系统生成的默认客户端 API Key
4. 使用 `Authorization: Bearer <key>` 调用网关接口

---

## 📚 Documentation（拆分版）

仓库已将文档拆分为多篇小文档，统一入口：

- [`Documentation/README.md`](./Documentation/README.md)

常用主题（建议按顺序阅读）：

- Getting Started（启动 / 初始化 / 客户端接入）
- Provider 管理、模型路由、模块 Marketplace、Codex OAuth
- 环境变量、对外 API、Admin API、模块 manifest、数据目录结构

---

## 📋 核心功能

### OpenAI 兼容网关
- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST/GET /v1/*`（forward/http_openai provider 透传）

### 管理后台 API（MVP）
- `POST /admin/api/setup`
- `POST /admin/api/login`
- `GET/POST /admin/api/providers`
- `GET/POST /admin/api/models`
- `GET /admin/api/dashboard`
- `GET /admin/api/logs`
- `GET /admin/api/marketplace/index`
- `POST /admin/api/marketplace/install`
- `POST /admin/api/modules/start`
- `POST /admin/api/modules/stop`

### Admin 登录安全
- 支持密码登录与 Passkey 登录（需安装并启用 `passkey-login` 模块）
- 支持 TOTP 2FA 策略与多验证器绑定（需安装并启用 `totp-2fa-login` 模块）
- 支持“密码后 2FA / Passkey 后 2FA / 仅 2FA 登录”策略组合

### 路由与调度
- 虚拟模型路由表（`models` + `model_routes`）
- 按优先级 / 权重 / 健康状态筛选
- 支持 `model@providerAlias` 变体路由

### 内置 Provider
- `forward`：`/v1/*` 透传
- `anthropic`：`/v1/chat/completions` 请求转换（流式/非流式）
- `grpc_chat`：占位实现（当前返回 `501_not_supported`）
- `codex`：在 Router 层完成 OpenAI Chat/Responses ↔ Codex（Responses）转换（流式/非流式）

---

## 🧩 模块规范与 Marketplace（文档）

模块打包、manifest 字段、索引来源与安装流程详见：

- [`Documentation/guide/03-modules-marketplace.md`](./Documentation/guide/03-modules-marketplace.md)
- [`Documentation/reference/04-module-manifest.md`](./Documentation/reference/04-module-manifest.md)

---

## 🗂️ 项目结构（文档）

- [`Documentation/development/02-repo-structure.md`](./Documentation/development/02-repo-structure.md)

---

## 🧪 测试（文档）

- [`Documentation/development/03-testing.md`](./Documentation/development/03-testing.md)

---

## 📌 当前 v0.1 边界

- `grpc_chat` 仍为占位适配器
- 管理页面是可用 MVP，不是完整富交互 UI
- API Key/Provider Secret 目前以明文方式写入 SQLite
- 仅记录请求元数据，不记录完整 prompt/response body
