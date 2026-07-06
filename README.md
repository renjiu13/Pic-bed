# Pic-Bed - 极轻量私有图床

专为低内存设备设计的单文件私有图床，完美支持玩客云、树莓派等 ARM32/ARM64 设备。

---

## ✨ 特性

- 🚀 **极低内存**：闲置 8~15MB，峰值不超 25MB
- 📦 **单文件部署**：纯静态编译，零依赖
- 🔧 **全平台支持**：Linux / Windows / macOS，共 6 种架构（amd64、arm64、armv7 等）
- 🔒 **安全加固**：路径防护、扩展名白名单、请求体大小限制、可选 Bearer 鉴权
- 🎯 **PicList 完美兼容**：支持自定义图床
- ⚙️ **丰富功能开关**：默认全关，按需开启
- 🔄 **在线更新**：一键检查并更新到最新版本
- 🖼️ **WebP 自动转换**：上传后可选自动转为 WebP（节省空间）
- 📝 **保留原始文件名**：可选不使用随机文件名

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

首次运行会自动生成 `config.json`，修改后**重启生效**（支持热重载）。

```json
{
  "port": 8080,
  "storage_dir": "./data",
  "max_size": 10,
  "api_key": "",
  "timeout": 60,
  "enable_log": false,
  "enable_delete": false,
  "enable_auto_clean": false,
  "keep_original_name": false,
  "enable_webp_convert": false,
  "webp_quality": 80,
  "keep_original_after_webp": false,
  "allowed_types": ["jpg", "jpeg", "png", "gif", "webp"],
  "auto_clean_hours": 720,
  "log_file": "./logs/app.log",
  "home_avatar_url": ""
}
```

**主要配置说明**：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `port` | 8080 | 监听端口 |
| `storage_dir` | `./data` | 图片存储目录 |
| `max_size` | 10 | 单文件大小上限（MB） |
| `api_key` | `""` | Bearer Token（空 = 关闭鉴权） |
| `timeout` | 60 | 请求超时时间（秒） |
| `enable_log` | false | 启用操作日志 |
| `enable_delete` | false | 允许 DELETE 删除图片 |
| `enable_auto_clean` | false | 启用自动清理 |
| `keep_original_name` | false | 保留原始文件名（否则随机） |
| `enable_webp_convert` | false | 上传后自动转为 WebP |
| `webp_quality` | 80 | WebP 转换质量（1-100） |
| `keep_original_after_webp` | false | WebP 转换后保留原图 |
| `allowed_types` | `["jpg",...]` | 允许上传的文件扩展名 |
| `auto_clean_hours` | 720 | 自动清理阈值（小时，默认 30 天） |
| `home_avatar_url` | `""` | 首页头像 URL（空则显示默认） |

---

## 💻 命令行用法

```bash
pic-bed -v           # 查看版本
pic-bed -h           # 查看帮助
pic-bed check-update # 检查更新
pic-bed update       # 在线更新
pic-bed              # 启动服务
```

---

## 📡 API

| 路径 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 首页 |
| `/upload` | GET/POST | 上传页面 + 上传接口（表单字段 `file`） |
| `/img/{年}/{月}/{文件名}` | GET | 预览图片（支持 WebP 自动重定向） |
| `/img/{年}/{月}/{文件名}` | DELETE | 删除图片（需开启 `enable_delete`） |

**上传示例**：

```bash
curl -F "file=@test.jpg" http://localhost:8080/upload
```

**带鉴权**：

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
| 请求头 | 鉴权时填 `{"Authorization": "Bearer 你的密钥"}` |
| 自定义前缀 | `http://你的IP:8080` |
| 返回 URL 路径 | `url` |

---

## 📄 License

MIT
