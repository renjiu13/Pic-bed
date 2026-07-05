package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

func TestHandleUploadReturnsWebPURLWhenEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfgData := map[string]interface{}{
		"storage_dir":         tmpDir,
		"enable_webp_convert": true,
		"webp_quality":        80,
		"allowed_types":       []string{"png"},
		"max_size":            10,
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
	if err := config.Reload(); err != nil {
		t.Fatalf("reload config: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "demo.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("not-a-real-image")); err != nil {
		t.Fatalf("write form data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	HandleUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected upload to succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := payload["url"].(string); !strings.HasSuffix(got, ".webp") {
		t.Fatalf("expected webp url, got %q", got)
	}
}

func TestHandleImageRedirectsOriginalFormatToWebP(t *testing.T) {
	tmpDir := t.TempDir()
	imageDir := filepath.Join(tmpDir, "2024", "01")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("make image dir: %v", err)
	}
	webpPath := filepath.Join(imageDir, "demo.webp")
	if err := os.WriteFile(webpPath, []byte("fake-webp"), 0o644); err != nil {
		t.Fatalf("write temp webp file: %v", err)
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

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected temporary redirect, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/img/2024/01/demo.webp" {
		t.Fatalf("unexpected redirect location: %q", got)
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
