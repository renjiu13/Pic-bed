package storage

import (
	"fmt"
	"io"
	"log"
	"sync"
)

var (
	defaultManager      *StorageManager
	defaultManagerMu    sync.Mutex
	defaultManagerOnce  sync.Once
	defaultManagerErr   error
)

// Init 初始化默认存储管理器
func Init(baseDir string) error {
	if baseDir == "" {
		return fmt.Errorf("storage base dir is empty")
	}

	defaultManagerOnce.Do(func() {
		sm, err := NewStorageManager(baseDir)
		if err != nil {
			defaultManagerErr = err
			return
		}
		defaultManager = sm
	})
	return defaultManagerErr
}

func ensureManager(baseDir string) (*StorageManager, error) {
	if defaultManager != nil {
		return defaultManager, nil
	}
	if err := Init(baseDir); err != nil {
		return nil, err
	}
	return defaultManager, nil
}

// SaveFile 保存文件（包级便捷函数）
func SaveFile(reader io.Reader, baseDir, year, month, fileName string) (string, error) {
	sm, err := ensureManager(baseDir)
	if err != nil {
		return "", err
	}
	return sm.SaveFile(reader, year, month, fileName)
}

// ConvertToWebP 转换为 WebP 格式（包级便捷函数）
func ConvertToWebP(inputPath string, quality float32, keepOriginal bool) (string, error) {
	if defaultManager == nil {
		return "", fmt.Errorf("storage not initialized")
	}
	return defaultManager.ConvertToWebP(inputPath, quality, keepOriginal)
}

// ConvertToWebPAsync 异步转换为 WebP 格式（包级便捷函数）
func ConvertToWebPAsync(inputPath string, quality float32, keepOriginal bool) error {
	if defaultManager == nil {
		return fmt.Errorf("storage not initialized")
	}
	return defaultManager.ConvertToWebPAsync(inputPath, quality, keepOriginal)
}

// DeleteFile 删除文件（包级便捷函数）
func DeleteFile(baseDir, year, month, fileName string) error {
	sm, err := ensureManager(baseDir)
	if err != nil {
		return err
	}
	return sm.DeleteFile(year, month, fileName)
}

// StartAutoClean 启动自动清理（包级便捷函数）
func StartAutoClean(baseDir string, hours int) {
	sm, err := ensureManager(baseDir)
	if err != nil {
		log.Printf("[storage] init failed: %v", err)
		return
	}
	sm.StartAutoClean(hours)
}

// StopAutoClean 停止自动清理（包级便捷函数）
func StopAutoClean() error {
	if defaultManager == nil {
		return nil
	}
	return defaultManager.StopAutoClean()
}

// Close 关闭存储管理器
func Close() error {
	return StopAutoClean()
}