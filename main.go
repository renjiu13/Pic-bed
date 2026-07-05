package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pic-bed/pic-bed/internal/config"
	"github.com/pic-bed/pic-bed/internal/handler"
	"github.com/pic-bed/pic-bed/internal/logger"
	"github.com/pic-bed/pic-bed/internal/storage"
)

func main() {
	if err := config.InitConfig(); err != nil {
		panic("Failed to load config: " + err.Error())
	}
	cfg := config.Get()

	// 初始化日志
	if cfg.EnableLog {
		// Yes 创建日志文件夹
		logDir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Warning: failed to create log directory: %v\n", err)
		}

		if err := logger.Init(cfg.LogFile, true); err != nil {
			fmt.Printf("Warning: log init failed: %v\n", err)
		}
	}

	// 初始化存储管理器
	if err := storage.Init(cfg.StorageDir); err != nil {
		panic("Failed to init storage: " + err.Error())
	}

	// 启动自动清理
	if cfg.EnableAutoClean && cfg.AutoCleanHours > 0 {
		storage.StartAutoClean(cfg.StorageDir, cfg.AutoCleanHours)
		fmt.Printf("[AutoClean] Enabled, clean files older than %d hours\n", cfg.AutoCleanHours)
	}

	// 注册路由
	http.HandleFunc("/", handler.HandleHome)
	http.HandleFunc("/upload", handler.AuthMiddleware(handler.HandleUpload))
	http.HandleFunc("/img/", handler.HandleImage)

	// 启动信息
	fmt.Println("=== 极轻量图床启动成功 ===")
	fmt.Printf("监听端口: %d\n", cfg.Port)
	fmt.Printf("存储目录: %s\n", cfg.StorageDir)
	fmt.Printf("单文件上限: %d MB\n", cfg.MaxSize)
	fmt.Printf("首页地址: http://0.0.0.0:%d/\n", cfg.Port)
	fmt.Printf("上传接口: POST http://0.0.0.0:%d/upload\n", cfg.Port)
	fmt.Printf("预览格式: GET  http://服务器IP:%d/img/年/月/文件名\n", cfg.Port)
	if cfg.APIKey != "" {
		fmt.Println("鉴权状态: 已开启 Bearer Token")
	} else {
		fmt.Println("鉴权状态: 未开启（公开上传）")
	}
	fmt.Printf("功能开关: 日志=%v 删除=%v 自动清理=%v\n",
		cfg.EnableLog, cfg.EnableDelete, cfg.EnableAutoClean)

	// 启动服务
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		panic("Server failed: " + err.Error())
	}
}
