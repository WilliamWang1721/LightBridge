# Passkey Login (Admin)

该模块为 LightBridge Admin 提供 Passkey（WebAuthn）登录能力。

## 运行方式

该模块由 LightBridge Core 以子进程形式拉起，并通过环境变量注入运行参数：

- `LIGHTBRIDGE_HTTP_PORT`：模块监听端口（仅绑定 127.0.0.1）
- `LIGHTBRIDGE_DATA_DIR`：模块私有数据目录（用于持久化 passkey 凭证）

## 端点

- `GET /health`：健康检查（200/ok）
- `POST /passkey/register/begin`
- `POST /passkey/register/finish`
- `POST /passkey/auth/begin`
- `POST /passkey/auth/finish`
- `GET /passkey/credentials?username=...`
- `POST /passkey/credentials/delete`

> 说明：这些端点仅供 Core 通过 `proxyModuleHTTP` 访问，不建议对外暴露。

