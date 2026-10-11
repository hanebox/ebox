#!/usr/bin/env bash
#
# ebox installer: requirement preflight, terminal UI, and install-time
# admin account creation.
#
#   curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash
#
# Run `install.sh --help` for every option. Prompts read from /dev/tty, so the
# script can be piped into bash without swallowing its own input.
#
set -euo pipefail

umask 077

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------

REPOSITORY="${EBOX_UPDATE_REPOSITORY:-buildandbuild/ebox}"
BINARY_PATH="/usr/local/bin/ebox"
ENV_DIR="/etc/ebox"
ENV_FILE="${ENV_DIR}/ebox.env"
UNIT_PATH="/etc/systemd/system/ebox.service"
DATA_DIR="/var/lib/ebox"
SERVICE_USER="ebox"
SERVICE_NAME="ebox"

VERSION="latest"
LOCAL_BINARY=""
PANEL_HOST="127.0.0.1"
PANEL_PORT="8000"
START_SERVICE=1
FORCE=0
NO_ADMIN=0
COLOR_MODE="auto"
ADMIN_NAME=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""

HAS_TTY=0
SPINNER=0
COLOR=0
UNICODE=0
TRUECOLOR=0
TMP_DIR=""
SPIN_PID=""

OS_NAME=""
ASSET_ARCH=""
DISTRO=""
EXISTING_INSTALL=0
PREVIOUS_BINARY=""
ADMIN_MODE="skip"
SKIP_ADMIN_REASON=""

PASSED=0
WARNED=0
FAILED=0

C_VIOLET=""
C_PURPLE=""
C_BLEND=""
C_PINK=""
C_GREEN=""
C_YELLOW=""
C_RED=""
C_DIM=""
C_BOLD=""
C_RESET=""

G_OK=""
G_ERR=""
G_WARN=""
G_INFO=""
G_ARROW=""
G_HINT=""
G_ELLIPSIS=""
G_BAR_ON=""
G_BAR_OFF=""
G_BOX_TL=""
G_BOX_TR=""
G_BOX_BL=""
G_BOX_BR=""
G_BOX_H=""
G_BOX_V=""
SPIN_FRAMES=()

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------

usage() {
  cat <<'USAGE'
ebox installer

  curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash

Options:
  --version v0.5.0          install a specific release (default: latest)
  --binary /path            install a local binary instead of downloading
  --host 127.0.0.1          address the panel binds to (default: 127.0.0.1)
  --port 8000               panel port (default: 8000)
  --data-dir /var/lib/ebox  database, deployments and backups
  --no-start                install without starting the service
  --no-admin                do not create the admin account (finish at /setup)
  --admin-name NAME         admin display name (for unattended installs)
  --admin-email EMAIL       admin email, used to sign in
  --admin-password PASS     admin password, at least 10 characters
                            (prefer the prompt; flags end up in shell history)
  --no-color                plain output (also honours NO_COLOR)
  --force                   no confirmations; continue past failed checks
  -h, --help                this help

The installer checks the host first (Docker, systemd, ports, disk, network),
then asks for the admin name, email and password and claims the panel as soon
as it starts, so the setup wizard is never left open to the network.
USAGE
}

usage_error() {
  printf '\n  %s%s%s %s\n\n' "$C_RED" "${G_ERR:-error}" "$C_RESET" "$1" >&2
  printf '  Run with --help for usage.\n\n' >&2
  exit 1
}

fail() {
  printf '\n  %s%s%s %s\n\n' "$C_RED" "$G_ERR" "$C_RESET" "$1" >&2
  exit 1
}

# ---------------------------------------------------------------------------
# Terminal toolkit
# ---------------------------------------------------------------------------

init_tui() {
  if [ "$COLOR_MODE" != "never" ] && [ -z "${NO_COLOR:-}" ] && [ -t 1 ] && [ "${TERM:-dumb}" != "dumb" ]; then
    COLOR=1
    case "${COLORTERM:-}" in
      truecolor|24bit) TRUECOLOR=1 ;;
    esac
  fi

  if [ -t 1 ] && [ "${TERM:-dumb}" != "dumb" ]; then
    SPINNER=1
  fi

  local loc="${LC_ALL:-${LC_CTYPE:-${LANG:-}}}"
  case "$loc" in
    *UTF-8*|*UTF8*|*utf8*|*utf-8*) UNICODE=1 ;;
  esac

  if [ "$COLOR" -eq 1 ]; then
    if [ "$TRUECOLOR" -eq 1 ]; then
      C_VIOLET=$'\033[38;2;123;107;255m'
      C_PURPLE=$'\033[38;2;168;85;247m'
      C_BLEND=$'\033[38;2;198;77;203m'
      C_PINK=$'\033[38;2;236;72;153m'
    else
      C_VIOLET=$'\033[38;5;105m'
      C_PURPLE=$'\033[38;5;141m'
      C_BLEND=$'\033[38;5;176m'
      C_PINK=$'\033[38;5;205m'
    fi
    C_GREEN=$'\033[32m'
    C_YELLOW=$'\033[33m'
    C_RED=$'\033[31m'
    C_DIM=$'\033[2m'
    C_BOLD=$'\033[1m'
    C_RESET=$'\033[0m'
  fi

  if [ "$UNICODE" -eq 1 ]; then
    G_OK="✔"
    G_ERR="✖"
    G_WARN="⚠"
    G_INFO="·"
    G_ARROW="▸"
    G_HINT="↳"
    G_ELLIPSIS="…"
    G_BAR_ON="█"
    G_BAR_OFF="░"
    G_BOX_TL="╭"
    G_BOX_TR="╮"
    G_BOX_BL="╰"
    G_BOX_BR="╯"
    G_BOX_H="─"
    G_BOX_V="│"
    SPIN_FRAMES=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
  else
    G_OK="[ok]"
    G_ERR="[!!]"
    G_WARN="[!]"
    G_INFO="-"
    G_ARROW=">"
    G_HINT="->"
    G_ELLIPSIS="..."
    G_BAR_ON="#"
    G_BAR_OFF="."
    G_BOX_TL="+"
    G_BOX_TR="+"
    G_BOX_BL="+"
    G_BOX_BR="+"
    G_BOX_H="-"
    G_BOX_V="|"
    SPIN_FRAMES=('|' / - '\')
  fi

  if { exec 3<>/dev/tty; } 2>/dev/null; then
    HAS_TTY=1
  fi
}

