#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

BACKEND_PID_FILE="$RUN_DIR/backend.pid"
BACKEND_LOG_FILE="$RUN_DIR/backend.log"

BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
BACKEND_DATA_DIR="${V2RAYN_DATA_DIR:-$RUN_DIR/data}"
BACKEND_ASSET_DIR="${V2RAYN_ASSET_DIR:-$ROOT_DIR/backend-go}"

mkdir -p "$RUN_DIR"
mkdir -p "$BACKEND_DATA_DIR"
mkdir -p "$BACKEND_ASSET_DIR"

warn_tun_requires_sudo() {
    if [[ "$EUID" -eq 0 ]]; then
        return 0
    fi
    local config_file="$BACKEND_DATA_DIR/config.json"
    if [[ ! -f "$config_file" ]] || ! command -v node >/dev/null 2>&1; then
        return 0
    fi

    local tun_mode
    tun_mode="$(node -e '
const fs = require("fs");
const path = process.argv[1];
try {
  const raw = fs.readFileSync(path, "utf8");
  const cfg = JSON.parse(raw);
  const mode = typeof cfg.tunMode === "string" ? cfg.tunMode : "";
  const enabled = cfg.enableTun === true;
  if ((mode && mode !== "off") || enabled) {
    process.stdout.write(mode || "on");
  }
} catch {}
' "$config_file")"

    if [[ -n "$tun_mode" ]]; then
        echo "warning: detected TUN mode (${tun_mode}) in $config_file"
        echo "warning: Linux TUN route takeover requires root privileges; use: sudo ./scripts/start-backend.sh"
    fi
}

is_running() {
    local pid="$1"
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

backend_api_ready() {
    if ! command -v curl >/dev/null 2>&1; then
        return 1
    fi
    curl -sS --max-time 1 "$BACKEND_URL/api/core/status" >/dev/null 2>&1
}

read_pid() {
    local pid_file="$1"
    if [[ -f "$pid_file" ]]; then
        tr -d '[:space:]' < "$pid_file"
    fi
}

cleanup_stale_pid() {
    local pid_file="$1"
    local pid
    pid="$(read_pid "$pid_file")"
    if [[ -n "$pid" ]] && ! is_running "$pid"; then
        rm -f "$pid_file"
    fi
}

start_backend() {
    if backend_api_ready; then
        echo "backend already running at http://$BACKEND_ADDR"
        return 0
    fi

    cleanup_stale_pid "$BACKEND_PID_FILE"
    local pid
    pid="$(read_pid "$BACKEND_PID_FILE")"
    if [[ -n "$pid" ]] && is_running "$pid"; then
        echo "backend already running (pid=$pid)"
        return 0
    fi

    : > "$BACKEND_LOG_FILE"
    setsid env \
        V2RAYN_API_ADDR="$BACKEND_ADDR" \
        V2RAYN_DATA_DIR="$BACKEND_DATA_DIR" \
        XRAY_LOCATION_ASSET="$BACKEND_ASSET_DIR" \
        V2RAY_LOCATION_ASSET="$BACKEND_ASSET_DIR" \
        bash -lc "cd '$ROOT_DIR/backend-go' && exec go run ./cmd/backend-api" >> "$BACKEND_LOG_FILE" 2>&1 &
    pid=$!
    echo "$pid" > "$BACKEND_PID_FILE"
    for _ in {1..20}; do
        if backend_api_ready; then
            echo "backend started on http://$BACKEND_ADDR (pid=$pid, data=$BACKEND_DATA_DIR)"
            return 0
        fi
        if ! is_running "$pid"; then
            break
        fi
        sleep 0.2
    done

    if ! is_running "$pid"; then
        echo "backend failed to start, check $BACKEND_LOG_FILE" >&2
        rm -f "$BACKEND_PID_FILE"
        return 1
    fi

    echo "backend process started but API is not reachable at http://$BACKEND_ADDR" >&2
    echo "backend may be unhealthy, check $BACKEND_LOG_FILE" >&2
    rm -f "$BACKEND_PID_FILE"
    return 1
}

warn_tun_requires_sudo
start_backend

echo "logs:"
echo "  backend: $BACKEND_LOG_FILE"
echo ""
echo "TUI workflow:"
echo "  start interactive TUI: ./scripts/start-tui.sh"
echo "  stop backend/TUN cleanup: ./scripts/stop-dev.sh"

if [[ "$EUID" -eq 0 ]]; then
    echo "note: started as root; stop with sudo ./scripts/stop-dev.sh to ensure full TUN cleanup"
fi