# TOTP 2FA Login (Admin)

该模块为 LightBridge Admin 提供基于 TOTP（RFC 6238）的二步验证能力。

## 运行方式

该模块由 LightBridge Core 以子进程形式拉起，并通过环境变量注入运行参数：

- `LIGHTBRIDGE_HTTP_PORT`：模块监听端口（仅绑定 127.0.0.1）
- `LIGHTBRIDGE_DATA_DIR`：模块私有数据目录（用于持久化 2FA 设备）

## 端点

- `GET /health`
- `POST /totp/enroll/begin`
- `POST /totp/enroll/confirm`
- `POST /totp/verify`
- `GET /totp/devices?username=...`
- `POST /totp/devices/delete`

> 说明：以上端点供 Core 网关通过 `proxyModuleHTTP` 调用，不建议直接对外暴露。
