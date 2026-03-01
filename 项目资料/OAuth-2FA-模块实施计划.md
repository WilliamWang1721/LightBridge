# OAuth 安全分类：2FA 模块实施计划（待你确认后执行）

> 文档状态：草案（只做规划，不改业务代码）  
> 日期：2026-02-28

## 1. 目标与边界

### 1.1 目标
在 LightBridge Admin 增加一个 **2FA（TOTP）模块**，归类到 OAuth/Auth 安全能力，支持：

1. 在 Settings 中创建 2FA，展示二维码供验证器扫码，生成 6 位动态码。
2. 登录流程支持：
   - 密码后追加 2FA（已开启 2FA 时强制）
   - 强制 2FA（无论密码或 Passkey 都要过 2FA）
   - 仅使用 2FA 代码登录（不输入密码）
3. 支持多个 2FA 验证器并行绑定。
4. 支持“密码 + 2FA”模式，同时支持“仅密码登录需要 2FA”的模式。
5. 与 Passkey 等其他 AUTH 模块兼容，避免页面入口冲突。

### 1.2 本期不做（明确边界）
1. 不做短信/邮件 OTP（仅 TOTP）。
2. 不做高复杂度风险引擎（如设备指纹风险评分），仅预留策略扩展点。
3. 不在本阶段改造客户端 API Key 鉴权（仅 Admin 登录链路）。

---

## 2. 现状评估（基于当前仓库）

1. 已有密码登录：`POST /admin/api/login`（`internal/gateway/admin.go`）。
2. 已有 Passkey 模块与登录链路：
   - 网关接口：`/admin/api/passkey/auth/begin|finish`（`internal/gateway/passkey.go`）
   - 登录页已存在“以其他方式登录”按钮（`internal/gateway/web/templates/login.html`）
3. 登录页已有 Passkey 触发逻辑，当前是“单按钮直接走 Passkey”，不是“下拉可选多方式”。
4. 模块系统支持 Auth 类模块安装/启停，并可通过 `GetInstalledModule` 判断是否安装且启用。

结论：
- 不应重做登录体系，应在现有 `password + passkey` 基础上扩展为 **统一可插拔认证入口**。
- 2FA 需要在网关加入“预认证态（pending auth）”处理，避免 Passkey 登录后直接落 Session 绕过强制 2FA。

---

## 3. 线上调研结论（官方文档 + 成熟案例）

### 3.1 标准/规范
1. TOTP 核心基于 RFC 6238（默认时间步长 30 秒、一次性口令）。
2. HOTP/TOTP 基础来自 RFC 4226，服务端要防重放、限速。
3. NIST SP 800-63B（800-63-4 版本）指出 OTP 不是抗钓鱼方案，适合作为提升层，但应配合更强因子（如 Passkey/WebAuthn）。
4. WebAuthn Level 3 规范明确 discoverable credentials（resident key/passkey）与 `userVerification` 行为，可用于与 2FA 策略协同。

### 3.2 成熟实践
1. GitHub：支持 TOTP/SMS/安全密钥/Passkey，并强调**备用恢复方式**与多方式并存，降低锁死风险。
2. Auth0：以策略驱动（Never/Adaptive/Always），并支持按场景强制 MFA，适合映射到本项目“仅密码需要2FA / 全方式强制2FA”。
3. OWASP MFA Cheat Sheet：强调失败处理、重置流程防滥用、建议配置多种因子/恢复方式。

### 3.3 对本项目的直接指导
1. 2FA 与 Passkey 不是替代关系，而是策略组合关系。
2. 必须做多验证器与备用机制（至少支持多设备）。
3. 对“强制 2FA”必须覆盖所有主登录方式，包含 Passkey。

---

## 4. 功能设计（对应你的需求逐条落地）

## 4.1 Setting：创建与绑定 2FA

流程：
1. 用户在 Settings 打开“2FA 安全”区。
2. 点击“添加验证器”后，后端生成 `secret`（Base32）并返回：
   - `otpauth://totp/...` URI
   - QR Code（SVG/PNG，前端展示）
3. 用户扫码后输入 6 位验证码进行确认。
4. 验证通过才真正绑定（避免“扫了但没激活”）。

支持项：
1. 一个账号可绑定多个验证器（设备名可自定义，如 iPhone、1Password）。
2. 可删除某个验证器。
3. 至少保留一个验证器时才允许启用“2FA-only 登录”（防锁死）。

## 4.2 登录验证流程

