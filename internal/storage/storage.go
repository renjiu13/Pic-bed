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

// StorageManager 管理文件存储
type StorageManager struct {
	baseDir     string
	mu          sync.RWMutex
	pathLocks   map[string]*sync.Mutex
	stopCleanCh chan struct{}
	cleanerDone chan struct{}
}

// NewStorageManager 创建存储管理器
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

// ValidateFileName 防止路径遍历
func (sm *StorageManager) ValidateFileName(fileName string) error {
	if strings.Contains(fileName, "..") || filepath.IsAbs(fileName) ||
		strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return fmt.Errorf("invalid filename")
	}
	return nil
}

// ValidatePath 检查路径是否在 baseDir 内
func (sm *StorageManager) ValidatePath(targetPath string) error {
	absTarget, _ := filepath.Abs(targetPath)
	absBase, _ := filepath.Abs(sm.baseDir)
	if !strings.HasPrefix(absTarget, absBase) {
		return fmt.Errorf("path traversal detected")
	}
	return nil
}

// SaveFile 流式保存文件
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

// CleanOldFiles 清理超过指定小时数的文件
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

// StartAutoClean 启动自动清理协程
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

// StopAutoClean 优雅关闭清理协程
func (sm *StorageManager) StopAutoClean() error {
	close(sm.stopCleanCh)
	select {
	case <-sm.cleanerDone:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("cleaner shutdown timeout")
	}
}

// ConvertToWebPAsync 在后台异步转换图片为 WebP，避免阻塞上传流程。
func (sm *StorageManager) ConvertToWebPAsync(srcPath string, quality float32) error {
	go func() {
		if _, err := sm.ConvertToWebP(srcPath, quality); err != nil {
			log.Printf("[storage] webp conversion failed for %s: %v", srcPath, err)
		}
	}()
	return nil
}

// ConvertToWebP 转换为 WebP 格式
func (sm *StorageManager) ConvertToWebP(srcPath string, quality float32) (string, error) {
	if quality < 0 || quality > 100 {
		return "", fmt.Errorf("quality must be 0-100")
	}

	ext := strings.ToLower(filepath.Ext(srcPath))

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

	return webpPath, nil
}

// Close 关闭管理器
func (sm *StorageManager) Close() error {
	return sm.StopAutoClean()
}
