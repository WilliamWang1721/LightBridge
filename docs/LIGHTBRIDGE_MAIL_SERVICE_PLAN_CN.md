# LightBridge Mail Service 长期实施计划

> 目标：为 LightBridge 增加一个可选安装、轻量外置、品牌统一、可长期稳定运行的邮件服务，用于关联账户管理中的 OAuth 账户、邮箱池管理、邮件读取、验证码/验证链接获取，以及后续自动化注册流程辅助。

---

## 0. 背景与最终命名

本方案中的所有用户可见名称统一为 **LightBridge Mail Service**，简称 **LBMS**。

底层可以集成 OutlookMail Plus / Outlook Email Plus，但这些名称只允许出现在 LBMS 内部 driver 配置、日志调试字段和开发者文档中。LightBridge 管理后台、API、账号 Extra 字段、环境变量、用户帮助文案、按钮和错误提示中，都不应出现 `Outlook Mail Pulse`、`OutlookMail Plus` 或其他底层服务品牌名。

统一命名规则：

| 场景 | 推荐命名 |
|---|---|
| 产品名称 | LightBridge Mail Service |
| 简称 | LBMS |
| 后端服务名 | `lightbridge-mail-service` |
| systemd 服务 | `LightBridge-mail-service.service` |
| Docker service | `mail-service` |
| API 前缀 | `/mail/v1` |
| OAuth 账户 Extra 字段 | `lbms_link` |
| 邮箱引用 URI | `lbms://mailbox/{mailbox_id}` |
| 底层驱动字段 | `driver: outlook_email_plus`，仅内部可见 |

---

## 1. 已确认的 LightBridge 现状约束

### 1.1 主体项目定位

LightBridge 是一个 AI API Gateway Platform，支持 systemd-friendly binary deployment、安装、升级、回滚和从 Sub2API 迁移。

### 1.2 部署形态

当前部署文件已经支持 Docker Compose 与二进制 systemd 两种路径。Docker Compose 以 LightBridge、PostgreSQL、Redis 为核心服务，并通过环境变量完成配置。LBMS 必须遵守这个模式：默认安装 LightBridge 时不附带邮件服务，只在用户主动启用时额外部署。

### 1.3 数据库和迁移约束

LightBridge 使用 PostgreSQL，迁移文件按顺序执行，并通过 `schema_migrations` 记录文件名和 checksum。已经应用过的 migration 不应修改，只能新增前向迁移。LBMS 第一阶段不应改动 LightBridge 主数据库 schema，只使用现有 `accounts.extra` 放一个极简链接。

### 1.4 Account schema 约束

LightBridge 的 `accounts` 表已有：

- `platform`：例如 `openai`、`gemini`、`anthropic`。
- `type`：例如 `oauth`、`api_key`、`cookie`。
- `credentials`：JSONB，用于凭证。
- `extra`：JSONB，用于平台扩展信息。

LBMS 只能在 `extra` 中写入非常小的引用，例如：

```json
{
  "lbms_link": "lbms://mailbox/mbx_01JABCDEF1234567890"
}
```

禁止在 LightBridge 主体 `extra` 里存放完整邮箱资料、底层 provider 配置、最近验证码、邮件内容、IMAP 参数、底层账号 ID、底层 API Key、同步状态详情等。

---

## 2. 总体架构

### 2.1 第一阶段推荐架构：外置 sidecar

```text
LightBridge 主服务
  ├─ OAuth 账户管理
  ├─ 原有 API Key / Admin JWT / 用户与分组体系
  ├─ 模块 UI 入口
  └─ accounts.extra.lbms_link 只保存极简邮箱链接

LightBridge Mail Service sidecar
  ├─ 自己的数据库
  ├─ 邮箱实体
  ├─ 邮箱池
  ├─ OAuth 账户绑定关系
  ├─ 验证码/验证链接获取
  ├─ 反向索引：邮箱 -> 多个 OAuth 账户
  ├─ LightBridge API Key 校验适配层
  └─ 底层 driver：outlook_email_plus

底层邮件服务
  └─ OutlookMail Plus / Outlook Email Plus
```

