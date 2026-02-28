# Development ①：架构总览（Architecture）

本篇从代码实现角度解释 LightBridge Core 的主要组件与调用链路，便于你快速定位“该改哪里/从哪里读数据”。

---

## 1. 组件一览（Core）

LightBridge Core（Go）由以下核心模块组成：

- `internal/gateway`：HTTP Server（/v1/*、/openai/*、/admin/*），鉴权、限流、日志
- `internal/store`：SQLite 数据访问层（Providers/Models/Routes/Keys/Modules/Logs/Settings）
- `internal/routing`：路由解析器 Resolver（根据 model 选择 Provider + UpstreamModel）
- `internal/providers`：协议适配器 Adapter（forward/anthropic/codex/http_openai/http_rpc/grpc_chat）
- `internal/modules`：
  - `Marketplace`：模块索引获取与安装（local / remote index.json / GitHub 目录）
  - `Manager`：模块子进程管理（端口分配、环境变量注入、健康检查、启停）
- `internal/app`：装配与启动（初始化目录、打开 DB、注册适配器、启动 server）

入口：

- `cmd/lightbridge/main.go`

---

## 2. 对外请求的核心链路（/v1/chat/completions）

简化后的链路：

1. `gateway.Server.handleV1Proxy`
2. `authenticateClientKey` 校验 `Authorization: Bearer <CLIENT_KEY>`
3. 读取 JSON body 提取 `model`
4. `routing.Resolver.Resolve(model)`
5. `store.GetProvider(route.ProviderID)` 获取 Provider 配置
6. `providers.Registry.Get(provider.Protocol)` 获取对应 Adapter
7. `adapter.Handle(...)` 执行：
   - forward/http_openai/http_rpc：透传
   - anthropic：协议转换
   - codex：转换为 Responses 并请求 `<endpoint>/responses`
8. 写入请求元数据日志 `store.InsertRequestLog`

失败与重试：

- 若非变体（无 `@`）且上游返回 5xx，会最多做 2 次“重新 resolve + 重试”（排除已失败 provider）

---

## 3. /openai 前缀与 App 适配的链路

`/openai/...` 只是一个路径前缀路由：

- `/openai/v1/*` → 转成 `/v1/*`
- `/openai/<app>/v1/*` → 转成 `/v1/*`，同时把 appID 放进请求上下文

额外行为：

- 若 app 绑定了 `key_id`：只有该 Client Key ID 可访问
- 若 app 配置了 model_mappings：在路由前先把 `model` 改写为映射后的值

配置存储：

- `settings` 表的 `voucher_config_v1`

---

## 4. 模块系统的关键点

模块是本机子进程：

- Core 分配端口并注入环境变量（HTTP/GRPC）
- 模块按 manifest 声明 health check
- 模块可暴露 provider alias（写入 providers 表），供路由与 `model@provider` 使用

目录：

- 安装目录：`<DATA_DIR>/modules/<id>/<version>/`
- 数据目录：`<DATA_DIR>/module_data/<id>/`

---

## 5. 适配器能力边界（非常重要）

并不是所有 Provider 都支持所有 `/v1/*`：

- `forward/http_openai/http_rpc`：透传（理论上支持最多端点）
- `anthropic/codex`：只支持 `POST /v1/chat/completions` 与 `POST /v1/responses`（其它返回 501）
- `grpc_chat`：当前为占位实现（返回 501）

因此：

- 你可以同时配置多个 Provider，并用路由决定某些模型走转换协议、其它端点走 forward 透传

