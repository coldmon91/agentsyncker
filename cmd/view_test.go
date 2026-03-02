package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunViewWithToolFlag(t *testing.T) {
	tmp := t.TempDir()
	backupRoot := filepath.Join(tmp, "backups")
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		t.Fatalf("create backup root failed: %v", err)
	}

	backupName := "gemini_snapshot_20260302_143000.tar.gz"
	if err := os.WriteFile(filepath.Join(backupRoot, backupName), []byte("x"), 0o644); err != nil {
		t.Fatalf("create backup file failed: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := runViewCommand(stdout, stderr, "gemini", backupRoot)
	if err != nil {
		t.Fatalf("expected nil error, got %v (stderr=%s)", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, backupName) {
		t.Fatalf("expected backup name in output, got: %s", output)
	}
	// Check for timestamp without 'time=' prefix as it was removed in the new UI
	if !strings.Contains(output, "2026-03-02 14:30:00") {
		t.Fatalf("expected formatted timestamp in output, got: %s", output)
	}
}
