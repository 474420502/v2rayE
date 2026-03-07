#!/bin/bash

# 构建脚本 - 同时构建 web 和 backend

set -e

echo "=== 开始构建 v2rayE ==="

# 构建 backend (Go)
echo ">>> 构建 backend-go..."
cd backend-go
go build -o server ./cmd/server
echo ">>> backend-go 构建完成"

# 返回根目录
cd ..

# 构建 web (Next.js)
echo ">>> 构建 web..."
cd web
npm run build
echo ">>> web 构建完成"

# 返回根目录
cd ..

echo "=== 构建完成 ==="
echo "  - backend: ./backend-go/server"
echo "  - web: ./web/.next"