### 2.2 为什么不直接改内核

当前目标是轻量外接，不改变内核。LBMS 第一阶段不能把邮件系统直接写进 LightBridge 主进程，也不能把底层邮件服务的表强塞到 LightBridge 主数据库。

第一阶段只做：

1. 一个独立运行的 LBMS sidecar。
2. 一个 LightBridge 前端模块或管理 UI 入口。
3. OAuth Account `extra.lbms_link` 的极简引用。
4. 可选反向代理，让用户通过同域 `/mail/v1` 访问 LBMS。

第二阶段如确实需要原生后端模块路由，再单独设计 LightBridge 内核能力：`backend.http.route` 或 `extension.route`，不能把这个改动混在第一阶段。

---

## 3. 核心产品原则

### 3.1 品牌统一

用户看到的永远是 LightBridge Mail Service。所有表单字段、设置页、API 文档、错误提示、日志中的普通级别信息，都用 LBMS 术语。

示例：

```text
正确：LightBridge Mail Service 连接失败
错误：OutlookMail Plus 连接失败

正确：邮箱服务 API Key 无效
错误：Outlook Email Plus X-API-Key invalid
```

底层 driver 错误只允许在 debug 日志中出现，并且要脱敏。

### 3.2 主体数据库轻量

LightBridge 主库只保存：

```json
{
  "lbms_link": "lbms://mailbox/mbx_xxx"
}
```

所有详细信息都归 LBMS 自己管理。

### 3.3 双向连接

必须支持：

- 从 OAuth 账号找到邮箱。
- 从邮箱找到所有绑定的 OAuth 账号。

### 3.4 一个邮箱绑定多个 OAuth

一个邮箱可以绑定多个 OAuth 账户，例如：

```text
aa@qq.com
  ├─ OpenAI OAuth #101
  ├─ OpenAI OAuth #102
  ├─ Gemini OAuth #205
  └─ Anthropic OAuth #311
```

第一阶段规则：

- 一个 OAuth 账户最多绑定一个邮箱。
- 一个邮箱可以绑定多个 OAuth 账户。
- 绑定关系由 LBMS 数据库维护。

---

## 4. 数据模型设计

### 4.1 LightBridge 主体账号 Extra

OAuth 账户 extra 只保存：

```json
{
  "lbms_link": "lbms://mailbox/mbx_01JABCDEF1234567890"
}
```

可选保留版本：

```json
{
  "lbms_link": "lbms://mailbox/mbx_01JABCDEF1234567890",
  "lbms_link_version": 1
}
```

不建议第一版添加更多字段。

### 4.2 LBMS 自有数据库表

LBMS 可以使用 SQLite 或 PostgreSQL。为长期稳定，推荐：

- Docker Compose 场景：优先 PostgreSQL，可复用 LightBridge PostgreSQL 但使用独立 schema 或独立 database。
- 二进制单机场景：允许 SQLite，便于轻量部署。
- 生产建议：PostgreSQL。

#### `lbms_mailboxes`

```sql
CREATE TABLE lbms_mailboxes (
  id                    TEXT PRIMARY KEY,
  email_address          TEXT NOT NULL,
  normalized_email       TEXT NOT NULL,
  display_name           TEXT,
  status                 TEXT NOT NULL DEFAULT 'active',
  source_driver          TEXT NOT NULL,
  source_mailbox_ref     TEXT,
  source_project_key     TEXT,
  tags                   JSONB NOT NULL DEFAULT '[]',
  metadata               JSONB NOT NULL DEFAULT '{}',
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at             TIMESTAMPTZ
);

CREATE UNIQUE INDEX lbms_mailboxes_normalized_email_active_unique
  ON lbms_mailboxes(normalized_email)
  WHERE deleted_at IS NULL;
```

