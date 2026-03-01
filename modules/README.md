# Module Marketplace

LightBridge 默认使用 GitHub 目录扫描作为 Marketplace 源：

- 默认：`LIGHTBRIDGE_MODULE_INDEX=github:WilliamWang1721/LightBridge/market/MODULES@main`
- Core 会扫描该目录下的 `*.zip` 并即时生成模块索引（无需提前生成 `index.json`）

静态索引（适合规模化）：

- `LIGHTBRIDGE_MODULE_INDEX=https://raw.githubusercontent.com/WilliamWang1721/LightBridge/main/market/MODULES/index.json`

本地扫描仍然支持（开发/离线兜底）：

- `LIGHTBRIDGE_MODULE_INDEX=local` 会扫描 `./MODULES`（若存在且大小写精确为 `MODULES`）→ `<DATA_DIR>/MODULES`
- 你可以用 `LIGHTBRIDGE_MODULES_DIR` 覆盖本地扫描目录
