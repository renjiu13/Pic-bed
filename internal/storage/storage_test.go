package storage

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type blockingReader struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingReader) Read(p []byte) (int, error) {
	select {
	case <-r.started:
	default:
		close(r.started)
	}
	<-r.release
	return 0, io.EOF
}

func TestConvertToWebPAsyncReturnsImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "sample.png")

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode source image: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close source image: %v", err)
	}

	sm, err := NewStorageManager(tmpDir)
	if err != nil {
		t.Fatalf("new storage manager: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- sm.ConvertToWebPAsync(srcPath, 80)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("async conversion returned error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected async conversion to return immediately")
	}
}

func TestSaveFileSerializesSamePath(t *testing.T) {
	sm, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("new storage manager: %v", err)
	}

	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	secondStarted := make(chan struct{})
	secondRelease := make(chan struct{})

	go func() {
		_, _ = sm.SaveFile(&blockingReader{started: firstStarted, release: firstRelease}, "2024", "01", "same.png")
	}()

	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first save did not start")
	}

	go func() {
		_, _ = sm.SaveFile(&blockingReader{started: secondStarted, release: secondRelease}, "2024", "01", "same.png")
	}()

	select {
	case <-secondStarted:
		t.Fatal("second save started before the first completed")
	case <-time.After(200 * time.Millisecond):
	}

	close(firstRelease)
	close(secondRelease)
}
