#!/bin/bash

# 构建脚本 - 默认构建 backend 与 TUI

set -e

echo "=== 开始构建 v2rayE ==="

echo ">>> 构建 backend-go..."
cd backend-go
go build -o server ./cmd/server
echo ">>> backend-go 构建完成"

cd ..

echo ">>> 构建 TUI..."
cd backend-go/cmd/tui
go build -o ../../../v2raye-tui .
echo ">>> TUI 构建完成"

cd ../..

echo "=== 构建完成 ==="
echo "  - backend: ./backend-go/server"
echo "  - tui: ./v2raye-tui"