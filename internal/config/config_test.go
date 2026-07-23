package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReloadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := InitConfig(); err != nil {
		t.Fatalf("init config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, configFileName), []byte(`{"port":9090}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := Reload(); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := Get().Port; got != 9090 {
		t.Fatalf("expected port 9090, got %d", got)
	}
}

// TestNormalizeCoexistsSwitches 验证：固定压缩 与 WebP 转换 可同时开启（协同，非互斥）
func TestNormalizeCoexistsSwitches(t *testing.T) {
	cfg := Config{
		EnableFixedSizeCompression: true,
		EnableWebPConvert:          true,
		KeepOriginalAfterWebP:      true,
	}
	normalizeConfig(&cfg)

	// 两个开关都应保留，不强制关闭任何一个
	if !cfg.EnableFixedSizeCompression {
		t.Fatalf("expected EnableFixedSizeCompression kept on")
	}
	if !cfg.EnableWebPConvert {
		t.Fatalf("expected EnableWebPConvert kept on (coexist, not exclusive)")
	}
	if !cfg.KeepOriginalAfterWebP {
		t.Fatalf("expected KeepOriginalAfterWebP kept on")
	}
}

// TestNormalizeFillsInvalidValues 验证非法值被兜底为默认值
func TestNormalizeFillsInvalidValues(t *testing.T) {
	cfg := Config{
		Port:                     0,
		MaxSize:                  -5,
		Timeout:                  0,
		StorageDir:               "",
		TargetFileSizeKB:         0,
		CompressionQualityStart:  200,
		WebPQuality:              -1,
		AutoCleanHours:           0,
		AllowedTypes:             nil,
	}
	normalizeConfig(&cfg)

	if cfg.Port != 8080 {
		t.Fatalf("Port got %d", cfg.Port)
	}
	if cfg.MaxSize != 10 {
		t.Fatalf("MaxSize got %d", cfg.MaxSize)
	}
	if cfg.Timeout != 30 {
		t.Fatalf("Timeout got %d", cfg.Timeout)
	}
	if cfg.StorageDir != "./data" {
		t.Fatalf("StorageDir got %q", cfg.StorageDir)
	}
	if cfg.TargetFileSizeKB != 500 {
		t.Fatalf("TargetFileSizeKB got %d", cfg.TargetFileSizeKB)
	}
	if cfg.CompressionQualityStart != 90 {
		t.Fatalf("CompressionQualityStart got %d", cfg.CompressionQualityStart)
	}
	if cfg.WebPQuality != 80 {
		t.Fatalf("WebPQuality got %v", cfg.WebPQuality)
	}
	if cfg.AutoCleanHours != 720 {
		t.Fatalf("AutoCleanHours got %d", cfg.AutoCleanHours)
	}
	if len(cfg.AllowedTypes) == 0 {
		t.Fatalf("AllowedTypes should be filled")
	}
}

