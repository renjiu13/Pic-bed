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