banner() {
  local word="${C_BOLD}${C_VIOLET}e${C_PURPLE}b${C_BLEND}o${C_PINK}x${C_RESET}"
  local tag="${C_DIM}self-hosted control plane for your server${C_RESET}"
  printf '\n'
  if [ "$UNICODE" -eq 0 ]; then
    local inner="  ${word}   "
    printf '  %s+---------+%s\n' "$C_VIOLET" "$C_RESET"
    printf '  %s|%s%s%s|%s   %s\n' "$C_PURPLE" "$C_RESET" "$inner" "$C_PURPLE" "$C_RESET" "$tag"
    printf '  %s+---------+%s\n' "$C_PINK" "$C_RESET"
    return 0
  fi
  printf '   %s▄▄▄▄▄▄▄▄▄%s\n' "$C_VIOLET" "$C_RESET"
  printf ' %s▄█▀▀▀▀▀▀▀▀▀█▄%s   %s\n' "$C_VIOLET" "$C_RESET" "$word"
  printf '%s▐█▌         ▐█▌%s  %s\n' "$C_PURPLE" "$C_RESET" "$tag"
  printf ' %s▀█▄▄▄▄▄▄▄▄▄█▀%s\n' "$C_PINK" "$C_RESET"
  printf '   %s▀▀▀▀▀▀▀▀▀%s\n' "$C_PINK" "$C_RESET"
}

group() {
  printf '\n  %s%s%s %s%s%s\n' "$C_VIOLET" "$G_ARROW" "$C_RESET" "$C_DIM" "$1" "$C_RESET"
}

step() {
  local n="$1" total="$2" title="$3" extra="${4:-}"
  printf '\n  %s%s%s %s[%s/%s]%s %s%s%s' \
    "$C_VIOLET" "$G_ARROW" "$C_RESET" "$C_DIM" "$n" "$total" "$C_RESET" "$C_BOLD" "$title" "$C_RESET"
  if [ -n "$extra" ]; then
    printf ' %s%s%s' "$C_DIM" "$extra" "$C_RESET"
  fi
  printf '\n'
}

row() {
  printf '  %s%-4s%s %s%-20s%s %s%s%s\n' "$2" "$1" "$C_RESET" "$C_BOLD" "$3" "$C_RESET" "$C_DIM" "$4" "$C_RESET"
}

c_ok() { PASSED=$((PASSED + 1)); row "$G_OK" "$C_GREEN" "$1" "$2"; }
c_warn() { WARNED=$((WARNED + 1)); row "$G_WARN" "$C_YELLOW" "$1" "$2"; }
c_info() { row "$G_INFO" "$C_DIM" "$1" "$2"; }

c_fail() {
  if [ "$FORCE" -eq 1 ]; then
    c_warn "$1" "$2 (forced)"
  else
    FAILED=$((FAILED + 1))
    row "$G_ERR" "$C_RED" "$1" "$2"
  fi
  hint "$3"
}

hint() {
  printf '       %s%s %s%s\n' "$C_DIM" "$G_HINT" "$1" "$C_RESET"
}

success_line() { printf '  %s%s%s %s\n' "$C_GREEN" "$G_OK" "$C_RESET" "$1"; }
warn_line() { printf '  %s%s%s %s\n' "$C_YELLOW" "$G_WARN" "$C_RESET" "$1"; }
note_line() { printf '  %s%s%s %s%s%s\n' "$C_DIM" "$G_INFO" "$C_RESET" "$C_DIM" "$1" "$C_RESET"; }

checks_summary() {
  printf '\n'
  printf '  %s%s%s %s%d passed%s    ' "$C_GREEN" "$G_OK" "$C_RESET" "$C_BOLD" "$PASSED" "$C_RESET"
  printf '%s%s%s %s%d warnings%s    ' "$C_YELLOW" "$G_WARN" "$C_RESET" "$C_BOLD" "$WARNED" "$C_RESET"
  printf '%s%s%s %s%d errors%s\n' "$C_RED" "$G_ERR" "$C_RESET" "$C_BOLD" "$FAILED" "$C_RESET"
}

exit_checks_failed() {
  checks_summary
  printf '\n  %s%s%s %sfix the errors above, then re-run the installer (or use --force to continue anyway)%s\n\n' \
    "$C_RED" "$G_ERR" "$C_RESET" "$C_DIM" "$C_RESET"
  exit 1
}

