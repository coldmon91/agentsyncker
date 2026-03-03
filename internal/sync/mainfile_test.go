package sync

import (
	"os"
	"path/filepath"
	"testing"

	"agentsyncker/internal/config"
)

func TestSyncMainFileReplacesTarget(t *testing.T) {
	tmp := t.TempDir()
	source := config.Tool{Name: "claude", HomeDir: filepath.Join(tmp, "src"), MainFile: "CLAUDE.md"}
	target := config.Tool{Name: "gemini", HomeDir: filepath.Join(tmp, "dst"), MainFile: "GEMINI.md"}

	if err := os.MkdirAll(source.HomeDir, 0o755); err != nil {
		t.Fatalf("create source dir failed: %v", err)
	}
	if err := os.MkdirAll(target.HomeDir, 0o755); err != nil {
		t.Fatalf("create target dir failed: %v", err)
	}

	if err := os.WriteFile(source.MainFilePath(), []byte("source-main"), 0o644); err != nil {
		t.Fatalf("write source main failed: %v", err)
	}
	if err := os.WriteFile(target.MainFilePath(), []byte("target-main"), 0o644); err != nil {
		t.Fatalf("write target main failed: %v", err)
	}

	if err := syncMainFile(source, target); err != nil {
		t.Fatalf("syncMainFile failed: %v", err)
	}

	content, err := os.ReadFile(target.MainFilePath())
	if err != nil {
		t.Fatalf("read target main failed: %v", err)
	}
	if string(content) != "source-main" {
		t.Fatalf("expected target main file to be replaced, got: %s", string(content))
	}
}
