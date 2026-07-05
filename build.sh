#!/bin/bash
set -e

echo "=== 极轻量图床 - 多架构编译 ==="

# 获取版本信息（优先使用环境变量，兼容 GitHub Actions）
if [ -z "$VERSION" ]; then
  VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
fi
if [ -z "$BUILD_TIME" ]; then
  BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
fi
if [ -z "$GIT_COMMIT" ]; then
  GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
fi

echo "版本: $VERSION"
echo "编译时间: $BUILD_TIME"
echo "Commit: $GIT_COMMIT"
echo ""

# 构建 ldflags
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/pic-bed/pic-bed/internal/version.Version=$VERSION"
LDFLAGS="$LDFLAGS -X github.com/pic-bed/pic-bed/internal/version.BuildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X github.com/pic-bed/pic-bed/internal/version.GitCommit=$GIT_COMMIT"

mkdir -p bin

echo "编译 linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/pic-bed-linux-amd64 ./cmd/pic-bed
echo "✓ linux amd64 完成"

echo "编译 linux arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o bin/pic-bed-linux-arm64 ./cmd/pic-bed
echo "✓ linux arm64 完成"

echo "编译 linux armv7 (玩客云)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="$LDFLAGS" -o bin/pic-bed-linux-armv7 ./cmd/pic-bed
echo "✓ linux armv7 完成"

echo "编译 windows amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/pic-bed-windows-amd64.exe ./cmd/pic-bed
echo "✓ windows amd64 完成"

echo "编译 darwin amd64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/pic-bed-darwin-amd64 ./cmd/pic-bed
echo "✓ darwin amd64 完成"

echo "编译 darwin arm64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o bin/pic-bed-darwin-arm64 ./cmd/pic-bed
echo "✓ darwin arm64 完成"

echo ""
echo "=== 编译完成 ==="
ls -lh bin/