spin_start() {
  local msg="$1"
  if [ "$SPINNER" -eq 0 ]; then
    printf '  %s%s %s%s\n' "$C_DIM" "$G_ELLIPSIS" "$msg" "$C_RESET"
    return 0
  fi
  printf '\033[?25l'
  (
    local i=0
    while :; do
      printf '\r  %s%s%s %s%s\033[K' "$C_VIOLET" "${SPIN_FRAMES[$i]}" "$C_RESET" "$C_DIM" "$msg"
      i=$(((i + 1) % ${#SPIN_FRAMES[@]}))
      sleep 0.12
    done
  ) &
  SPIN_PID=$!
}

spin_stop() {
  if [ -n "$SPIN_PID" ]; then
    kill "$SPIN_PID" 2>/dev/null || true
    wait "$SPIN_PID" 2>/dev/null || true
    SPIN_PID=""
    printf '\r\033[K'
    if [ "$SPINNER" -eq 1 ]; then
      printf '\033[?25h'
    fi
  fi
}

human() {
  awk -v b="${1:-0}" 'BEGIN {
    if (b < 1024) { printf "%d B", b; exit }
    split("KB MB GB TB", u, " ")
    i = 1
    while (b >= 1024 && i < 5) { b /= 1024; i++ }
    printf "%.1f %s", b, u[i]
  }'
}

vlen() {
  local s
  s=$(printf '%s' "$1" | sed $'s/\033\\[[0-9;]*m//g')
  printf '%s' "${#s}"
}

rep() {
  local ch="$1" n="$2" out="" i
  for ((i = 0; i < n; i++)); do
    out+="$ch"
  done
  printf '%s' "$out"
}

box() {
  local title="$1"
  shift
  local lines=("$@")
  local w=0 line len dashes inner
  for line in "${lines[@]}"; do
    len=$(vlen "$line")
    if [ "$len" -gt "$w" ]; then w=$len; fi
  done
  len=$(vlen "$title")
  if [ "$len" -gt "$w" ]; then w=$len; fi
  inner=$((w + 2))

  if [ -n "$title" ]; then
    dashes=$((inner - len - 3))
    if [ "$dashes" -lt 1 ]; then dashes=1; fi
    printf '  %s%s%s %s%s%s %s%s%s\n' \
      "$C_DIM" "$G_BOX_TL$G_BOX_H" "$C_RESET" \
      "$C_BOLD" "$title" "$C_RESET" \
      "$C_DIM" "$(rep "$G_BOX_H" "$dashes")$G_BOX_TR" "$C_RESET"
  else
    printf '  %s%s%s\n' "$C_DIM" "$G_BOX_TL$(rep "$G_BOX_H" "$inner")$G_BOX_TR" "$C_RESET"
  fi

  for line in "${lines[@]}"; do
    len=$(vlen "$line")
    printf '  %s%s%s %s %s %s%s%s\n' \
      "$C_DIM" "$G_BOX_V" "$C_RESET" "$line" "$(rep ' ' $((w - len)))" "$C_DIM" "$G_BOX_V" "$C_RESET"
  done

  printf '  %s%s%s\n' "$C_DIM" "$G_BOX_BL$(rep "$G_BOX_H" "$inner")$G_BOX_BR" "$C_RESET"
}

render_bar() {
  local pct="$1" size="$2" total="$3" label="$4"
  local width=28 filled i bar=""
  filled=$((pct * width / 100))
  for ((i = 0; i < width; i++)); do
    if [ "$i" -lt "$filled" ]; then
      bar+="$G_BAR_ON"
    else
      bar+="$G_BAR_OFF"
    fi
  done
  printf '\r  %s%s%s  %s%s%s %s%3d%%%s  %s%s / %s%s\033[K' \
    "$C_BOLD" "$label" "$C_RESET" \
    "$C_VIOLET" "$bar" "$C_RESET" \
    "$C_BOLD" "$pct" "$C_RESET" \
    "$C_DIM" "$(human "$size")" "$(human "$total")" "$C_RESET"
}

confirm() {
  local answer=""
  printf '  %s?%s %s' "$C_YELLOW" "$C_RESET" "$1" >&2
  IFS= read -r -u 3 answer || answer=""
  case "$answer" in
    ''|[Yy]|[Yy][Ee][Ss]) return 0 ;;
    *) return 1 ;;
  esac
}

cleanup() {
  if [ -n "$SPIN_PID" ]; then
    kill "$SPIN_PID" 2>/dev/null || true
    wait "$SPIN_PID" 2>/dev/null || true
  fi
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
  if [ "$SPINNER" -eq 1 ]; then
    printf '\033[?25h'
  fi
  :
}

trap cleanup EXIT
trap 'exit 130' INT TERM

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

