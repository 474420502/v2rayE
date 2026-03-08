#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

BACKEND_PID_FILE="$RUN_DIR/backend.pid"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
BACKEND_DATA_DIR="${V2RAYN_DATA_DIR:-$RUN_DIR/data}"
BACKEND_PORT="${BACKEND_ADDR##*:}"

can_use_passwordless_sudo() {
    [[ "$EUID" -eq 0 ]] && return 0
    command -v sudo >/dev/null 2>&1 || return 1
    sudo -n true >/dev/null 2>&1
}

kill_pid_or_group() {
    local signal_name="$1"
    local pid="$2"

    if kill "-${signal_name}" -- "-${pid}" 2>/dev/null || kill "-${signal_name}" "$pid" 2>/dev/null; then
        return 0
    fi

    if can_use_passwordless_sudo; then
        sudo -n kill "-${signal_name}" -- "-${pid}" 2>/dev/null || sudo -n kill "-${signal_name}" "$pid" 2>/dev/null || true
        return 0
    fi

    return 1
}

run_ip_cmd() {
    if ip "$@" >/dev/null 2>&1; then
        return 0
    fi
    if can_use_passwordless_sudo; then
        sudo -n ip "$@" >/dev/null 2>&1 || true
        return 0
    fi
    return 1
}

read_tun_name() {
    local config_file="$BACKEND_DATA_DIR/config.json"
    if [[ -f "$config_file" ]] && command -v node >/dev/null 2>&1; then
        local tun_name
        tun_name="$(node -e '
const fs = require("fs");
const path = process.argv[1];
let tun = "";
try {
  const cfg = JSON.parse(fs.readFileSync(path, "utf8"));
  if (typeof cfg.tunName === "string" && cfg.tunName.trim()) {
    tun = cfg.tunName.trim();
  }
} catch {}
process.stdout.write(tun);
' "$config_file")"
        if [[ -n "$tun_name" ]]; then
            echo "$tun_name"
            return 0
        fi
    fi
    echo "xraye0"
}

cleanup_tun_fallback() {
    if ! command -v ip >/dev/null 2>&1; then
        return 0
    fi
    local tun_name
    tun_name="$(read_tun_name)"

    run_ip_cmd route del default dev "$tun_name"
    run_ip_cmd link set dev "$tun_name" down
    run_ip_cmd link del dev "$tun_name"

    # Extra safety for old/default interface names used in previous runs.
    for candidate in xraye0 xray0; do
        [[ "$candidate" == "$tun_name" ]] && continue
        run_ip_cmd route del default dev "$candidate"
        run_ip_cmd link set dev "$candidate" down
        run_ip_cmd link del dev "$candidate"
    done
}

post_backend_api() {
    local path="$1"
    local payload="$2"
    if ! command -v curl >/dev/null 2>&1; then
        return 0
    fi

    local -a curl_args
    curl_args=(
        -sS
        --max-time 2
        -H
        "Content-Type: application/json"
    )
    if [[ -n "${V2RAYN_API_TOKEN:-}" ]]; then
        curl_args+=(-H "Authorization: Bearer ${V2RAYN_API_TOKEN}")
    fi

    curl "${curl_args[@]}" -X POST "$BACKEND_URL$path" -d "$payload" >/dev/null 2>&1 || true
}

graceful_backend_shutdown() {
    local pid
    pid="$(read_pid "$BACKEND_PID_FILE")"
    if [[ -z "$pid" ]] || ! is_running "$pid"; then
        return 0
    fi

    # Let the backend process handle StopCore() during SIGTERM so runtime
    # restore intent survives restart workflows.
    post_backend_api "/api/system-proxy/apply" '{"mode":"forced_clear","exceptions":""}'
}

