# Guide ③：模块与 Marketplace（安装 / 配置 / 启停 / 卸载）

LightBridge 采用“微内核 + 模块”的扩展方式：

- Core 保持轻量（网关、路由、管理后台、SQLite）
- Provider 能力通过模块扩展（本机子进程，Core 分配端口并做健康检查）

本篇讲清楚 Marketplace 的索引来源、模块安装目录、以及常用运维操作。

---

## 1. Marketplace 索引来源（`LIGHTBRIDGE_MODULE_INDEX`）

Core 会根据 `LIGHTBRIDGE_MODULE_INDEX`（或 Admin API 传入的 url 参数）获取模块列表。

支持三种主要来源：

### 1.1 远程 `index.json`（默认 / Phase 2）

推荐写法：

- `https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json`

说明：

- Core 通过 HTTP GET 拉取 `index.json`，无需 GitHub API 目录扫描。
- `index.json` 内容为 ModuleIndex JSON（字段见参考文档）。

### 1.2 GitHub 目录扫描（Phase 1，开发/救援路径）

推荐写法：

- `github:WilliamWang1721/LightBridge/market/MODULES@main`

说明：

- Core 会通过 GitHub API 列出该目录下的 `*.zip`，下载并读取 `manifest.json`，即时生成索引（无需提前生成 `index.json`）。
- 鉴权（可选，用于私仓或提高限额）：
  - `LIGHTBRIDGE_GITHUB_TOKEN`（优先）
  - 或 `GITHUB_TOKEN`
  - 或 `GH_TOKEN`

### 1.3 `local`（开发 / 离线兜底）

会扫描本地目录中的 `*.zip` 作为模块包。

扫描目录优先级（从高到低）：

1. `LIGHTBRIDGE_MODULES_DIR`（显式指定）
2. 当前工作目录下存在 `./MODULES`（注意：必须大小写完全匹配 `MODULES`）
3. `<DATA_DIR>/MODULES`

> 注意：`file://` 形式的 index URL 不支持；本地请用 `local`。

---

## 2. 模块安装后存放在哪里？

安装后 Core 会把模块解压到：

- `<DATA_DIR>/modules/<module_id>/<version>/...`

并把“安装记录”写入 SQLite（InstalledModules 表）。

模块运行时配置与私有数据在：

- `<DATA_DIR>/module_data/<module_id>/config.json`
- `<DATA_DIR>/module_data/<module_id>/...`（模块自行写入）

完整结构见：[数据目录结构](../reference/05-data-dir-layout.md)

---

## 3. 在管理后台安装模块（推荐）

入口：

- `http://<ADDR>/admin/marketplace`

常见流程：

1. 选择/确认索引来源（默认就是远程 Marketplace）
2. 在模块列表中点击 Install
3. 若模块处于 enabled 状态，系统会尝试启动模块并做健康检查

安装行为（Core 内部）：

- 下载/读取 ZIP
- 校验 SHA256
- 解压
- 在安装目录中寻找 `manifest.json`（优先 `dist/manifest.json`）
- 校验 manifest 字段与协议
- 写入 SQLite 安装记录

---

## 4. 启动/停止模块（Runtime）

模块是“本机子进程”，Core 启动时会：

1. 为模块分配空闲端口：
   - `LIGHTBRIDGE_HTTP_PORT`
   - `LIGHTBRIDGE_GRPC_PORT`
2. 注入运行时环境变量（见参考文档）
3. 启动进程
4. 根据 manifest 的 health 配置进行健康检查
5. 健康后写入 runtime（PID/端口/启动时间）

你可以通过 API 查看模块运行状态：

```bash
curl -s "http://127.0.0.1:3210/admin/api/modules" \
  -H "Cookie: lightbridge_admin=<your_cookie_here>"
```

（返回里会包含 `runtime.http_port` / `runtime.grpc_port`）

---

## 5. 启用/禁用模块（Enabled）

Enabled 与 Runtime 的区别：

- Enabled：是否允许该模块被启动（持久化在 DB）
- Runtime：当前是否正在运行（PID/端口）

禁用模块通常会：

- 停止进程
- 将其暴露的 Provider alias 标记为 disabled/down（以避免被路由选中）

---

## 6. 模块配置（config.json / schema / defaults）

Core 会为每个模块维护一个配置文件：

- `<DATA_DIR>/module_data/<module_id>/config.json`

并通过 Admin API 提供读取/写入：

- `GET  /admin/api/modules/config?module_id=...`（返回 config/schema/defaults/config_path）
- `POST /admin/api/modules/config`（可携带 `restart=true`）

模块侧读取方式：

- 从环境变量 `LIGHTBRIDGE_CONFIG_PATH` 获取路径并读取 JSON

---

## 7. 卸载模块（可选清理数据）

卸载会做三件事：

1. 停止模块进程
2. 删除 DB 安装记录
3. 删除安装目录 `<DATA_DIR>/modules/<id>/...`

可选：

- `purge_data=true` 时，连同 `<DATA_DIR>/module_data/<id>/` 一并删除（会丢失 OAuth Token / 模块私有数据）

---

## 8. 升级模块（Upgrade）

升级会：

- 从 index 中重新选择指定模块最新/指定版本
- 重新下载安装
- 替换安装记录
-（如 enabled）停止旧进程并启动新版本

对应 API：

- `POST /admin/api/modules/upgrade`

---

## 下一步

- 想接入 Codex OAuth？继续看：[Codex OAuth（openai-codex-oauth）](./04-codex-oauth.md)
- 想了解模块 manifest 规范？看参考：[模块规范](../reference/04-module-manifest.md)
