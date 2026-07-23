// Package compress 通过调整 WebP 质量参数将图片压缩到目标大小（不缩放）。
package compress

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/gowebp"
)

// Config 压缩配置
type Config struct {
	TargetSizeKB      int  // 目标大小（KB）
	InitialQuality    int  // 初始（最大）质量 1-100
	EnableCompression bool // 是否启用压缩
}

// CompressToTarget 通过二分查找最优 WebP 质量，将图片压缩到目标大小。
// 仅调整质量，不缩放图像。返回压缩后的 WebP 文件路径。
// 若未启用压缩或格式不支持（gif/webp），原样返回 inputPath。
func CompressToTarget(inputPath string, cfg Config) (string, error) {
	if !cfg.EnableCompression {
		return inputPath, nil
	}

	ext := strings.ToLower(filepath.Ext(inputPath))
	// GIF / WebP 不处理（无法用 stdlib 解码，且无意义）
	if ext == ".gif" || ext == ".webp" {
		return inputPath, nil
	}

	// 小图保护：原图已 <= 目标大小则原样保留，避免无意义重编码（甚至放大）
	targetBytes := int64(cfg.TargetSizeKB) * 1024
	if info, err := os.Stat(inputPath); err == nil && info.Size() <= targetBytes {
		return inputPath, nil
	}

	// 只解码一次，后续多次编码复用（比每次重新读文件更高效）
	img, err := decodeImage(inputPath)
	if err != nil {
		return "", fmt.Errorf("decode %s: %w", inputPath, err)
	}
	if img == nil {
		return inputPath, nil
	}

	outputPath := strings.TrimSuffix(inputPath, ext) + ".webp"

	maxQuality := cfg.InitialQuality
	if maxQuality < 1 {
		maxQuality = 1
	}
	if maxQuality > 100 {
		maxQuality = 100
	}

	// 二分查找满足目标大小的最大质量
	quality := findOptimalQuality(img, outputPath, maxQuality, targetBytes)

	// 用最终质量再编码一次，确保输出文件与选定质量一致
	if err := encodeWebP(img, outputPath, float32(quality)); err != nil {
		return "", fmt.Errorf("final encode: %w", err)
	}

	// 压缩成功后删除原图（此时原图文件句柄早已在 decodeImage 中关闭，可安全删除）
	if removeErr := os.Remove(inputPath); removeErr != nil && !os.IsNotExist(removeErr) {
		// 删除失败不影响压缩结果（webp 已生成），仅记录日志
		log.Printf("[compress] failed to remove original %s: %v", inputPath, removeErr)
	}

	return outputPath, nil
}

// decodeImage 解码为 image.Image（读完后立即关闭文件，避免 Windows 文件占用）
func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".png":
		return png.Decode(f)
	default:
		return nil, fmt.Errorf("unsupported type: %s", ext)
	}
}

// findOptimalQuality 二分查找使输出大小 <= targetBytes 的最大质量。
// WebP 文件大小随质量单调（非严格）递增，故可二分。
// 若所有质量都超出目标，返回 1（最小质量，文件最小）。
func findOptimalQuality(img image.Image, outputPath string, maxQuality int, targetBytes int64) int {
	quality := 1
	left, right := 1, maxQuality

	for left <= right {
		mid := (left + right) / 2
		if err := encodeWebP(img, outputPath, float32(mid)); err != nil {
			right = mid - 1
			continue
		}
		info, err := os.Stat(outputPath)
		if err != nil || info.Size() > targetBytes {
			// 太大或失败，降低质量继续找
			right = mid - 1
			continue
		}
		// 满足目标，尝试更高质量
		quality = mid
		left = mid + 1
	}

	return quality
}

// encodeWebP 以指定质量编码为 WebP
func encodeWebP(img image.Image, outputPath string, quality float32) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	return gowebp.Encode(out, img, &gowebp.Options{
		Lossy:   true,
		Quality: quality,
	})
}
