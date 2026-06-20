package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pic-bed/pic-bed/internal/config"
	"github.com/pic-bed/pic-bed/internal/logger"
	"github.com/pic-bed/pic-bed/internal/security"
	"github.com/pic-bed/pic-bed/internal/storage"
)

// UploadResponse 上传响应
type UploadResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

// ListResponse 文件列表响应
type ListResponse struct {
	Success bool               `json:"success"`
	Files   []storage.FileInfo `json:"files"`
	Count   int                `json:"count"`
}

// getClientIP 获取访问者真实 IP（优先读反向代理头，回退 RemoteAddr）
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// HandleUpload 处理上传
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	ip := getClientIP(r)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, false, "", "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	maxBytes := int64(cfg.MaxSize) * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		logger.LogError(ip, "", "parse form failed: "+err.Error())
		respondJSON(w, false, "", "file too large or parse failed", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		logger.LogError(ip, "", "get file failed: "+err.Error())
		respondJSON(w, false, "", "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 基于扩展名验证文件类型（移除魔数验证）
	ext, err := security.ValidateFileType(fileHeader.Filename, cfg.AllowedTypes)
	if err != nil {
		logger.LogError(ip, fileHeader.Filename, "invalid format: "+err.Error())
		respondJSON(w, false, "", err.Error(), http.StatusBadRequest)
		return
	}

	year, month := security.GetYearMonth()
	fileName := security.GenerateFileName(fileHeader.Filename, ext, cfg.KeepOriginalName)

	relativeURL, err := storage.SaveFile(file, cfg.StorageDir, year, month, fileName)
	if err != nil {
		logger.LogError(ip, fileName, "save failed: "+err.Error())
		respondJSON(w, false, "", "save failed", http.StatusInternalServerError)
		return
	}

	logger.LogUpload(ip, fileName, relativeURL, fileHeader.Size)
	respondJSON(w, true, relativeURL, "upload success", http.StatusOK)
}

// HandleImage 统一处理图片相关请求（GET预览 / DELETE删除）
// 解决Go标准库同一路径不能多次注册的问题
func HandleImage(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	ip := getClientIP(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	pathSuffix := strings.TrimPrefix(r.URL.Path, "/img/")
	parts := strings.SplitN(pathSuffix, "/", 3)
	if len(parts) != 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	year, month, fileName := parts[0], parts[1], parts[2]

	// 路径安全校验
	if strings.ContainsAny(year, "./\\") || strings.ContainsAny(month, "./\\") || strings.ContainsAny(fileName, "/\\") {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	fullPath := filepath.Join(cfg.StorageDir, year, month, fileName)
	if !security.IsPathSafe(cfg.StorageDir, fullPath) {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	// 根据HTTP方法分发处理
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// GET/HEAD: 预览图片
		logger.LogAccess(ip, fileName)
		http.ServeFile(w, r, fullPath)

	case http.MethodDelete:
		// DELETE: 删除图片（需开启开关）
		w.Header().Set("Content-Type", "application/json")
		if !cfg.EnableDelete {
			respondJSON(w, false, "", "delete disabled", http.StatusForbidden)
			return
		}
		if err := storage.DeleteFile(cfg.StorageDir, year, month, fileName); err != nil {
			logger.LogError(ip, fileName, "delete failed: "+err.Error())
			respondJSON(w, false, "", err.Error(), http.StatusNotFound)
			return
		}
		logger.LogDelete(ip, fileName)
		respondJSON(w, true, "", "delete success", http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleHome 首页欢迎页面（Jubilee风格）
func HandleHome(w http.ResponseWriter, r *http.Request) {
	// 只处理根路径
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cfg := config.Get()
	clientIP := getClientIP(r)
	avatarURL := cfg.HomeAvatarURL
	
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to awang! :)</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 40px 20px;
            background: #fff;
        }
        .container {
            text-align: center;
            max-width: 500px;
        }
        h1 {
            font-size: 2rem;
            font-weight: 700;
            color: #000;
            margin-bottom: 20px;
        }
        .avatar {
            width: 300px;
            height: 300px;
            margin: 0 auto 20px;
            border-radius: 20px;
            overflow: hidden;
            display: flex;
            align-items: center;
            justify-content: center;
            position: relative;
        }
        .avatar.default {
            background: linear-gradient(135deg, #ff6b9d 0%, #ffa3c4 100%);
            font-size: 120px;
        }
        .avatar.default::before {
            content: "✨";
            position: absolute;
            top: 20px;
            left: 30px;
            font-size: 40px;
            animation: twinkle 2s ease-in-out infinite;
        }
        .avatar.default::after {
            content: "⭐";
            position: absolute;
            top: 50px;
            right: 40px;
            font-size: 25px;
            animation: twinkle 2s ease-in-out infinite 0.5s;
        }
        .avatar img {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }
        .avatar .face {
            font-size: 150px;
            z-index: 1;
        }
        @keyframes twinkle {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.5; transform: scale(0.8); }
        }
        p {
            font-size: 1.1rem;
            color: #333;
            margin-bottom: 10px;
            line-height: 1.6;
        }
        .ip {
            font-size: 1rem;
            color: #666;
            font-family: monospace;
            margin-top: 10px;
        }
        .footer {
            margin-top: 40px;
            font-size: 0.85rem;
            color: #999;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to awang! :)</h1>
        
        {{if .AvatarURL}}
        <div class="avatar">
            <img src="{{.AvatarURL}}" alt="avatar">
        </div>
        {{else}}
        <div class="avatar default">
            <span class="face">😎</span>
        </div>
        {{end}}
        <p>I serve photos. You'll need a URL.</p>
        <p></p>
        <p class="ip">{{.ClientIP}}</p>
    </div>
    <div class="footer">
        Pic Bed · 极轻量私有图床
    </div>
</body>
</html>`
	t, _ := template.New("home").Parse(tmpl)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, map[string]interface{}{
		"ClientIP":  clientIP,
		"AvatarURL": avatarURL,
	})
}

// HandleFileList 文件列表页面
func HandleFileList(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	if !cfg.EnableFileList {
		http.Error(w, "file list disabled", http.StatusForbidden)
		return
	}

	if r.URL.Query().Get("format") == "json" {
		files, err := storage.ListFiles(cfg.StorageDir)
		if err != nil {
			respondJSON(w, false, nil, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListResponse{
			Success: true,
			Files:   files,
			Count:   len(files),
		})
		return
	}

	// HTML 页面 - Apple 风格
	files, _ := storage.ListFiles(cfg.StorageDir)
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>图片库 · Pic Bed</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif;
            background: #f5f5f7;
            color: #1d1d1f;
            min-height: 100vh;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 60px 24px 80px;
        }

        /* 头部 */
        .header {
            margin-bottom: 48px;
        }

        .header h1 {
            font-size: 40px;
            font-weight: 700;
            letter-spacing: -0.02em;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #1d1d1f 0%, #434344 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .header p {
            font-size: 17px;
            color: #86868b;
            font-weight: 400;
        }

        /* 统计栏 */
        .stats {
            display: flex;
            gap: 24px;
            margin-bottom: 40px;
            flex-wrap: wrap;
        }

        .stat-card {
            flex: 1;
            min-width: 160px;
            background: #ffffff;
            border-radius: 18px;
            padding: 24px;
            box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
            transition: all 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
        }

        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
        }

        .stat-card .number {
            font-size: 32px;
            font-weight: 700;
            letter-spacing: -0.01em;
            color: #1d1d1f;
            margin-bottom: 4px;
        }

        .stat-card .label {
            font-size: 14px;
            color: #86868b;
            font-weight: 500;
        }

        /* 图片网格 */
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
        }

        .card {
            background: #ffffff;
            border-radius: 16px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
            transition: all 0.4s cubic-bezier(0.25, 0.1, 0.25, 1);
            cursor: pointer;
        }

        .card:hover {
            transform: translateY(-6px) scale(1.02);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.12);
        }

        .card .img-wrapper {
            width: 100%;
            aspect-ratio: 1;
            overflow: hidden;
            background: #f5f5f7;
            position: relative;
        }

        .card .img-wrapper::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(to bottom, transparent 60%, rgba(0,0,0,0.15) 100%);
            opacity: 0;
            transition: opacity 0.3s ease;
        }

        .card:hover .img-wrapper::after {
            opacity: 1;
        }

        .card img {
            width: 100%;
            height: 100%;
            object-fit: cover;
            transition: transform 0.5s cubic-bezier(0.25, 0.1, 0.25, 1);
        }

        .card:hover img {
            transform: scale(1.08);
        }

        .card .info {
            padding: 16px;
        }

        .card .filename {
            font-size: 14px;
            color: #1d1d1f;
            font-weight: 600;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-bottom: 6px;
            letter-spacing: -0.01em;
        }

        .card .meta {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            color: #86868b;
            font-weight: 500;
        }

        .card .actions {
            display: flex;
            gap: 8px;
            margin-top: 14px;
            opacity: 0;
            transform: translateY(4px);
            transition: all 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
        }

        .card:hover .actions {
            opacity: 1;
            transform: translateY(0);
        }

        .btn {
            flex: 1;
            padding: 8px 14px;
            border: none;
            border-radius: 10px;
            font-size: 13px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s ease;
            font-family: inherit;
            letter-spacing: -0.01em;
        }

        .btn-copy {
            background: #0071e3;
            color: white;
        }

        .btn-copy:hover {
            background: #0077ed;
            transform: scale(1.02);
        }

        .btn-copy:active {
            transform: scale(0.98);
            background: #006edb;
        }

        /* 空状态 */
        .empty {
            text-align: center;
            padding: 120px 20px;
        }

        .empty .icon {
            font-size: 64px;
            margin-bottom: 24px;
            opacity: 0.6;
        }

        .empty h2 {
            font-size: 24px;
            font-weight: 600;
            color: #1d1d1f;
            margin-bottom: 8px;
            letter-spacing: -0.01em;
        }

        .empty p {
            font-size: 15px;
            color: #86868b;
        }

        /* Toast 提示 */
        .toast {
            position: fixed;
            bottom: 40px;
            left: 50%;
            transform: translateX(-50%) translateY(100px);
            background: rgba(29, 29, 31, 0.92);
            backdrop-filter: saturate(180%) blur(20px);
            -webkit-backdrop-filter: saturate(180%) blur(20px);
            color: white;
            padding: 14px 28px;
            border-radius: 100px;
            font-size: 14px;
            font-weight: 500;
            opacity: 0;
            transition: all 0.4s cubic-bezier(0.25, 0.1, 0.25, 1);
            z-index: 1000;
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
            letter-spacing: -0.01em;
        }

        .toast.show {
            transform: translateX(-50%) translateY(0);
            opacity: 1;
        }

        /* 页脚 */
        .footer {
            text-align: center;
            color: #86868b;
            margin-top: 80px;
            font-size: 13px;
            font-weight: 500;
        }

        /* 响应式 */
        @media (max-width: 640px) {
            .container {
                padding: 40px 16px 60px;
            }
            .header h1 {
                font-size: 32px;
            }
            .stats {
                gap: 12px;
            }
            .stat-card {
                padding: 16px;
                min-width: 140px;
            }
            .stat-card .number {
                font-size: 24px;
            }
            .grid {
                grid-template-columns: repeat(2, 1fr);
                gap: 12px;
            }
            .card .actions {
                opacity: 1;
                transform: none;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>图片库</h1>
            <p>你所有的图片，一目了然。</p>
        </div>

        <div class="stats">
            <div class="stat-card">
                <div class="number">{{.Count}}</div>
                <div class="label">张图片</div>
            </div>
            <div class="stat-card">
                <div class="number">{{.TotalSize}}</div>
                <div class="label">总大小</div>
            </div>
        </div>

        {{if .Files}}
        <div class="grid">
            {{range .Files}}
            <div class="card" onclick="copyUrl('{{.Path}}')">
                <div class="img-wrapper">
                    <img src="{{.Path}}" alt="{{.Name}}" loading="lazy">
                </div>
                <div class="info">
                    <div class="filename" title="{{.Name}}">{{.Name}}</div>
                    <div class="meta">
                        <span>{{.SizeStr}}</span>
                        <span>{{.TimeStr}}</span>
                    </div>
                    <div class="actions">
                        <button class="btn btn-copy" onclick="event.stopPropagation(); copyUrl('{{.Path}}')">
                            复制链接
                        </button>
                    </div>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <div class="empty">
            <div class="icon">🖼️</div>
            <h2>还没有图片</h2>
            <p>上传你的第一张图片吧</p>
        </div>
        {{end}}

        <div class="footer">
            Pic Bed · 极轻量私有图床
        </div>
    </div>

    <div class="toast" id="toast">✓ 链接已复制</div>

    <script>
        function copyUrl(path) {
            const url = window.location.origin + path;
            navigator.clipboard.writeText(url).then(() => {
                showToast();
            }).catch(() => {
                const input = document.createElement('input');
                input.value = url;
                document.body.appendChild(input);
                input.select();
                document.execCommand('copy');
                document.body.removeChild(input);
                showToast();
            });
        }

        function showToast() {
            const toast = document.getElementById('toast');
            toast.classList.add('show');
            setTimeout(() => {
                toast.classList.remove('show');
            }, 2000);
        }
    </script>
</body>
</html>`

	// 预处理文件数据
	type fileView struct {
		Name    string
		Path    string
		SizeStr string
		TimeStr string
	}

	var totalSize int64
	var fileViews []fileView
	for _, f := range files {
		totalSize += f.Size
		fileViews = append(fileViews, fileView{
			Name:    f.Name,
			Path:    f.Path,
			SizeStr: formatSize(f.Size),
			TimeStr: f.ModTime.Format("2006-01-02 15:04"),
		})
	}

	t, _ := template.New("list").Parse(tmpl)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, map[string]interface{}{
		"Files":     fileViews,
		"Count":     len(files),
		"TotalSize": formatSize(totalSize),
	})
}

// AuthMiddleware 鉴权中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.Get()
		if cfg.APIKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondJSON(w, false, "", "missing Authorization", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != cfg.APIKey {
			respondJSON(w, false, "", "invalid API key", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func respondJSON(w http.ResponseWriter, success bool, url interface{}, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"url":     url,
		"message": msg,
	})
}
