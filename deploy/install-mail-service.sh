#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="LightBridge-mail-service"
INSTALL_DIR="/opt/LightBridge"
DATA_DIR="/var/lib/LightBridge/mail-service"
ENV_DIR="/etc/LightBridge"
ENV_FILE="${ENV_DIR}/mail-service.env"
SYSTEMD_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
BINARY_SOURCE=""

usage() {
  cat <<'USAGE'
Install LightBridge Mail Service as an optional systemd sidecar.

Usage:
  sudo ./install-mail-service.sh --binary ./lightbridge-mail-service

Options:
  --binary PATH   Path to a prebuilt lightbridge-mail-service binary.
  -h, --help      Show help.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary)
      BINARY_SOURCE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "${BINARY_SOURCE}" ]]; then
  echo "--binary is required" >&2
  usage
  exit 1
fi

if [[ ! -f "${BINARY_SOURCE}" ]]; then
  echo "Binary not found: ${BINARY_SOURCE}" >&2
  exit 1
fi

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root or with sudo." >&2
  exit 1
fi

if ! id -u LightBridge >/dev/null 2>&1; then
  useradd --system --home-dir "${INSTALL_DIR}" --shell /usr/sbin/nologin LightBridge
fi

mkdir -p "${INSTALL_DIR}" "${DATA_DIR}" "${ENV_DIR}"
install -m 0755 "${BINARY_SOURCE}" "${INSTALL_DIR}/lightbridge-mail-service"
chown -R LightBridge:LightBridge "${DATA_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
  cat > "${ENV_FILE}" <<'ENV'
LBMS_HOST=0.0.0.0
LBMS_PORT=8091
LBMS_API_KEY=change-me-to-a-long-random-value
LBMS_DATA_PATH=/var/lib/LightBridge/mail-service/lbms-store.json
LBMS_DRIVER=outlook_email_plus
LBMS_DRIVER_BASE_URL=http://127.0.0.1:5000
LBMS_DRIVER_API_KEY=change-me-driver-key
LBMS_REQUEST_TIMEOUT_SECONDS=10
LBMS_VERIFICATION_CACHE_SECONDS=30
LIGHTBRIDGE_BASE_URL=http://127.0.0.1:8080
ENV
  chmod 0600 "${ENV_FILE}"
fi

cat > "${SYSTEMD_FILE}" <<'UNIT'
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
UNIT

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"

echo "Installed ${SERVICE_NAME}."
echo "Edit ${ENV_FILE}, then start with: sudo systemctl start ${SERVICE_NAME}.service"
