# Development ②：仓库结构（Repo Structure）

本篇帮助你快速定位代码与资源的位置（适合第一次进入本仓库的开发者）。

---

## 1. 顶层目录

- `cmd/lightbridge/`：Core 入口（main）
- `internal/`：Core 主体实现（不对外暴露的 Go 包）
- `modules/`：内置/示例模块（包含 `openai-codex-oauth`）
- `tests/`：测试与测试数据
- `参考资料/`：项目开发文档与项目日志（开发过程记录）
- `参考项目/`：参考/借鉴的开源实现（供复用，不建议直接改）

---

## 2. internal 目录（重点）

- `internal/app/`
  - `config.go`：读取环境变量与默认配置
  - `app.go`：初始化目录/DB/组件装配/启动 server
- `internal/gateway/`
  - `server.go`：HTTP 路由（/v1、/openai、/admin）
  - `admin.go`：Admin API（providers/models/modules/keys/...）
  - `web/`：管理后台模板与静态资源（embed）
- `internal/store/`
  - `store.go`：SQLite CRUD（providers/models/routes/keys/modules/logs/settings）
- `internal/routing/`
  - `resolver.go`：模型到 Provider 的解析与选择（priority/weight/failover 逻辑）
- `internal/providers/`
  - `http_forward.go`：forward/http_openai/http_rpc 透传
  - `anthropic.go`：Anthropic 协议转换
  - `codex.go`：Codex 协议转换（对上游 /responses）
  - `grpc_chat.go`：占位适配器
- `internal/modules/`
  - `marketplace.go`：索引获取与模块安装（local/remote/github）
  - `manager.go`：模块子进程启停与健康检查
- `internal/db/`
  - SQLite 初始化与迁移
- `internal/translator/`
  - 协议转换逻辑（OpenAI ↔ Claude / Codex）

---

## 3. modules 目录（模块）

示例模块：

- `modules/openai-codex-oauth/`
  - `package.sh`：打包 zip + 生成 index.json（用于 local marketplace）
  - `dist/`：打包产物（zip、manifest、README）
  - `cmd/openai-codex-oauth/`：模块可执行入口

---

## 4. 参考资料（务必遵守）

按仓库 `AGENT.md` 约定：

- 开发相关指导在 `参考资料/项目开发文档.md`
- 每次完成任务要在 `参考资料/项目日志.md` 追加记录（按既有格式）

