package storage

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KarpelesLab/gowebp"
)

// SaveFile 流式保存文件
func SaveFile(reader io.Reader, baseDir, year, month, fileName string) (string, error) {
	targetDir := filepath.Join(baseDir, year, month)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("create dir failed: %w", err)
	}

	fullPath := filepath.Join(targetDir, fileName)
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
func DeleteFile(baseDir, year, month, fileName string) error {
	filePath := filepath.Join(baseDir, year, month, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}
	return os.Remove(filePath)
}

// CleanOldFiles 自动清理超过指定小时数的文件
func CleanOldFiles(baseDir string, hours int) (int, int64, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	deletedCount := 0
	deletedSize := int64(0)

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if err := os.Remove(path); err == nil {
				deletedCount++
				deletedSize += size
			}
		}
		return nil
	})

	return deletedCount, deletedSize, err
}

// StartAutoClean 启动自动清理协程
func StartAutoClean(baseDir string, hours int) {
	if hours <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
		defer ticker.Stop()
		for range ticker.C {
			count, size, _ := CleanOldFiles(baseDir, hours)
			if count > 0 {
				fmt.Printf("[AutoClean] Deleted %d files, freed %d bytes\n", count, size)
			}
		}
	}()
}

// ConvertToWebP 将图片转换为 WebP 格式并保存；GIF 和已是 WebP 的文件跳过
func ConvertToWebP(srcPath string, quality float32) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))

	if ext == ".gif" || ext == ".webp" {
		return srcPath, nil
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
		return "", err
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
		return srcPath, nil
	}

	os.Remove(srcPath)
	return webpPath, nil
}
