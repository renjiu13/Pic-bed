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
- 🖼️ **智能图片处理**：固定大小压缩 + WebP 转换，二者可协同工作
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
  "timeout": 30,
  "allowed_types": ["jpg", "jpeg", "png", "gif", "webp"],

  "enable_log": false,
  "log_file": "./pic-bed.log",
  "enable_delete": false,
  "enable_auto_clean": false,
  "auto_clean_hours": 720,
  "keep_original_name": false,

  "enable_fixed_size_compression": false,
  "target_file_size_kb": 500,
  "compression_quality_start": 90,

  "enable_webp_convert": false,
  "webp_quality": 80,
  "keep_original_after_webp": false,

  "home_avatar_url": ""
}
```

### 基础配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `port` | 8080 | 监听端口 |
| `storage_dir` | `./data` | 图片存储目录 |
| `max_size` | 10 | 单文件大小上限（MB） |
| `api_key` | `""` | Bearer Token（空 = 关闭鉴权） |
| `timeout` | 30 | 请求超时时间（秒） |
| `allowed_types` | `["jpg",...]` | 允许上传的文件扩展名 |
| `keep_original_name` | false | 保留原始文件名（否则随机） |
| `home_avatar_url` | `""` | 首页头像 URL（空则显示默认 emoji） |

### 日志与清理

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `enable_log` | false | 启用操作日志 |
| `log_file` | `./pic-bed.log` | 日志文件路径 |
| `enable_delete` | false | 允许 DELETE 删除图片 |
| `enable_auto_clean` | false | 启用自动清理 |
| `auto_clean_hours` | 720 | 自动清理阈值（小时，默认 30 天） |

### 📦 固定大小压缩

通过二分查找 WebP 质量参数，将图片压缩到目标大小。**仅调整质量，不缩放图像**，压缩成功后自动删除原图。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `enable_fixed_size_compression` | false | 开启固定大小压缩 |
| `target_file_size_kb` | 500 | 目标大小（KB） |
| `compression_quality_start` | 90 | 起始（最大）质量 1-100 |
| `compress_queue_size` | 100 | 异步压缩队列大小（0=同步阻塞，>0=异步不阻塞上传） |

**工作规则**：

- 支持 jpg / jpeg / png，压缩为 WebP 后删除原图
- gif / webp 自动跳过（不处理）
- **小图保护**：原图已 ≤ 目标大小时原样保留，不重编码
- 解码失败时保留原图文件
- **异步模式**（`compress_queue_size > 0`）：上传立即返回原图 URL，后台串行压缩（弱设备友好）。压缩完成后访问原图 URL 自动重定向到 WebP
- **同步模式**（`compress_queue_size = 0`）：上传时直接压缩，请求阻塞直到完成

### 🔄 WebP 转换

将上传的图片异步转换为 WebP 格式。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `enable_webp_convert` | false | 上传后自动转为 WebP |
| `webp_quality` | 80 | WebP 转换质量（1-100） |
| `keep_original_after_webp` | false | 转换后保留原图 |

### 🛡️ 内存守护

专为弱内存设备（玩客云、树莓派等）设计。定期检查进程内存占用，超过阈值时优雅关闭服务并自动重启，防止 OOM。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `enable_memory_guard` | false | 启用内存守护 |
| `memory_limit_mb` | 200 | 内存上限（MB），超过则重启 |
| `memory_check_interval` | 30 | 检查间隔（秒） |

**工作机制**：

1. 每隔 `memory_check_interval` 秒检查进程内存（`runtime.MemStats.Sys`）
2. 超过 `memory_limit_mb` 时触发：优雅关闭 HTTP 服务 → 等待压缩队列完成 → 释放资源 → 自动重启
3. Linux 使用 `syscall.Exec` 原地重启（PID 不变，systemd 友好）；其他平台启动新进程后退出

**弱设备推荐配置**：

```json
{
  "compress_queue_size": 50,
  "enable_memory_guard": true,
  "memory_limit_mb": 150,
  "memory_check_interval": 20
}
```

### 📦🔄 压缩与转换协同模式

固定压缩与 WebP 转换**可同时开启**，按优先级协同工作，而非互斥：

```
上传图片 → 保存原图 → 立即返回原图 URL（请求结束）
  │
  ├─ 固定压缩开启？
  │   ├─ 异步模式（queue_size > 0）→ 入队，后台串行压缩
  │   │   ├─ 成功（大图）→ 删原图，生成 webp（访问原图 URL 自动重定向）
  │   │   ├─ 跳过（小图/gif/webp）→ 原图保留
  │   │   └─ 失败（损坏文件）→ 原图保留
  │   ├─ 同步模式（queue_size = 0）→ 直接压缩（同上结果）
  │   │   └─ 跳过时 → 继续 ↓
  │   └─ 否 → 继续 ↓
  │
  ├─ WebP 转换开启 且 固定压缩未处理？
  │   └─ 是 → 异步转换为 WebP
  └─ 否 → 原样返回
