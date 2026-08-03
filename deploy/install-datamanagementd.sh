#!/usr/bin/env bash

set -euo pipefail

# 用法：
#   sudo ./install-datamanagementd.sh --binary /path/to/datamanagementd

BIN_PATH=""
INSTALL_DIR="/opt/LightBridge"
DATA_DIR="/var/lib/LightBridge/datamanagement"
SERVICE_FILE_NAME="LightBridge-datamanagementd.service"

function print_help() {
  cat <<'EOF'
用法:
  install-datamanagementd.sh --binary <datamanagementd二进制路径>

参数:
  --binary  指定已构建的 datamanagementd 二进制路径
  -h, --help 显示帮助

示例:
  sudo ./install-datamanagementd.sh --binary /path/to/datamanagementd
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary)
      BIN_PATH="${2:-}"
      shift 2
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "未知参数: $1"
      print_help
      exit 1
      ;;
  esac
done

if [[ -z "$BIN_PATH" ]]; then
  echo "错误: 必须提供 --binary"
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "错误: 请使用 root 权限执行（例如 sudo）"
  exit 1
fi

if [[ ! -f "$BIN_PATH" ]]; then
  echo "错误: 二进制文件不存在: $BIN_PATH"
  exit 1
fi

if ! id LightBridge >/dev/null 2>&1; then
  echo "[2/6] 创建系统用户 LightBridge..."
  useradd --system --no-create-home --shell /usr/sbin/nologin LightBridge
else
  echo "[2/6] 系统用户 LightBridge 已存在，跳过创建"
fi

echo "[3/6] 安装 datamanagementd 二进制..."
mkdir -p "$INSTALL_DIR"
install -m 0755 "$BIN_PATH" "$INSTALL_DIR/datamanagementd"

echo "[4/6] 准备数据目录..."
mkdir -p "$DATA_DIR"
chown -R LightBridge:LightBridge /var/lib/LightBridge
chmod 0750 "$DATA_DIR"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_TEMPLATE="$SCRIPT_DIR/$SERVICE_FILE_NAME"
if [[ ! -f "$SERVICE_TEMPLATE" ]]; then
  echo "错误: 未找到服务模板 $SERVICE_TEMPLATE"
  exit 1
fi

echo "[5/6] 安装 systemd 服务..."
cp "$SERVICE_TEMPLATE" "/etc/systemd/system/$SERVICE_FILE_NAME"
systemctl daemon-reload
systemctl enable --now LightBridge-datamanagementd

echo "[6/6] 完成，当前状态："
systemctl --no-pager --full status LightBridge-datamanagementd || true

cat <<'EOF'

下一步建议：
1. 查看日志：sudo journalctl -u LightBridge-datamanagementd -f
2. 在 LightBridge（容器部署时）挂载 socket:
   /tmp/LightBridge-datamanagement.sock:/tmp/LightBridge-datamanagement.sock
3. 进入管理后台“数据管理”页面确认 agent=enabled

EOF
