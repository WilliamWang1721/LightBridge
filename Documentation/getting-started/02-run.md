# Getting Started ②：启动与运行（Run）

本篇从“首次启动”出发，带你跑起 LightBridge，并解释最重要的环境变量与目录结构。

---

## 1. 最快启动（开发模式）

在仓库根目录执行：

```bash
go run ./cmd/lightbridge
```

默认监听：

- `127.0.0.1:3210`

验证健康检查：

```bash
curl -s http://127.0.0.1:3210/healthz
```

期望输出（示例）：

```json
{"name":"lightbridge","status":"ok"}
```

---

## 2. 管理后台入口

打开浏览器：

- `http://127.0.0.1:3210/admin`

首次运行会自动跳转到：

- `http://127.0.0.1:3210/admin/setup`

（下一篇会讲“首次初始化”）

---

## 3. 必懂的环境变量（最小集合）

LightBridge 目前从环境变量读取配置（见 `internal/app/config.go`），常用如下：

### 3.1 `LIGHTBRIDGE_ADDR`

指定监听地址（host:port），例如：

```bash
export LIGHTBRIDGE_ADDR=127.0.0.1:3210
go run ./cmd/lightbridge
```

如果你希望局域网访问（有安全风险，务必配合防火墙/反向代理/鉴权管理）：

```bash
export LIGHTBRIDGE_ADDR=0.0.0.0:3210
go run ./cmd/lightbridge
```

### 3.2 `LIGHTBRIDGE_DATA_DIR`

指定数据目录（强烈建议在生产环境显式指定）。

示例：

```bash
export LIGHTBRIDGE_DATA_DIR="$HOME/.lightbridge-data"
go run ./cmd/lightbridge
```

注意：

- 即使你传入相对路径，LightBridge 启动时也会尽量将其转换为绝对路径。

### 3.3 `LIGHTBRIDGE_MODULE_INDEX`

指定 Marketplace 的“索引来源”。常见取值：

- `local`（默认）：扫描本地 `MODULES` 目录里的 `*.zip`
- 远程 `index.json`：例如 `https://example.com/index.json`
- GitHub 目录：`github:<owner>/<repo>/MODULES@main`（更详细见模块文档）

### 3.4 `LIGHTBRIDGE_COOKIE_SECRET`

管理后台会使用 Cookie 维持登录会话。

- **不设置**：每次启动随机生成，会导致“重启后需要重新登录”（旧 Cookie 会失效）
- **设置为固定值**：可跨重启保持会话（仍受 Cookie 过期时间影响）

示例：

```bash
export LIGHTBRIDGE_COOKIE_SECRET="replace-with-a-long-random-string"
go run ./cmd/lightbridge
```

---

## 4. 默认数据目录到底在哪里？

默认数据目录基于 Go 的 `os.UserConfigDir()`：

- macOS：通常为 `~/Library/Application Support/LightBridge`
- Linux：通常为 `$XDG_CONFIG_HOME/LightBridge`，若未设置则常见为 `~/.config/LightBridge`
- Windows：通常在用户配置目录下（例如 `%AppData%\\LightBridge`）

数据库文件路径为：

- `<DATA_DIR>/lightbridge.db`

也可以通过 `LIGHTBRIDGE_DATA_DIR` 完全覆盖（见上文）。

---

## 5. 运行过程中会自动创建哪些目录？

启动时会自动创建（若不存在）：

- `<DATA_DIR>/`（权限默认 `0700`）
- `<DATA_DIR>/modules/`（已安装模块存放）
- `<DATA_DIR>/MODULES/`（本地 Marketplace 扫描目录）
- `<DATA_DIR>/module_data/`（模块运行时配置/私有数据，按模块分目录）

详细结构见：[数据目录结构](../reference/05-data-dir-layout.md)

---

## 6. 停止服务

在运行 `go run` 的终端中：

- `Ctrl + C` 发送 `SIGINT` 即可退出

数据会保留在 `<DATA_DIR>`，下次启动会复用同一个数据库与模块安装状态。

---

## 下一步

- 首次初始化（创建管理员 + 默认 Client Key）：[03-first-setup.md](./03-first-setup.md)

