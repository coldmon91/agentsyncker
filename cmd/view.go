package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"proman/internal/backup"
	"proman/internal/config"
	"proman/internal/detector"

	"github.com/spf13/cobra"
)

func newViewCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var toolName string
	var backupRoot string

	cmd := &cobra.Command{
		Use:   "view",
		Short: "View backup history for a tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runViewCommand(stdout, stderr, toolName, backupRoot)
		},
	}

	cmd.Flags().StringVar(&toolName, "tool", "", "tool name")
	cmd.Flags().StringVar(&backupRoot, "backup-root", "", "backup root directory (default ~/.proman/backups)")

	return cmd
}

func runViewCommand(stdout io.Writer, stderr io.Writer, toolName string, backupRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory failed: %w", err)
	}

	var tool config.Tool
	if toolName == "" {
		installed, err := detector.InstalledTools(config.DefaultTools(home))
		if err != nil {
			return fmt.Errorf("detect installed tools failed: %w", err)
		}
		tool, err = chooseToolInteractive(installed, os.Stdin, stdout, "Select tool to view history:")
		if err != nil {
			return fmt.Errorf("tool selection failed: %w", err)
		}
	} else {
		tool, err = config.FindTool(home, toolName)
		if err != nil {
			return fmt.Errorf("invalid tool: %w", err)
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

	entries, err := manager.List(tool.Name)
	if err != nil {
		return fmt.Errorf("list backups failed: %w", err)
	}

	printTitle(stdout, fmt.Sprintf("Backup History for %s", tool.DisplayName))
	if len(entries) == 0 {
		printInfo(stdout, "No backups found.")
		return nil
	}

	for _, entry := range entries {
		preRestore := ""
		if entry.PreRestore {
			preRestore = " [pre-restore]"
		}
		_, _ = fmt.Fprintf(stdout, "%s  %s  %s%s\n",
			styleInfo.Render(entry.Name),
			styleDim.Render(entry.Timestamp.Format("2006-01-02 15:04:05")),
			entry.Asset,
			styleDim.Render(preRestore),
		)
	}
	return nil
}
