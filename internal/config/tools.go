package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type CommandFormat string

const (
	CommandFormatMarkdown CommandFormat = "md"
	CommandFormatTOML     CommandFormat = "toml"
)

type Tool struct {
	Name        string
	DisplayName string
	HomeDir     string
	MainFile    string
	CommandDir  string
	SkillDir    string
	AgentDir    string
	CmdFormat   CommandFormat
}

func ExpandHome(path string, home string) string {
	if path == "~" {
		return filepath.Clean(home)
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Clean(filepath.Join(home, strings.TrimPrefix(path, "~/")))
	}
	return filepath.Clean(path)
}

func DefaultTools(home string) []Tool {
	return []Tool{
		{
			Name:        "claude",
			DisplayName: "Claude Code",
			HomeDir:     ExpandHome("~/.claude", home),
			MainFile:    "CLAUDE.md",
			CommandDir:  "commands",
			SkillDir:    "skills",
			CmdFormat:   CommandFormatMarkdown,
		},
		{
			Name:        "codex",
			DisplayName: "Codex CLI",
			HomeDir:     ExpandHome("~/.codex", home),
			MainFile:    "AGENTS.md",
			CommandDir:  "prompts",
			SkillDir:    "skills",
			CmdFormat:   CommandFormatMarkdown,
		},
		{
			Name:        "gemini",
			DisplayName: "Gemini CLI",
			HomeDir:     ExpandHome("~/.gemini", home),
			MainFile:    "GEMINI.md",
			CommandDir:  "commands",
			SkillDir:    "skills",
			CmdFormat:   CommandFormatTOML,
		},
		{
			Name:        "opencode",
			DisplayName: "OpenCode",
			HomeDir:     ExpandHome("~/.config/opencode", home),
			MainFile:    "AGENTS.md",
			CommandDir:  "commands",
			SkillDir:    "skills",
			AgentDir:    "agents",
			CmdFormat:   CommandFormatMarkdown,
		},
	}
}

func ToolMap(home string) map[string]Tool {
	tools := DefaultTools(home)
	out := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		out[tool.Name] = tool
	}
	return out
}

func FindTool(home string, name string) (Tool, error) {
	tool, ok := ToolMap(home)[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Tool{}, fmt.Errorf("unknown tool %q", name)
	}
	return tool, nil
}

func (t Tool) MainFilePath() string {
	return filepath.Join(t.HomeDir, t.MainFile)
}

func (t Tool) CommandDirPath() string {
	return filepath.Join(t.HomeDir, t.CommandDir)
}

func (t Tool) SkillDirPath() string {
	return filepath.Join(t.HomeDir, t.SkillDir)
}

func (t Tool) AgentDirPath() string {
	if t.AgentDir == "" {
		return ""
	}
	return filepath.Join(t.HomeDir, t.AgentDir)
}

func (t Tool) DirAssets() map[string]string {
	assets := map[string]string{
		filepath.Base(t.CommandDir): t.CommandDirPath(),
		filepath.Base(t.SkillDir):   t.SkillDirPath(),
	}
	if t.AgentDir != "" {
		assets[filepath.Base(t.AgentDir)] = t.AgentDirPath()
	}
	return assets
}

func (t Tool) ResolveAssetPath(asset string, extension string) (string, error) {
	asset = strings.TrimSpace(asset)
	if extension == ".bak" && asset == filepath.Base(t.MainFile) {
		return t.MainFilePath(), nil
	}
	if extension == ".tar.gz" {
		for assetName, target := range t.DirAssets() {
			if assetName == asset {
				return target, nil
			}
		}
	}
	return "", fmt.Errorf("unsupported asset %q for tool %q", asset, t.Name)
}
