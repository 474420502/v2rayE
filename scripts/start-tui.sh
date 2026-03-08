#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT_DIR/v2raye"

if [[ ! -x "$BIN" ]]; then
    echo "[compat] building unified executable: ./v2raye"
    "$ROOT_DIR/scripts/build.sh"
fi

echo "[compat] start-tui.sh is deprecated; running ./v2raye"
exec "$BIN" "$@"
