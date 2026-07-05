package storage

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/KarpelesLab/gowebp"
)

// StorageManager 存储管理器
type StorageManager struct {
	baseDir     string
	mu          sync.RWMutex
	pathLocks   map[string]*sync.Mutex
	stopCleanCh chan struct{}
	cleanerDone chan struct{}
}

// NewStorageManager 创建新的存储管理器
func NewStorageManager(baseDir string) (*StorageManager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	return &StorageManager{
		baseDir:     baseDir,
		pathLocks:   make(map[string]*sync.Mutex),
		stopCleanCh: make(chan struct{}),
		cleanerDone: make(chan struct{}),
	}, nil
}

func (sm *StorageManager) lockForPath(path string) *sync.Mutex {
	sm.mu.RLock()
	lock, ok := sm.pathLocks[path]
	sm.mu.RUnlock()
	if ok {
		return lock
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	if lock, ok = sm.pathLocks[path]; !ok {
		lock = &sync.Mutex{}
		sm.pathLocks[path] = lock
	}
	return lock
}

// ValidateFileName 校验文件名
func (sm *StorageManager) ValidateFileName(fileName string) error {
	if strings.Contains(fileName, "..") || filepath.IsAbs(fileName) ||
		strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return fmt.Errorf("invalid filename")
	}
	return nil
}

// ValidatePath 校验路径是否在 baseDir 内
func (sm *StorageManager) ValidatePath(targetPath string) error {
	absTarget, _ := filepath.Abs(targetPath)
	absBase, _ := filepath.Abs(sm.baseDir)
	if !strings.HasPrefix(absTarget, absBase) {
		return fmt.Errorf("path traversal detected")
	}
	return nil
}

// SaveFile 保存文件
func (sm *StorageManager) SaveFile(reader io.Reader, year, month, fileName string) (string, error) {
	if err := sm.ValidateFileName(fileName); err != nil {
		return "", err
	}

	targetDir := filepath.Join(sm.baseDir, year, month)
	if err := sm.ValidatePath(targetDir); err != nil {
		return "", err
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("create dir failed: %w", err)
	}

	fullPath := filepath.Join(targetDir, fileName)
	if err := sm.ValidatePath(fullPath); err != nil {
		return "", err
	}

	pathLock := sm.lockForPath(fullPath)
	pathLock.Lock()
	defer pathLock.Unlock()

	outFile, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("write file failed: %w", err)
	}

	return fmt.Sprintf("/img/%s/%s/%s", year, month, fileName), nil
}

// DeleteFile 删除文件
func (sm *StorageManager) DeleteFile(year, month, fileName string) error {
	if err := sm.ValidateFileName(fileName); err != nil {
		return err
	}

	filePath := filepath.Join(sm.baseDir, year, month, fileName)
	if err := sm.ValidatePath(filePath); err != nil {
		return err
	}

	pathLock := sm.lockForPath(filePath)
	pathLock.Lock()
	defer pathLock.Unlock()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}

	return os.Remove(filePath)
}

// CleanOldFiles 清理指定小时数之前的旧文件
func (sm *StorageManager) CleanOldFiles(hours int) (int, int64, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	deletedCount := 0
	deletedSize := int64(0)

	err := filepath.Walk(sm.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if delErr := os.Remove(path); delErr == nil {
				deletedCount++
				deletedSize += size
			}
		}
		return nil
	})

	return deletedCount, deletedSize, err
}

// StartAutoClean 启动自动清理
func (sm *StorageManager) StartAutoClean(hours int) {
	if hours <= 0 {
		return
	}

	go func() {
		defer close(sm.cleanerDone)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				count, size, _ := sm.CleanOldFiles(hours)
				if count > 0 {
					log.Printf("[AutoClean] Deleted %d files, freed %d bytes\n", count, size)
				}
			case <-sm.stopCleanCh:
				return
			}
		}
	}()
}

// StopAutoClean 停止自动清理
func (sm *StorageManager) StopAutoClean() error {
	close(sm.stopCleanCh)
	select {
	case <-sm.cleanerDone:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("cleaner shutdown timeout")
	}
}

// ConvertToWebPAsync 异步转换为 WebP 格式，不阻塞请求。
// keepOriginal 为可选参数，兼容旧调用方式。
func (sm *StorageManager) ConvertToWebPAsync(srcPath string, quality float32, keepOriginal ...bool) error {
	preserveOriginal := false
	if len(keepOriginal) > 0 {
		preserveOriginal = keepOriginal[0]
	}

	go func() {
		if _, err := sm.ConvertToWebP(srcPath, quality, preserveOriginal); err != nil {
			log.Printf("[storage] webp conversion failed for %s: %v", srcPath, err)
		}
	}()
	return nil
}

// ConvertToWebP 转换为 WebP 格式
// keepOriginal: 转换成功后是否保留原图
func (sm *StorageManager) ConvertToWebP(srcPath string, quality float32, keepOriginal bool) (string, error) {
	if quality < 0 || quality > 100 {
		return "", fmt.Errorf("quality must be 0-100")
	}

	ext := strings.ToLower(filepath.Ext(srcPath))

	// GIF 和 WebP 不转换
	if ext == ".gif" || ext == ".webp" {
		return srcPath, nil
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", srcPath)
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var img image.Image
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".png":
		img, err = png.Decode(f)
	default:
		return srcPath, nil
	}

	if err != nil {
		return "", fmt.Errorf("decode failed: %w", err)
	}

	webpPath := strings.TrimSuffix(srcPath, ext) + ".webp"

	out, err := os.Create(webpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	err = gowebp.Encode(out, img, &gowebp.Options{
		Lossy:   true,
		Quality: quality,
	})
	if err != nil {
		os.Remove(webpPath)
		return "", err
	}

	if info, _ := os.Stat(webpPath); info == nil || info.Size() == 0 {
		os.Remove(webpPath)
		return "", fmt.Errorf("webp file invalid")
	}

	// ✨ 根据配置决定是否删除原图
	if !keepOriginal {
		// 使用文件锁确保并发安全
		pathLock := sm.lockForPath(srcPath)
		pathLock.Lock()
		defer pathLock.Unlock()

		// 二次确认文件存在后再删除
		if _, err := os.Stat(srcPath); err == nil {
			os.Remove(srcPath)
		}
	}

	return webpPath, nil
}

// Close 关闭存储管理器
func (sm *StorageManager) Close() error {
	return sm.StopAutoClean()
}
