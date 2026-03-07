#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://${BACKEND_ADDR}"
API_TOKEN="${V2RAYN_API_TOKEN:-}"
TUN_MODE_ON="${TUN_MODE_ON:-system}"

usage() {
    cat <<'USAGE'
Usage:
  sudo ./scripts/tun-cycle-check.sh

What this script does:
  1) Reads current route/config/status snapshot
  2) Enables TUN mode (default: system) and restarts core
  3) Disables TUN mode (off) and restarts core
    4) Verifies default route no longer points to xraye0 and tunMode is off

Environment overrides:
  V2RAYN_API_ADDR   backend api address, default 127.0.0.1:18000
  V2RAYN_API_TOKEN  bearer token if API auth is enabled
  TUN_MODE_ON       one of system|mixed|gvisor, default system
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

require_cmd() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "missing required command: $cmd" >&2
        exit 1
    fi
}

require_cmd curl
require_cmd ip
require_cmd node

if [[ "$EUID" -ne 0 ]]; then
    echo "this script should be run with sudo/root for TUN route operations" >&2
    exit 1
fi

api_call() {
    local method="$1"
    local path="$2"
    local payload="${3:-}"

    local -a args
    args=(
        -sS
        --max-time 10
        -X "$method"
        -H "Content-Type: application/json"
    )

    if [[ -n "$API_TOKEN" ]]; then
        args+=(-H "Authorization: Bearer ${API_TOKEN}")
    fi

    if [[ -n "$payload" ]]; then
        args+=(-d "$payload")
    fi

    curl "${args[@]}" "${BACKEND_URL}${path}"
}

extract_field() {
    local json="$1"
    local field="$2"
    printf '%s' "$json" | node -e '
const fs = require("fs");
const input = fs.readFileSync(0, "utf8");
const field = process.argv[1];
let value = "";
try {
  const obj = JSON.parse(input);
  if (obj && typeof obj === "object" && obj.data && typeof obj.data === "object") {
    const v = obj.data[field];
    value = v === undefined || v === null ? "" : String(v);
  }
} catch {}
process.stdout.write(value);
' "$field"
}

print_header() {
    local title="$1"
    echo
    echo "========== ${title} =========="
}

print_snapshot() {
    local label="$1"
    local cfg_json status_json route_line

    cfg_json="$(api_call GET /api/config)"
    status_json="$(api_call GET /api/core/status)"
    route_line="$(ip -4 route show default | head -n 1 || true)"

    print_header "$label"
    echo "route: ${route_line:-<none>}"
    echo "config.tunMode: $(extract_field "$cfg_json" tunMode)"
    echo "config.enableTun: $(extract_field "$cfg_json" enableTun)"
    echo "config.coreEngine: $(extract_field "$cfg_json" coreEngine)"
    echo "status.running: $(extract_field "$status_json" running)"
    echo "status.error: $(extract_field "$status_json" error)"
}

echo "backend: ${BACKEND_URL}"

print_snapshot "Before"

print_header "Enable TUN and restart core"
api_call PUT /api/config "{\"tunMode\":\"${TUN_MODE_ON}\",\"enableTun\":true,\"tunStack\":\"${TUN_MODE_ON}\",\"tunAutoRoute\":true}" >/dev/null
api_call POST /api/core/restart "{}" >/dev/null || true
sleep 2
print_snapshot "After TUN On"

print_header "Disable TUN and restart core"
api_call PUT /api/config '{"tunMode":"off","enableTun":false}' >/dev/null
api_call POST /api/core/restart '{}' >/dev/null || true
sleep 2
print_snapshot "After TUN Off"

final_cfg="$(api_call GET /api/config)"
final_route="$(ip -4 route show default | head -n 1 || true)"
final_mode="$(extract_field "$final_cfg" tunMode)"

pass=1
if [[ "$final_mode" != "off" ]]; then
    echo "FAIL: expected config.tunMode=off, got ${final_mode:-<empty>}" >&2
    pass=0
fi
if [[ "$final_route" == *"dev xraye0"* ]]; then
    echo "FAIL: default route still points to xraye0" >&2
    pass=0
fi

print_header "Result"
if [[ "$pass" -eq 1 ]]; then
    echo "PASS: tunMode is off and default route is not xraye0"
    exit 0
fi

echo "FAIL: see snapshots above"
exit 1