### A. 密码验证后追加 2FA
1. 密码正确后，不立即创建 session。
2. 返回 `requires_2fa=true` + `auth_ticket`。
3. 前端弹出 2FA 输入框，提交验证码。
4. 验证通过后才签发 session。

### B. 强制验证（密码/Passkey 均需 2FA）
1. 对密码流和 Passkey 流统一套用 `enforce_all_methods=true`。
2. Passkey 成功后同样进入 `auth_ticket` 阶段，而非直接登录。

### C. 独立登录（仅 2FA）
1. 登录页“使用其他方式登录”下拉中增加“使用 2FA 代码登录”。
2. 进入 2FA-only 流程：`username + 2FA code`（不输入密码）。
3. 仅当用户在设置里主动开启 `allow_totp_only_login=true` 时显示。

## 4.3 组合模式（策略矩阵）

定义策略字段（建议）：
1. `enabled`：是否启用 2FA
2. `require_for_password`：密码登录后是否必须 2FA
3. `require_for_passkey`：Passkey 登录后是否必须 2FA
4. `allow_totp_only_login`：是否允许仅 2FA 登录

可表达你的全部场景：
1. 仅密码需要 2FA：`require_for_password=true, require_for_passkey=false`
2. 全方式强制 2FA：`require_for_password=true, require_for_passkey=true`
3. 密码+2FA 组合：`enabled=true, require_for_password=true`
4. 仅 2FA 登录：`allow_totp_only_login=true`

---

## 5. 登录入口与冲突处理（你重点要求）

## 5.1 Admin Login 入口改造

当前是“单按钮触发 Passkey”，改为：
1. 按钮文案：`使用其他方式登录`。
2. 点击后展示下拉菜单（动态项）：
   - 使用 Passkey 登录（安装且启用 passkey 模块时）
   - 使用 2FA 代码登录（安装且启用 2FA 模块，且策略允许时）

## 5.2 与其他 AUTH 模块冲突避免

新增统一接口：`GET /admin/api/auth/methods`（未登录可访问）。
- 返回当前可用的登录方式列表，由后端统一判定模块状态与策略。
- 登录页、Passkey 页、未来其他登录页都只消费该接口，不各自硬编码。

这样可避免：
1. Passkey 页面与 2FA 页面各自重复实现逻辑。
2. 模块增减后前端入口错乱。
3. 多模块同时声明“other login”导致按钮冲突。

---

## 6. 模块显示逻辑（你要求）

1. 若未安装 2FA 模块：
   - Settings 不显示 2FA 区块。
   - 登录下拉不显示“2FA 登录”。
2. 若安装但未启用：
   - 可在 Settings 展示“模块已安装但未启用”的提示（仅提示，不展示绑定操作）。
3. 若安装并启用：
   - 显示完整 2FA 设置能力。

---

## 7. 技术架构设计

## 7.1 模块化边界

建议新增模块：`totp-2fa-login`（Auth 标签）。

模块负责：
1. 生成/校验 TOTP。
2. 管理 2FA 设备（多验证器）。
3. 维护临时 challenge state。

网关负责：
1. 登录编排（password/passkey/totp-only 的统一状态机）。
2. session 签发。
3. 策略持久化与页面渲染。

## 7.2 数据模型

网关侧 `settings`（建议 key）：
1. `admin_2fa_enabled`
2. `admin_2fa_require_for_password`
3. `admin_2fa_require_for_passkey`
4. `admin_2fa_allow_totp_only`

模块侧存储（`$LIGHTBRIDGE_DATA_DIR`）：
- `totp_secrets.json`（或 sqlite）
  - `username`
  - `device_id`
  - `device_label`
  - `secret_enc`（建议加密存储）
  - `created_at`
  - `last_used_at`

安全要求：
1. secret 不可明文日志输出。
2. 验证码仅用于即时校验，不持久化。
3. 对校验接口限流（每账户/每IP）。

## 7.3 核心 API（草案）

网关对前端：
1. `GET /admin/api/auth/methods`
2. `POST /admin/api/2fa/challenge/start`
3. `POST /admin/api/2fa/challenge/verify`
4. `POST /admin/api/2fa/totp-only/login`
5. `GET /admin/api/2fa/policy`（需登录）
6. `POST /admin/api/2fa/policy`（需登录）
7. `POST /admin/api/2fa/enroll/begin`（需登录）
8. `POST /admin/api/2fa/enroll/confirm`（需登录）
9. `GET /admin/api/2fa/devices`（需登录）
10. `POST /admin/api/2fa/devices/delete`（需登录）

