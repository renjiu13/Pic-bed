package handler

import (
	"encoding/json"
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
    <meta name="viewport" content="width=800">
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

func respondJSON(w http.ResponseWriter, success bool, url interface{}, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"url":     url,
		"message": msg,
	})
}
