package tools

import (
	"os"
	"strings"
	"testing"
)

func TestViewImageValid(t *testing.T) {
	dataURL, meta := ViewImage("/tmp/test.png")
	if dataURL == "" {
		t.Fatalf("expected non-empty data URL, got empty; meta=%s", meta)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("unexpected data URL prefix: %q", dataURL[:50])
	}
	if meta == "" {
		t.Fatal("expected non-empty metadata")
	}
	t.Logf("OK: %s → %d bytes data URL", meta, len(dataURL))
}

func TestViewImageMissing(t *testing.T) {
	_, meta := ViewImage("/tmp/does-not-exist-12345.png")
	if !strings.Contains(meta, "error") {
		t.Fatalf("expected error for missing file, got: %s", meta)
	}
	t.Logf("OK: %s", meta)
}

func TestViewImageEmptyPath(t *testing.T) {
	_, meta := ViewImage("")
	if !strings.Contains(meta, "empty path") {
		t.Fatalf("expected empty-path error, got: %s", meta)
	}
	t.Logf("OK: %s", meta)
}

func TestViewImageNotAnImage(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/fake.png"
	if err := os.WriteFile(path, []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	_, result := ViewImage(path)
	if !strings.Contains(result, "error") {
		t.Fatalf("expected error for non-image file, got: %s", result)
	}
	t.Logf("OK: %s", result)
}

func TestViewImageSchemaShape(t *testing.T) {
	sch := ViewImageSchema()
	fn := sch["function"].(map[string]any)
	if fn["name"] != ViewImageName {
		t.Fatalf("schema name = %v, want %q", fn["name"], ViewImageName)
	}
	params := fn["parameters"].(map[string]any)
	required := params["required"].([]string)
	found := false
	for _, r := range required {
		if r == "path" {
			found = true
		}
	}
	if !found {
		t.Fatal("schema missing 'path' in required")
	}
}