valid_email() {
  [[ "$1" =~ ^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$ ]]
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --version)
        [ $# -ge 2 ] || usage_error "--version needs a tag, e.g. v0.5.0"
        VERSION="$2"
        shift 2
        ;;
      --binary)
        [ $# -ge 2 ] || usage_error "--binary needs a path"
        LOCAL_BINARY="$2"
        shift 2
        ;;
      --host)
        [ $# -ge 2 ] || usage_error "--host needs an address"
        PANEL_HOST="$2"
        shift 2
        ;;
      --port)
        [ $# -ge 2 ] || usage_error "--port needs a number"
        PANEL_PORT="$2"
        shift 2
        ;;
      --data-dir)
        [ $# -ge 2 ] || usage_error "--data-dir needs a path"
        DATA_DIR="${2%/}"
        shift 2
        ;;
      --admin-name)
        [ $# -ge 2 ] || usage_error "--admin-name needs a value"
        ADMIN_NAME="$2"
        shift 2
        ;;
      --admin-email)
        [ $# -ge 2 ] || usage_error "--admin-email needs a value"
        ADMIN_EMAIL="$2"
        shift 2
        ;;
      --admin-password)
        [ $# -ge 2 ] || usage_error "--admin-password needs a value"
        ADMIN_PASSWORD="$2"
        shift 2
        ;;
      --no-start) START_SERVICE=0; shift ;;
      --no-admin) NO_ADMIN=1; shift ;;
      --no-color) COLOR_MODE="never"; shift ;;
      --force) FORCE=1; shift ;;
      -h|--help) usage; exit 0 ;;
      *) usage_error "unknown option: $1" ;;
    esac
  done

  case "$PANEL_PORT" in
    ''|*[!0-9]*) usage_error "--port must be a number" ;;
  esac
  if [ "$PANEL_PORT" -lt 1 ] || [ "$PANEL_PORT" -gt 65535 ]; then
    usage_error "--port must be between 1 and 65535"
  fi
  case "$DATA_DIR" in
    /*) ;;
    *) usage_error "--data-dir must be an absolute path" ;;
  esac
  if [ -n "$LOCAL_BINARY" ] && [ ! -f "$LOCAL_BINARY" ]; then
    usage_error "--binary ${LOCAL_BINARY} does not exist"
  fi
  if [ -n "$ADMIN_EMAIL" ] && ! valid_email "$ADMIN_EMAIL"; then
    usage_error "--admin-email does not look like an email address"
  fi
  if [ -n "$ADMIN_PASSWORD" ] && [ "${#ADMIN_PASSWORD}" -lt 10 ]; then
    usage_error "--admin-password must be at least 10 characters"
  fi
  if [ "$NO_ADMIN" -eq 1 ] && { [ -n "$ADMIN_NAME" ] || [ -n "$ADMIN_EMAIL" ] || [ -n "$ADMIN_PASSWORD" ]; }; then
    printf '  %s%s%s %s--no-admin wins over --admin-*; no account will be created%s\n\n' \
      "$C_YELLOW" "$G_WARN" "$C_RESET" "$C_DIM" "$C_RESET" >&2
  fi
}

# ---------------------------------------------------------------------------
# Checks
# ---------------------------------------------------------------------------

pkg_hint() {
  local pkg="$1"
  if command -v apt-get >/dev/null 2>&1; then
    printf 'apt-get install -y %s' "$pkg"
  elif command -v dnf >/dev/null 2>&1; then
    printf 'dnf install -y %s' "$pkg"
  elif command -v pacman >/dev/null 2>&1; then
    printf 'pacman -S --noconfirm %s' "$pkg"
  elif command -v zypper >/dev/null 2>&1; then
    printf 'zypper install -y %s' "$pkg"
  else
    printf 'install %s with your package manager' "$pkg"
  fi
}

listening_ports() {
  if command -v ss >/dev/null 2>&1; then
    ss -ltnH 2>/dev/null || ss -ltn 2>/dev/null | tail -n +2
  elif command -v netstat >/dev/null 2>&1; then
    netstat -ltn 2>/dev/null | tail -n +2
  else
    return 1
  fi
}

port_in_use() {
  local port="$1" output
  if ! output="$(listening_ports)"; then
    printf 'unknown'
    return 0
  fi
  if printf '%s\n' "$output" | awk 'NF >= 4 { print $4 }' | grep -qE "[:.]${port}\$"; then
    printf 'yes'
  else
    printf 'no'
  fi
}

ebox_traefik_running() {
  command -v docker >/dev/null 2>&1 &&
    docker ps --filter 'name=^ebox-traefik$' --format '{{.Names}}' 2>/dev/null | grep -q .
}

gather_system_info() {
  OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
  DISTRO="$(sed -n 's/^PRETTY_NAME="\(.*\)"$/\1/p' /etc/os-release 2>/dev/null | head -1)"
  if [ -z "$DISTRO" ]; then
    DISTRO="$(uname -s)"
  fi
  case "$(uname -m)" in
    x86_64|amd64) ASSET_ARCH="amd64" ;;
    aarch64|arm64) ASSET_ARCH="arm64" ;;
    *) ASSET_ARCH="" ;;
  esac
}

require_root() {
  if [ "$(id -u)" -eq 0 ]; then
    return 0
  fi
  printf '\n  %s%s%s the installer needs root to create the service and write %s\n' \
    "$C_RED" "$G_ERR" "$C_RESET" "$ENV_DIR" >&2
  printf '      %s%s fix: curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash%s\n\n' \
    "$C_DIM" "$G_HINT" "$C_RESET" >&2
  exit 1
}

detect_existing() {
  if [ -f "$UNIT_PATH" ] || [ -x "$BINARY_PATH" ] || [ -f "$ENV_FILE" ]; then
    EXISTING_INSTALL=1
  fi
}

run_checks() {
  group "Host"

  if [ "$OS_NAME" != "linux" ]; then
    c_fail "operating system" "Linux is required (found ${OS_NAME})" "run this installer on the Linux host you want to manage"
    exit_checks_failed
  fi
  c_ok "operating system" "${DISTRO} · kernel $(uname -r)"

  if [ -z "$ASSET_ARCH" ]; then
    c_fail "architecture" "unsupported: $(uname -m)" "ebox releases cover x86_64 and arm64"
    exit_checks_failed
  fi
  c_ok "architecture" "$(uname -m) · ebox-linux-${ASSET_ARCH}"

  if [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1; then
    c_ok "init system" "systemd $(systemctl --version 2>/dev/null | awk 'NR == 1 { print $2 }')"
  else
    c_fail "init system" "systemd was not detected" "ebox installs a systemd service; run this on a systemd host"
  fi

  group "Tooling"

  if command -v curl >/dev/null 2>&1; then
    c_ok "curl" "v$(curl --version 2>/dev/null | awk 'NR == 1 { print $2 }')"
  else
    c_fail "curl" "not found" "$(pkg_hint curl)"
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    c_ok "sha256sum" "downloads can be verified"
  else
    c_warn "sha256sum" "downloads cannot be verified" "$(pkg_hint coreutils)"
  fi

  if command -v openssl >/dev/null 2>&1; then
    c_ok "openssl" "generates the APP_KEY secret"
  else
    c_warn "openssl" "APP_KEY falls back to /dev/urandom" "$(pkg_hint openssl)"
  fi

  if command -v git >/dev/null 2>&1; then
    c_ok "git" "v$(git --version 2>/dev/null | awk '{ print $3 }') · source deployments"
  else
    c_warn "git" "source deployments will be disabled" "$(pkg_hint git)"
  fi

  local ssh_pkg="openssh-client"
  if command -v dnf >/dev/null 2>&1 || command -v zypper >/dev/null 2>&1; then
    ssh_pkg="openssh-clients"
  fi
  if command -v ssh >/dev/null 2>&1; then
    c_ok "ssh" "ssh git remotes"
  else
    c_warn "ssh" "ssh git remotes will be disabled" "$(pkg_hint "$ssh_pkg")"
  fi

  group "Docker"

  if command -v docker >/dev/null 2>&1; then
    c_ok "docker cli" "$(docker --version 2>/dev/null | sed -n 's/^Docker version \([^,]*\).*/\1/p')"
  else
    c_fail "docker cli" "not installed" "curl -fsSL https://get.docker.com | sh"
  fi

  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    c_ok "docker daemon" "reachable"
  else
    c_fail "docker daemon" "not reachable" "systemctl enable --now docker"
  fi

  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    local compose_version
    compose_version="$(docker compose version --short 2>/dev/null)"
    c_ok "docker compose" "${compose_version:-installed} · builds and proxy"
  else
    c_fail "docker compose" "the compose v2 plugin is required" "$(pkg_hint docker-compose-plugin)"
  fi

  if getent group docker >/dev/null 2>&1; then
    c_ok "docker group" "${SERVICE_USER} joins it to reach the socket"
  else
    c_fail "docker group" "the docker group does not exist" "reinstall Docker or check /etc/group"
  fi

  group "Resources"

  local probe_dir="$DATA_DIR"
  while [ ! -d "$probe_dir" ] && [ "$probe_dir" != "/" ]; do
    probe_dir="$(dirname "$probe_dir")"
  done
  local avail_kb
  avail_kb="$(df -Pk "$probe_dir" 2>/dev/null | awk 'NR == 2 { print $4 }')"
  case "$avail_kb" in
    ''|*[!0-9]*)
      c_warn "disk space" "could not read free space at ${probe_dir}" "check df -h ${probe_dir}"
      ;;
    *)
      if [ "$avail_kb" -lt 2097152 ]; then
        c_fail "disk space" "$(human $((avail_kb * 1024))) free at ${probe_dir}" "free at least 2 GB; images and builds need room"
      elif [ "$avail_kb" -lt 5242880 ]; then
        c_warn "disk space" "$(human $((avail_kb * 1024))) free at ${probe_dir}" "Docker images and builds fill this up quickly"
      else
        c_ok "disk space" "$(human $((avail_kb * 1024))) free at ${probe_dir}"
      fi
      ;;
  esac

  local mem_kb
  mem_kb="$(awk '/^MemTotal:/{ print $2 }' /proc/meminfo 2>/dev/null)"
  case "$mem_kb" in
    ''|*[!0-9]*) ;;
    *)
      if [ "$mem_kb" -lt 1048576 ]; then
        c_warn "memory" "$(human $((mem_kb * 1024))) ram" "builds may be slow; 1 GB is the practical minimum"
      else
        c_ok "memory" "$(human $((mem_kb * 1024))) ram"
      fi
      ;;
  esac

  group "Network"

  if [ -n "$LOCAL_BINARY" ]; then
    c_info "github.com" "not required (installing a local binary)"
  elif curl -fsI --max-time 10 https://api.github.com >/dev/null 2>&1; then
    c_ok "github.com" "reachable for release downloads"
  else
    c_fail "github.com" "not reachable" "check DNS/proxy, or install a local build with --binary"
  fi

  group "Ports"

  local port_state
  port_state="$(port_in_use "$PANEL_PORT")"
  case "$port_state" in
    no)
      c_ok "port ${PANEL_PORT}" "free for the panel"
      ;;
    yes)
      if [ "$EXISTING_INSTALL" -eq 1 ]; then
        c_info "port ${PANEL_PORT}" "held by the current ebox service (it will restart)"
      else
        c_fail "port ${PANEL_PORT}" "already in use" "pick another with --port, or stop the process using it"
      fi
      ;;
    *)
      c_info "ports" "install iproute2 (ss) to enable port checks"
      ;;
  esac

  if [ "$port_state" != "unknown" ]; then
    check_proxy_port 80
    check_proxy_port 443
    check_proxy_port 8080
  fi

  group "Installation"

  if [ "$EXISTING_INSTALL" -eq 1 ]; then
    c_info "existing install" "found at ${BINARY_PATH} · will be upgraded"
  else
    c_info "existing install" "none · fresh installation"
  fi

  checks_summary

  if [ "$FAILED" -gt 0 ]; then
    printf '\n  %s%s%s %sfix the errors above, then re-run the installer (or use --force to continue anyway)%s\n\n' \
      "$C_RED" "$G_ERR" "$C_RESET" "$C_DIM" "$C_RESET"
    exit 1
  fi
  if [ "$WARNED" -gt 0 ]; then
    note_line "continuing with ${WARNED} warning(s)"
  fi
}