#### `lbms_oauth_bindings`

```sql
CREATE TABLE lbms_oauth_bindings (
  id                         TEXT PRIMARY KEY,
  mailbox_id                 TEXT NOT NULL REFERENCES lbms_mailboxes(id),
  lightbridge_account_id      BIGINT NOT NULL,
  lightbridge_platform        TEXT NOT NULL,
  lightbridge_account_type    TEXT NOT NULL,
  lightbridge_account_name    TEXT,
  status                     TEXT NOT NULL DEFAULT 'active',
  created_by                 BIGINT,
  created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at                 TIMESTAMPTZ
);

CREATE UNIQUE INDEX lbms_oauth_bindings_account_active_unique
  ON lbms_oauth_bindings(lightbridge_account_id)
  WHERE deleted_at IS NULL;

CREATE INDEX lbms_oauth_bindings_mailbox_id_idx
  ON lbms_oauth_bindings(mailbox_id)
  WHERE deleted_at IS NULL;
```

含义：

- `lightbridge_account_id` 在 active binding 中唯一。
- `mailbox_id` 不唯一，所以一个邮箱可绑定多个 OAuth。

#### `lbms_mail_events`

用于审计邮件读取、验证码获取、验证链接获取。

```sql
CREATE TABLE lbms_mail_events (
  id                    TEXT PRIMARY KEY,
  mailbox_id             TEXT NOT NULL,
  lightbridge_account_id  BIGINT,
  event_type             TEXT NOT NULL,
  request_id             TEXT,
  actor_type             TEXT NOT NULL,
  actor_id               TEXT,
  success                BOOLEAN NOT NULL,
  error_code             TEXT,
  latency_ms             INT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX lbms_mail_events_mailbox_created_idx
  ON lbms_mail_events(mailbox_id, created_at DESC);
```

#### `lbms_driver_accounts`

用于隐藏底层邮件服务细节。

```sql
CREATE TABLE lbms_driver_accounts (
  id                    TEXT PRIMARY KEY,
  driver                TEXT NOT NULL,
  base_url              TEXT NOT NULL,
  encrypted_api_key      TEXT NOT NULL,
  status                TEXT NOT NULL DEFAULT 'active',
  health_status          TEXT,
  last_health_checked_at TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 5. API 设计

### 5.1 统一鉴权头

LBMS 对外支持：

```http
Authorization: Bearer <LightBridge API Key or LBMS Admin Token>
X-API-Key: <LightBridge API Key or LBMS Admin Token>
```

第一阶段可以先支持 LBMS 自己的 `LBMS_API_KEY`。第二阶段支持 LightBridge API Key 透传校验。

### 5.2 健康检查

```http
GET /mail/v1/health
```

返回：

```json
{
  "success": true,
  "data": {
    "service": "LightBridge Mail Service",
    "status": "ok",
    "driver_status": "ok",
    "version": "0.1.0"
  }
}
```

### 5.3 邮箱列表

```http
GET /mail/v1/mailboxes?keyword=aa@qq.com&status=active&page=1&page_size=20
```

### 5.4 创建或关联邮箱

```http
POST /mail/v1/mailboxes/link-or-create
Content-Type: application/json

