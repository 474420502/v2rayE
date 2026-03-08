#!/bin/bash

# 构建脚本 - 构建统一可执行文件

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "=== 开始构建 v2rayE ==="

echo ">>> 构建统一入口..."
cd "$ROOT_DIR/backend-go"
go build -o "$ROOT_DIR/v2raye" ./cmd/v2raye
echo ">>> 统一入口构建完成"

echo "=== 构建完成 ==="
echo "  - executable: ./v2raye"
echo "  - server mode: ./v2raye --server"
echo "  - tui mode: ./v2raye"