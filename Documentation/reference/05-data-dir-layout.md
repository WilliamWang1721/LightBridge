# Reference ⑤：数据目录结构（Data Dir Layout）

本篇描述 `<DATA_DIR>` 的真实目录结构与各文件用途，便于你：

- 找到数据库/模块/配置文件
- 备份与迁移
- 排查“模块端口/配置/凭据”问题

---

## 1. `<DATA_DIR>` 是什么？

Core 的数据目录由配置决定：

- 默认：`os.UserConfigDir()/LightBridge`
- 覆盖：`LIGHTBRIDGE_DATA_DIR=/path/to/dir`

数据库路径固定为：

- `<DATA_DIR>/lightbridge.db`

---

## 2. 目录结构（实际会被创建）

Core 启动时会确保以下目录存在：

```text
<DATA_DIR>/
  lightbridge.db
  modules/
  MODULES/
  module_data/
```

其中：

### 2.1 `lightbridge.db`

SQLite 数据库，包含：

- 管理员账号（密码 hash）
- Client API Keys（明文）
- Providers（config_json 里常含上游 key，明文）
- Models / Routes
- 模块安装与运行时信息
- 请求元数据日志

### 2.2 `modules/`（模块安装目录）

模块安装解压位置：

```text
<DATA_DIR>/modules/<module_id>/<version>/...
```

其中会包含模块的 `manifest.json`、可执行文件等。

### 2.3 `MODULES/`（本地 Marketplace 扫描目录）

默认情况下 Core 使用远程 Marketplace；当 `LIGHTBRIDGE_MODULE_INDEX=local` 时，Core 才会扫描一个本地目录下的 `*.zip`：

优先级：

1. `LIGHTBRIDGE_MODULES_DIR`
2. `./MODULES`（若存在且大小写为 `MODULES`）
3. `<DATA_DIR>/MODULES`

因此你可以把模块 zip 放到：

- `<DATA_DIR>/MODULES/*.zip`

或在仓库根目录建一个：

- `./MODULES/*.zip`

### 2.4 `module_data/`（模块数据与配置）

每个模块有独立的“模块数据目录”：

```text
<DATA_DIR>/module_data/<module_id>/
  config.json
  ... (module specific files)
```

Core 会把该目录路径注入给模块：

- `LIGHTBRIDGE_DATA_DIR=<DATA_DIR>/module_data/<module_id>`

模块的配置文件路径：

- `LIGHTBRIDGE_CONFIG_PATH=<DATA_DIR>/module_data/<module_id>/config.json`

---

## 3. 权限与安全提示

Core 会尝试用较严格权限创建 `<DATA_DIR>`（例如 `0700`）。

你仍应将其视为敏感目录：

- DB 和模块数据里可能包含明文 key/token
- 备份/迁移时注意保护备份文件

详见：[安全注意事项](../guide/07-security.md)
