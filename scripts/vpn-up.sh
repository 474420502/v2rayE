#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
TOKEN="${V2RAYN_TUI_TOKEN:-${V2RAYN_API_TOKEN:-}}"
PROXY_MODE="${V2RAYN_VPN_PROXY_MODE:-global}"
PROXY_EXCEPTIONS="${V2RAYN_VPN_PROXY_EXCEPTIONS:-}"

api_post() {
    local path="$1"
    local body="$2"

    local -a curl_args
    curl_args=(
        -sS
        --max-time 5
        -H "Content-Type: application/json"
    )
    if [[ -n "$TOKEN" ]]; then
        curl_args+=( -H "Authorization: Bearer ${TOKEN}" )
    fi

    curl "${curl_args[@]}" -X POST "$BACKEND_URL$path" -d "$body"
}

api_get() {
    local path="$1"

    local -a curl_args
    curl_args=(
        -sS
        --max-time 5
    )
    if [[ -n "$TOKEN" ]]; then
        curl_args+=( -H "Authorization: Bearer ${TOKEN}" )
    fi

    curl "${curl_args[@]}" "$BACKEND_URL$path"
}

if ! command -v curl >/dev/null 2>&1; then
    echo "error: curl is required" >&2
    exit 1
fi

echo "[1/5] ensure backend"
"$ROOT_DIR/scripts/start-backend.sh" >/dev/null

echo "[2/5] start core"
start_resp="$(api_post "/api/core/start" '{}')"
if ! printf '%s' "$start_resp" | rg -q '"running"[[:space:]]*:[[:space:]]*true'; then
    echo "error: core start failed"
    printf '%s\n' "$start_resp"
    exit 1
fi

echo "[3/5] apply system proxy (mode=$PROXY_MODE)"
proxy_payload=$(printf '{"mode":"%s","exceptions":"%s"}' "$PROXY_MODE" "$PROXY_EXCEPTIONS")
proxy_resp="$(api_post "/api/system-proxy/apply" "$proxy_payload")"
if ! printf '%s' "$proxy_resp" | rg -q '"code"[[:space:]]*:[[:space:]]*0'; then
    echo "error: apply system proxy failed"
    printf '%s\n' "$proxy_resp"
    exit 1
fi

echo "[4/5] check core status"
status_resp="$(api_get "/api/core/status")"
printf '%s\n' "$status_resp"

echo "[5/5] check network availability"
net_resp="$(api_get "/api/network/availability")"
printf '%s\n' "$net_resp"

echo "VPN is up."