{
  "email_address": "aa@qq.com",
  "lightbridge_account_id": 101,
  "lightbridge_platform": "openai",
  "lightbridge_account_type": "oauth"
}
```

返回：

```json
{
  "success": true,
  "data": {
    "mailbox_id": "mbx_01JABCDEF1234567890",
    "lbms_link": "lbms://mailbox/mbx_01JABCDEF1234567890",
    "email_address": "aa@qq.com",
    "binding_id": "bind_01J..."
  }
}
```

### 5.5 OAuth 账户获取验证码

```http
GET /mail/v1/accounts/{account_id}/verification-code?since_minutes=10&code_length=6
Authorization: Bearer <token>
```

返回：

```json
{
  "success": true,
  "data": {
    "mailbox_id": "mbx_01JABCDEF1234567890",
    "email_address": "aa@qq.com",
    "code": "123456",
    "received_at": "2026-06-12T10:20:30Z",
    "confidence": "high"
  }
}
```

### 5.6 邮箱获取验证码

```http
GET /mail/v1/mailboxes/{mailbox_id}/verification-code?since_minutes=10
```

### 5.7 查看邮箱绑定的 OAuth 账户

```http
GET /mail/v1/mailboxes/{mailbox_id}/bindings
```

返回：

```json
{
  "success": true,
  "data": {
    "mailbox_id": "mbx_01JABCDEF1234567890",
    "email_address": "aa@qq.com",
    "bindings": [
      {
        "lightbridge_account_id": 101,
        "platform": "openai",
        "type": "oauth",
        "name": "OpenAI Account A",
        "status": "active"
      },
      {
        "lightbridge_account_id": 102,
        "platform": "gemini",
        "type": "oauth",
        "name": "Gemini Account B",
        "status": "active"
      }
    ]
  }
}
```

### 5.8 解绑

```http
DELETE /mail/v1/accounts/{account_id}/mailbox-link
```

解绑时：

1. 删除或软删除 LBMS binding。
2. LightBridge account extra 中的 `lbms_link` 由前端或后端适配器清空。
3. 不删除 mailbox 本身。
4. 如果 mailbox 没有任何 active binding，则状态变为 `available` 或继续 `active`，由策略决定。

---

## 6. UI 设计：精确到页面

### 6.1 管理后台一级菜单

新增菜单：

```text
管理后台
  └─ LightBridge Mail Service
```

菜单显示规则：

- 管理员可见。
- 普通用户默认不可见。
- 如果未来支持用户自管邮箱，则普通用户只看到“我的邮箱服务”。

### 6.2 LightBridge Mail Service 首页

路径建议：

```text
/admin/mail-service
```

页面布局：

```text
[标题] LightBridge Mail Service
[副标题] 统一管理 OAuth 账户关联邮箱、邮箱池、验证码和验证链接。

[状态卡片区]
  - 服务状态：正常 / 异常 / 未配置
  - Driver 状态：正常 / 异常 / 未连接
  - 邮箱总数
  - 已绑定 OAuth 数
  - 最近 24h 验证码读取次数
  - 最近错误数

[主要操作]
  - 测试连接
  - 新增邮箱
  - 导入邮箱池
  - 查看审计日志
  - 打开设置
```

### 6.3 设置页

路径：

```text
/admin/mail-service/settings
```

分区：

#### A. 基础信息

字段：

| 字段 | UI 类型 | 说明 |
|---|---|---|
| 服务名称 | 只读文本 | LightBridge Mail Service |
| 服务地址 | 输入框 | LBMS sidecar 地址，例如 `http://127.0.0.1:8091` |
| 公开 API 前缀 | 只读 | `/mail/v1` |
| 启用状态 | Switch | 开启/关闭 LBMS 集成 |

#### B. Driver 配置

用户可见文案仍写“邮件服务驱动”，不要写 Outlook。

| 字段 | UI 类型 | 说明 |
|---|---|---|
| 驱动类型 | Select | `默认邮件驱动`，高级模式才显示内部值 |
| Driver Base URL | 输入框 | 只在高级模式显示 |
| Driver API Key | 密码框 | 保存后脱敏 |
| 连接超时 | 数字输入 | 默认 10s |
| 请求重试次数 | 数字输入 | 默认 2 |

按钮：

```text
[测试连接]
[保存设置]
[重置为默认]
```

测试连接结果：

```text
成功：LightBridge Mail Service 已连接，邮件驱动可用。
失败：LightBridge Mail Service 暂不可用，请检查服务地址、密钥和网络。
```

#### C. 安全策略

字段：