check_proxy_port() {
  local port="$1" state
  state="$(port_in_use "$port")"
  case "$state" in
    no) c_ok "port ${port}" "free for the Traefik proxy" ;;
    yes)
      if ebox_traefik_running; then
        c_info "port ${port}" "held by ebox's Traefik container"
      else
        c_warn "port ${port}" "in use by another process" "free it before enabling the proxy for domains"
      fi
      ;;
    *) ;;
  esac
}

# ---------------------------------------------------------------------------
# Admin account
# ---------------------------------------------------------------------------

probe_host() {
  case "$PANEL_HOST" in
    0.0.0.0|'') printf '127.0.0.1' ;;
    '::') printf '[::1]' ;;
    *) printf '%s' "$PANEL_HOST" ;;
  esac
}

existing_panel_port() {
  local port=""
  if [ -f "$ENV_FILE" ]; then
    port="$(sed -n 's/^APP_PORT=\([0-9]*\)$/\1/p' "$ENV_FILE" | head -1)"
  fi
  printf '%s' "${port:-$PANEL_PORT}"
}

panel_state() {
  local host="$1" port="$2" body=""
  body="$(curl -fsS --noproxy '*' --max-time 3 "http://${host}:${port}/api/v1/setup/status" 2>/dev/null)" || {
    printf 'unknown'
    return 0
  }
  case "$body" in
    *'"initialized":true'*) printf 'true' ;;
    *'"initialized":false'*) printf 'false' ;;
    *) printf 'unknown' ;;
  esac
}

prompt_admin() {
  local attempts=0 pass1="" pass2=""

  printf '\n  %sAdmin account%s\n' "$C_BOLD" "$C_RESET"
  printf '  %sebox claims the panel with this account right after it starts, so the\n  setup wizard is never left open to the network.%s\n' "$C_DIM" "$C_RESET"

  if [ -z "$ADMIN_NAME" ]; then
    printf '\n  %sDisplay name%s %s[Admin]%s: ' "$C_BOLD" "$C_RESET" "$C_DIM" "$C_RESET" >&2
    IFS= read -r -u 3 ADMIN_NAME || ADMIN_NAME=""
  fi
  if [ -z "$ADMIN_NAME" ]; then
    ADMIN_NAME="Admin"
  fi

  attempts=0
  while [ -z "$ADMIN_EMAIL" ]; do
    printf '  %sEmail%s %s(used to sign in)%s: ' "$C_BOLD" "$C_RESET" "$C_DIM" "$C_RESET" >&2
    IFS= read -r -u 3 ADMIN_EMAIL || ADMIN_EMAIL=""
    if valid_email "$ADMIN_EMAIL"; then
      break
    fi
    printf '  %s%s%s that does not look like an email address\n' "$C_RED" "$G_ERR" "$C_RESET" >&2
    ADMIN_EMAIL=""
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 3 ]; then
      return 1
    fi
  done

  attempts=0
  while [ -z "$ADMIN_PASSWORD" ]; do
    printf '  %sPassword%s %s(at least 10 characters)%s: ' "$C_BOLD" "$C_RESET" "$C_DIM" "$C_RESET" >&2
    IFS= read -rs -u 3 pass1 || pass1=""
    printf '\n' >&2
    printf '  %sConfirm password%s: ' "$C_BOLD" "$C_RESET" >&2
    IFS= read -rs -u 3 pass2 || pass2=""
    printf '\n' >&2
    if [ "${#pass1}" -lt 10 ]; then
      printf '  %s%s%s use at least 10 characters\n' "$C_RED" "$G_ERR" "$C_RESET" >&2
    elif [ "$pass1" != "$pass2" ]; then
      printf '  %s%s%s the passwords do not match\n' "$C_RED" "$G_ERR" "$C_RESET" >&2
    else
      ADMIN_PASSWORD="$pass1"
    fi
    attempts=$((attempts + 1))
    if [ -z "$ADMIN_PASSWORD" ] && [ "$attempts" -ge 3 ]; then
      return 1
    fi
  done

  return 0
}

