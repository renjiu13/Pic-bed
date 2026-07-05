package handler

import (
	"strings"
	"testing"
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