| 字段 | UI 类型 | 默认值 |
|---|---|---|
| 允许通过 LightBridge API Key 访问 | Switch | 关闭，第二阶段开启 |
| 允许管理员 JWT 访问 | Switch | 开启 |
| 允许普通用户读取自己的绑定邮箱 | Switch | 关闭 |
| 验证码结果缓存秒数 | 数字输入 | 30 |
| 单邮箱每分钟读取限制 | 数字输入 | 10 |
| 单 API Key 每分钟读取限制 | 数字输入 | 60 |
| 邮件内容是否允许显示全文 | Switch | 关闭 |

#### D. 数据保留

字段：

| 字段 | UI 类型 | 默认值 |
|---|---|---|
| 邮件事件日志保留天数 | 数字输入 | 30 |
| 错误日志保留天数 | 数字输入 | 90 |
| 验证码结果是否落库 | Switch | 关闭 |
| 邮件正文是否落库 | Switch | 关闭 |

### 6.4 邮箱池页面

路径：

```text
/admin/mail-service/mailboxes
```

表格列：

| 列 | 说明 |
|---|---|
| 邮箱地址 | `aa@qq.com` |
| 状态 | active / available / disabled / error |
| 绑定 OAuth 数 | 支持点击进入绑定列表 |
| 最近邮件时间 | 最近读取到的邮件时间 |
| 最近验证码时间 | 最近成功提取验证码时间 |
| 标签 | 项目、用途、平台 |
| 操作 | 查看 / 绑定 OAuth / 获取验证码 / 禁用 / 删除 |

顶部筛选：

```text
[关键词输入框：邮箱地址 / OAuth 名称]
[状态 Select]
[平台 Select：全部 / OpenAI / Gemini / Anthropic / 其他]
[是否已绑定 Select]
[搜索]
[重置]
```

批量操作：

```text
[批量导入]
[批量禁用]
[批量打标签]
[导出邮箱列表]
```

### 6.5 邮箱详情页 / 抽屉

点击邮箱地址打开右侧抽屉：

```text
邮箱详情：aa@qq.com

[基础信息]
  邮箱 ID: mbx_xxx
  状态: active
  创建时间
  更新时间
  Driver 状态：正常

[绑定的 OAuth 账户]
  表格：平台 / 账号名称 / Account ID / 状态 / 操作

[最近邮件]
  列表：主题 / 发件人 / 收件时间 / 是否包含验证码 / 操作

[验证码]
  [获取最新验证码]
  [等待新验证码]
  [复制验证码]

[危险操作]
  [禁用邮箱]
  [解绑所有 OAuth]
  [删除邮箱]
```

### 6.6 OAuth 账户创建/编辑表单扩展

在 OAuth 类型账户表单中增加区块：

```text
LightBridge Mail Service

[ ] 关联邮箱服务

邮箱地址： [aa@qq.com                    ]
绑定方式： [查找或创建邮箱 v]
同步策略： [创建 OAuth 账户后建立双向绑定 v]

[测试读取] [选择已有邮箱]
```

绑定方式选项：

1. 查找或创建邮箱。
2. 只查找已有邮箱。
3. 从邮箱池领取一个邮箱。
4. 暂不绑定，仅保存 OAuth 账户。

保存流程：

1. 用户点击保存 OAuth 账户。
2. 前端先创建或更新 LightBridge OAuth account。
3. 前端调用 LBMS `link-or-create`。
4. LBMS 返回 `lbms_link`。
5. 前端更新 LightBridge account extra，只写入 `lbms_link`。
6. LBMS 写入反向 binding。

失败处理：

- 如果 OAuth 账户保存成功、LBMS 绑定失败：页面提示“OAuth 账户已保存，但邮箱绑定失败”，并提供“重试绑定”按钮。
- 不应回滚 OAuth 账户创建，避免用户丢失凭证。

### 6.7 OAuth 账户详情页按钮

在 OAuth 账号详情页增加卡片：