网关对 2FA 模块：
1. `/totp/enroll/begin`
2. `/totp/enroll/confirm`
3. `/totp/verify`
4. `/totp/devices`
5. `/totp/devices/delete`

---

## 8. 交互与用户体验（UX）

1. 扫码流程必须有“我已扫码，输入 6 位码确认”步骤，避免误绑定。
2. 2FA 输入错误时提供明确提示：
   - 代码错误/已过期/请求过于频繁。
3. 在强制模式下，登录页明确提示“当前账户需要二次验证”。
4. 2FA-only 模式开启时，在设置页展示风险提示与一键回退入口。
5. 首次开启强制模式前，检测是否至少绑定 1 个验证器，未满足则禁止保存。

---

## 9. 实施阶段（确认后执行）

### 阶段 1：基础能力
1. 创建 2FA 模块骨架（manifest、health、存储、TOTP verify）。
2. 完成 Settings 绑定/删除多个验证器。

### 阶段 2：登录编排
1. 引入 `auth_ticket` 预认证态。
2. 改造密码登录与 Passkey 登录，使其可进入 2FA challenge。
3. 新增 2FA-only 登录流程。

### 阶段 3：统一登录入口
1. 登录页改为“其他方式下拉”。
2. 加入 `/admin/api/auth/methods` 统一入口。

### 阶段 4：测试与文档
1. 单元测试：TOTP 时间窗口、错误码、策略判定。
2. 集成测试：password/passkey/2fa-only 三链路。
3. Playwright 自检（见下一节）。

---

## 10. Playwright 自检清单（完成后必须执行）

1. 未安装 2FA 模块：
   - Settings 无 2FA 区块；登录下拉无 2FA 选项。
2. 安装并启用模块：
   - 可生成二维码并成功绑定，显示设备列表。
3. 多验证器：
   - 连续绑定两个设备，任一设备代码可通过登录。
4. 密码 + 2FA：
   - 密码正确后出现 2FA challenge，验证成功登录。
5. 仅密码需要 2FA：
   - Passkey 登录直接成功，密码登录需 2FA。
6. 全方式强制 2FA：
   - 密码和 Passkey 都进入 2FA challenge。
7. 2FA-only：
   - 登录页可从“其他方式”选择 2FA 登录并成功。
8. 冲突验证：
   - Passkey 页面也能看到统一“其他方式”入口，不冲突。

---

## 11. 风险与缓解

1. 风险：启用 2FA-only 但设备丢失导致锁死。
   - 缓解：启用前必须 >=1 设备，建议后续加入恢复码。
2. 风险：Passkey 现有流程“先创建 session”导致绕过强制 2FA。
   - 缓解：引入统一 `auth_ticket` 状态机后再签发 session。
3. 风险：时间漂移导致 TOTP 误判。
   - 缓解：允许有限时间窗口（如 ±1 step），并在 UI 给出设备时间校准提示。

---

## 12. 预估改动文件（确认后再动手）

核心网关：
1. `internal/gateway/admin.go`
2. `internal/gateway/passkey.go`
3. `internal/gateway/server.go`
4. `internal/store/store.go`
5. `internal/gateway/web/templates/login.html`
6. `internal/gateway/web/templates/login_passkey.html`
7. `internal/gateway/web/templates/settings.html`（若当前尚未完成）

新模块（建议）：
1. `modules/totp-2fa-login/manifest.json`
2. `modules/totp-2fa-login/cmd/totp-2fa-login/*.go`
3. `modules/totp-2fa-login/README.md`

---

## 13. 参考资料（本次已调研）

1. RFC 6238 (TOTP): https://www.rfc-editor.org/rfc/rfc6238
2. RFC 4226 (HOTP): https://datatracker.ietf.org/doc/html/rfc4226
3. NIST SP 800-63B (63-4): https://pages.nist.gov/800-63-4/sp800-63b.html
4. OWASP MFA Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html
5. W3C WebAuthn Level 3: https://www.w3.org/TR/webauthn-3/
6. GitHub 2FA 文档（Passkey + fallback 实践）:
   https://docs.github.com/en/authentication/securing-your-account-with-two-factor-authentication-2fa/about-two-factor-authentication
7. Auth0 MFA 策略（Never/Adaptive/Always）:
   https://auth0.com/docs/secure/multi-factor-authentication/enable-mfa

