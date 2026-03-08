#!/bin/bash

# 构建脚本 - 默认构建 backend 与 TUI

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "=== 开始构建 v2rayE ==="

echo ">>> 构建 backend-go..."
cd "$ROOT_DIR/backend-go"
go build -o backend-api ./cmd/backend-api
echo ">>> backend-go 构建完成"

echo ">>> 构建 TUI..."
cd "$ROOT_DIR/backend-go/cmd/tui"
go build -o "$ROOT_DIR/v2raye-tui" .
echo ">>> TUI 构建完成"

echo "=== 构建完成 ==="
echo "  - backend api: ./backend-go/backend-api"
echo "  - tui: ./v2raye-tui"