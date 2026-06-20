# Pic-Bed

极轻量私有图床，单文件运行，内存占用约 8~15MB。支持 Linux / Windows / macOS，兼容 PicList。

当前版本：<!-- version -->v1.0.0<!-- /version --> · [全部版本](https://github.com/renjiu13/Pic-bed/releases)

## 快速安装

复制对应平台命令，一行完成下载并启动。首次运行自动生成 `config.json`。

**Linux amd64**（云服务器 / PC）

```bash
mkdir -p ~/pic-bed && cd ~/pic-bed && curl -fsSL https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-linux-amd64 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

**Linux arm64**（树莓派 4/5、ARM 服务器）

```bash
mkdir -p ~/pic-bed && cd ~/pic-bed && curl -fsSL https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-linux-arm64 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

**Linux armv7**（玩客云、树莓派 3）

```bash
mkdir -p ~/pic-bed && cd ~/pic-bed && curl -fsSL https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-linux-armv7 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

**Windows**（PowerShell）

```powershell
irm https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-windows-amd64.exe -OutFile pic-bed.exe; .\pic-bed.exe
```

**macOS Apple Silicon**

```bash
mkdir -p ~/pic-bed && cd ~/pic-bed && curl -fsSL https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-darwin-arm64 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

**macOS Intel**

```bash
mkdir -p ~/pic-bed && cd ~/pic-bed && curl -fsSL https://github.com/renjiu13/Pic-bed/releases/download/<!-- version -->v1.0.0<!-- /version -->/pic-bed-darwin-amd64 -o pic-bed && chmod +x pic-bed && ./pic-bed
```

启动后访问 `http://服务器IP:8080/` 查看首页，`POST /upload` 上传图片。

## 配置

配置文件 `config.json`，完整示例见 [`examples/config.example.json`](examples/config.example.json)。

| 配置项 | 默认 | 说明 |
|--------|------|------|
| `port` | `8080` | 监听端口 |
| `max_size` | `10` | 单文件上限（MB） |
| `api_key` | `""` | 非空则上传需 `Bearer` 鉴权 |
| `enable_log` | `false` | 操作日志 |
| `enable_delete` | `false` | 允许 `DELETE /img/...` |
| `enable_auto_clean` | `false` | 自动清理过期文件 |
| `auto_clean_hours` | `720` | 清理阈值（小时） |
| `home_avatar_url` | `""` | 首页头像 URL |

修改后重启生效。

## API

```bash
# 上传
curl -F "file=@test.jpg" http://服务器IP:8080/upload
# → {"success":true,"url":"/img/2026/06/xxx.jpg","message":"upload success"}

# 预览
# GET http://服务器IP:8080/img/2026/06/xxx.jpg

# 删除（需 enable_delete: true）
curl -X DELETE http://服务器IP:8080/img/2026/06/xxx.jpg
```

## PicList

| 配置项 | 值 |
|--------|-----|
| 接口网址 | `http://服务器IP:8080/upload` |
| 请求方法 | POST |
| 表单参数名 | `file` |
| 自定义前缀 | `http://服务器IP:8080` |
| 返回 URL 路径 | `url` |

开启鉴权时，请求头填 `{"Authorization": "Bearer 你的密钥"}`。

## 开机自启（Linux）

```bash
sudo cp examples/pic-bed.service /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable --now pic-bed
```

## 自行编译

```bash
bash build.sh   # 产物在 bin/
```

## License

MIT
