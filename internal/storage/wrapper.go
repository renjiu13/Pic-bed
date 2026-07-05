package storage

import (
	"fmt"
	"io"
	"log"
	"sync"
)

var (
	defaultManager     *StorageManager
	defaultManagerMu   sync.Mutex
	defaultManagerOnce sync.Once
	defaultManagerErr  error
)

// Init 初始化默认存储管理器。
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

// SaveFile 通过默认管理器保存文件。
func SaveFile(reader io.Reader, baseDir, year, month, fileName string) (string, error) {
	sm, err := ensureManager(baseDir)
	if err != nil {
		return "", err
	}
	return sm.SaveFile(reader, year, month, fileName)
}

// ConvertToWebP 通过默认管理器转换图片为 WebP。
func ConvertToWebP(inputPath string, quality float32) (string, error) {
	if defaultManager == nil {
		return "", fmt.Errorf("storage not initialized")
	}
	return defaultManager.ConvertToWebP(inputPath, quality)
}

// ConvertToWebPAsync 通过默认管理器在后台异步转换图片为 WebP。
func ConvertToWebPAsync(inputPath string, quality float32) error {
	if defaultManager == nil {
		return fmt.Errorf("storage not initialized")
	}
	return defaultManager.ConvertToWebPAsync(inputPath, quality)
}

// DeleteFile 通过默认管理器删除文件。
func DeleteFile(baseDir, year, month, fileName string) error {
	sm, err := ensureManager(baseDir)
	if err != nil {
		return err
	}
	return sm.DeleteFile(year, month, fileName)
}

// StartAutoClean 启动默认管理器的自动清理。
func StartAutoClean(baseDir string, hours int) {
	sm, err := ensureManager(baseDir)
	if err != nil {
		log.Printf("[storage] init failed: %v", err)
		return
	}
	sm.StartAutoClean(hours)
}

// StopAutoClean 停止默认管理器的自动清理。
func StopAutoClean() error {
	if defaultManager == nil {
		return nil
	}
	return defaultManager.StopAutoClean()
}

// Close 关闭默认存储管理器。
func Close() error {
	return StopAutoClean()
}
