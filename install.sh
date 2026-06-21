#!/bin/bash
set -e

echo "=== Pic-Bed 一键安装 ==="
echo ""

# ========== 权限检测 ==========
if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
  echo "👑 当前为 root 用户"
else
  if command -v sudo &> /dev/null; then
    SUDO="sudo"
    echo "🔐 非 root 用户，将使用 sudo"
  else
    echo "❌ 当前不是 root 用户，且系统未安装 sudo"
    echo "请切换到 root 用户后重新执行：su root"
    exit 1
  fi
fi

# ========== 架构检测 ==========
ARCH=$(uname -m)
case $ARCH in
  x86_64)   PLATFORM="linux-amd64" ;;
  aarch64)  PLATFORM="linux-arm64" ;;
  armv7l)   PLATFORM="linux-armv7" ;;
  *)
    echo ""
    echo "❌ 不支持的架构: $ARCH"
    echo "支持的架构: x86_64, aarch64, armv7l"
    exit 1
    ;;
esac

echo "📦 检测到架构: $ARCH ($PLATFORM)"

# ========== 获取最新版本 ==========
echo ""
echo "🔍 获取最新版本..."
VERSION=$(curl -s https://api.github.com/repos/renjiu13/Pic-bed/releases/latest | grep tag_name | cut -d '"' -f4)

if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
  echo "❌ 获取版本信息失败，请检查网络连接或 GitHub 是否可访问"
  exit 1
fi

echo "✨ 最新版本: $VERSION"

# ========== 安装目录 ==========
INSTALL_DIR="/opt/pic-bed"
echo ""
echo "📂 安装目录: $INSTALL_DIR"

# 检测是否已安装
if [ -f "$INSTALL_DIR/pic-bed" ]; then
  OLD_VERSION=$($INSTALL_DIR/pic-bed -v 2>/dev/null || echo "未知版本")
  echo "⚠️  检测到已安装版本: $OLD_VERSION"
  echo "将覆盖升级到 $VERSION"
  echo ""
fi

# ========== 创建目录 ==========
$SUDO mkdir -p $INSTALL_DIR

# ========== 下载 ==========
echo "⬇️  下载中..."
$SUDO curl -L --progress-bar \
  "https://github.com/renjiu13/Pic-bed/releases/download/$VERSION/pic-bed-$PLATFORM" \
  -o $INSTALL_DIR/pic-bed

$SUDO chmod +x $INSTALL_DIR/pic-bed

# ========== 完成 ==========
echo ""
echo "✅ 安装完成！"
echo ""
echo "🚀 启动服务："
echo "   cd $INSTALL_DIR && ./pic-bed"
echo ""
echo "⚙️  配置文件："
echo "   首次运行自动生成 $INSTALL_DIR/config.json"
echo ""
echo "🔧 设置开机自启（systemd）："
echo "   参考 README 中的 systemd 配置说明"
echo ""