```text
LightBridge Mail Service

邮箱：aa@qq.com
绑定状态：已绑定
绑定数量提示：该邮箱还绑定了 3 个 OAuth 账户

[获取验证码]
[等待新邮件]
[查看最近邮件]
[复制验证链接]
[更换邮箱]
[解绑]
```

如果未绑定：

```text
LightBridge Mail Service

当前 OAuth 账户未绑定邮箱。
[绑定邮箱]
[从邮箱池领取]
```

### 6.8 获取验证码交互

点击“获取验证码”：

```text
Modal: 获取验证码

邮箱：aa@qq.com
时间范围：[最近 10 分钟 v]
验证码长度：[自动 v]
来源：[邮件标题 + 正文 v]

[获取]
```

成功：

```text
验证码：123456
[复制验证码]
[查看来源邮件]
```

失败：

```text
未找到验证码。
建议：
1. 点击“等待新邮件”持续监听。
2. 检查邮箱是否收到邮件。
3. 扩大时间范围。
```

### 6.9 等待新邮件交互

点击“等待新邮件”：

```text
Modal: 等待新邮件

等待时间：[60 秒]
匹配内容：[验证码 / 验证链接 / 任意新邮件]

状态：正在等待新邮件...
进度条：0-60 秒
[取消]
```

成功后自动显示验证码或验证链接。

---

## 7. 部署计划

### 7.1 Docker Compose 可选部署

新增文件：

```text
deploy/docker-compose.mail-service.yml
```

示例：

```yaml
services:
  mail-service:
    image: weishaw/lightbridge-mail-service:latest
    container_name: LightBridge-mail-service
    restart: unless-stopped
    ports:
      - "${LBMS_BIND_HOST:-127.0.0.1}:${LBMS_PORT:-8091}:8091"
    environment:
      - LBMS_HOST=0.0.0.0
      - LBMS_PORT=8091
      - LBMS_DATABASE_URL=${LBMS_DATABASE_URL:-sqlite:///data/lbms.db}
      - LBMS_PUBLIC_NAME=LightBridge Mail Service
      - LBMS_DRIVER=outlook_email_plus
      - LBMS_DRIVER_BASE_URL=${LBMS_DRIVER_BASE_URL:-http://outlook-mail-plus:5000}
      - LBMS_DRIVER_API_KEY=${LBMS_DRIVER_API_KEY:?LBMS_DRIVER_API_KEY is required}
      - LBMS_API_KEY=${LBMS_API_KEY:?LBMS_API_KEY is required}
      - LIGHTBRIDGE_BASE_URL=${LIGHTBRIDGE_BASE_URL:-http://LightBridge:8080}
    volumes:
      - ./mail_service_data:/data
    depends_on:
      - LightBridge
```

启动方式：

```bash
cd deploy
docker compose -f docker-compose.local.yml -f docker-compose.mail-service.yml up -d
```

### 7.2 二进制部署

新增文件：

```text
deploy/install-mail-service.sh
deploy/LightBridge-mail-service.service
```

安装脚本风格对齐现有 datamanagementd：

```bash
sudo ./install-mail-service.sh --binary ./lightbridge-mail-service
```

systemd 服务：

```ini
[Unit]
Description=LightBridge Mail Service
After=network.target LightBridge.service
Wants=network.target

[Service]
Type=simple
User=LightBridge
Group=LightBridge
WorkingDirectory=/opt/LightBridge
EnvironmentFile=-/etc/LightBridge/mail-service.env
ExecStart=/opt/LightBridge/lightbridge-mail-service
Restart=always
RestartSec=5s
LimitNOFILE=100000
NoNewPrivileges=true
PrivateTmp=true
ReadWritePaths=/var/lib/LightBridge/mail-service

[Install]
WantedBy=multi-user.target
```

---

## 8. 稳定性和长期运行策略

### 8.1 进程稳定性

LBMS 必须具备：

- systemd `Restart=always`。
- Docker `restart: unless-stopped`。
- 健康检查接口。
- Driver 健康检查。
- 数据库连接池。
- 请求超时。
- 熔断和降级。