collect_admin() {
  if [ "$NO_ADMIN" -eq 1 ]; then
    ADMIN_MODE="skip"
    SKIP_ADMIN_REASON="--no-admin"
    return 0
  fi
  if [ "$START_SERVICE" -eq 0 ]; then
    ADMIN_MODE="skip"
    SKIP_ADMIN_REASON="--no-start"
    return 0
  fi

  if [ "$EXISTING_INSTALL" -eq 1 ]; then
    local state
    state="$(panel_state "$(probe_host)" "$(existing_panel_port)")"
    if [ "$state" = "true" ]; then
      ADMIN_MODE="skip"
      SKIP_ADMIN_REASON="the panel already has an admin account"
      return 0
    fi
  fi

  if [ -n "$ADMIN_NAME" ] && [ -n "$ADMIN_EMAIL" ] && [ -n "$ADMIN_PASSWORD" ]; then
    ADMIN_MODE="create"
    return 0
  fi

  if [ "$HAS_TTY" -eq 0 ]; then
    if [ -n "$ADMIN_NAME$ADMIN_EMAIL$ADMIN_PASSWORD" ]; then
      fail "provide --admin-name, --admin-email and --admin-password together, or pass --no-admin"
    fi
    ADMIN_MODE="skip"
    SKIP_ADMIN_REASON="no interactive terminal"
    return 0
  fi

  if ! prompt_admin; then
    fail "could not collect the admin credentials; re-run with --no-admin to use the web wizard instead"
  fi
  ADMIN_MODE="create"
  return 0
}

confirm_upgrade() {
  if [ "$EXISTING_INSTALL" -eq 0 ]; then
    return 0
  fi
  if [ "$FORCE" -eq 1 ] || [ "$HAS_TTY" -eq 0 ]; then
    note_line "upgrading the existing installation"
    return 0
  fi
  printf '\n'
  if ! confirm "An existing installation was found. Upgrade it? [Y/n] "; then
    printf '\n  cancelled\n\n'
    exit 0
  fi
  return 0
}

# ---------------------------------------------------------------------------
# Install steps
# ---------------------------------------------------------------------------

create_user_and_dirs() {
  local nologin="/usr/sbin/nologin"
  if [ ! -x "$nologin" ]; then
    nologin="/sbin/nologin"
  fi

  if id "$SERVICE_USER" >/dev/null 2>&1; then
    success_line "user ${SERVICE_USER} already exists"
  else
    useradd --system --home-dir "$DATA_DIR" --shell "$nologin" "$SERVICE_USER"
    success_line "created the ${SERVICE_USER} system user"
  fi

  if getent group docker >/dev/null 2>&1; then
    if id -nG "$SERVICE_USER" 2>/dev/null | grep -qw docker; then
      success_line "${SERVICE_USER} is in the docker group"
    else
      usermod -aG docker "$SERVICE_USER"
      success_line "added ${SERVICE_USER} to the docker group"
    fi
  else
    warn_line "the docker group is missing; ${SERVICE_USER} cannot reach the Docker socket"
  fi

  install -d -m 0755 -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"
  install -d -m 0755 -o "$SERVICE_USER" -g "$SERVICE_USER" "${DATA_DIR}/traefik"
  install -d -m 0755 "$ENV_DIR"
  success_line "prepared ${DATA_DIR} and ${ENV_DIR}"
}

resolve_release() {
  if [ "$VERSION" != "latest" ]; then
    return 0
  fi
  local body="" rc=0
  spin_start "resolving the latest release of ${REPOSITORY}"
  body="$(curl -fsSL --max-time 20 "https://api.github.com/repos/${REPOSITORY}/releases/latest" 2>"$TMP_DIR/github.log")" || rc=$?
  spin_stop
  if [ "$rc" -ne 0 ]; then
    if grep -qi 'rate limit' "$TMP_DIR/github.log" 2>/dev/null; then
      fail "the GitHub API rate limit was hit; pass --version vX.Y.Z to skip the lookup"
    fi
    fail "could not resolve the latest release of ${REPOSITORY}; pass --version vX.Y.Z"
  fi
  VERSION="$(printf '%s' "$body" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  if [ -z "$VERSION" ]; then
    fail "no release was found for ${REPOSITORY}"
  fi
  success_line "release ${VERSION}"
}

download() {
  local url="$1" dest="$2" label="$3"
  local total="" size=0 pct=0 rc=0 start=$SECONDS

  total="$(curl -fsSIL --max-time 20 "$url" 2>/dev/null | tr -d '\r' | awk 'tolower($1) == "content-length:" { v = $2 } END { if (v) print v }')"
  case "$total" in
    ''|*[!0-9]*) total=0 ;;
  esac

  if [ "$SPINNER" -eq 1 ] && [ "$total" -gt 0 ]; then
    curl -fSL --retry 3 --retry-delay 2 --connect-timeout 20 -o "$dest" "$url" >/dev/null 2>"$TMP_DIR/curl.log" &
    local pid=$!
    while kill -0 "$pid" 2>/dev/null; do
      size="$(stat -c %s "$dest" 2>/dev/null)" || size=0
      case "$size" in
        ''|*[!0-9]*) size=0 ;;
      esac
      if [ "$size" -gt "$total" ]; then
        size="$total"
      fi
      pct=$((size * 100 / total))
      render_bar "$pct" "$size" "$total" "$label"
      sleep 0.2
    done
    wait "$pid" || rc=$?
    render_bar 100 "$total" "$total" "$label"
    printf '\n'
  else
    spin_start "downloading ${label}"
    curl -fSL --retry 3 --retry-delay 2 --connect-timeout 20 -o "$dest" "$url" 2>"$TMP_DIR/curl.log" || rc=$?
    spin_stop
  fi

  if [ "$rc" -eq 0 ]; then
    success_line "downloaded ${label} ($(human "$(stat -c %s "$dest")") in $((SECONDS - start))s)"
  fi
  return "$rc"
}

