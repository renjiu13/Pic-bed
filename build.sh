#!/bin/bash
set -e

echo "=== 极轻量图床 - 多架构编译 ==="

mkdir -p bin

echo "编译 linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/pic-bed-linux-amd64 ./cmd/pic-bed
echo "✓ linux amd64 完成"

echo "编译 linux arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/pic-bed-linux-arm64 ./cmd/pic-bed
echo "✓ linux arm64 完成"

echo "编译 linux armv7 (玩客云)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o bin/pic-bed-linux-armv7 ./cmd/pic-bed
echo "✓ linux armv7 完成"

echo "编译 windows amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/pic-bed-windows-amd64.exe ./cmd/pic-bed
echo "✓ windows amd64 完成"

echo "编译 darwin amd64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/pic-bed-darwin-amd64 ./cmd/pic-bed
echo "✓ darwin amd64 完成"

echo "编译 darwin arm64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/pic-bed-darwin-arm64 ./cmd/pic-bed
echo "✓ darwin arm64 完成"

echo ""
echo "=== 编译完成 ==="
ls -lh bin/