### 8.2 请求超时建议

| 操作 | 默认超时 | 最大超时 |
|---|---:|---:|
| 健康检查 | 3s | 5s |
| 最新邮件 | 10s | 30s |
| 获取验证码 | 10s | 30s |
| 等待新邮件 | 60s | 120s |
| 邮箱池领取 | 10s | 30s |

### 8.3 重试策略

- GET 类读取：最多重试 2 次。
- POST 绑定类：只在幂等键存在时重试。
- 邮箱池领取：必须有 idempotency key，避免重复领取。
- 删除/解绑：使用软删除，失败可重试。

### 8.4 幂等键

所有写操作支持：

```http
Idempotency-Key: lbms_{uuid}
```

### 8.5 审计日志

记录：

- 谁请求了验证码。
- 通过哪个 API Key 请求。
- 请求哪个 mailbox。
- 是否关联 account_id。
- 成功/失败。
- 延迟。
- 错误码。

不记录：

- 完整邮件正文。
- 完整验证码结果长期存储。
- 底层 driver API Key。

### 8.6 数据清理任务

后台定时任务：

| 任务 | 频率 |
|---|---|
| 清理过期事件日志 | 每天 03:00 |
| 检查 driver 健康 | 每 1 分钟 |
| 同步邮箱状态 | 每 10 分钟 |
| 清理孤立 binding | 每天 04:00 |

---

## 9. 实施阶段

### Phase 0：文档和边界确认

交付：

- 本文档。
- 明确品牌命名。
- 明确 Extra 只存 `lbms_link`。
- 明确邮箱与 OAuth 账户为一对多。
- 明确 sidecar first，不改内核。

验收：

- 所有对外文案均为 LightBridge Mail Service。
- 不出现 Outlook Mail Pulse 字段命名。
- 方案中没有把邮箱细节写入 LightBridge 主体数据库。

### Phase 1：LBMS sidecar 最小可用

目录建议：

```text
mailservice/
  cmd/lightbridge-mail-service/main.go
  internal/config/
  internal/http/
  internal/store/
  internal/driver/
  internal/driver/outlookemailplus/
  internal/binding/
  internal/audit/
```

实现 API：

- `GET /mail/v1/health`
- `POST /mail/v1/mailboxes/link-or-create`
- `GET /mail/v1/accounts/{account_id}/verification-code`
- `GET /mail/v1/mailboxes/{mailbox_id}/bindings`
- `DELETE /mail/v1/accounts/{account_id}/mailbox-link`

验收：

- 可以启动。
- 可以连接底层 driver。
- 可以创建 mailbox。
- 可以建立 OAuth binding。
- 可以通过 account_id 获取验证码。

### Phase 2：LightBridge 管理 UI

实现页面：

- `/admin/mail-service`
- `/admin/mail-service/settings`
- `/admin/mail-service/mailboxes`
- 邮箱详情抽屉。

验收：

- 管理员可以配置服务地址和 API Key。
- 可以测试连接。
- 可以看到邮箱池。
- 可以看到邮箱绑定的 OAuth 账户。

### Phase 3：OAuth 账户表单集成

实现：

- OAuth 新增/编辑页中的 LBMS 区块。
- 保存时自动创建/绑定邮箱。
- `extra.lbms_link` 写入 LightBridge 账号。
- 绑定失败的重试按钮。

验收：

- 创建 OpenAI OAuth 时可以绑定邮箱。
- 创建 Gemini OAuth 时可以绑定同一个邮箱。
- 一个邮箱可显示多个 OAuth binding。
- LightBridge account extra 仍只有 `lbms_link`。

### Phase 4：统一 API Key 鉴权

实现：

- LBMS 接收 LightBridge API Key。
- 调用 LightBridge 或只读数据库验证 API Key。
- 校验用户、分组、account 访问权限。
- 记录审计日志。

验收：

