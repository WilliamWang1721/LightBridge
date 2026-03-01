# market/MODULES（远程 Marketplace 源目录）

说明：

- 本目录用于承载 LightBridge 的“远程模块源”（供 GitHub 目录扫描或静态 `index.json` 使用）。
- 之所以不放在仓库根 `MODULES/`：macOS 默认大小写不敏感文件系统下，根目录已存在 `modules/` 源码目录，无法再创建同级的 `MODULES/`。

## Phase 1（GitHub 目录扫描）

- 将模块打包为 ZIP（包含 `manifest.json` + `bin/<os>/<arch>/...`）。
- 在本目录中**每个模块 ID 仅保留 1 个 ZIP（最新）**，建议命名为：
  - `market/MODULES/<module_id>.zip`

默认配置：

- `LIGHTBRIDGE_MODULE_INDEX=github:WilliamWang1721/LightBridge/market/MODULES@main`

## Phase 2（静态 index.json + GitHub Releases）

- 维护 `market/MODULES/index.json`（每个 `id` 只出现一次，代表最新版本）。
- ZIP 不再提交进 git，改为上传到 GitHub Releases。

配置示例：

- `LIGHTBRIDGE_MODULE_INDEX=https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json`

