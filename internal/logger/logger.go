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
func Init(path string, enable bool) error {
	if !enable {
		return nil
	}

	var err error
	once.Do(func() {
		logDir := filepath.Dir(path)
		if logDir != "." {
			if mkdirErr := os.MkdirAll(logDir, 0755); mkdirErr != nil {
				err = fmt.Errorf("create log dir failed: %w", mkdirErr)
				return
			}
		}
		logPath = path
	})
	if err != nil {
		return err
	}
	return rotateLogFile()
}

// rotateLogFile 切换日志文件（按日期）
func rotateLogFile() error {
	mu.Lock()
	defer mu.Unlock()
	return rotateLogFileLocked()
}

func rotateLogFileLocked() error {
	today := time.Now().Format("2006-01-02")

	if today == currentDate && logFile != nil {
		return nil
	}

	if logFile != nil {
		_ = logFile.Close()
	}

	if logPath == "" {
		return nil
	}

	logDir := filepath.Dir(logPath)
	logName := filepath.Base(logPath)
	ext := filepath.Ext(logName)
	baseName := logName[:len(logName)-len(ext)]

	newLogPath := filepath.Join(logDir, fmt.Sprintf("%s-%s%s", baseName, today, ext))

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
	mu.Lock()
	defer mu.Unlock()

	if logger == nil || logPath == "" {
		return
	}

	today := time.Now().Format("2006-01-02")
	if today != currentDate {
		_ = rotateLogFileLocked()
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

	var err error
	if logFile != nil {
		err = logFile.Close()
		logFile = nil
	}
	logger = nil
	currentDate = ""
	return err
}
