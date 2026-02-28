# Passkey 登录模块（计划）

> 目标：在 LightBridge Admin 中新增一个“Passkey 登录”安全认证模块，使管理员可使用 Passkey（WebAuthn）完成身份验证；支持系统内置认证（Face ID / Touch ID / Windows Hello）与第三方凭证管理器（如 1Password）。

## 0. 项目现状（基于代码仓库审阅）

- Admin 登录页：`/admin/login`，模板为 `internal/gateway/web/templates/login.html`，登录 API 为 `POST /admin/api/login`（`internal/gateway/admin.go`）。
- Admin UI 为 Go `net/http` 服务端渲染模板（`internal/gateway/server.go` 的 `routeAdminPages` / `routeAdminAPI`）。
- 模块系统：
  - 安装信息持久化在 SQLite：`modules_installed` / `module_runtime`（`internal/db/db.go`、`internal/store/store.go`）。
  - 模块安装/启动/停止 API：`/admin/api/modules/*`（`internal/gateway/server.go`）。
  - 当前 manifest 校验仅支持 `services[].kind=provider`（`internal/modules/marketplace.go: validateManifest`），因此本模块需以“provider 模块”形式打包，但 **不暴露 provider alias**，避免对 Provider 列表产生副作用。
- 侧边栏“设置 Settings”目前指向 `/admin/logs`（多处模板一致），需要落地一个遵循新 UI 标准的 Settings 页面作为入口承载 Passkey 配置项。

## 1. 需求拆解与产品行为

### 1.1 登录入口与流程

- 入口位置：`/admin/login` 页面最下方的“以其他方式登录”按钮（现为“以自定义方式登录”且 disabled）。
- 启用条件：当检测到 **Passkey 登录模块已安装**（且建议要求 enabled）时按钮可点击；否则保持 disabled。
- 点击后行为：触发 WebAuthn `navigator.credentials.get()`，弹出系统 Passkey 认证框；成功后创建 Admin Session 并跳转 `/admin/dashboard`。

### 1.2 UI 与页面设计

- Passkey 专门登录页（或同页切换视图）：
  - 复用现有登录页的字体、卡片、按钮样式（见 `login.html`）。
  - 展示“使用 Passkey 登录”的说明、错误态、返回密码登录入口。
- 设置页：
  - 从侧边栏 Setting 标签进入（建议新增 `/admin/settings` 并将各模板侧边栏链接统一指向该路由）。
  - 若已安装 Passkey 模块：展示 Passkey 管理区（创建/列出/删除），以及“仅 Passkey 登录 / 允许密码+Passkey”选项。
  - 若未安装：不渲染任何 Passkey 相关内容（页面可留作后续扩展）。

### 1.3 兼容性要求（落地策略）

- 不强制硬件设备：WebAuthn 选项不限定 `authenticatorAttachment`；`userVerification` 使用 `preferred`（默认优先使用 Face ID/Touch ID/Windows Hello）。
- 支持第三方（如 1Password）：同样依赖“不限制认证器类型 + 使用标准 WebAuthn 参数”，并优先采用可发现凭证（resident key / discoverable credential）路径。

## 2. 技术方案（推荐）

### 2.1 总体架构

采用“**核心网关 + 可安装模块**”模式：

1) **Passkey 模块（可安装）**：在 `modules/passkey-login/` 提供可执行文件（HTTP 服务），负责：
- 生成注册/登录 challenge（WebAuthn options）
- 校验 WebAuthn attestation/assertion
- 在模块数据目录（`$LIGHTBRIDGE_DATA_DIR`）持久化 Passkey 凭证

2) **核心网关（gateway）**：在 `internal/gateway` 增加 Admin API 与页面逻辑，负责：
- 检测模块是否安装以启用 UI
- 代理调用模块 HTTP 接口（复用 `proxyModuleHTTP` 方式）
- 在 Passkey 登录成功后由网关创建 Admin Session Cookie（`sessionManager.newSession`）
- 维护“是否允许密码登录”的全局设置（存储在 `settings` 表；不需要二次验证，因为进入设置页本身已登录）

### 2.2 模块 manifest 约束与设计

由于当前模块系统只允许 `services[].kind=provider`，Passkey 模块 manifest 需满足：

- `services` 至少一个条目，`kind=provider`，`protocol=http_rpc`（或 `http_openai` 均可）。
- `expose_provider_aliases` 留空（避免注册 provider）。
- 健康检查提供 `/health`，保证启动稳定。

### 2.3 数据持久化（模块侧）

模块侧数据文件建议：

