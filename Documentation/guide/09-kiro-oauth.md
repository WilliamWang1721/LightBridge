# Guide ⑨：Kiro OAuth Provider（kiro-oauth-provider 模块）

本篇用于在 LightBridge 中接入 `kiro-oauth-provider`，并完成 `kiro` Provider 的 OAuth 配置与调用验证。

---

## 0. 完成后你会得到什么

- 已安装并运行模块：`kiro-oauth-provider`
- 自动注册的 Provider 别名：`kiro`
  - `protocol = http_openai`
  - `endpoint = http://127.0.0.1:<module_http_port>`
- 可用认证路径：
  - Google OAuth（PKCE）
  - AWS Device Code（Builder ID / IAM Identity Center OIDC）
  - 凭据导入（JSON / token）
- 可直接调用：`model@kiro`

---

## 1. 安装与启动模块

1. 打包模块：

```bash
bash modules/kiro-oauth-provider/package.sh
```

2. 将 zip 放入本地 Marketplace 目录（示例）：

```bash
mkdir -p MODULES
cp modules/kiro-oauth-provider/dist/*.zip MODULES/
```

3. 打开 `/admin/marketplace`，索引源选择 `local`，安装并启动 `kiro-oauth-provider`。

4. 用 `/admin/api/modules` 检查运行态，确认 `runtime.http_port > 0`。

---

## 2. Providers 页面进行 OAuth

入口：`/admin/providers`

建议走“添加 Provider → 自动读取模块 Provider”选择 `kiro`，会打开 Kiro OAuth 弹窗。

### 2.1 Google OAuth（推荐）

1. 点击 `Google OAuth`
2. 浏览器授权后，Core 本地回调监听会处理：
   - `127.0.0.1:19876-19880`
   - path: `/oauth/callback`
3. 若自动回调失败，复制完整回调 URL 到弹窗 “回调 URL” 区域，点击 “从回调 URL 获取 Token”。

### 2.2 AWS Device Code

1. 点击 `AWS Device Code`
2. 弹窗显示 `device_url` + `user_code`
3. 浏览器完成设备授权后回到弹窗刷新状态。

### 2.3 导入凭据

- 可直接填 `refresh_token`/`access_token` 导入
- 或批量导入 JSON 凭据文件

---

## 3. 保存 Provider

当状态中至少有一个可用账号后，点击 `保存`。

保存逻辑会确保：

- Provider ID 固定为 `kiro`
- Type 固定为 `module`
- Protocol 固定为 `http_openai`
- Endpoint 自动对齐模块 runtime HTTP 端口

---

## 4. 验证调用

### 4.1 非流式

```bash
curl -s "http://127.0.0.1:3210/v1/chat/completions" \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"claude-sonnet-4-5@kiro",
    "messages":[{"role":"user","content":"hello"}],
    "stream":false
  }'
```

### 4.2 流式

```bash
curl -N "http://127.0.0.1:3210/v1/chat/completions" \
  -H "Authorization: Bearer <CLIENT_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"claude-sonnet-4-5@kiro",
    "messages":[{"role":"user","content":"write a quick summary"}],
    "stream":true
  }'
```

---

## 5. 用量与配额

- 请求级 usage：响应里会返回 `usage.prompt_tokens/completion_tokens/total_tokens`（估算口径）
- 配额级 usage：弹窗“配额标准化”来自 `/admin/api/kiro/usage/limits`
  - `quota.items[]`
  - `used_percent` / `remaining_percent`
  - `reset_at`

---

## 6. Failover 语义

当 Kiro 账号池不可用或配额耗尽时，模块返回：

- HTTP `503`
- OpenAI error code: `insufficient_quota`

这样对于“非 `model@kiro` 强制变体”的路由，Core 可触发 5xx failover 重选 Provider。

---

## 7. 常见问题

1. 生成 OAuth 链接失败：先确认 `kiro-oauth-provider` 已启动且 `runtime.http_port` 有值。
2. 回调失败：手动粘贴回调 URL 到弹窗并执行 exchange。
3. 一直 401：使用弹窗 `刷新状态` + `导入` 或 `刷新`，确认账号存在 `refresh_token`。
4. 402/503：说明账号配额不足，切换其他账号或等待 reset。

