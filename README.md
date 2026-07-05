# Pic-Bed - 极轻量私有图床

专为低内存设备设计的单文件私有图床，完美支持玩客云等 ARM32 设备。

---

## ✨ 特性

- 🚀 **极低内存**：闲置 8~15MB，峰值不超 25MB
- 📦 **单文件部署**：纯静态编译，零依赖
- 🔧 **全平台支持**：Linux / Windows / macOS，6 种架构
- 🔒 **安全加固**：路径防护、扩展名白名单、可选 Bearer 鉴权
- 🎯 **PicList 兼容**：完美支持 PicList 自定义图床
- ⚙️ **按需开启**：8 大功能开关，默认全关

---

## 🚀 一键安装

### Linux / 玩客云 / 树莓派

```bash
curl -sSL https://raw.githubusercontent.com/renjiu13/Pic-bed/main/install.sh | bash
```

安装完成后启动：
```bash
cd /opt/pic-bed && ./pic-bed
```

### Windows (PowerShell)

```powershell
$v = (Invoke-RestMethod https://api.github.com/repos/renjiu13/Pic-bed/releases/latest).tag_name
Invoke-WebRequest "https://github.com/renjiu13/Pic-bed/releases/download/$v/pic-bed-windows-amd64.exe" -OutFile "pic-bed.exe"
.\pic-bed.exe
```

### macOS

```bash
# Apple Silicon (M 系列)
curl -L https://github.com/renjiu13/Pic-bed/releases/latest/download/pic-bed-darwin-arm64 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

---

## ⚙️ 配置

首次运行自动生成 `config.json`：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `port` | 8080 | 监听端口 |
| `storage_dir` | ./data | 图片存储目录 |
| `max_size` | 10 | 单文件上限（MB） |
| `api_key` | "" | Bearer Token，空则关闭鉴权 |
| `enable_log` | false | 操作日志 |
| `enable_delete` | false | 允许删除图片 |
| `enable_auto_clean` | false | 自动清理旧文件 |
| `auto_clean_hours` | 720 | 自动清理阈值（小时，默认 30 天） |
| `home_avatar_url` | "" | 首页头像 URL |

修改配置后重启生效。

---

## 🔧 systemd 开机自启

```bash
# 下载服务文件
wget https://raw.githubusercontent.com/renjiu13/Pic-bed/main/examples/pic-bed.service -O /etc/systemd/system/pic-bed.service

# 修改路径后启动
systemctl daemon-reload
systemctl enable --now pic-bed
systemctl status pic-bed
```

---

## 📡 API

| 路径 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 首页 |
| `/upload` | POST | 上传图片（表单字段：`file`） |
| `/img/{年}/{月}/{文件名}` | GET | 预览图片 |
| `/img/{年}/{月}/{文件名}` | DELETE | 删除图片（需开启） |

**上传示例：**
```bash
curl -F "file=@test.jpg" http://localhost:8080/upload
```

**开启鉴权后：**
```bash
curl -F "file=@test.jpg" -H "Authorization: Bearer 你的密钥" http://localhost:8080/upload
```

---

## 🎯 PicList 配置

| 配置项 | 值 |
|--------|-----|
| 接口网址 | `http://你的IP:8080/upload` |
| 请求方法 | POST |
| 表单参数名 | `file` |
| 请求头 | `{}`（鉴权时填 `{"Authorization": "Bearer 你的密钥"}`） |
| 自定义前缀 | `http://你的IP:8080` |
| 返回 URL 路径 | `url` |

---

## 📥 手动下载

最新版本：<!-- version -->v1.0.5<!-- /version -->

前往 [Releases](https://github.com/renjiu13/Pic-bed/releases) 下载对应平台二进制。

| 平台 | 文件 |
|------|------|
| Linux amd64 | [pic-bed-linux-amd64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-linux-amd64) |
| Linux arm64 | [pic-bed-linux-arm64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-linux-arm64) |
| Linux armv7 | [pic-bed-linux-armv7](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-linux-armv7) |
| Windows | [pic-bed-windows-amd64.exe](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-windows-amd64.exe) |
| macOS Intel | [pic-bed-darwin-amd64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-darwin-amd64) |
| macOS Apple Silicon | [pic-bed-darwin-arm64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/pic-bed-darwin-arm64) |

校验和：[SHA256SUMS](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.5/SHA256SUMS)

---

## 🔒 安全说明

✅ 已实现：扩展名白名单、路径遍历防护、请求体硬限制、可选 Bearer 鉴权

❌ 未实现：魔数检测、上传频率限制、IP 白名单、图片压缩

---

## 📄 License

MIT
