package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentsyncker/internal/backup"
	"agentsyncker/internal/config"
)

func TestEngineSyncClaudeToGemini(t *testing.T) {
	tmp := t.TempDir()
	source := config.Tool{
		Name:       "claude",
		HomeDir:    filepath.Join(tmp, ".claude"),
		MainFile:   "CLAUDE.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		CmdFormat:  config.CommandFormatMarkdown,
	}
	target := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, ".gemini"),
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		CmdFormat:  config.CommandFormatTOML,
	}

	setupToolFixture(t, source,
		"source-main",
		"---\ndescription: \"run tests\"\n---\nexecute tests\n",
		"skill-a",
	)
	setupToolFixture(t, target,
		"target-main",
		"description = \"old\"\nprompt = \"\"\"\nold\n\"\"\"\n",
		"old-skill",
	)

	backupManager, err := backup.NewManager(filepath.Join(tmp, "backups"))
	if err != nil {
		t.Fatalf("new backup manager failed: %v", err)
	}
	engine := NewEngine(backupManager)

	results, err := engine.Sync(source, []config.Tool{target})
	if err != nil {
		t.Fatalf("engine sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].CommandsFiles != 1 {
		t.Fatalf("expected one command file, got %d", results[0].CommandsFiles)
	}
	if results[0].SkillsFiles != 1 {
		t.Fatalf("expected one skill file, got %d", results[0].SkillsFiles)
	}
	if len(results[0].Backups) != 1 {
		t.Fatalf("expected one snapshot backup, got %d", len(results[0].Backups))
	}

	mainContent, err := os.ReadFile(target.MainFilePath())
	if err != nil {
		t.Fatalf("read target main failed: %v", err)
	}
	if !strings.Contains(string(mainContent), "PROMAN-SYNC-START") {
		t.Fatalf("expected sync block in main file: %s", string(mainContent))
	}
	if !strings.Contains(string(mainContent), "source-main") {
		t.Fatalf("expected source main content in target main file: %s", string(mainContent))
	}

	cmdContent, err := os.ReadFile(filepath.Join(target.CommandDirPath(), "test.toml"))
	if err != nil {
		t.Fatalf("read synced command failed: %v", err)
	}
	if !strings.Contains(string(cmdContent), `description = "run tests"`) {
		t.Fatalf("unexpected command conversion: %s", string(cmdContent))
	}

	skillContent, err := os.ReadFile(filepath.Join(target.SkillDirPath(), "example", "SKILL.md"))
	if err != nil {
		t.Fatalf("read synced skill failed: %v", err)
	}
	if string(skillContent) != "skill-a" {
		t.Fatalf("unexpected skill content: %s", string(skillContent))
	}
}

func TestEngineSyncGeminiToClaude(t *testing.T) {
	tmp := t.TempDir()
	source := config.Tool{
		Name:       "gemini",
		HomeDir:    filepath.Join(tmp, ".gemini"),
		MainFile:   "GEMINI.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		CmdFormat:  config.CommandFormatTOML,
	}
	target := config.Tool{
		Name:       "claude",
		HomeDir:    filepath.Join(tmp, ".claude"),
		MainFile:   "CLAUDE.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		CmdFormat:  config.CommandFormatMarkdown,
	}

	setupToolFixture(t, source,
		"source-main",
		"description = \"run tests\"\nprompt = \"\"\"\nexecute tests\n\"\"\"\n",
		"skill-g",
	)
	setupToolFixture(t, target,
		"target-main",
		"---\ndescription: \"old\"\n---\nold\n",
		"old-skill",
	)

	backupManager, err := backup.NewManager(filepath.Join(tmp, "backups"))
	if err != nil {
		t.Fatalf("new backup manager failed: %v", err)
	}
	engine := NewEngine(backupManager)

	if _, err := engine.Sync(source, []config.Tool{target}); err != nil {
		t.Fatalf("engine sync failed: %v", err)
	}

	cmdContent, err := os.ReadFile(filepath.Join(target.CommandDirPath(), "test.md"))
	if err != nil {
		t.Fatalf("read synced command failed: %v", err)
	}
	if !strings.Contains(string(cmdContent), "description: \"run tests\"") {
		t.Fatalf("unexpected command conversion: %s", string(cmdContent))
	}
}

