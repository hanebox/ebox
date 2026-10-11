#!/usr/bin/env bash
#
# ebox installer for a Linux host with systemd and Docker.
#
#   curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash
#
# Options:
#   --version v0.5.0     install a specific release (default: latest)
#   --binary /path       install a local binary instead of downloading
#   --port 8000          panel port (default: 8000, or the current value)
#   --host 127.0.0.1     panel address (default: 127.0.0.1)
#   --data-dir /var/lib/ebox
#   --no-start           install without starting the service
#   --force              overwrite an existing installation
set -euo pipefail

REPOSITORY="${EBOX_UPDATE_REPOSITORY:-buildandbuild/ebox}"
BINARY_PATH="/usr/local/bin/ebox"
ENV_DIR="/etc/ebox"
ENV_FILE="${ENV_DIR}/ebox.env"
UNIT_PATH="/etc/systemd/system/ebox.service"
DATA_DIR="/var/lib/ebox"
SERVICE_USER="ebox"
VERSION="latest"
PANEL_HOST="127.0.0.1"
PANEL_PORT="8000"
START_SERVICE=1
FORCE=0
LOCAL_BINARY=""

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
fail() {
  printf '\033[1;31merror:\033[0m %s\n' "$*" >&2
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:?}"; shift 2 ;;
    --binary) LOCAL_BINARY="${2:?}"; shift 2 ;;
    --port) PANEL_PORT="${2:?}"; shift 2 ;;
    --host) PANEL_HOST="${2:?}"; shift 2 ;;
    --data-dir) DATA_DIR="${2:?}"; shift 2 ;;
    --no-start) START_SERVICE=0; shift ;;
    --force) FORCE=1; shift ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done

[ "$(id -u)" -eq 0 ] || fail "run this installer as root (sudo)"

require() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required but was not found in PATH"
}

require systemctl
require curl
require tar

if ! command -v docker >/dev/null 2>&1; then
  fail "Docker is required. Install it first: https://docs.docker.com/engine/install/"
fi

if systemctl is-active --quiet docker; then
  log "Docker is running"
else
  warn "the docker service is not active; the panel will fail to start until it is"
fi

if [ -f "$UNIT_PATH" ] && [ "$FORCE" -eq 0 ] && [ "$LOCAL_BINARY" = "" ]; then
  log "An existing installation was found at ${UNIT_PATH}"
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ASSET_ARCH="amd64" ;;
  aarch64|arm64) ASSET_ARCH="arm64" ;;
  *) fail "unsupported architecture: ${ARCH}" ;;
esac

OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
[ "$OS_NAME" = "linux" ] || fail "this installer targets Linux (found ${OS_NAME})"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if [ -n "$LOCAL_BINARY" ]; then
  [ -f "$LOCAL_BINARY" ] || fail "no such file: ${LOCAL_BINARY}"
  log "Using local binary ${LOCAL_BINARY}"
  cp "$LOCAL_BINARY" "${TMP_DIR}/ebox"
else
  if [ "$VERSION" = "latest" ]; then
    log "Resolving the newest release of ${REPOSITORY}"
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPOSITORY}/releases/latest" |
      sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
    [ -n "$VERSION" ] || fail "no release was found for ${REPOSITORY}"
  fi

  ASSET="ebox-${OS_NAME}-${ASSET_ARCH}"
  BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"

  log "Downloading ${ASSET} ${VERSION}"
  curl -fsSL "${BASE_URL}/${ASSET}" -o "${TMP_DIR}/ebox" ||
    fail "the release asset ${ASSET} could not be downloaded from ${BASE_URL}"

  if curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt" 2>/dev/null; then
    log "Verifying the checksum"
    EXPECTED="$(awk -v asset="$ASSET" '{ name=$2; sub(/^\*/, "", name); if (name == asset) print $1 }' "${TMP_DIR}/checksums.txt" | head -1)"
    if [ -n "$EXPECTED" ]; then
      ACTUAL="$(sha256sum "${TMP_DIR}/ebox" | awk '{print $1}')"
      [ "$EXPECTED" = "$ACTUAL" ] || fail "checksum mismatch: expected ${EXPECTED}, got ${ACTUAL}"
    else
      warn "the checksum file has no entry for ${ASSET}"
    fi
  else
    warn "this release has no checksums.txt, skipping verification"
  fi