- 用户可用 LightBridge API Key 请求 `/mail/v1/accounts/{id}/verification-code`。
- 无权限 account 返回 403。
- 过期/禁用 API Key 返回 401。

### Phase 5：邮箱池自动化

实现：

- 从邮箱池领取邮箱。
- 创建 OAuth 后 complete。
- 失败后 release。
- 支持 project_key。

验收：

- 能从池中领取邮箱。
- 同一个邮箱可被多个 OAuth 绑定，前提是策略允许。
- 失败流程不会造成邮箱永久锁死。

### Phase 6：生产强化

实现：

- Prometheus metrics。
- 日志脱敏。
- 配置热加载。
- 数据备份文档。
- 迁移脚本。
- 灾难恢复文档。

验收：

- 服务重启后绑定关系不丢失。
- driver 临时不可用时 UI 明确提示，不影响 LightBridge 主服务。
- 邮件服务停用时 OAuth 账户仍可正常用于网关调度。

---

## 10. 回滚策略

### 10.1 UI 回滚

禁用模块或隐藏菜单，不影响主服务。

### 10.2 Sidecar 回滚

停止服务：

```bash
sudo systemctl stop LightBridge-mail-service
```

Docker：

```bash
docker compose -f docker-compose.local.yml -f docker-compose.mail-service.yml stop mail-service
```

### 10.3 数据回滚

由于 LightBridge 主体只写入 `extra.lbms_link`，回滚只需：

- 保留 extra，不影响账号调度。
- 或批量清理 `extra.lbms_link`。

### 10.4 不可接受的回滚方式

禁止为了回滚 LBMS 去删除 OAuth 账户、清空 credentials 或修改调度字段。

---

## 11. 测试计划

### 11.1 单元测试

- 邮箱标准化。
- `lbms://mailbox/{id}` 解析。
- 一个邮箱多 binding。
- 一个 account 只能有一个 active binding。
- driver 错误脱敏。

### 11.2 集成测试

- LBMS -> driver health。
- link-or-create。
- 获取验证码。
- 等待新邮件。
- 解绑。

### 11.3 UI 测试

- 设置页保存。
- 测试连接。
- 邮箱池筛选。
- OAuth 表单绑定邮箱。
- 同邮箱绑定多个 OAuth 的显示。
- 获取验证码 modal。

### 11.4 稳定性测试

- driver 不可用 10 分钟后恢复。
- LBMS 重启。
- LightBridge 重启。
- 数据库断开后恢复。
- 同时 100 个验证码请求。

---

## 12. 关键验收清单

- [ ] 用户可见品牌只有 LightBridge Mail Service / LBMS。
- [ ] LightBridge 主体账号 `extra` 只存 `lbms_link`。
- [ ] LBMS 自己维护 mailbox 和 OAuth binding。
- [ ] 一个 mailbox 可以绑定多个 OAuth account。
- [ ] 从 OAuth 账户能查邮箱。
- [ ] 从邮箱能查所有 OAuth 账户。
- [ ] 可以在 OAuth 账户详情页获取验证码。
- [ ] 可以在邮箱详情页查看绑定账户。
- [ ] LBMS 停止不会影响 LightBridge 主网关调度。
- [ ] 默认安装 LightBridge 不会安装 LBMS。
- [ ] Docker 和 systemd 都有可选部署路径。
- [ ] 所有敏感信息脱敏。
- [ ] 所有写操作幂等。
- [ ] 有审计日志和数据清理任务。

---

## 13. 最终执行原则

实现时以“长期稳定执行”为第一优先级：

1. 不改 LightBridge 内核，先 sidecar。
2. 不污染主数据库，Extra 只存链接。
3. 不暴露底层邮件服务品牌，统一 LBMS。
4. 不把验证码和邮件正文长期写入主库。
5. 不让邮件服务失败影响 OAuth 账号调度。
6. 不做不可回滚的强耦合设计。
7. 每个阶段都必须有独立验收和回滚路径。
