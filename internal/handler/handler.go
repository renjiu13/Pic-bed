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

// HandleUpload 处理上传（GET 返回上传页面，POST 处理上传）
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	ip := getClientIP(r)

	w.Header().Set("Access-Control-Allow-Origin", "*")

	// GET 请求返回上传页面
	if r.Method == http.MethodGet {
		renderUploadPage(w, cfg)
		return
	}

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

	if cfg.EnableWebPConvert {
		fullPath := filepath.Join(cfg.StorageDir, year, month, fileName)
		webpPath, convErr := storage.ConvertToWebP(fullPath, cfg.WebPQuality)
		if convErr == nil && webpPath != fullPath {
			fileName = filepath.Base(webpPath)
			relativeURL = fmt.Sprintf("/img/%s/%s/%s", year, month, fileName)
		}
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
        .footer a:hover {
            opacity: 0.6;
            transition: opacity 0.2s;
        }
        .upload-btn-mini {
            position: fixed;
            top: 20px;
            right: 20px;
            width: 28px;
            height: 28px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #999;
            text-decoration: none;
            font-size: 20px;
            line-height: 1;
            transition: color 0.2s ease;
            background: transparent;
            border: none;
            cursor: pointer;
        }
        .upload-btn-mini:hover {
            color: #000;
        }
    </style>
</head>
<body>
    <a href="/upload" class="upload-btn-mini" title="上传图片">⬆</a>
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
        <a href="https://github.com/renjiu13/Pic-bed" 
           target="_blank" 
           style="color: inherit; text-decoration: none;">
            Pic Bed
        </a> · 极轻量私有图床
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

// renderUploadPage 渲染上传页面
func renderUploadPage(w http.ResponseWriter, cfg config.Config) {
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=800">
    <title>图片上传 - Pic Bed</title>
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
            width: 100%;
        }
        h1 {
            font-size: 1.8rem;
            font-weight: 700;
            color: #000;
            margin-bottom: 30px;
        }
        .upload-area {
            border: 2px dashed #ddd;
            border-radius: 16px;
            padding: 40px 20px;
            margin-bottom: 20px;
            transition: border-color 0.2s;
            cursor: pointer;
        }
        .upload-area:hover {
            border-color: #ff6b9d;
        }
        .upload-area.dragover {
            border-color: #ff6b9d;
            background: #fff5f8;
        }
        .upload-icon {
            font-size: 48px;
            margin-bottom: 10px;
        }
        .upload-text {
            color: #666;
            font-size: 0.95rem;
            margin-bottom: 5px;
        }
        .upload-hint {
            color: #999;
            font-size: 0.8rem;
        }
        .file-input {
            display: none;
        }
        .api-key-section {
            margin-bottom: 20px;
            text-align: left;
        }
        .api-key-section label {
            display: block;
            font-size: 0.85rem;
            color: #666;
            margin-bottom: 6px;
        }
        .api-key-section input {
            width: 100%;
            padding: 10px 14px;
            border: 1px solid #ddd;
            border-radius: 8px;
            font-size: 0.9rem;
            outline: none;
            transition: border-color 0.2s;
        }
        .api-key-section input:focus {
            border-color: #ff6b9d;
        }
        .upload-btn {
            width: 100%;
            padding: 12px 24px;
            background: linear-gradient(135deg, #ff6b9d 0%, #ffa3c4 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        .upload-btn:hover {
            opacity: 0.9;
        }
        .upload-btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .result-area {
            margin-top: 24px;
            text-align: left;
            display: none;
        }
        .result-area.show {
            display: block;
        }
        .result-title {
            font-size: 0.9rem;
            font-weight: 600;
            color: #333;
            margin-bottom: 10px;
        }
        .result-preview {
            width: 100%;
            max-height: 200px;
            object-fit: contain;
            border-radius: 8px;
            margin-bottom: 12px;
            background: #f5f5f5;
        }
        .result-url {
            display: flex;
            gap: 8px;
            margin-bottom: 8px;
        }
        .result-url input {
            flex: 1;
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 0.8rem;
            font-family: monospace;
            background: #f9f9f9;
        }
        .copy-btn {
            padding: 8px 14px;
            background: #f0f0f0;
            border: none;
            border-radius: 6px;
            font-size: 0.8rem;
            cursor: pointer;
            transition: background 0.2s;
        }
        .copy-btn:hover {
            background: #e0e0e0;
        }
        .error-msg {
            color: #e74c3c;
            font-size: 0.85rem;
            margin-top: 10px;
        }
        .footer {
            margin-top: 40px;
            font-size: 0.85rem;
            color: #999;
        }
        .footer a {
            color: inherit;
            text-decoration: none;
        }
        .footer a:hover {
            opacity: 0.6;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>📤 图片上传</h1>
        
        {{if .NeedAPIKey}}
        <div class="api-key-section">
            <label for="apiKey">API Key</label>
            <input type="password" id="apiKey" placeholder="请输入 API Key">
        </div>
        {{end}}
        
        <div class="upload-area" id="uploadArea">
            <div class="upload-icon">🖼️</div>
            <div class="upload-text">点击或拖拽图片到此处上传</div>
            <div class="upload-hint">支持 {{.AllowedTypes}} · 最大 {{.MaxSize}}MB</div>
        </div>
        <input type="file" class="file-input" id="fileInput" accept="image/*">
        
        <button class="upload-btn" id="uploadBtn" disabled>选择图片后上传</button>
        
        <div class="result-area" id="resultArea">
            <div class="result-title">✅ 上传成功</div>
            <img class="result-preview" id="resultPreview" alt="预览">
            <div class="result-url">
                <input type="text" id="resultURL" readonly>
                <button class="copy-btn" onclick="copyURL()">复制</button>
            </div>
        </div>
        
        <div class="error-msg" id="errorMsg"></div>
    </div>
    
    <div class="footer">
        <a href="https://github.com/renjiu13/Pic-bed" target="_blank">Pic Bed</a> · 极轻量私有图床
    </div>

    <script>
        const uploadArea = document.getElementById('uploadArea');
        const fileInput = document.getElementById('fileInput');
        const uploadBtn = document.getElementById('uploadBtn');
        const resultArea = document.getElementById('resultArea');
        const resultPreview = document.getElementById('resultPreview');
        const resultURL = document.getElementById('resultURL');
        const errorMsg = document.getElementById('errorMsg');
        const apiKeyInput = document.getElementById('apiKey');
        
        let selectedFile = null;
        
        // 点击上传区域
        uploadArea.addEventListener('click', () => fileInput.click());
        
        // 拖拽
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });
        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            if (e.dataTransfer.files.length > 0) {
                handleFileSelect(e.dataTransfer.files[0]);
            }
        });
        
        // 文件选择
        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                handleFileSelect(e.target.files[0]);
            }
        });
        
        function handleFileSelect(file) {
            selectedFile = file;
            uploadBtn.disabled = false;
            uploadBtn.textContent = '上传 ' + file.name;
            errorMsg.textContent = '';
            resultArea.classList.remove('show');
        }
        
        // 上传
        uploadBtn.addEventListener('click', async () => {
            if (!selectedFile) {
                fileInput.click();
                return;
            }
            
            uploadBtn.disabled = true;
            uploadBtn.textContent = '上传中...';
            errorMsg.textContent = '';
            
            const formData = new FormData();
            formData.append('file', selectedFile);
            
            try {
                const headers = {};
                {{if .NeedAPIKey}}
                const apiKey = apiKeyInput.value.trim();
                if (!apiKey) {
                    throw new Error('请输入 API Key');
                }
                headers['Authorization'] = 'Bearer ' + apiKey;
                {{end}}
                
                const response = await fetch('/upload', {
                    method: 'POST',
                    headers: headers,
                    body: formData
                });
                
                const data = await response.json();
                
                if (data.success) {
                    resultPreview.src = data.url;
                    resultURL.value = window.location.origin + data.url;
                    resultArea.classList.add('show');
                    selectedFile = null;
                    fileInput.value = '';
                    uploadBtn.textContent = '选择图片后上传';
                    uploadBtn.disabled = true;
                } else {
                    throw new Error(data.message || '上传失败');
                }
            } catch (err) {
                errorMsg.textContent = '❌ ' + err.message;
                uploadBtn.textContent = '重新上传';
                uploadBtn.disabled = false;
            }
        });
        
        function copyURL() {
            resultURL.select();
            document.execCommand('copy');
            const btn = event.target;
            const originalText = btn.textContent;
            btn.textContent = '已复制';
            setTimeout(() => btn.textContent = originalText, 1500);
        }
    </script>
</body>
</html>`

	t, _ := template.New("upload").Parse(tmpl)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	needAPIKey := cfg.APIKey != ""
	allowedTypesStr := strings.Join(cfg.AllowedTypes, ", ")

	t.Execute(w, map[string]interface{}{
		"NeedAPIKey":   needAPIKey,
		"AllowedTypes": allowedTypesStr,
		"MaxSize":      cfg.MaxSize,
	})
}

// AuthMiddleware 鉴权中间件（GET 请求放行，用于访问上传页面）
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.Get()
		if cfg.APIKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// GET 请求放行（上传页面可公开访问，实际上传才需要鉴权）
		if r.Method == http.MethodGet {
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
