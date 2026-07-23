package handler

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pic-bed/pic-bed/internal/compress"
	"github.com/pic-bed/pic-bed/internal/config"
	"github.com/pic-bed/pic-bed/internal/logger"
	"github.com/pic-bed/pic-bed/internal/security"
	"github.com/pic-bed/pic-bed/internal/storage"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	homeTemplate   *template.Template
	uploadTemplate *template.Template
	templateOnce   sync.Once
)

// compressQueue 异步压缩队列（由 main.go 调用 SetCompressQueue 注入）
// 为 nil 时走同步压缩
var compressQueue *compress.Queue

// SetCompressQueue 注入异步压缩队列（main.go 启动时调用）
func SetCompressQueue(q *compress.Queue) {
	compressQueue = q
}

func initTemplates() {
	templateOnce.Do(func() {
		homeTemplate = template.Must(template.ParseFS(templateFS, "templates/home.html"))
		uploadTemplate = template.Must(template.ParseFS(templateFS, "templates/upload.html"))
	})
}

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

	fullPath := filepath.Join(cfg.StorageDir, year, month, fileName)

	// webpHandled 标记是否已安排图片处理，用于决定是否需要 WebP 转换兜底
	webpHandled := false

	// 📦 固定大小压缩：异步入队，上传立即返回（不阻塞请求）
	// 对 jpg/png 返回 .webp URL（压缩完成直接命中，未完成时 HandleImage 自动回退原图）
	if cfg.EnableFixedSizeCompression {
		compressCfg := compress.Config{
			TargetSizeKB:      cfg.TargetFileSizeKB,
			InitialQuality:    cfg.CompressionQualityStart,
			EnableCompression: true,
		}

		if cfg.CompressQueueSize > 0 && compressQueue != nil {
			// 异步模式：入队，立即返回
			queued := compressQueue.Enqueue(compress.Task{
				InputPath: fullPath,
				Cfg:       compressCfg,
			})
			if !queued {
				logger.LogError(ip, fileName, "compress queue full, image kept as original")
			}
			// 优先返回 .webp URL：压缩完成后直接命中；
			// 压缩未完成/跳过时 HandleImage 自动回退到原图
			if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" && ext != ".gif" && ext != ".webp" {
				relativeURL = fmt.Sprintf("/img/%s/%s/%s.webp", year, month, strings.TrimSuffix(filepath.Base(fileName), ext))
			}
			webpHandled = true // 已入队，不走 webp 兜底（队列会处理）
		} else {
			// 同步模式（CompressQueueSize=0）：兼容旧行为，直接压缩
			compressedPath, compErr := compress.CompressToTarget(fullPath, compressCfg)
			if compErr != nil {
				logger.LogError(ip, fileName, "compress failed: "+compErr.Error())
				webpHandled = true
			} else if compressedPath != fullPath {
				relativeURL = fmt.Sprintf("/img/%s/%s/%s", year, month, filepath.Base(compressedPath))
				webpHandled = true
			}
		}
	}

	// 🔄 WebP 转换：与固定压缩协同——
	//   固定压缩未开启时独立生效；
	//   固定压缩同步模式跳过该格式时兜底转换；
	//   固定压缩异步模式已入队时不重复处理。
	if cfg.EnableWebPConvert && !webpHandled {
		if err := storage.ConvertToWebPAsync(fullPath, cfg.WebPQuality, cfg.KeepOriginalAfterWebP); err != nil {
			logger.LogError(ip, fileName, "webp convert queue failed: "+err.Error())
		}
		if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" && ext != ".webp" {
			relativeURL = fmt.Sprintf("/img/%s/%s/%s.webp", year, month, strings.TrimSuffix(filepath.Base(fileName), ext))
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
		if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" && ext != ".webp" {
			webpPath := strings.TrimSuffix(fullPath, ext) + ".webp"
			if _, err := os.Stat(webpPath); err == nil {
				http.Redirect(w, r, fmt.Sprintf("/img/%s/%s/%s", year, month, filepath.Base(webpPath)), http.StatusTemporaryRedirect)
				return
			}
		} else if strings.EqualFold(filepath.Ext(fileName), ".webp") {
			fallbackCandidates := []string{strings.TrimSuffix(fullPath, filepath.Ext(fullPath)) + ".png", strings.TrimSuffix(fullPath, filepath.Ext(fullPath)) + ".jpg", strings.TrimSuffix(fullPath, filepath.Ext(fullPath)) + ".jpeg", strings.TrimSuffix(fullPath, filepath.Ext(fullPath)) + ".gif"}
			for _, candidate := range fallbackCandidates {
				if _, err := os.Stat(candidate); err == nil {
					fullPath = candidate
					break
				}
			}
		}
		if info, err := os.Stat(fullPath); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("ETag", fmt.Sprintf(`"%x"`, info.ModTime().Unix()))
			w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
			if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == fmt.Sprintf(`"%x"`, info.ModTime().Unix()) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
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

	initTemplates()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := homeTemplate.Execute(w, map[string]interface{}{
		"ClientIP":  clientIP,
		"AvatarURL": avatarURL,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderUploadPage 渲染上传页面
func renderUploadPage(w http.ResponseWriter, cfg config.Config) {
	initTemplates()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	needAPIKey := cfg.APIKey != ""
	allowedTypesStr := strings.Join(cfg.AllowedTypes, ", ")

	if err := uploadTemplate.Execute(w, map[string]interface{}{
		"NeedAPIKey":   needAPIKey,
		"AllowedTypes": allowedTypesStr,
		"MaxSize":      cfg.MaxSize,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
