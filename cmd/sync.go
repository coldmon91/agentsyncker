package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"proman/internal/backup"
	"proman/internal/config"
	"proman/internal/detector"
	"proman/internal/sync"

	"github.com/spf13/cobra"
)

func newSyncCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var sourceName string
	var targetNames string
	var backupRoot string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize prompt assets from source tool to targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncCommand(stdout, stderr, sourceName, targetNames, backupRoot)
		},
	}

	cmd.Flags().StringVar(&sourceName, "source", "", "source tool (claude|codex|gemini|opencode)")
	cmd.Flags().StringVar(&targetNames, "target", "", "target tools separated by commas")
	cmd.Flags().StringVar(&backupRoot, "backup-root", "", "backup root directory (default ~/.proman/backups)")

	return cmd
}

func runSyncCommand(stdout io.Writer, stderr io.Writer, sourceName string, targetNames string, backupRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory failed: %w", err)
	}

	var source config.Tool
	var targets []config.Tool

	if sourceName == "" || targetNames == "" {
		installed, err := detector.InstalledTools(config.DefaultTools(home))
		if err != nil {
			return fmt.Errorf("detect installed tools failed: %w", err)
		}

		source, targets, err = chooseSourceAndTargetsInteractive(installed, os.Stdin, stdout)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}
	} else {
		source, err = config.FindTool(home, sourceName)
		if err != nil {
			return fmt.Errorf("invalid source: %w", err)
		}

		targets, err = resolveTargets(home, targetNames)
		if err != nil {
			return fmt.Errorf("invalid targets: %w", err)
		}
	}

	resolvedBackupRoot := backupRoot
	if resolvedBackupRoot == "" {
		resolvedBackupRoot = backup.DefaultRoot(home)
	}
	resolvedBackupRoot = filepath.Clean(resolvedBackupRoot)

	manager, err := backup.NewManager(resolvedBackupRoot)
	if err != nil {
		return fmt.Errorf("initialize backup manager failed: %w", err)
	}

	engine := sync.NewEngine(manager)
	results, err := engine.Sync(source, targets)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	printTitle(stdout, fmt.Sprintf("Sync Results (Source: %s)", source.DisplayName))

	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, styleInfo.Render(fmt.Sprintf("Target: %s", result.Target)))
		if len(result.Backups) > 0 {
			printInfo(stdout, fmt.Sprintf("Created backups: %s", strings.Join(result.Backups, ", ")))
		}
		printSuccess(stdout, fmt.Sprintf("Synchronized %d commands, %d skills, and %d agents", result.CommandsFiles, result.SkillsFiles, result.AgentsFiles))
		_, _ = fmt.Fprintln(stdout)
	}
	return nil
}

func resolveTargets(home string, input string) ([]config.Tool, error) {
	names := strings.Split(input, ",")
	seen := map[string]struct{}{}
	result := make([]config.Tool, 0, len(names))
	for _, name := range names {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		tool, err := config.FindTool(home, trimmed)
		if err != nil {
			return nil, err
		}
		result = append(result, tool)
		seen[trimmed] = struct{}{}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid targets")
	}
	return result, nil
}
