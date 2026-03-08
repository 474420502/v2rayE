#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://$BACKEND_ADDR"
# Token precedence: V2RAYN_TUI_TOKEN (preferred), then V2RAYN_API_TOKEN, then legacy V2RAYE_TUI_TOKEN.
TOKEN="${V2RAYN_TUI_TOKEN:-${V2RAYN_API_TOKEN:-${V2RAYE_TUI_TOKEN:-}}}"

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

if ! backend_api_ready; then
    "$ROOT_DIR/scripts/start-dev.sh"
fi

cd "$ROOT_DIR/backend-go/cmd/tui"
exec go run . --base-url "$BACKEND_URL" --token "$TOKEN" "$@"