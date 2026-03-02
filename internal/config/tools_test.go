package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultTools(t *testing.T) {
	home := "/tmp/home"
	tools := DefaultTools(home)
	if len(tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(tools))
	}

	found := map[string]Tool{}
	for _, tool := range tools {
		found[tool.Name] = tool
	}

	claude := found["claude"]
	if claude.HomeDir != filepath.Clean("/tmp/home/.claude") {
		t.Fatalf("unexpected claude home dir: %s", claude.HomeDir)
	}
	if claude.CmdFormat != CommandFormatMarkdown {
		t.Fatalf("expected markdown format, got %s", claude.CmdFormat)
	}

	gemini := found["gemini"]
	if gemini.MainFile != "GEMINI.md" {
		t.Fatalf("unexpected gemini main file: %s", gemini.MainFile)
	}
	if gemini.CmdFormat != CommandFormatTOML {
		t.Fatalf("expected toml format, got %s", gemini.CmdFormat)
	}
}

func TestFindTool(t *testing.T) {
	home := "/tmp/home"
	tool, err := FindTool(home, "GeMiNi")
	if err != nil {
		t.Fatalf("find tool returned error: %v", err)
	}
	if tool.Name != "gemini" {
		t.Fatalf("expected gemini, got %s", tool.Name)
	}

	if _, err := FindTool(home, "unknown"); err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestResolveAssetPath(t *testing.T) {
	tool := Tool{
		Name:       "gemini",
		HomeDir:    "/tmp/home/.gemini",
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
	}

	filePath, err := tool.ResolveAssetPath("GEMINI.md", ".bak")
	if err != nil {
		t.Fatalf("resolve main file failed: %v", err)
	}
	if filePath != "/tmp/home/.gemini/GEMINI.md" {
		t.Fatalf("unexpected file path: %s", filePath)
	}

	dirPath, err := tool.ResolveAssetPath("commands", ".tar.gz")
	if err != nil {
		t.Fatalf("resolve command dir failed: %v", err)
	}
	if dirPath != "/tmp/home/.gemini/commands" {
		t.Fatalf("unexpected dir path: %s", dirPath)
	}

	if _, err := tool.ResolveAssetPath("commands", ".bak"); err == nil {
		t.Fatal("expected error for unsupported file backup asset")
	}
}