is_running() {
    local pid="$1"
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

read_pid() {
    local pid_file="$1"
    if [[ -f "$pid_file" ]]; then
        tr -d '[:space:]' < "$pid_file"
    fi
}

stop_by_pid_file() {
    local name="$1"
    local pid_file="$2"
    local pid
    pid="$(read_pid "$pid_file")"

    if [[ -z "$pid" ]]; then
        echo "$name not running"
        return 0
    fi

    if ! is_running "$pid"; then
        rm -f "$pid_file"
        echo "$name already stopped"
        return 0
    fi

    kill_pid_or_group TERM "$pid" || true

    for _ in {1..20}; do
        if ! is_running "$pid"; then
            rm -f "$pid_file"
            echo "$name stopped"
            return 0
        fi
        sleep 0.2
    done

    kill_pid_or_group KILL "$pid" || true
    sleep 0.2
    if is_running "$pid"; then
        echo "$name still running (pid=$pid)" >&2
        echo "hint: process may be root-owned, retry with sudo: sudo ./scripts/stop-dev.sh" >&2
        return 1
    fi

    rm -f "$pid_file"
    echo "$name stopped (forced)"
}

collect_pids_by_pattern() {
    ps -eo pid=,cmd= | awk -v root="$ROOT_DIR/backend-go" '
        (index($0, root) > 0 || $0 ~ /go run \.\/cmd\/(backend-api|server)/ || $0 ~ /\/tmp\/go-build[^ ]*\/exe\/(backend-api|server)/) &&
        ($0 ~ /go run \.\/cmd\/(backend-api|server)/ || $0 ~ /\/cmd\/(backend-api|server)/ || $0 ~ /backend-go\/cmd\/(backend-api|server)/ || $0 ~ /\/tmp\/go-build[^ ]*\/exe\/(backend-api|server)/ || $0 ~ /(^|[[:space:]])(backend-api|server)([[:space:]]|$)/) { print $1 }
    '
}

collect_pids_with_fuser() {
    local port="$1"
    local raw
    if [[ "$2" == "sudo" ]]; then
        raw="$(sudo -n fuser -n tcp "$port" 2>/dev/null || true)"
    else
        raw="$(fuser -n tcp "$port" 2>/dev/null || true)"
    fi

    if [[ -n "$raw" ]]; then
        printf '%s\n' "$raw" | tr ' ' '\n' | rg '^[0-9]+$' || true
    fi
}

collect_pids_by_port() {
    local port="$1"
    if [[ -z "$port" ]]; then
        return 0
    fi

    if command -v fuser >/dev/null 2>&1; then
        collect_pids_with_fuser "$port" "" || true
        if can_use_passwordless_sudo; then
            collect_pids_with_fuser "$port" "sudo" || true
        fi
        return 0
    fi

    if command -v ss >/dev/null 2>&1; then
        if can_use_passwordless_sudo; then
            sudo -n ss -ltnp 2>/dev/null | rg ":${port}" | rg -o 'pid=[0-9]+' | sed 's/pid=//' || true
            return 0
        fi
        ss -ltnp 2>/dev/null | rg ":${port}" | rg -o 'pid=[0-9]+' | sed 's/pid=//' || true
    fi
}

extra_port_cleanup() {
    local -a pids=()
    mapfile -t pids < <(collect_pids_by_port "$BACKEND_PORT" | rg '^[0-9]+$' | sort -u)
    if [[ "${#pids[@]}" -eq 0 ]]; then
        return 0
    fi
    kill_pids_force "port:$BACKEND_PORT" "${pids[@]}" || true
}

kill_pids_force() {
    local role="$1"
    shift
    local pids=("$@")
    if [[ "${#pids[@]}" -eq 0 ]]; then
        echo "$role not running"
        return 0
    fi

    local pid
    for pid in "${pids[@]}"; do
        kill_pid_or_group TERM "$pid" || true
    done

    for _ in {1..20}; do
        local alive=0
        for pid in "${pids[@]}"; do
            if is_running "$pid"; then
                alive=1
                break
            fi
        done
        if [[ "$alive" -eq 0 ]]; then
            echo "$role stopped"
            return 0
        fi
        sleep 0.2
    done

    for pid in "${pids[@]}"; do
        kill_pid_or_group KILL "$pid" || true
    done
    sleep 0.2

    local still_alive=0
    for pid in "${pids[@]}"; do
        if is_running "$pid"; then
            still_alive=1
            echo "$role still running (pid=$pid)" >&2
        fi
    done
    if [[ "$still_alive" -eq 0 ]]; then
        echo "$role stopped (forced)"
        return 0
    fi

    return 1
}

force_cleanup_role() {
    local -a collected=()

    while IFS= read -r pid; do
        [[ -n "$pid" ]] && collected+=("$pid")
    done < <(collect_pids_by_pattern)

    while IFS= read -r pid; do
        [[ -n "$pid" ]] && collected+=("$pid")
    done < <(collect_pids_by_port "$BACKEND_PORT")

    if [[ "${#collected[@]}" -eq 0 ]]; then
        echo "backend not running"
        return 0
    fi

    mapfile -t collected < <(printf '%s\n' "${collected[@]}" | rg '^[0-9]+$' | sort -u)
    kill_pids_force "backend" "${collected[@]}"
}

stop_ok=0

graceful_backend_shutdown
if stop_by_pid_file "backend" "$BACKEND_PID_FILE"; then
    stop_ok=1
fi

if force_cleanup_role; then
    stop_ok=1
fi

extra_port_cleanup

rm -f "$BACKEND_PID_FILE"
cleanup_tun_fallback

if [[ "$EUID" -ne 0 ]] && ! can_use_passwordless_sudo; then
    echo "note: if TUN was enabled and cleanup did not complete, run: sudo ./scripts/stop-dev.sh"
fi

if [[ "$stop_ok" -eq 0 ]]; then
    exit 1
fi