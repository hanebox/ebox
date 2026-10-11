#!/usr/bin/env bash
#
# Remove an ebox installation.
#
#   sudo bash installer/uninstall.sh              # keep data in /var/lib/ebox
#   sudo bash installer/uninstall.sh --purge      # remove data too
#
set -euo pipefail

UNIT_PATH="/etc/systemd/system/ebox.service"
BINARY_PATH="/usr/local/bin/ebox"
ENV_DIR="/etc/ebox"
DATA_DIR="/var/lib/ebox"
SERVICE_USER="ebox"
PURGE=0

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
fail() {
  printf '\033[1;31merror:\033[0m %s\n' "$*" >&2
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --purge) PURGE=1; shift ;;
    --data-dir) DATA_DIR="${2:?}"; shift 2 ;;
    -h|--help) sed -n '2,9p' "$0"; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done

[ "$(id -u)" -eq 0 ] || fail "run this uninstaller as root (sudo)"

if systemctl list-unit-files | grep -q '^ebox.service'; then
  log "Stopping and disabling ebox"
  systemctl disable --now ebox || true
else
  log "No ebox systemd unit was found"
fi

for path in "$UNIT_PATH" "$BINARY_PATH" "${BINARY_PATH}.previous"; do
  if [ -e "$path" ]; then
    log "Removing ${path}"
    rm -f "$path"
  fi
done

systemctl daemon-reload

if [ "$PURGE" -eq 1 ]; then
  log "Removing ${DATA_DIR} (database, deployments, backups)"
  rm -rf "$DATA_DIR"

  log "Removing ${ENV_DIR}"
  rm -rf "$ENV_DIR"

  if id "$SERVICE_USER" >/dev/null 2>&1; then
    log "Removing the ${SERVICE_USER} user"
    userdel "$SERVICE_USER" 2>/dev/null || true
  fi

  log "Removing ebox containers, images and volumes"
  docker ps -aq --filter "label=ebox.managed=true" | xargs -r docker rm -f >/dev/null 2>&1 || true
  docker images --filter "reference=ebox/*" -q | sort -u | xargs -r docker rmi -f >/dev/null 2>&1 || true
  docker volume ls -q --filter "label=ebox.managed=true" | xargs -r docker volume rm -f >/dev/null 2>&1 || true

  echo
  echo "ebox is removed, including its data and containers."
else
  echo
  cat <<EOF
ebox is removed. Your data is still in ${DATA_DIR} and ${ENV_DIR}.

  Remove it too:  sudo bash installer/uninstall.sh --purge
EOF
fi
