package sync

import (
	"fmt"
	"os"

	"proman/internal/backup"
	"proman/internal/config"
)

type Result struct {
	Target        string
	Backups       []string
	CommandsFiles int
	SkillsFiles   int
	AgentsFiles   int
}

type Engine struct {
	BackupManager *backup.Manager
}

func NewEngine(backupManager *backup.Manager) *Engine {
	return &Engine{BackupManager: backupManager}
}

func (e *Engine) Sync(source config.Tool, targets []config.Tool) ([]Result, error) {
	if e.BackupManager == nil {
		return nil, fmt.Errorf("backup manager is required")
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}

	if _, err := os.Stat(source.MainFilePath()); err != nil {
		return nil, fmt.Errorf("source main file missing: %w", err)
	}
	if _, err := os.Stat(source.CommandDirPath()); err != nil {
		return nil, fmt.Errorf("source command dir missing: %w", err)
	}
	if _, err := os.Stat(source.SkillDirPath()); err != nil {
		return nil, fmt.Errorf("source skill dir missing: %w", err)
	}

	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		if target.Name == source.Name {
			return nil, fmt.Errorf("source and target cannot be the same: %s", source.Name)
		}

		result := Result{Target: target.Name}
		if backupName, err := e.BackupManager.BackupToolSnapshot(target); err != nil {
			return nil, fmt.Errorf("backup snapshot for %s failed: %w", target.Name, err)
		} else if backupName != "" {
			result.Backups = append(result.Backups, backupName)
		}

		if err := syncMainFile(source, target); err != nil {
			return nil, fmt.Errorf("sync main file to %s failed: %w", target.Name, err)
		}

		commandsCount, err := syncCommands(source, target)
		if err != nil {
			return nil, fmt.Errorf("sync commands to %s failed: %w", target.Name, err)
		}
		result.CommandsFiles = commandsCount

		skillsCount, err := syncSkills(source, target)
		if err != nil {
			return nil, fmt.Errorf("sync skills to %s failed: %w", target.Name, err)
		}
		result.SkillsFiles = skillsCount
		if source.AgentDirPath() != "" && target.AgentDirPath() != "" {
			agentsCount, err := syncAgents(source, target)
			if err != nil {
				return nil, fmt.Errorf("sync agents to %s failed: %w", target.Name, err)
			}
			result.AgentsFiles = agentsCount
		}

		results = append(results, result)
	}

	return results, nil
}
