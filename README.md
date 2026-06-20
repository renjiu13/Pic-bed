# Pic-Bed - 极轻量私有图床

专为低内存设备设计的单文件私有图床，支持玩客云等 ARM32 设备。纯 Go 编写，`CGO_ENABLED=0` 静态编译，零外部依赖。

## ✨ 核心特性

- 🚀 **极低内存占用**：闲置约 8~15MB，峰值不超过 25MB
- 📦 **单文件部署**：下载二进制 + `config.json` 即可运行
- 🔧 **多平台支持**：Linux（amd64 / arm64 / armv7）、Windows、macOS
- 🔒 **基础安全**：扩展名白名单、路径遍历防护、请求体大小硬限制、可选 Bearer 鉴权
- 🎯 **PicList 兼容**：支持 PicList 自定义图床配置
- ⚙️ **按需开启**：日志、删除、自动清理等功能默认关闭，手动启用

## 📥 下载

最新版本：<!-- version -->v1.0.0<!-- /version -->

前往 [Releases 页面](https://github.com/renjiu13/Pic-bed/releases) 下载，或直接点击下表：

| 平台 | 文件 | 适用设备 |
|------|------|----------|
| Linux amd64 | [pic-bed-linux-amd64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-linux-amd64) | 云服务器、PC |
| Linux arm64 | [pic-bed-linux-arm64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-linux-arm64) | 树莓派 4/5、ARM 服务器 |
| Linux armv7 | [pic-bed-linux-armv7](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-linux-armv7) | **玩客云**、树莓派 3 |
| Windows amd64 | [pic-bed-windows-amd64.exe](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-windows-amd64.exe) | Windows x86-64 |
| macOS Intel | [pic-bed-darwin-amd64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-darwin-amd64) | Mac Intel |
| macOS Apple Silicon | [pic-bed-darwin-arm64](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-darwin-arm64) | Mac M 系列 |
| SHA256 校验和 | [SHA256SUMS](https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/SHA256SUMS) | 文件完整性校验 |

### 校验下载文件

**Linux / macOS：**

```bash
sha256sum -c SHA256SUMS
```

**Windows (PowerShell)：**

```powershell
Get-FileHash .\pic-bed-windows-amd64.exe -Algorithm SHA256
```

将输出结果与 `SHA256SUMS` 文件中的对应行对比。

## 📡 路由一览

| 路径 | 方法 | 说明 | 前置条件 |
|------|------|------|----------|
| `/` | GET | 首页欢迎页，支持自定义头像 | 无 |
| `/upload` | POST | 上传图片，表单字段名 `file` | 无；`api_key` 非空时需 Bearer 鉴权 |
| `/img/{年}/{月}/{文件名}` | GET / HEAD | 预览图片 | 无 |
| `/img/{年}/{月}/{文件名}` | DELETE | 删除图片 | `enable_delete: true` |

> 图片路径统一为 `/img/年/月/文件名`，例如 `/img/2026/06/abc123.jpg`

## 🔧 完整配置说明

首次运行会在工作目录自动生成 `config.json`。也可参考 `examples/config.example.json`。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `port` | int | `8080` | HTTP 监听端口 |
| `storage_dir` | string | `"./data"` | 图片存储目录 |
| `max_size` | int | `10` | 单文件大小上限（MB） |
| `api_key` | string | `""` | Bearer Token；**空字符串表示关闭鉴权** |
| `timeout` | int | `30` | 读写超时（秒） |
| `enable_log` | bool | `false` | 是否记录操作日志 |
| `enable_delete` | bool | `false` | 是否允许 DELETE 删除图片 |
| `enable_auto_clean` | bool | `false` | 是否启用自动清理 |
| `keep_original_name` | bool | `false` | 保留原始文件名 + 随机后缀 |
| `allowed_types` | []string | `jpg,jpeg,png,gif,webp` | 允许上传的扩展名白名单 |
| `auto_clean_hours` | int | `720` | 自动清理阈值（小时），默认 720 = 30 天 |
| `log_file` | string | `"./pic-bed.log"` | 日志文件路径 |
| `home_avatar_url` | string | `""` | 首页头像 URL，空则显示默认 emoji |

示例配置：

```json
{
  "port": 8080,
  "storage_dir": "./data",
  "max_size": 10,
  "api_key": "",
  "timeout": 30,
  "enable_log": false,
  "enable_delete": false,
  "enable_auto_clean": false,
  "keep_original_name": false,
  "allowed_types": ["jpg", "jpeg", "png", "gif", "webp"],
  "auto_clean_hours": 720,
  "log_file": "./pic-bed.log",
  "home_avatar_url": ""
}
```

修改配置后需重启服务生效。

### 首页头像

访问 `http://服务器IP:8080/` 查看欢迎页。通过 `home_avatar_url` 自定义头像：

- 留空：显示默认 emoji 头像
- 网络图片：`"home_avatar_url": "https://example.com/avatar.jpg"`
- 本地上传图片：`"home_avatar_url": "/img/2026/06/xxx.jpg"`

### 自动清理

1. 设置 `enable_auto_clean: true`
2. 按需调整 `auto_clean_hours`（默认 `720`，即 30 天）
3. 重启服务后，后台每小时检查一次，删除超过阈值的文件

> 注意：时间单位为**小时**（`auto_clean_hours`），不是天。旧版文档中的 `auto_clean_days` 已废弃。

## 🔒 安全说明

当前已实现的安全措施：

| 措施 | 实现方式 |
|------|----------|
| 文件类型限制 | 扩展名白名单（`allowed_types`） |
| 路径遍历防护 | `IsPathSafe` 校验，禁止跳出存储目录 |
| 请求体限制 | `http.MaxBytesReader` 硬限制上传大小 |
| 可选鉴权 | `api_key` 非空时，上传接口需 `Authorization: Bearer <key>` |
| 文件名消毒 | 保留原名模式下过滤危险字符 |

当前**未实现**（请勿在文档中误认为已具备）：

- ❌ 魔数 / MIME 真实类型检测（仅靠扩展名，可被伪装文件绕过）
- ❌ 图片压缩（`enable_compress` 不存在，无此配置项）
- ❌ 上传频率限制
- ❌ IP 白名单

## 📡 API 示例

### 上传图片

```bash
curl -F "file=@test.jpg" http://服务器IP:8080/upload
```

响应：

```json
{"success":true,"url":"/img/2026/06/171846123456789abcdef.jpg","message":"upload success"}
```

### 预览图片

```
GET http://服务器IP:8080/img/2026/06/文件名.jpg
```

### 删除图片（需 `enable_delete: true`）

```bash
curl -X DELETE http://服务器IP:8080/img/2026/06/文件名.jpg
```

## 🔐 开启鉴权

1. 在 `config.json` 中设置 `api_key` 为自定义密钥
2. 上传时携带请求头：`Authorization: Bearer 你的密钥`
3. PicList 请求头改为：`{"Authorization": "Bearer 你的密钥"}`

## 🎯 PicList 配置

| 配置项 | 值 |
|--------|-----|
| 接口网址 | `http://服务器IP:8080/upload` |
| 请求方法 | POST |
| 表单参数名 | `file` |
| 请求头 | `{}`（开启鉴权时填 Bearer Token） |
| 请求体 | `{}` |
| 自定义前缀 | `http://服务器IP:8080` |
| 网站路径 | 留空 |
| 返回数据 URL 路径 | `url` |

> PicList 部分版本不支持 `$.url` 写法，直接填 `url` 即可。

## 🚀 快速部署

### Linux（玩客云 ARMv7）

```bash
mkdir -p /opt/pic-bed && cd /opt/pic-bed
wget https://github.com/renjiu13/Pic-bed/releases/download/v1.0.0/pic-bed-linux-armv7 -O pic-bed
chmod +x pic-bed
./pic-bed   # 首次运行自动生成 config.json
```

### Windows

```powershell
# 从 Releases 下载 pic-bed-windows-amd64.exe 到工作目录
.\pic-bed-windows-amd64.exe
# 同目录生成 config.json 后即可使用
```

### systemd 开机自启（Linux）

```bash
cp examples/pic-bed.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now pic-bed
systemctl status pic-bed
journalctl -u pic-bed -f
```

## 🔨 自行编译

程序入口为 `./cmd/pic-bed`（`build.sh` 编译此路径）。

### 一键多架构编译（Linux / macOS 终端）

```bash
chmod +x build.sh
./build.sh
```

产物：

| 文件名 | 平台 |
|--------|------|
| `pic-bed-linux-amd64` | Linux x86-64 |
| `pic-bed-linux-arm64` | Linux ARM64 |
| `pic-bed-linux-armv7` | Linux ARMv7（玩客云） |
| `pic-bed-windows-amd64.exe` | Windows x86-64 |
| `pic-bed-darwin-amd64` | macOS Intel |
| `pic-bed-darwin-arm64` | macOS Apple Silicon |

### 单独交叉编译

```bash
# Windows exe
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o pic-bed.exe ./cmd/pic-bed

# 本机直接运行（开发调试）
go run ./cmd/pic-bed
```

## 🚢 自动发布（维护者）

推送 `v*` 标签后，GitHub Actions 会自动：

1. 编译 6 个平台的二进制
2. UPX 压缩
3. 生成 `SHA256SUMS` 校验和
4. 创建 GitHub Release 并上传产物
5. 自动更新 README 中的版本号和下载链接

```bash
git tag v1.0.0
git push origin v1.0.0
```

也可在 GitHub Actions 页面手动触发（`workflow_dispatch`）。

## 📝 日志格式

开启 `enable_log` 后，日志写入 `log_file` 指定路径：

```
2026/06/15 22:00:00 [UPLOAD] IP:192.168.1.100 File:xxx.jpg - URL:/img/2026/06/xxx.jpg Size:123456 bytes
2026/06/15 22:00:01 [DELETE] IP:192.168.1.100 File:xxx.jpg - File deleted
2026/06/15 22:00:02 [ACCESS] IP:192.168.1.100 File:xxx.jpg - File accessed
```

## 🏗️ 项目结构

```
pic-bed/
├── cmd/pic-bed/          # 主程序入口（编译入口）
├── internal/
│   ├── config/           # 配置加载
│   ├── handler/          # HTTP 路由处理
│   ├── logger/           # 操作日志
│   ├── security/         # 扩展名校验、路径防护
│   └── storage/          # 文件存储与自动清理
├── examples/
│   ├── config.example.json
│   └── pic-bed.service
├── .github/workflows/
│   └── release.yml       # 自动构建发布
├── build.sh              # 多架构编译脚本
├── go.mod
└── README.md
```

## 📊 性能参考

- 闲置内存：约 8~12MB
- 峰值内存：约 15~25MB
- 二进制体积：约 6~7MB（`-ldflags="-s -w"` 编译）
- 并发：Go 原生协程，适合中小流量私有图床

## 📄 License

MIT
