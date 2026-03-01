# Reference ①：环境变量一览（Environment Variables）

本篇列出当前仓库代码实际读取的环境变量（Core 与模块运行时）。

---

## A. Core（LightBridge 主进程）

### `LIGHTBRIDGE_ADDR`

- 作用：指定监听地址（`host:port`）
- 默认：`127.0.0.1:3210`

示例：

```bash
export LIGHTBRIDGE_ADDR=0.0.0.0:3210
```

### `LIGHTBRIDGE_DATA_DIR`

- 作用：覆盖数据目录
- 默认：`<os.UserConfigDir()>/LightBridge`（不同 OS 路径不同）

示例：

```bash
export LIGHTBRIDGE_DATA_DIR="$HOME/.lightbridge-data"
```

### `LIGHTBRIDGE_MODULE_INDEX`

- 作用：Marketplace 索引来源
- 默认：`github:WilliamWang1721/LightBridge/market/MODULES@main`

可选：

- GitHub 目录扫描（`github:<owner>/<repo>/<path>@<ref>`）
- 远程 `index.json` URL
- `local`（开发/离线兜底）

### `LIGHTBRIDGE_COOKIE_SECRET`

- 作用：管理后台 Cookie 会话签名密钥
- 默认：空（Core 启动时随机生成）

建议：

- 生产环境设置固定值以便跨重启保持会话（或相反：不设置以强制重启失效）

### `LIGHTBRIDGE_MODULES_DIR`

- 作用：指定 local Marketplace 扫描目录（扫描 `*.zip`）
- 说明：仅在 `LIGHTBRIDGE_MODULE_INDEX=local` 时生效
- 默认：未设置时按优先级扫描 `./MODULES`（若存在且大小写匹配）→ `<DATA_DIR>/MODULES`

### `LIGHTBRIDGE_GITHUB_API_BASE`

- 作用：GitHub 目录扫描时使用的 API Base
- 默认：`https://api.github.com`

### `LIGHTBRIDGE_GITHUB_TOKEN`

- 作用：GitHub API Token（用于私仓或提高限额）
- 优先级：`LIGHTBRIDGE_GITHUB_TOKEN` > `GITHUB_TOKEN` > `GH_TOKEN`

---

## B. Module Runtime（Core 启动模块时注入）

以下变量由 Core 在启动模块子进程时注入：

### `LIGHTBRIDGE_MODULE_ID`

- 模块 ID（来自 manifest）

### `LIGHTBRIDGE_DATA_DIR`

- 模块数据目录（注意：这是模块自己的 data dir，不是 Core 的 data dir）
- 实际路径：`<DATA_DIR>/module_data/<module_id>`

### `LIGHTBRIDGE_CONFIG_PATH`

- 模块配置文件路径
- 实际路径：`<DATA_DIR>/module_data/<module_id>/config.json`

### `LIGHTBRIDGE_HTTP_PORT`

- Core 为模块分配的 HTTP 端口（模块应监听 `127.0.0.1:<port>`）

### `LIGHTBRIDGE_GRPC_PORT`

- Core 为模块分配的 gRPC 端口（若模块提供 gRPC 服务）

### `LIGHTBRIDGE_LOG_LEVEL`

- Core 预留的日志级别字段（当前默认注入 `info`）