verify_checksum() {
  local asset="$1" expected="" actual=""
  if [ -n "$LOCAL_BINARY" ]; then
    return 0
  fi
  if ! curl -fsSL "https://github.com/${REPOSITORY}/releases/download/${VERSION}/checksums.txt" -o "$TMP_DIR/checksums.txt" 2>/dev/null; then
    warn_line "this release has no checksums.txt; skipping verification"
    return 0
  fi
  expected="$(awk -v asset="$asset" '{ name = $2; sub(/^\*/, "", name); if (name == asset) print $1 }' "$TMP_DIR/checksums.txt" | head -1)"
  if [ -z "$expected" ]; then
    warn_line "checksums.txt has no entry for ${asset}; skipping verification"
    return 0
  fi
  if ! command -v sha256sum >/dev/null 2>&1; then
    warn_line "sha256sum is not available; skipping verification"
    return 0
  fi
  actual="$(sha256sum "$TMP_DIR/ebox" | awk '{ print $1 }')"
  if [ "$expected" != "$actual" ]; then
    fail "checksum mismatch: expected ${expected}, got ${actual}"
  fi
  success_line "sha256 verified (${actual:0:16}${G_ELLIPSIS})"
}

write_environment() {
  local existing_key="" existing_traefik="" key="" traefik=""
  if [ -f "$ENV_FILE" ]; then
    existing_key="$(sed -n 's/^APP_KEY=\(.*\)$/\1/p' "$ENV_FILE" | head -1)"
    existing_traefik="$(sed -n 's/^EBOX_TRAEFIK_DIR=\(.*\)$/\1/p' "$ENV_FILE" | head -1)"
  fi

  key="$existing_key"
  if [ -z "$key" ]; then
    if command -v openssl >/dev/null 2>&1; then
      key="base64:$(openssl rand -base64 32)"
    else
      key="base64:$(head -c 32 /dev/urandom | base64 | tr -d '\n')"
    fi
  fi
  traefik="${existing_traefik:-${DATA_DIR}/traefik}"

  install -m 0600 /dev/null "$ENV_FILE"
  {
    printf '# Generated by the ebox installer on %s.\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf '# APP_KEY encrypts stored secrets. Back it up: losing it makes stored\n'
    printf '# credentials unreadable.\n'
    printf 'APP_KEY=%s\n' "$key"
    printf 'APP_ENV=production\n'
    printf 'APP_DEBUG=false\n'
    printf '\n'
    printf '# The Go server binds APP_HOST/APP_PORT; EBOX_HOST/EBOX_PORT feed proxy\n'
    printf '# routing and the URLs the panel shows.\n'
    printf 'APP_HOST=%s\n' "$PANEL_HOST"
    printf 'APP_PORT=%s\n' "$PANEL_PORT"
    printf 'EBOX_HOST=%s\n' "$PANEL_HOST"
    printf 'EBOX_PORT=%s\n' "$PANEL_PORT"
    printf '\n'
    printf '# Database, app checkouts, volumes, backups and the Traefik stack.\n'
    printf 'EBOX_DATA_DIR=%s\n' "$DATA_DIR"
    printf 'EBOX_TRAEFIK_DIR=%s\n' "$traefik"
    printf 'EBOX_DOCKER_NETWORK=ebox\n'
    printf '\n'
    printf '# Local port the Traefik API is published on.\n'
    printf 'EBOX_TRAEFIK_API_PORT=8080\n'
    printf '\n'
    printf '# GitHub repository used for release checks and self-update.\n'
    printf 'EBOX_UPDATE_REPOSITORY=%s\n' "$REPOSITORY"
  } >"$ENV_FILE"
}

write_unit() {
  cat >"$UNIT_PATH" <<EOF
[Unit]
Description=ebox personal server control plane
Documentation=https://github.com/buildandbuild/ebox
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${DATA_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${BINARY_PATH}
Restart=always
RestartSec=3
# The panel talks to the Docker socket.
SupplementaryGroups=docker
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true
# Database, deployments, backups and the Traefik stack.
ReadWritePaths=${DATA_DIR} ${ENV_DIR}

[Install]
WantedBy=multi-user.target
EOF
  chmod 0644 "$UNIT_PATH"
}

install_binary() {
  if [ -f "$BINARY_PATH" ]; then
    if cp -f "$BINARY_PATH" "${BINARY_PATH}.previous" 2>/dev/null; then
      PREVIOUS_BINARY="${BINARY_PATH}.previous"
    fi
  fi
  install -m 0755 "$TMP_DIR/ebox" "$BINARY_PATH"
}

wait_for_panel() {
  local host="$1" port="$2" timeout="$3" i=0
  spin_start "waiting for the panel to answer"
  while [ "$i" -lt "$timeout" ]; do
    if curl -fsS --noproxy '*' --max-time 2 "http://${host}:${port}/api/v1/setup/status" >/dev/null 2>&1; then
      spin_stop
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  spin_stop
  return 1
}

start_service() {
  systemctl daemon-reload || fail "could not reload systemd (is this a systemd host?)"
  if ! systemctl enable "$SERVICE_NAME" >/dev/null 2>&1; then
    warn_line "could not enable ${SERVICE_NAME} at boot"
  fi

  if ! systemctl restart "$SERVICE_NAME"; then
    printf '\n'
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager 2>/dev/null || true
    rollback_binary
    fail "the ${SERVICE_NAME} service failed to start; inspect it with: journalctl -u ${SERVICE_NAME} -n 50"
  fi

  if ! wait_for_panel "$(probe_host)" "$PANEL_PORT" 60; then
    printf '\n'
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager 2>/dev/null || true
    rollback_binary
    fail "the panel did not come up within 60 seconds (see the log above)"
  fi

  success_line "the panel is up on http://$(probe_host):${PANEL_PORT}"
}

rollback_binary() {
  if [ -n "$PREVIOUS_BINARY" ] && [ -f "$PREVIOUS_BINARY" ]; then
    warn_line "restoring the previous binary"
    install -m 0755 "$PREVIOUS_BINARY" "$BINARY_PATH" || true
    systemctl restart "$SERVICE_NAME" >/dev/null 2>&1 || true
  fi
}

json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}