- `passkeys.json`：存储 credential 列表（按 username 分组）
  - `credential_id`（base64url）
  - `public_key`（建议存 COSE key 原始 bytes 的 base64url，或解析后的 x/y）
  - `sign_count`
  - `created_at` / `last_used_at`
  - `label`（用户在设置页可选填）

并在删除最后一个 passkey 时，网关侧自动将“仅 Passkey 登录”回退为“允许密码+Passkey”，避免锁死。

### 2.4 网关侧设置项（核心）

在 `settings` 表新增 key（全局）：

- `admin_password_login_enabled`：`"1"`/`"0"`
  - 默认 `"1"`：允许密码登录（同时 Passkey 可用）
  - 为 `"0"`：仅允许 Passkey 登录；`/admin/api/login` 即使密码正确也返回 403

启用 `"0"` 前需满足：当前至少存在 1 个 Passkey（由设置页调用模块 list 接口验证）。

## 3. API 设计（网关对前端）

> 说明：所有 `/admin/api/passkey/*` 由 gateway 暴露给前端；gateway 内部再通过 `proxyModuleHTTP(ctx, passkeyModuleID, ...)` 转发至模块。

### 3.1 未登录可用（用于登录流程）

- `POST /admin/api/passkey/auth/begin`
  - 入参：`{ username?: string }`（可选；不传则走 discoverable）
  - 出参：`{ publicKey: PublicKeyCredentialRequestOptionsJSON, state: string }`
- `POST /admin/api/passkey/auth/finish`
  - 入参：`{ state: string, credential: PublicKeyCredentialJSON, remember?: boolean }`
  - 成功：gateway 写入 session cookie，返回 `{ ok: true, next: "/admin/dashboard" }`

### 3.2 已登录可用（用于设置/注册）

- `POST /admin/api/passkey/register/begin`
- `POST /admin/api/passkey/register/finish`
- `GET  /admin/api/passkey/credentials`
- `POST /admin/api/passkey/credentials/delete`
- `GET  /admin/api/passkey/password_login_enabled`
- `POST /admin/api/passkey/password_login_enabled`

## 4. 页面改动清单

### 4.1 Admin Login（`login.html`）

- 将底部按钮文案替换为“以其他方式登录”。
- 根据 `passkeyInstalled` 决定 disabled。
- 点击后执行：
  1) 调用 `/admin/api/passkey/auth/begin`
  2) `navigator.credentials.get({ publicKey })`
  3) 调用 `/admin/api/passkey/auth/finish`，成功后跳转

### 4.2 Settings 页面（新增）

- 新增 `internal/gateway/web/templates/settings.html`（与 router/auth/dashboard 同一套 UI 标准）。
- 侧边栏 Setting 统一指向 `/admin/settings`。
- 若未安装 Passkey 模块：不渲染 Passkey 区域。
- 若已安装：渲染
  - 创建 Passkey（调用 register begin/finish）
  - Passkey 列表与删除
  - “允许密码登录”开关（满足安全前置条件后才允许关闭）

## 5. 测试与验收

### 5.1 手动验收（最重要）

- 未安装模块：
  - `/admin/login` 的“以其他方式登录”按钮 disabled
  - `/admin/settings` 不出现 Passkey 相关内容
- 安装模块后：
  - 点击按钮能弹出系统 Passkey 认证框并登录成功
  - 设置页可创建 Passkey（无需输入密码）
  - 可删除 Passkey；删除最后一个时自动回退“允许密码登录”
- 兼容性：
  - macOS Safari/Chrome 使用 Touch ID/Face ID（如设备支持）
  - 浏览器扩展/第三方（如 1Password）可正常弹出并完成流程

### 5.2 自动化（建议）

- gateway：对 `admin_password_login_enabled` 的行为添加单测（密码登录被禁用时返回 403）。
- 模块：对 challenge/state 过期、签名校验失败等关键路径添加单测（可先覆盖 JSON 存取与状态机）。

## 6. 实施步骤（执行顺序）

1) 新增 `modules/passkey-login`（HTTP 服务 + manifest + package.sh）。
2) gateway 增加 `/admin/api/passkey/*` 路由与处理（代理模块 + session 创建）。
3) 修改 `login.html`：启用入口、接入 Passkey 登录流程、错误态提示。
4) 新增 `settings.html` + 更新侧边栏 Setting 链接；实现 Passkey 管理 UI。
5) 加入设置项 `admin_password_login_enabled` 并在 `POST /admin/api/login` 执行拦截逻辑。
6) 补充测试与文档（`README`/`参考资料` 更新）。

