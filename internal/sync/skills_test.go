package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMirrorDirectory(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source")
	target := filepath.Join(tmp, "target")

	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir source failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "SKILL.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write source failed: %v", err)
	}

	count, err := mirrorDirectory(source, target)
	if err != nil {
		t.Fatalf("mirror directory failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 copied file, got %d", count)
	}

	content, err := os.ReadFile(filepath.Join(target, "nested", "SKILL.md"))
	if err != nil {
		t.Fatalf("read target file failed: %v", err)
	}
	if string(content) != "content" {
		t.Fatalf("unexpected mirrored content: %s", string(content))
	}
}
