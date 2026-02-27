<div align="center">

# LightBridge 🚀

**一个微内核式 AI 网关：统一 OpenAI 兼容下游接口、支持多提供商路由，并可通过模块市场扩展能力。**

</div>

<div align="center">

[![Go](https://img.shields.io/badge/Go-%E2%89%A51.23-00ADD8?logo=go)](https://go.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57?logo=sqlite)](https://www.sqlite.org/)
[![Gateway](https://img.shields.io/badge/API-OpenAI%20Compatible-10a37f)](#-核心功能)
[![Status](https://img.shields.io/badge/Status-MVP%20v0.1-orange)](#-当前-v01-边界)

[**👉 主 README**](./README.md) | [中文副本](./README-ZH.md)

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
- **默认回退策略**：`claude-*` 自动回退 `anthropic`，其他模型默认回退 `forward`。

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
- [📋 核心功能](#-核心功能)
- [🧩 模块清单字段](#-模块清单字段已实现)
- [🗂️ 项目结构](#️-项目结构)
- [🧪 测试](#-测试)
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
LIGHTBRIDGE_MODULE_INDEX=local
LIGHTBRIDGE_MODULES_DIR=/path/to/MODULES # optional
LIGHTBRIDGE_COOKIE_SECRET=your-secret
```

默认数据目录：
- macOS/Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/LightBridge`

Marketplace 默认源：
- `local`：扫描 `./MODULES`（优先）或 `${LIGHTBRIDGE_DATA_DIR}/MODULES` 里的 `*.zip` 模块包
- 也可将 `LIGHTBRIDGE_MODULE_INDEX` 设置为一个远程 `index.json` URL（如 GitHub Pages/Raw/Releases）

---

## ⚙️ 首次初始化

1. 打开 `http://127.0.0.1:3210/admin/setup`
2. 创建管理员账号和密码
3. 复制系统生成的默认客户端 API Key
4. 使用 `Authorization: Bearer <key>` 调用网关接口

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

### 路由与调度
- 虚拟模型路由表（`models` + `model_routes`）
- 按优先级 / 权重 / 健康状态筛选
- 支持 `model@providerAlias` 变体路由

### 内置 Provider
- `forward`：`/v1/*` 透传
- `anthropic`：`/v1/chat/completions` 请求转换（流式/非流式）
- `grpc_chat`：占位实现（当前返回 `501_not_supported`）

---

## 🧩 模块清单字段（已实现）

必填字段：
- `id`, `name`, `version`, `license`, `min_core_version`
- `entrypoints`（`<os>/<arch>`、`<os>` 或 `default`）
- `services[]`
- `config_schema`
- `config_defaults`

`services[]` 中 provider 相关字段：
- `kind: "provider"`
- `protocol: "http_openai" | "http_rpc" | "grpc_chat"`
- `health`
- `expose_provider_aliases[]`

模块启动时核心注入环境变量：
- `LIGHTBRIDGE_MODULE_ID`
- `LIGHTBRIDGE_DATA_DIR`
- `LIGHTBRIDGE_CONFIG_PATH`
- `LIGHTBRIDGE_HTTP_PORT`
- `LIGHTBRIDGE_GRPC_PORT`
- `LIGHTBRIDGE_LOG_LEVEL`

---

## 🗂️ 项目结构

- `cmd/lightbridge/main.go`：程序入口
- `internal/app`：应用装配与启动
- `internal/db`：SQLite 初始化与迁移
- `internal/store`：数据访问层
- `internal/routing`：路由解析与模型聚合
- `internal/providers`：provider 适配器
- `internal/modules`：模块市场与运行时管理
- `internal/gateway`：网关与管理后台
- `tests/testdata/module-sample`：模块测试样例

---

## 🧪 测试

```bash
go test ./...
```

当前覆盖：
- 路由解析与回退策略
- 模型列表拼装（含 `model@provider`）
- SQLite 迁移幂等性
- Forward 透传（流式/非流式）
- Anthropic 转换（流式/非流式）
- 模块安装/启动与网关调用链路

---

## 📌 当前 v0.1 边界

- `grpc_chat` 仍为占位适配器
- 管理页面是可用 MVP，不是完整富交互 UI
- API Key/Provider Secret 目前以明文方式写入 SQLite
- 仅记录请求元数据，不记录完整 prompt/response body
