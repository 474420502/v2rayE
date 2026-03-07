#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

BACKEND_PID_FILE="$RUN_DIR/backend.pid"
FRONTEND_PID_FILE="$RUN_DIR/frontend.pid"
BACKEND_LOG_FILE="$RUN_DIR/backend.log"
FRONTEND_LOG_FILE="$RUN_DIR/frontend.log"

BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
BACKEND_DATA_DIR="${V2RAYN_DATA_DIR:-$RUN_DIR/data}"
FRONTEND_PORT="${PORT:-3000}"
EFFECTIVE_FRONTEND_PORT="$FRONTEND_PORT"
COUPLED_LIFECYCLE="${V2RAYE_COUPLED_LIFECYCLE:-1}"

mkdir -p "$RUN_DIR"
mkdir -p "$BACKEND_DATA_DIR"

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
        echo "warning: Linux TUN route takeover requires root privileges; use: sudo ./scripts/start-dev.sh"
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
    setsid env V2RAYN_API_ADDR="$BACKEND_ADDR" V2RAYN_DATA_DIR="$BACKEND_DATA_DIR" bash -lc "cd '$ROOT_DIR/backend-go' && exec go run ./cmd/server" >> "$BACKEND_LOG_FILE" 2>&1 &
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

is_port_busy() {
    local port="$1"
    ss -ltn 2>/dev/null | awk '{print $4}' | rg -q ":${port}$"
}

pick_frontend_port() {
    local preferred="$1"
    if ! is_port_busy "$preferred"; then
        echo "$preferred"
        return 0
    fi

    local candidate
    for candidate in $(seq $((preferred + 1)) $((preferred + 20))); do
        if ! is_port_busy "$candidate"; then
            echo "$candidate"
            return 0
        fi
    done

    echo "$preferred"
    return 1
}

start_frontend() {
    cleanup_stale_pid "$FRONTEND_PID_FILE"
    local pid
    pid="$(read_pid "$FRONTEND_PID_FILE")"
    if [[ -n "$pid" ]] && is_running "$pid"; then
        echo "frontend already running (pid=$pid)"
        return 0
    fi

    local selected_port
    if ! selected_port="$(pick_frontend_port "$FRONTEND_PORT")"; then
        echo "frontend port range ${FRONTEND_PORT}-$((FRONTEND_PORT + 20)) appears busy" >&2
        return 1
    fi
    EFFECTIVE_FRONTEND_PORT="$selected_port"
    if [[ "$EFFECTIVE_FRONTEND_PORT" != "$FRONTEND_PORT" ]]; then
        echo "warning: port $FRONTEND_PORT is in use, frontend will use $EFFECTIVE_FRONTEND_PORT"
    fi

    : > "$FRONTEND_LOG_FILE"
    if [[ "$EUID" -eq 0 ]] && [[ -n "${SUDO_USER:-}" ]]; then
        # Keep frontend dev artifacts owned by the login user, even when backend needs sudo.
        chown -R "${SUDO_USER}:${SUDO_USER}" "$ROOT_DIR/web/.next" >/dev/null 2>&1 || true
        setsid sudo -u "$SUDO_USER" env PORT="$EFFECTIVE_FRONTEND_PORT" V2RAYN_BACKEND_URL="http://$BACKEND_ADDR" bash -lc "cd '$ROOT_DIR/web' && exec npm run dev" >> "$FRONTEND_LOG_FILE" 2>&1 &
    else
        setsid env PORT="$EFFECTIVE_FRONTEND_PORT" V2RAYN_BACKEND_URL="http://$BACKEND_ADDR" bash -lc "cd '$ROOT_DIR/web' && exec npm run dev" >> "$FRONTEND_LOG_FILE" 2>&1 &
    fi
    pid=$!
    echo "$pid" > "$FRONTEND_PID_FILE"
    sleep 2
    if ! is_running "$pid"; then
        echo "frontend failed to start, check $FRONTEND_LOG_FILE" >&2
        rm -f "$FRONTEND_PID_FILE"
        return 1
    fi
    echo "frontend started on http://127.0.0.1:$EFFECTIVE_FRONTEND_PORT (pid=$pid, backend=http://$BACKEND_ADDR)"
}

warn_tun_requires_sudo

start_backend
if ! start_frontend; then
    if [[ "$COUPLED_LIFECYCLE" == "0" ]]; then
        echo "warning: frontend did not start, but backend is still running at http://$BACKEND_ADDR"
        echo "hint: resolve frontend conflict (see $FRONTEND_LOG_FILE) and rerun ./scripts/start-dev.sh"
        echo "note: set V2RAYE_COUPLED_LIFECYCLE=1 to enforce all-or-nothing startup"
        exit 1
    fi

    echo "error: frontend start failed, rolling back backend for coupled lifecycle"
    "$ROOT_DIR/scripts/stop-dev.sh" >/dev/null 2>&1 || true
    echo "hint: fix frontend conflict (see $FRONTEND_LOG_FILE), then rerun ./scripts/start-dev.sh"
    exit 1
fi

echo "logs:"
echo "  backend:  $BACKEND_LOG_FILE"
echo "  frontend: $FRONTEND_LOG_FILE"

if [[ "$EUID" -eq 0 ]]; then
    echo "note: started as root; stop with sudo ./scripts/stop-dev.sh to ensure full TUN cleanup"
fi