package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pic-bed/pic-bed/internal/config"
	"github.com/pic-bed/pic-bed/internal/handler"
	"github.com/pic-bed/pic-bed/internal/logger"
	"github.com/pic-bed/pic-bed/internal/selfupdate"
	"github.com/pic-bed/pic-bed/internal/storage"
	"github.com/pic-bed/pic-bed/internal/version"
)

func main() {
	// 定义命令行参数
	showVersion := flag.Bool("v", false, "显示版本信息")
	showVersionLong := flag.Bool("version", false, "显示版本信息")
	showHelp := flag.Bool("h", false, "显示帮助信息")
	showHelpLong := flag.Bool("help", false, "显示帮助信息")

	// 自定义 Usage
	flag.Usage = func() {
		fmt.Println("Pic-bed - 极轻量私有图床")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Println("  pic-bed [选项] [命令]")
		fmt.Println()
		fmt.Println("选项:")
		fmt.Println("  -v, --version    显示版本信息")
		fmt.Println("  -h, --help       显示帮助信息")
		fmt.Println()
		fmt.Println("命令:")
		fmt.Println("  check-update     检查是否有新版本")
		fmt.Println("  update           在线更新到最新版本")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  pic-bed              # 启动图床服务")
		fmt.Println("  pic-bed -v           # 查看版本")
		fmt.Println("  pic-bed check-update # 检查更新")
		fmt.Println("  pic-bed update       # 在线更新")
	}

	flag.Parse()

	// 处理版本显示
	if *showVersion || *showVersionLong {
		fmt.Println(version.FullInfo())
		return
	}

	// 处理帮助显示
	if *showHelp || *showHelpLong {
		flag.Usage()
		return
	}

	// 处理子命令
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "check-update":
			cmdCheckUpdate()
			return
		case "update":
			cmdUpdate()
			return
		default:
			fmt.Printf("未知命令: %s\n\n", args[0])
			flag.Usage()
			os.Exit(1)
		}
	}

	// 默认：启动服务
	startServer()
}

// cmdCheckUpdate 检查更新命令
func cmdCheckUpdate() {
	fmt.Println("正在检查更新...")
	fmt.Printf("当前版本: %s\n", version.Version)

	result, err := selfupdate.CheckUpdate()
	if err != nil {
		fmt.Printf("检查更新失败: %v\n", err)
		os.Exit(1)
	}

	if result.HasUpdate {
		fmt.Printf("发现新版本: %s\n", result.Latest)
		fmt.Printf("运行 'pic-bed update' 进行更新\n")
	} else {
		fmt.Println("✓ 已是最新版本")
	}
}

// cmdUpdate 在线更新命令
func cmdUpdate() {
	fmt.Println("正在检查更新...")
	fmt.Printf("当前版本: %s\n", version.Version)

	result, err := selfupdate.CheckUpdate()
	if err != nil {
		fmt.Printf("检查更新失败: %v\n", err)
		os.Exit(1)
	}

	if !result.HasUpdate {
		fmt.Println("✓ 已是最新版本")
		return
	}

	fmt.Printf("发现新版本: %s\n", result.Latest)

	if result.DownloadURL == "" {
		fmt.Println("错误: 未找到对应平台的下载文件")
		os.Exit(1)
	}

	if err := selfupdate.DoUpdate(result.DownloadURL); err != nil {
		fmt.Printf("更新失败: %v\n", err)
		os.Exit(1)
	}
}

// startServer 启动图床服务
func startServer() {
	if err := config.InitConfig(); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	cfg := config.Get()

	// 初始化日志
	if cfg.EnableLog {
		if err := logger.Init(cfg.LogFile, true); err != nil {
			fmt.Printf("Warning: log init failed: %v\n", err)
		}
	}

	// 初始化存储管理器
	if err := storage.Init(cfg.StorageDir); err != nil {
		panic("Failed to init storage: " + err.Error())
	}

	go func() {
		hup := make(chan os.Signal, 1)
		signal.Notify(hup, syscall.SIGHUP)
		for range hup {
			if err := config.Reload(); err != nil {
				fmt.Printf("Config reload failed: %v\n", err)
				continue
			}
			fmt.Println("Config reloaded")
		}
	}()

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
	fmt.Printf("版本: %s\n", version.Version)
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("Server failed: " + err.Error())
		}
	}()

	<-quit
	fmt.Println("\nShutdown signal received, exiting gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
	}

	_ = logger.Close()
	_ = storage.Close()
}
