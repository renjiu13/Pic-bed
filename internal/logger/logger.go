package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile     *os.File
	logger      *log.Logger
	logPath     string
	currentDate string
	mu          sync.Mutex
	once        sync.Once
)

// Init 初始化日志
func Init(logPath string, enable bool) error {
	if !enable {
		return nil
	}

	var err error
	once.Do(func() {
		logDir := filepath.Dir(logPath)
		if logDir != "." {
			os.MkdirAll(logDir, 0755)
		}
		// 保存路径模板（不含日期）
		// 例如: /data/logs/picbed.log -> /data/logs/picbed-YYYY-MM-DD.log
	})

	// 成功初始化后再设置路径
	if err == nil {
		mu.Lock()
		logPath = logPath
		mu.Unlock()
		// 初始化第一个日志文件
		err = rotateLogFile()
	}

	return err
}

// rotateLogFile 切换日志文件（按日期）
func rotateLogFile() error {
	mu.Lock()
	defer mu.Unlock()

	today := time.Now().Format("2006-01-02")

	// 如果日期未改变且文件已打开，无需轮转
	if today == currentDate && logFile != nil {
		return nil
	}

	// 关闭旧文件
	if logFile != nil {
		logFile.Close()
	}

	// 生成新日志文件名（含日期）
	logDir := filepath.Dir(logPath)
	logName := filepath.Base(logPath)
	ext := filepath.Ext(logName)
	baseName := logName[:len(logName)-len(ext)]

	newLogPath := filepath.Join(logDir, fmt.Sprintf("%s-%s%s", baseName, today, ext))

	// 打开新文件
	var err error
	logFile, err = os.OpenFile(newLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	logger = log.New(logFile, "", log.LstdFlags)
	currentDate = today

	return nil
}

// ensureLogger 确保日志文件已初始化（检查是否需要轮转）
func ensureLogger() {
	if logger == nil {
		return
	}

	today := time.Now().Format("2006-01-02")
	if today != currentDate {
		rotateLogFile()
	}
}

// Log 记录操作日志
func Log(action, ip, filename, detail string) {
	ensureLogger()

	if logger == nil {
		return
	}

	msg := fmt.Sprintf("[%s] IP:%s File:%s - %s", action, ip, filename, detail)
	logger.Println(msg)
}

// LogUpload 记录上传日志
func LogUpload(ip, filename, url string, size int64) {
	Log("UPLOAD", ip, filename, fmt.Sprintf("URL:%s Size:%d bytes", url, size))
}

// LogDelete 记录删除日志
func LogDelete(ip, filename string) {
	Log("DELETE", ip, filename, "File deleted")
}

// LogAccess 记录访问日志
func LogAccess(ip, filename string) {
	Log("ACCESS", ip, filename, "File accessed")
}

// LogError 记录错误日志
func LogError(ip, filename, err string) {
	Log("ERROR", ip, filename, err)
}

// GetToday 获取当前日期字符串
func GetToday() string {
	return time.Now().Format("2006-01-02")
}

// Close 关闭日志文件
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
