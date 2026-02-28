# Guide ⑦：数据目录与安全注意事项（Security）

本篇聚焦“真实部署时的安全边界”：哪些东西存在哪里、是否明文、默认监听范围、以及你应该怎么做最基本的防护。

---

## 1. 默认监听是“仅本机”（安全默认）

默认：

- `LIGHTBRIDGE_ADDR=127.0.0.1:3210`

这意味着：

- 只有本机能访问 `/admin` 与 `/v1/*`
- 适合本地开发/单机使用

如果你改成 `0.0.0.0:3210` 让局域网可访问：

- 等同于把管理后台和网关暴露给网络
- 你必须额外做防护（至少：防火墙、反向代理、TLS、访问控制）

---

## 2. 数据目录里有什么敏感信息？

数据目录（`<DATA_DIR>`）包含：

- `lightbridge.db`（SQLite）
  - 管理员账号（密码为 hash，不是明文）
  - Client API Keys（**明文存储**）
  - Providers 配置（`config_json` 里常包含 `api_key`，**明文存储**）
  - 模块安装状态、运行时端口等
- `modules/`（模块解压后的文件）
- `module_data/`（模块配置与模块私有数据）
  - 例如 `openai-codex-oauth` 会在此写入 `credentials.json`（包含 OAuth token）

结论：

- **把 `<DATA_DIR>` 当作“包含密钥的敏感目录”对待**

---

## 3. 文件权限与主机隔离建议

Core 启动时会尽量保证：

- `<DATA_DIR>` 权限为 `0700`（仅当前用户可读写执行）

建议你进一步做到：

- 不要把 `<DATA_DIR>` 放到多人共享/同步盘
- 若部署在服务器上，使用专用系统用户运行 LightBridge
- 备份数据库时，确保备份文件同样受控（备份文件里也有明文 key）

---

## 4. Admin Cookie Secret（会话安全）

管理后台会用 Cookie 维持会话。

- 若你未设置 `LIGHTBRIDGE_COOKIE_SECRET`，Core 会每次启动随机生成 secret
  - 优点：重启会让旧 Cookie 全失效（更安全）
  - 缺点：你每次重启都要重新登录

若你希望会话跨重启保持：

- 设置 `LIGHTBRIDGE_COOKIE_SECRET` 为一个长随机字符串

---

## 5. HTTP / HTTPS 与反向代理

Core 自身默认提供 HTTP 服务，不内置 TLS。

若要公网/跨机器访问，建议：

1. Core 仍监听在内网（或 localhost）
2. 用 Nginx/Caddy/Traefik 做反向代理并启用 HTTPS
3. 对 `/admin/*` 加额外认证（如 Basic Auth / SSO / IP 白名单）
4. 对 `/v1/*` 使用 Client API Key，并按需求配合限流与监控

---

## 6. “不记录 prompt/response”并不等于“完全无风险”

Core 当前只记录请求元数据，不记录完整 prompt/response body。

但你仍需注意：

- 上游 provider 的 endpoint、api_key、token 仍在本地存储（明文）
- 模块可能会记录自己的日志或缓存（取决于模块实现）

---

## 下一步

- 查看数据目录结构与各文件含义：[数据目录结构](../reference/05-data-dir-layout.md)
- 常见安全相关故障排查（比如“被暴露导致被刷”）：[故障排查](./08-troubleshooting.md)

