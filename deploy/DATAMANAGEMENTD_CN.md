# datamanagementd 部署说明（数据管理）

本文说明如何在宿主机部署 `datamanagementd`，并与主进程联动开启“数据管理”功能。

## 1. 关键约束

- 主进程固定探测路径：`/tmp/LightBridge-datamanagement.sock`
- 仅当该 Unix Socket 可连通且 `Health` 成功时，后台“数据管理”才会启用
- `datamanagementd` 使用 SQLite 持久化元数据，不依赖主库

## 2. 获取并运行发行版二进制

本仓库不包含 `datamanagementd` 的源码模块。请从与当前 LightBridge 版本匹配的可信发行渠道取得二进制，并先核对版本与校验和，再安装到宿主机：

```bash
/path/to/datamanagementd -version
sha256sum /path/to/datamanagementd

mkdir -p /var/lib/LightBridge/datamanagement
chown -R LightBridge:LightBridge /var/lib/LightBridge/datamanagement
```

手动启动示例：

```bash
/path/to/datamanagementd \
  -socket-path /tmp/LightBridge-datamanagement.sock \
  -sqlite-path /var/lib/LightBridge/datamanagement/datamanagementd.db \
  -version 1.0.0
```

## 3. systemd 托管（推荐）

仓库已提供示例服务文件：`deploy/LightBridge-datamanagementd.service`

```bash
sudo cp deploy/LightBridge-datamanagementd.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now LightBridge-datamanagementd
sudo systemctl status LightBridge-datamanagementd
```

查看日志：

```bash
sudo journalctl -u LightBridge-datamanagementd -f
```

也可以使用一键安装脚本（自动安装二进制 + 注册 systemd）：

```bash
sudo ./deploy/install-datamanagementd.sh --binary /path/to/datamanagementd
```

## 4. Docker 部署联动

若 `LightBridge` 运行在 Docker 容器中，需要将宿主机 Socket 挂载到容器同路径：

```yaml
services:
  LightBridge:
    volumes:
      - /tmp/LightBridge-datamanagement.sock:/tmp/LightBridge-datamanagement.sock
```

建议在 `docker-compose.override.yml` 中维护该挂载，避免覆盖主 compose 文件。

## 5. 依赖检查

`datamanagementd` 执行备份时依赖以下工具：

- `pg_dump`
- `redis-cli`
- `docker`（仅 `source_mode=docker_exec` 时）

缺失依赖会导致对应任务失败，并在任务详情中体现错误信息。
