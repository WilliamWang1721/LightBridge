# Kiro OAuth Provider Module (LightBridge) — v0.1.0

`kiro-oauth-provider` 为 LightBridge 提供 `http_openai` 协议的 Kiro Provider，支持：

- Google OAuth（PKCE）
- AWS Builder ID / IAM Identity Center（Device Code）
- 凭据导入（JSON / token 字段）
- 多账号池（启用/禁用/删除/激活）
- Kiro 配额查询与标准化
- Chat Completions stream / non-stream（含最小 tools）
- 请求级 usage 估算（`usage.prompt_tokens/completion_tokens/total_tokens`）

## HTTP Endpoints

- `GET /health`
- `GET /v1/models`
- `POST /v1/chat/completions`
- `GET /auth/status`
- `POST /auth/oauth/start`
- `POST /auth/oauth/exchange`
- `POST /auth/device/start`
- `POST /auth/import`
- `POST /auth/refresh`
- `POST /auth/accounts/enable`
- `POST /auth/accounts/disable`
- `POST /auth/accounts/delete`
- `POST /auth/accounts/activate`
- `GET /usage/limits`

## 存储

凭据和账号池保存在：

- `LIGHTBRIDGE_DATA_DIR/accounts.json`

写入方式为原子写（`*.tmp + rename`）。

## 配置

配置文件由 Core 写入到 `LIGHTBRIDGE_CONFIG_PATH`。

关键字段见 `manifest.json`：

- `base_url` / `amazonq_base_url`
- `auth_service_endpoint`
- `aws_oidc_endpoint`
- `social_refresh_url` / `idc_refresh_url`
- `selection_strategy`（`fill_first` / `round_robin`）
- `models`

## 复用声明（按 AGENT 要求）

本模块遵循“复用优先”，主要复用来源如下：

1. 本仓库 `modules/openai-codex-oauth`：
   - 模块骨架（main/config/util 路由组织与错误输出风格）
   - 原子写保存模式（tmp + rename）
2. 本仓库 `modules/chatbox-persistent`：
   - 文件 store 与并发读写风格（`RWMutex + snapshot + atomic save`）
3. `参考项目/AIClient-2-API(Extra)`（GPLv3，允许拷贝）：
   - Kiro OAuth 三路径（Google PKCE / AWS Device / refresh）实现思路
   - `buildCodewhispererRequest` 请求结构
   - `parseAwsEventStreamBuffer` 混合流 JSON 提取算法
   - Kiro 配额字段结构与标准化映射
4. `参考项目/CLIProxyAPI(extra)`（MIT）：
   - 可重试/不可重试错误分类思路（本模块做了等价简化）
5. `参考项目/one-api(extra)`（MIT）：
   - 配额语义和 `insufficient_quota` 错误码口径对齐

许可证护栏：

- 已遵循：`参考项目/new-api(extra)`（AGPLv3）仅参考行为，不复制源码。

## 打包

```bash
bash modules/kiro-oauth-provider/package.sh
```

生成：

- `modules/kiro-oauth-provider/dist/*.zip`
- `modules/kiro-oauth-provider/dist/index.json`

