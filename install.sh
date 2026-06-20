#!/bin/bash
set -e

echo "=== Pic-Bed 一键安装 ==="
echo ""

# 检测架构
ARCH=$(uname -m)
case $ARCH in
  x86_64)   PLATFORM="linux-amd64" ;;
  aarch64)  PLATFORM="linux-arm64" ;;
  armv7l)   PLATFORM="linux-armv7" ;;
  *)
    echo "❌ 不支持的架构: $ARCH"
    echo "支持的架构: x86_64, aarch64, armv7l"
    exit 1
    ;;
esac

echo "📦 检测到架构: $ARCH ($PLATFORM)"

# 获取最新版本
echo "🔍 获取最新版本..."
VERSION=$(curl -s https://api.github.com/repos/renjiu13/Pic-bed/releases/latest | grep tag_name | cut -d '"' -f4)

if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
  echo "❌ 获取版本信息失败，请检查网络连接"
  exit 1
fi

echo "✨ 最新版本: $VERSION"

# 安装目录
INSTALL_DIR="/opt/pic-bed"
echo ""
echo "📂 安装目录: $INSTALL_DIR"

# 创建目录
sudo mkdir -p $INSTALL_DIR
cd $INSTALL_DIR

# 下载
echo "⬇️  下载中..."
sudo curl -L "https://github.com/renjiu13/Pic-bed/releases/download/$VERSION/pic-bed-$PLATFORM" -o pic-bed
sudo chmod +x pic-bed

echo ""
echo "✅ 安装完成！"
echo ""
echo "🚀 启动命令："
echo "   cd $INSTALL_DIR && ./pic-bed"
echo ""
echo "⚙️  配置文件首次运行自动生成在: $INSTALL_DIR/config.json"
echo ""
echo "🔧 开机自启（systemd）："
echo "   参考 README 中的 systemd 配置"
echo ""
