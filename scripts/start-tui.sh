#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
# Token precedence: V2RAYN_TUI_TOKEN (preferred), then V2RAYN_API_TOKEN.
TOKEN="${V2RAYN_TUI_TOKEN:-${V2RAYN_API_TOKEN:-}}"
AUTO_UP="${V2RAYN_TUI_AUTO_UP:-1}"

backend_api_ready() {
    if ! command -v curl >/dev/null 2>&1; then
        return 1
    fi

    local -a curl_args
    curl_args=(
        -sS
        --max-time 1
    )
    if [[ -n "$TOKEN" ]]; then
        curl_args+=(-H "Authorization: Bearer ${TOKEN}")
    fi

    curl "${curl_args[@]}" "$BACKEND_URL/api/core/status" >/dev/null 2>&1
}

api_post() {
    local path="$1"
    local body="$2"

    local -a curl_args
    curl_args=(
        -sS
        --max-time 3
        -H "Content-Type: application/json"
    )
    if [[ -n "$TOKEN" ]]; then
        curl_args+=( -H "Authorization: Bearer ${TOKEN}" )
    fi

    curl "${curl_args[@]}" -X POST "$BACKEND_URL$path" -d "$body"
}

ensure_core_running() {
    if [[ "$AUTO_UP" != "1" ]]; then
        return 0
    fi
    if ! command -v curl >/dev/null 2>&1; then
        echo "warning: curl not found, skip auto core start" >&2
        return 0
    fi

    local response
    response="$(api_post "/api/core/start" '{}')" || {
        echo "warning: failed to auto start core; open TUI and start core manually" >&2
        return 0
    }

    if printf '%s' "$response" | rg -q '"running"[[:space:]]*:[[:space:]]*true'; then
        echo "core ready: running"
        return 0
    fi

    echo "warning: core is not running yet; open TUI Dashboard to inspect startup error" >&2
}

if ! backend_api_ready; then
    "$ROOT_DIR/scripts/start-backend.sh"
fi

ensure_core_running

cd "$ROOT_DIR/backend-go/cmd/tui"
exec go run . --base-url "$BACKEND_URL" --token "$TOKEN" "$@"