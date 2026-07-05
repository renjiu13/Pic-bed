package logger

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInitStoresProvidedPath(t *testing.T) {
	logFile = nil
	logger = nil
	logPath = ""
	currentDate = ""
	mu = sync.Mutex{}
	once = sync.Once{}

	tempDir, err := os.MkdirTemp("", "picbed-logger-")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	logPathValue := filepath.Join(tempDir, "picbed.log")

	t.Cleanup(func() {
		_ = Close()
		logFile = nil
		logger = nil
		logPath = ""
		currentDate = ""
		mu = sync.Mutex{}
		once = sync.Once{}
		_ = os.Remove(logPathValue)
		_ = os.RemoveAll(tempDir)
	})

	if err := Init(logPathValue, true); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if logPath != logPathValue {
		t.Fatalf("expected logPath to be %q, got %q", logPathValue, logPath)
	}
}
