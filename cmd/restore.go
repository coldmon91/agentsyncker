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

func newRestoreCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var toolName string
	var backupName string
	var backupRoot string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore tool data from a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestoreCommand(stdout, stderr, toolName, backupName, backupRoot)
		},
	}

	cmd.Flags().StringVar(&toolName, "tool", "", "tool name")
	cmd.Flags().StringVar(&backupName, "backup", "", "backup filename")
	cmd.Flags().StringVar(&backupRoot, "backup-root", "", "backup root directory (default ~/.proman/backups)")

	return cmd
}

func runRestoreCommand(stdout io.Writer, stderr io.Writer, toolName string, backupName string, backupRoot string) error {
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
		tool, err = chooseToolInteractive(installed, os.Stdin, stdout, "Select tool to restore:")
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

	selectedBackup := backupName
	if selectedBackup == "" {
		entries, err := manager.List(tool.Name)
		if err != nil {
			return fmt.Errorf("list backups failed: %w", err)
		}
		selectedBackup, err = chooseBackupInteractive(entries, os.Stdin, stdout)
		if err != nil {
			return fmt.Errorf("backup selection failed: %w", err)
		}
	}

	targetPath, err := manager.Restore(tool, selectedBackup)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	printTitle(stdout, "Restore Complete")
	printSuccess(stdout, fmt.Sprintf("Successfully restored %s data from backup: %s", tool.DisplayName, selectedBackup))
	printInfo(stdout, fmt.Sprintf("Restored path: %s", targetPath))
	return nil
}
