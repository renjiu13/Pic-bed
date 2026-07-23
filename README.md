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

**工作规则**：

- 支持 jpg / jpeg / png，压缩为 WebP 后删除原图
- gif / webp 自动跳过（不处理）
- **小图保护**：原图已 ≤ 目标大小时原样保留，不重编码
- 解码失败时保留原图文件

### 🔄 WebP 转换

将上传的图片异步转换为 WebP 格式。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `enable_webp_convert` | false | 上传后自动转为 WebP |
| `webp_quality` | 80 | WebP 转换质量（1-100） |
| `keep_original_after_webp` | false | 转换后保留原图 |

### 📦🔄 压缩与转换协同模式

固定压缩与 WebP 转换**可同时开启**，按优先级协同工作，而非互斥：

```
上传图片
  │
  ├─ 固定压缩开启？
  │   ├─ 是 → 尝试压缩到目标大小
  │   │       ├─ 成功（大图）→ 返回 webp，流程结束
  │   │       ├─ 跳过（小图/gif/webp）→ 继续 ↓
  │   │       └─ 失败（损坏文件）→ 流程结束，不兜底
  │   └─ 否 → 继续 ↓
  │
  ├─ WebP 转换开启 且 固定压缩未处理？
  │   └─ 是 → 异步转换为 WebP
  └─ 否 → 原样返回
```

| 场景 | 固定压缩 | WebP 转换 | 最终结果 |
|------|---------|-----------|---------|
| 大图（jpg/png） | 二分压到目标大小 | 跳过 | 精确控大小的 webp |
| 小图（已 ≤ 目标） | 跳过 | 兜底转换 | 转为 webp |
| gif / webp | 跳过 | 跳过 | 原样保留 |
| 损坏文件 | 失败 | 跳过 | 保留原图 |
| 只开固定压缩 | 正常工作 | 不执行 | — |
| 只开 WebP 转换 | 不执行 | 正常工作 | — |

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
| `/img/{年}/{月}/{文件名}` | DELETE | 删除图片（需开启 `enable_delete`） |

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
