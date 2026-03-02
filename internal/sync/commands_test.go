package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentsyncker/internal/config"
)

func TestSyncCommandsMarkdownToTOML(t *testing.T) {
	tmp := t.TempDir()
	source := config.Tool{
		Name:       "claude",
		HomeDir:    filepath.Join(tmp, "source"),
		CommandDir: "commands",
		CmdFormat:  config.CommandFormatMarkdown,
	}
	target := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, "target"),
		CommandDir: "commands",
		CmdFormat:  config.CommandFormatTOML,
	}

	sourceCmdPath := filepath.Join(source.CommandDirPath(), "test.md")
	if err := os.MkdirAll(filepath.Dir(sourceCmdPath), 0o755); err != nil {
		t.Fatalf("mkdir source cmd dir failed: %v", err)
	}
	if err := os.WriteFile(sourceCmdPath, []byte("---\ndescription: \"desc\"\n---\nrun tests\n"), 0o644); err != nil {
		t.Fatalf("write source cmd failed: %v", err)
	}

	count, err := syncCommands(source, target)
	if err != nil {
		t.Fatalf("syncCommands failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one command file, got %d", count)
	}

	targetCmdPath := filepath.Join(target.CommandDirPath(), "test.toml")
	content, err := os.ReadFile(targetCmdPath)
	if err != nil {
		t.Fatalf("read target command failed: %v", err)
	}
	if !strings.Contains(string(content), `description = "desc"`) {
		t.Fatalf("unexpected converted content: %s", string(content))
	}
}