func TestEngineSyncAgentsWhenBothToolsSupportAgentDir(t *testing.T) {
	tmp := t.TempDir()
	source := config.Tool{
		Name:       "source",
		HomeDir:    filepath.Join(tmp, ".source"),
		MainFile:   "AGENTS.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		AgentDir:   "agents",
		CmdFormat:  config.CommandFormatMarkdown,
	}
	target := config.Tool{
		Name:       "target",
		HomeDir:    filepath.Join(tmp, ".target"),
		MainFile:   "AGENTS.md",
		CommandDir: "commands",
		SkillDir:   "skills",
		AgentDir:   "agents",
		CmdFormat:  config.CommandFormatMarkdown,
	}

	setupToolFixture(t, source,
		"source-main",
		"---\ndescription: \"desc\"\n---\ncommand\n",
		"skill",
	)
	setupToolFixture(t, target,
		"target-main",
		"---\ndescription: \"old\"\n---\nold\n",
		"old-skill",
	)

	sourceAgentPath := filepath.Join(source.AgentDirPath(), "planner.md")
	if err := os.MkdirAll(filepath.Dir(sourceAgentPath), 0o755); err != nil {
		t.Fatalf("create source agent dir failed: %v", err)
	}
	if err := os.WriteFile(sourceAgentPath, []byte("agent-content"), 0o644); err != nil {
		t.Fatalf("write source agent file failed: %v", err)
	}

	targetAgentPath := filepath.Join(target.AgentDirPath(), "old.md")
	if err := os.MkdirAll(filepath.Dir(targetAgentPath), 0o755); err != nil {
		t.Fatalf("create target agent dir failed: %v", err)
	}
	if err := os.WriteFile(targetAgentPath, []byte("old-agent"), 0o644); err != nil {
		t.Fatalf("write target agent file failed: %v", err)
	}

	backupManager, err := backup.NewManager(filepath.Join(tmp, "backups"))
	if err != nil {
		t.Fatalf("new backup manager failed: %v", err)
	}
	engine := NewEngine(backupManager)

	results, err := engine.Sync(source, []config.Tool{target})
	if err != nil {
		t.Fatalf("engine sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].AgentsFiles != 1 {
		t.Fatalf("expected one agent file, got %d", results[0].AgentsFiles)
	}
	if len(results[0].Backups) != 1 {
		t.Fatalf("expected one snapshot backup, got %d", len(results[0].Backups))
	}

	content, err := os.ReadFile(filepath.Join(target.AgentDirPath(), "planner.md"))
	if err != nil {
		t.Fatalf("read synced agent failed: %v", err)
	}
	if string(content) != "agent-content" {
		t.Fatalf("unexpected synced agent content: %s", string(content))
	}
}

func setupToolFixture(t *testing.T, tool config.Tool, mainContent string, commandContent string, skillContent string) {
	t.Helper()
	if err := os.MkdirAll(tool.CommandDirPath(), 0o755); err != nil {
		t.Fatalf("create command dir failed: %v", err)
	}
	if err := os.MkdirAll(tool.SkillDirPath(), 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(tool.MainFilePath()), 0o755); err != nil {
		t.Fatalf("create main dir failed: %v", err)
	}

	if err := os.WriteFile(tool.MainFilePath(), []byte(mainContent), 0o644); err != nil {
		t.Fatalf("write main file failed: %v", err)
	}

	commandExt := ".md"
	if tool.CmdFormat == config.CommandFormatTOML {
		commandExt = ".toml"
	}
	if err := os.WriteFile(filepath.Join(tool.CommandDirPath(), "test"+commandExt), []byte(commandContent), 0o644); err != nil {
		t.Fatalf("write command file failed: %v", err)
	}

	skillPath := filepath.Join(tool.SkillDirPath(), "example", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("create skill nested dir failed: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}
}
