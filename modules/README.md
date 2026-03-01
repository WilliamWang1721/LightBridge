# Module Marketplace

LightBridge 默认使用静态 `index.json` 作为 Marketplace 源（Phase 2）：

- 默认：`LIGHTBRIDGE_MODULE_INDEX=https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json`
- Core 会通过 HTTP GET 拉取索引，不依赖 GitHub 目录扫描

GitHub 目录扫描（Phase 1，开发/救援路径）：

- `LIGHTBRIDGE_MODULE_INDEX=github:WilliamWang1721/LightBridge/market/MODULES@main`（扫描目录下的 `*.zip` 并即时生成索引）

本地扫描仍然支持（开发/离线兜底）：

- `LIGHTBRIDGE_MODULE_INDEX=local` 会扫描 `./MODULES`（若存在且大小写精确为 `MODULES`）→ `<DATA_DIR>/MODULES`
- 你可以用 `LIGHTBRIDGE_MODULES_DIR` 覆盖本地扫描目录
