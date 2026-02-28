# Guide ④：Codex OAuth（openai-codex-oauth 模块）

本篇以“实际可操作流程”为主，带你在 LightBridge 中通过模块 `openai-codex-oauth` 以 OAuth 方式接入 Codex，并最终得到一个可用的 Provider：`codex`。

---

## 0. 你将得到什么？

完成本篇后，你会拥有：

- 已安装并运行的模块：`openai-codex-oauth`
- 一个可路由的 Provider（推荐 ID）：`codex`
  - `protocol = codex`
  - `endpoint = http://127.0.0.1:<module_http_port>`（由 Core 自动推导）
- 可在下游请求中使用：
  - `model@codex`（变体强制走 Codex）
  - 或配置全局路由，让 `gpt-*` 默认走 codex（见路由文档）

---

## 1. 前置条件

1. LightBridge Core 已启动，并完成 Setup Wizard（有管理员账号 & Client API Key）
2. 你能打开管理后台：
   - `http://127.0.0.1:3210/admin`
3. 你准备使用 OAuth 登录（浏览器需要能访问 OpenAI 的登录页）

---

## 2. 安装模块（Marketplace）

入口：

- `/admin/marketplace`

### 2.1 使用 local Marketplace（最简单）

local 模式会扫描：

- `./MODULES`（若存在且大小写为 `MODULES`）
- 或 `<DATA_DIR>/MODULES`
- 或你指定的 `LIGHTBRIDGE_MODULES_DIR`

如果你在仓库内开发模块，可直接按模块 README 打包：

```bash
bash modules/openai-codex-oauth/package.sh
mkdir -p MODULES
cp modules/openai-codex-oauth/dist/*.zip MODULES/
```

然后回到 Marketplace 页面：

- 选择索引来源 `local`
- 找到 `openai-codex-oauth` 并安装

> 更完整的 Marketplace 说明见：[模块与 Marketplace](./03-modules-marketplace.md)

---

## 3. 启动模块并确认运行状态

安装后在 Marketplace 或 Modules 列表中启动模块（不同 UI 版本入口略有差异）。

你也可以用 API 确认模块是否运行：

```bash
curl -s "http://127.0.0.1:3210/admin/api/modules" \
  -H "Cookie: lightbridge_admin=<your_cookie_here>"
```

找到 `openai-codex-oauth`：

- `runtime.http_port > 0` 表示模块已拿到 HTTP 端口并通过健康检查

---

## 4. 在 Providers 页面打开 Codex OAuth 弹窗

入口：

- `/admin/providers`

常见操作：

1. 点击「添加」
2. 选择（或自动读取到）Codex（OpenAI）
3. 打开 Codex OAuth 弹窗

如果弹窗提示“请先在 Marketplace 安装 openai-codex-oauth”：

- 说明模块尚未安装/未启动，回到第 2～3 步检查

---

## 5. OAuth 登录：推荐路径（自动回调）

点击弹窗内的：

- 「生成 OAuth 链接」

点击后，弹窗会自动显示「回调 URL」快速输入区（位于高级选项之外），用于回调异常时手动换取 Token。

后台会做两件事：

1. Core 会尝试在本机启动一个回调监听：
   - `127.0.0.1:1455`
   - `[::1]:1455`
2. 请求模块生成 OAuth URL，并指定 `redirect_uri`：
   - `http://localhost:1455/auth/callback`

随后浏览器会打开 OAuth 登录页面。你完成授权后：

- 浏览器会跳转到 `http://localhost:1455/auth/callback?code=...&state=...`
- Core 本地回调服务会把 `code/state` 转发给模块 `/auth/oauth/exchange` 换取 Token
- 页面会显示成功提示，并引导你回到 LightBridge Providers 页面

---

## 6. OAuth 失败时的备用方案（手动粘贴回调 URL）

如果你的环境无法启动本地回调服务（常见原因：`1455` 端口被占用），你仍可使用弹窗的备用功能：

1. 点击「生成 OAuth 链接」后，在弹窗主区域找到「回调 URL」输入框（无需展开高级选项）
2. 把浏览器地址栏里的完整回调 URL 粘贴进去（包含 `code` 与 `state`）
3. 点击「从回调 URL 获取 Token」

该操作会调用：

- `POST /admin/api/codex/oauth/exchange`

由 Core 代理转发给模块完成换取。

---

## 7. Device Flow（设备码登录）

若 OAuth 网页跳转受限，你可以使用 Device Flow：

1. 在 Codex OAuth 弹窗中点击「生成 Device Code」（或相似文案）
2. 弹窗会显示：
   - `verification_url`
   - `user_code`
3. 打开 `verification_url` 并输入 `user_code`
4. 返回弹窗刷新状态，直到显示已授权

底层 API：

- `POST /admin/api/codex/device/start`（由 Core 代理到模块）
- `GET  /admin/api/codex/oauth/status`（查看当前状态）

---

## 8. 手动导入 Token（高级）

如果你已经有可用的：

- `refresh_token` 或 `access_token`

可以在弹窗高级选项中手动导入。

底层 API：

- `POST /admin/api/codex/oauth/import`

---

## 9. 保存 Provider（关键一步）

当弹窗状态显示已获取 Token 后：

- 点击弹窗内的「保存」

系统会确保写入一个 Provider（推荐 ID：`codex`），并自动推导 endpoint：

- `protocol = codex`
- `endpoint = http://127.0.0.1:<module_http_port>`

为什么推荐 Provider ID 使用 `codex`？

- Core 的 fallback 规则会把很多 `gpt-*` / `o*-*` 模型优先推断到 provider ID `codex`
- 这样你在没写 routes 的情况下，也更可能默认走 Codex（当然仍建议显式配置 routes）

---

## 10. 验证调用

### 10.1 强制走 codex（变体语法）

```bash
curl -s "http://127.0.0.1:3210/v1/chat/completions" \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"gpt-4o-mini@codex",
    "messages":[{"role":"user","content":"Hello from codex"}]
  }'
```

### 10.2 配置全局 routes（推荐长期做法）

把 `gpt-4o-mini` 的 routes 指向 `codex`，见：

- [模型路由与故障转移](./02-routing.md)

---

## 11. Token 存储位置（重要）

模块会把凭据写入它自己的数据目录（由 Core 注入 `LIGHTBRIDGE_DATA_DIR`）：

- `<DATA_DIR>/module_data/openai-codex-oauth/credentials.json`

权限/安全注意事项见：[数据目录与安全注意事项](./07-security.md)