fi

chmod 0755 "${TMP_DIR}/ebox"

log "Creating the ${SERVICE_USER} system user"
if id "$SERVICE_USER" >/dev/null 2>&1; then
  log "User ${SERVICE_USER} already exists"
else
  useradd --system --home-dir "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER" ||
    useradd --system --home-dir "$DATA_DIR" --shell /sbin/nologin "$SERVICE_USER"
fi

if getent group docker >/dev/null 2>&1; then
  usermod -aG docker "$SERVICE_USER"
else
  warn "the docker group was not found; add ${SERVICE_USER} to the docker group manually"
fi

log "Creating directories"
install -d -m 0755 -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"
install -d -m 0755 "$ENV_DIR"

if [ -f "$ENV_FILE" ]; then
  log "Keeping the existing environment file at ${ENV_FILE}"
  # shellcheck disable=SC1090
  EXISTING_KEY="$(sed -n 's/^APP_KEY=\(.*\)$/\1/p' "$ENV_FILE" | head -1)"
else
  EXISTING_KEY=""
fi

if [ -z "$EXISTING_KEY" ]; then
  if command -v openssl >/dev/null 2>&1; then
    EXISTING_KEY="base64:$(openssl rand -base64 32)"
  else
    EXISTING_KEY="base64:$(head -c 32 /dev/urandom | base64 | tr -d '\n')"
  fi
fi

install -m 0600 /dev/null "$ENV_FILE"
cat >"$ENV_FILE" <<EOF
# Generated by the ebox installer on $(date -u +%Y-%m-%dT%H:%M:%SZ).
# APP_KEY encrypts stored secrets. Back it up: losing it makes stored
# credentials unreadable.
APP_KEY=${EXISTING_KEY}
APP_ENV=production
APP_DEBUG=false

EBOX_DATA_DIR=${DATA_DIR}
EBOX_HOST=${PANEL_HOST}
EBOX_PORT=${PANEL_PORT}
EBOX_DOCKER_NETWORK=ebox
EBOX_TRAEFIK_API_PORT=8080
EBOX_UPDATE_REPOSITORY=${REPOSITORY}
EOF

log "Installing the panel binary to ${BINARY_PATH}"
if systemctl is-active --quiet ebox && [ -f "$BINARY_PATH" ]; then
  cp -f "$BINARY_PATH" "${BINARY_PATH}.previous"
fi
install -m 0755 "${TMP_DIR}/ebox" "$BINARY_PATH"

log "Installing the systemd unit"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "${SCRIPT_DIR}/ebox.service" ]; then
  install -m 0644 "${SCRIPT_DIR}/ebox.service" "$UNIT_PATH"
else
  cat >"$UNIT_PATH" <<'EOF'
[Unit]
Description=ebox personal server control plane
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=ebox
Group=ebox
EnvironmentFile=/etc/ebox/ebox.env
ExecStart=/usr/local/bin/ebox
Restart=always
RestartSec=3
SupplementaryGroups=docker
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/ebox /etc/ebox

[Install]
WantedBy=multi-user.target
EOF
fi

systemctl daemon-reload
systemctl enable ebox >/dev/null

if [ "$START_SERVICE" -eq 1 ]; then
  log "Starting ebox"
  systemctl restart ebox
fi

IP_ADDRESS="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$IP_ADDRESS" ] || IP_ADDRESS="<server ip>"

cat <<EOF

ebox ${VERSION} is installed.

  Open the setup wizard:  http://${PANEL_HOST}:${PANEL_PORT}/setup
  From another machine:   http://${IP_ADDRESS}:${PANEL_PORT}/setup

  Configuration:          ${ENV_FILE}
  Data, backups:          ${DATA_DIR}
  Service control:        systemctl {status,restart,stop} ebox
  Logs:                   journalctl -u ebox -f

The first visit creates the admin account. There are no default credentials.
EOF
