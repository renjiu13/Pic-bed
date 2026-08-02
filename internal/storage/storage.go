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
	// pathLocks 清理相关
	lockCleanStop chan struct{}
	lockCleanDone chan struct{}
}

// NewStorageManager 创建新的存储管理器
func NewStorageManager(baseDir string) (*StorageManager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	sm := &StorageManager{
		baseDir:       baseDir,
		pathLocks:     make(map[string]*sync.Mutex),
		stopCleanCh:   make(chan struct{}),
		cleanerDone:   make(chan struct{}),
		lockCleanStop: make(chan struct{}),
		lockCleanDone: make(chan struct{}),
	}

	// 启动 pathLocks 定期清理（每 10 分钟清理一次空闲锁）
	go sm.lockCleaner()

	return sm, nil
}

// lockCleaner 定期清理 pathLocks 中未被持有的锁
// 防止 map 随上传/删除无限增长
func (sm *StorageManager) lockCleaner() {
	defer close(sm.lockCleanDone)
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanIdleLocks()
		case <-sm.lockCleanStop:
			return
		}
	}
}

// cleanIdleLocks 清理当前未被持有的路径锁
// 通过 TryLock 检测：能锁住说明没有其他 goroutine 持有，可安全删除
func (sm *StorageManager) cleanIdleLocks() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 用写锁遍历，尝试 TryLock 每个锁
	// TryLock 成功 = 没人持有，可安全删除
	// TryLock 失败 = 有人正在使用，跳过
	for path, lock := range sm.pathLocks {
		if lock.TryLock() {
			lock.Unlock()
			delete(sm.pathLocks, path)
		}
	}
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

// webpSem 限制 WebP 转换并发数，防止高并发上传时 goroutine 飙升
// 弱设备（玩客云等）设为 2：既保证上传不阻塞，又限制内存峰值
var webpSem = make(chan struct{}, 2)

// ConvertToWebPAsync 异步转换为 WebP 格式，不阻塞请求。
// 使用信号量控制并发（最多 2 个同时转换），防止弱设备内存飙升。
// keepOriginal 为可选参数，兼容旧调用方式。
func (sm *StorageManager) ConvertToWebPAsync(srcPath string, quality float32, keepOriginal ...bool) error {
	preserveOriginal := false
	if len(keepOriginal) > 0 {
		preserveOriginal = keepOriginal[0]
	}

	go func() {
		// 信号量：获取名额才能执行，满了就等（不丢弃，保证最终一致性）
		webpSem <- struct{}{}
		defer func() { <-webpSem }()

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

	// ✅ 第一步：读取并解码图片，读完就关闭文件（避免 Windows 上文件占用导致删除失败）
	var img image.Image
	var decodeErr error
	func() {
		f, err := os.Open(srcPath)
		if err != nil {
			decodeErr = err
			return
		}
		defer f.Close() // 匿名函数结束就关闭文件

		switch ext {
		case ".jpg", ".jpeg":
			img, decodeErr = jpeg.Decode(f)
		case ".png":
			img, decodeErr = png.Decode(f)
		default:
			return
		}
	}()

	if decodeErr != nil {
		return "", fmt.Errorf("decode failed: %w", decodeErr)
	}
	if img == nil {
		return srcPath, nil
	}

	// ✅ 第二步：生成 WebP 文件
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

	// ✅ 第三步：删除原图（这时候原图已经关闭，可以安全删除）
	if !keepOriginal {
		pathLock := sm.lockForPath(srcPath)
		pathLock.Lock()
		defer pathLock.Unlock()

		if _, err := os.Stat(srcPath); err == nil {
			if removeErr := os.Remove(srcPath); removeErr != nil {
				log.Printf("[storage] failed to remove original file %s: %v", srcPath, removeErr)
			}
		}
	}

	return webpPath, nil
}

// Close 关闭存储管理器
func (sm *StorageManager) Close() error {
	// 停止锁清理器
	close(sm.lockCleanStop)
	<-sm.lockCleanDone

	return sm.StopAutoClean()
}