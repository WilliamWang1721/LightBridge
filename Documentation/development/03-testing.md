# Development ③：测试指南（Testing）

本篇说明如何在本仓库运行测试，以及在修改路由/Provider/模块相关代码后建议验证哪些内容。

---

## 1. 运行全部测试

在仓库根目录：

```bash
go test ./...
```

---

## 2. 常见测试位置（你改哪里就先跑哪里）

- 路由解析：
  - `internal/routing/*_test.go`
- Store / 迁移 / CRUD：
  - `internal/store/*_test.go`
- 模块 Marketplace：
  - `internal/modules/*_test.go`
- 网关集成链路（若有）：
  - `internal/gateway/*_test.go`

---

## 3. 手工回归（建议清单）

当你改动了以下能力时，建议手工回归一次：

### 3.1 `/admin` 初始化与登录

- 首次启动跳转 `/admin/setup`
- setup 后能拿到 `default_client_key`
- 重启后会话是否按预期失效/保留（取决于 `LIGHTBRIDGE_COOKIE_SECRET`）

### 3.2 Provider 保存与调用

- `/admin/providers` 保存 forward/anthropic/codex provider
- 用 curl 调用 `/v1/models`、`/v1/chat/completions` 验证链路

### 3.3 模块安装与启动

- local marketplace 扫描 zip 是否正常
- 安装 → 启动 → 健康检查 → Provider alias 自动出现
- 卸载（purge_data true/false）是否按预期

---

## 4. 数据隔离建议（跑测试/回归时）

为了避免污染你本机真实数据目录，建议在回归时显式指定：

```bash
export LIGHTBRIDGE_DATA_DIR="$(mktemp -d)"
go run ./cmd/lightbridge
```

这样退出后可以直接删除该目录。