create_admin() {
  local host="$1" port="$2" payload="" code=""
  payload="$(printf '{"name":"%s","email":"%s","password":"%s"}' \
    "$(json_escape "$ADMIN_NAME")" "$(json_escape "$ADMIN_EMAIL")" "$(json_escape "$ADMIN_PASSWORD")")"

  code="$(printf '%s' "$payload" | curl -sS --noproxy '*' --connect-timeout 5 --max-time 20 \
    -o "$TMP_DIR/setup.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' --data-binary @- \
    "http://${host}:${port}/api/v1/setup" 2>/dev/null)" || code="000"

  case "$code" in
    200) return 0 ;;
    403) return 2 ;;
    422) return 3 ;;
    000) return 5 ;;
    *) return 4 ;;
  esac
}

api_message() {
  local msg=""
  if [ -s "$TMP_DIR/setup.json" ]; then
    msg="$(sed -n 's/.*"message":"\([^"]*\)".*/\1/p' "$TMP_DIR/setup.json" | head -1)"
  fi
  printf '%s' "$msg"
}

maybe_create_admin() {
  if [ "$ADMIN_MODE" != "create" ]; then
    case "$SKIP_ADMIN_REASON" in
      "--no-admin") warn_line "admin account creation skipped (--no-admin)" ;;
      "--no-start") note_line "the service was not started; create the admin account at /setup when it is" ;;
      "no interactive terminal") warn_line "no terminal available, so no admin account was created" ;;
      *) note_line "${SKIP_ADMIN_REASON}" ;;
    esac
    return 0
  fi

  local host port state rc=0 msg=""
  host="$(probe_host)"
  port="$PANEL_PORT"

  state="$(panel_state "$host" "$port")"
  if [ "$state" = "true" ]; then
    success_line "the panel already has an admin account; keeping it"
    ADMIN_PASSWORD=""
    return 0
  fi

  create_admin "$host" "$port" || rc=$?
  case "$rc" in
    0) success_line "admin account created for ${ADMIN_EMAIL}" ;;
    2) success_line "the panel already has an admin account; keeping it" ;;
    3)
      msg="$(api_message)"
      fail "the panel rejected the account details${msg:+: ${msg}}"
      ;;
    4)
      msg="$(api_message)"
      fail "could not create the admin account${msg:+: ${msg}}"
      ;;
    *) fail "could not reach the panel at ${host}:${port} to create the admin account" ;;
  esac
  ADMIN_PASSWORD=""
  return 0
}

perform_install() {
  local asset="" url=""

  step 1 7 "Creating the system user and directories"
  create_user_and_dirs

  step 2 7 "Fetching the ebox binary"
  if [ -n "$LOCAL_BINARY" ]; then
    cp -f "$LOCAL_BINARY" "$TMP_DIR/ebox" || fail "could not read ${LOCAL_BINARY}"
    chmod 0755 "$TMP_DIR/ebox"
    VERSION="local build"
    success_line "using ${LOCAL_BINARY} ($(human "$(stat -c %s "$TMP_DIR/ebox")"))"
  else
    resolve_release
    asset="ebox-${OS_NAME}-${ASSET_ARCH}"
    url="https://github.com/${REPOSITORY}/releases/download/${VERSION}/${asset}"
    if ! download "$url" "$TMP_DIR/ebox" "${asset} ${VERSION}"; then
      if [ -s "$TMP_DIR/curl.log" ]; then
        tail -n 3 "$TMP_DIR/curl.log" >&2
      fi
      fail "could not download ${url}"
    fi
    verify_checksum "$asset"
  fi
  chmod 0755 "$TMP_DIR/ebox"

  step 3 7 "Writing the configuration"
  write_environment
  success_line "${ENV_FILE} (0600)"

  step 4 7 "Installing the binary"
  install_binary
  success_line "${BINARY_PATH}"

  step 5 7 "Installing the systemd unit"
  write_unit
  success_line "${UNIT_PATH}"

  if [ "$START_SERVICE" -eq 0 ]; then
    step 6 7 "Starting the service" "(skipped: --no-start)"
    step 7 7 "Creating the admin account" "(skipped: --no-start)"
    return 0
  fi

  step 6 7 "Starting the service"
  start_service

  step 7 7 "Creating the admin account"
  maybe_create_admin
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

first_ip() {
  local ip=""
  ip="$(hostname -I 2>/dev/null | awk '{ print $1 }')"
  printf '%s' "${ip:-<server-ip>}"
}

finish() {
  local url="" account=""
  url="http://$(probe_host):${PANEL_PORT}"

  if [ "$ADMIN_MODE" = "create" ] && [ -n "$ADMIN_EMAIL" ]; then
    account="$ADMIN_EMAIL"
  else
    account="not created ${G_HINT} finish at ${url}/setup"
  fi

  printf '\n'
  box "ebox ${VERSION} - ${OS_NAME}/${ASSET_ARCH}" \
    "Panel       ${url}" \
    "Account     ${account}" \
    "Data        ${DATA_DIR}" \
    "Service     systemctl status ${SERVICE_NAME}" \
    "Logs        journalctl -u ${SERVICE_NAME} -f" \
    "Update      panel Settings ${G_ARROW} Updates, or re-run this installer" \
    "Uninstall   sudo bash installer/uninstall.sh"

  case "$PANEL_HOST" in
    127.0.0.1|localhost|::1) ;;
    *)
      printf '\n'
      warn_line "the panel binds ${PANEL_HOST}; set a panel domain in Settings before exposing it"
      ;;
  esac

  if [ "$PANEL_HOST" = "0.0.0.0" ]; then
    note_line "reachable on this network at http://$(first_ip):${PANEL_PORT}"
  fi

  if [ "$ADMIN_MODE" != "create" ]; then
    printf '\n'
    warn_line "no admin account exists yet ${G_HINT} open ${url}/setup"
    printf '      %sanyone who can reach the panel before you do can claim it%s\n' "$C_DIM" "$C_RESET"
  fi

  printf '\n  %sTraefik (the proxy) is provisioned by the panel from the Proxy page on first use.%s\n\n' "$C_DIM" "$C_RESET"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  parse_args "$@"
  init_tui
  banner

  require_root
  gather_system_info
  detect_existing

  TMP_DIR="$(mktemp -d)"

  run_checks
  confirm_upgrade
  collect_admin
  perform_install
  finish
}

main "$@"
