package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pic-bed/pic-bed/internal/config"
)

func TestEmbeddedTemplatesExist(t *testing.T) {
	for _, path := range []string{"templates/home.html", "templates/upload.html"} {
		data, err := templateFS.ReadFile(path)
		if err != nil {
			t.Fatalf("expected embedded template %s to exist: %v", path, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			t.Fatalf("embedded template %s is empty", path)
		}
	}
}

func TestHandleImageSetsCacheHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	imageDir := filepath.Join(tmpDir, "2024", "01")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("make image dir: %v", err)
	}
	imagePath := filepath.Join(imageDir, "demo.png")
	if err := os.WriteFile(imagePath, []byte("not-a-real-image"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cfgData := map[string]interface{}{
		"storage_dir":   tmpDir,
		"enable_delete": true,
	}
	cfgBytes, err := json.Marshal(cfgData)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), cfgBytes, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := config.InitConfig(); err != nil {
		t.Fatalf("init config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/img/2024/01/demo.png", nil)
	rec := httptest.NewRecorder()

	HandleImage(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected cache-control header: %q", got)
	}
	if got := rec.Header().Get("ETag"); got == "" {
		t.Fatalf("expected etag header to be set")
	}
}
