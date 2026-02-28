# LightBridge Documentation（拆分版）

本目录为 **LightBridge 的完整 Documentation**（按主题拆分为多篇小文档），用于替代“单个超长文档”的维护方式。

> 适用范围：以本仓库当前代码为准（Go Core / MVP v0.1）。  
> 若你发现文档与实现不一致：以实现为准，并欢迎提交修正。

---

## 推荐阅读路径（从 0 到可用）

1. Getting Started：
   - [01-prerequisites.md](./getting-started/01-prerequisites.md)
   - [02-run.md](./getting-started/02-run.md)
   - [03-first-setup.md](./getting-started/03-first-setup.md)
   - [04-client-config.md](./getting-started/04-client-config.md)
2. Guide（按你要做的事）：
   - [Provider 管理](./guide/01-providers.md)
   - [模型路由与故障转移](./guide/02-routing.md)
   - [模块 Marketplace：安装/配置/启停/卸载](./guide/03-modules-marketplace.md)
   - [Codex OAuth（openai-codex-oauth 模块）](./guide/04-codex-oauth.md)
   - [OpenAI 别名与“应用适配”(Apps)](./guide/05-openai-alias-and-apps.md)
   - [日志与限流](./guide/06-logs-and-rate-limit.md)
   - [数据目录与安全注意事项](./guide/07-security.md)
   - [故障排查](./guide/08-troubleshooting.md)
3. Reference（需要精确信息时查）：
   - [环境变量一览](./reference/01-env-vars.md)
   - [对外 OpenAI 兼容 API](./reference/02-http-api-public.md)
   - [管理后台 Admin API](./reference/03-http-api-admin.md)
   - [模块规范：manifest / index / ZIP](./reference/04-module-manifest.md)
   - [数据目录结构](./reference/05-data-dir-layout.md)
4. Development（参与开发/二次开发）：
   - [架构总览](./development/01-architecture.md)
   - [仓库结构](./development/02-repo-structure.md)
   - [测试指南](./development/03-testing.md)
   - [开发模块（Provider 模块）](./development/04-module-development.md)

---

## 文档目录（文件树）

```text
Documentation/
  README.md
  getting-started/
    01-prerequisites.md
    02-run.md
    03-first-setup.md
    04-client-config.md
  guide/
    01-providers.md
    02-routing.md
    03-modules-marketplace.md
    04-codex-oauth.md
    05-openai-alias-and-apps.md
    06-logs-and-rate-limit.md
    07-security.md
    08-troubleshooting.md
  reference/
    01-env-vars.md
    02-http-api-public.md
    03-http-api-admin.md
    04-module-manifest.md
    05-data-dir-layout.md
  development/
    01-architecture.md
    02-repo-structure.md
    03-testing.md
    04-module-development.md
```

---

## 约定（阅读/操作时的统一说明）

- 文档中使用的占位符：
  - `<ADDR>`：LightBridge 监听地址（默认 `127.0.0.1:3210`）
  - `<BASE>`：对外 Base URL（常见：`http://<ADDR>/v1` 或 `http://<ADDR>/openai/v1`）
  - `<DATA_DIR>`：数据目录（可用 `LIGHTBRIDGE_DATA_DIR` 覆盖）
  - `<ADMIN_URL>`：管理后台入口（`http://<ADDR>/admin`）
  - `<CLIENT_KEY>`：Client API Key（在 Setup Wizard 或 Router/AUTH 页面创建）
- 约定 HTTP 示例均使用 `curl`，并显式携带：
  - `Authorization: Bearer <CLIENT_KEY>`
  - `Content-Type: application/json`（当 body 为 JSON 时）

