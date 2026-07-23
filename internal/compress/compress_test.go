package compress

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	// 用固定种子伪随机噪声，PNG 几乎不可压缩，确保大于测试目标
	r := rand.New(rand.NewSource(42))
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(r.Intn(256)),
				G: uint8(r.Intn(256)),
				B: uint8(r.Intn(256)),
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func TestCompressToTargetDisabledReturnsInput(t *testing.T) {
	got, err := CompressToTarget("whatever.png", Config{EnableCompression: false})
	if err != nil || got != "whatever.png" {
		t.Fatalf("expected passthrough, got %q err=%v", got, err)
	}
}

func TestCompressToTargetSkipsGifAndWebp(t *testing.T) {
	for _, name := range []string{"a.gif", "b.webp"} {
		got, err := CompressToTarget(name, Config{EnableCompression: true, TargetSizeKB: 10, InitialQuality: 80})
		if err != nil || got != name {
			t.Fatalf("expected passthrough for %s, got %q err=%v", name, got, err)
		}
	}
}

func TestCompressToTargetProducesWebPUnderTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.png")
	writeTestPNG(t, src)

	out, err := CompressToTarget(src, Config{
		EnableCompression: true,
		TargetSizeKB:      50,
		InitialQuality:    90,
	})
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if filepath.Ext(out) != ".webp" {
		t.Fatalf("expected webp output, got %q", out)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output is empty")
	}
	if info.Size() > 50*1024 {
		t.Fatalf("output %d bytes exceeds target 50KB", info.Size())
	}

	// 压缩成功后原图应被删除
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected original %s to be removed, stat err=%v", src, err)
	}
}

func TestCompressToTargetKeepsOriginalOnDecodeError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "broken.png")
	// 写入 >1KB 的非图片数据，使其大于目标(1KB)从而跳过小图保护、走到解码失败
	junk := make([]byte, 2048)
	for i := range junk {
		junk[i] = byte(i % 256)
	}
	if err := os.WriteFile(src, junk, 0o644); err != nil {
		t.Fatalf("write broken png: %v", err)
	}

	_, err := CompressToTarget(src, Config{
		EnableCompression: true,
		TargetSizeKB:      1,
		InitialQuality:    90,
	})
	if err == nil {
		t.Fatalf("expected decode error, got nil")
	}
	// 解码失败时原图应保留
	if _, statErr := os.Stat(src); statErr != nil {
		t.Fatalf("expected original kept on decode error, stat err=%v", statErr)
	}
}

func TestCompressToTargetSkipsSmallImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tiny.png")
	writeTestPNG(t, src)

	// 目标设为 100MB，原图必然 <= 目标，应原样返回（不生成 webp、不删原图）
	out, err := CompressToTarget(src, Config{
		EnableCompression: true,
		TargetSizeKB:      100 * 1024,
		InitialQuality:    90,
	})
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if out != src {
		t.Fatalf("expected passthrough for small image, got %q", out)
	}
	// 原图应保留
	if _, statErr := os.Stat(src); statErr != nil {
		t.Fatalf("expected original kept, stat err=%v", statErr)
	}
	// 不应生成 webp
	webpPath := filepath.Join(dir, "tiny.webp")
	if _, statErr := os.Stat(webpPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no webp generated, stat err=%v", statErr)
	}
}