```

| 场景 | 固定压缩 | WebP 转换 | 上传返回 | 最终结果 |
|------|---------|-----------|---------|---------|
| 大图（jpg/png） | 异步压缩 | 跳过 | `.webp` URL | 压缩完成直接命中；未完成时回退原图 |
| 小图（已 ≤ 目标） | 跳过 | 兜底转换 | 原图/webp URL | 转为 webp |
| gif / webp | 跳过 | 跳过 | 原图 URL | 原样保留 |
| 损坏文件 | 失败 | 跳过 | `.webp` URL | 访问时回退到原图 |
| 只开固定压缩 | 正常工作 | 不执行 | `.webp` URL | 异步压缩完成直接命中 |
| 只开 WebP 转换 | 不执行 | 正常工作 | webp URL | — |

> **异步压缩优先返回 `.webp` URL**：jpg/png 上传后立即返回 `.webp` 后缀链接。压缩完成前访问该 URL，`HandleImage` 自动回退到原图；压缩完成后直接命中 WebP。gif/webp 保持原后缀。

> `keep_original_after_webp` 仅在 WebP 转换路径生效；固定压缩路径始终删除原图。

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
| `/img/{年}/{月}/{文件名}` | DELETE | 删除图片（需开启 `enable_delete`，配置 `api_key` 时需鉴权） |

**上传示例**：

```bash
curl -F "file=@test.jpg" http://localhost:8080/upload
```

**带鉴权**：

```bash
curl -F "file=@test.jpg" -H "Authorization: Bearer 你的密钥" http://localhost:8080/upload
```

**响应格式**：

```json
{
  "success": true,
  "url": "/img/2026/07/abc123.webp",
  "message": "upload success"
}
```

**删除示例**：

```bash
# 无鉴权
curl -X DELETE http://localhost:8080/img/2026/07/abc123.webp

# 带鉴权
curl -X DELETE -H "Authorization: Bearer 你的密钥" http://localhost:8080/img/2026/07/abc123.webp
```

> 删除时支持联动清理：删除 `.webp` 时自动删除对应的原图，删除原图时自动清理对应的 `.webp`。压缩未完成时删除 `.webp` 会自动回退删除原图。

---

## 🎯 PicList 配置

### 上传配置

| 配置项 | 值 |
|--------|-----|
| 接口网址 | `http://你的IP:8080/upload` |
| 请求方法 | POST |
| 表单参数名 | `file` |
| 请求头 | 鉴权时填 `{"Authorization": "Bearer 你的密钥"}` |
| 自定义前缀 | `http://你的IP:8080` |
| 返回 URL 路径 | `url` |

### 删除配置

PicList 删除时通过本地服务 `POST http://127.0.0.1:36677/delete` 调用，需在 PicList 自定义图床设置中配置删除接口：

| 配置项 | 值 |
|--------|-----|
| 删除方法 | DELETE |
| 删除地址 | 图片 URL 本身（如 `http://你的IP:8080/img/2026/07/abc123.webp`） |
| 请求头 | 鉴权时填 `{"Authorization": "Bearer 你的密钥"}` |

> Pic-bed 需开启 `enable_delete: true`。配置了 `api_key` 时，DELETE 请求也必须携带 `Authorization: Bearer 你的密钥` 头。

---

## 📄 License

MIT
