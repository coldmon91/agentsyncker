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

func newBackupCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var toolName string
	var listOnly bool
	var deleteMode bool
	var backupRoot string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create, view, or delete backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupCommand(stdout, stderr, toolName, listOnly, deleteMode, backupRoot)
		},
	}

	cmd.Flags().StringVar(&toolName, "tool", "", "tool name")
	cmd.Flags().BoolVar(&listOnly, "list", false, "list backups for the tool")
	cmd.Flags().BoolVar(&deleteMode, "delete", false, "delete backups for the tool (interactive)")
	cmd.Flags().StringVar(&backupRoot, "backup-root", "", "backup root directory (default ~/.proman/backups)")

	return cmd
}

func runBackupCommand(stdout io.Writer, stderr io.Writer, toolName string, listOnly bool, deleteMode bool, backupRoot string) error {
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
		tool, err = chooseToolInteractive(installed, os.Stdin, stdout, "Select tool for backup operations:")
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

	if listOnly && deleteMode {
		return fmt.Errorf("--list and --delete cannot be used together")
	}

	if listOnly {
		entries, err := manager.List(tool.Name)
		if err != nil {
			return fmt.Errorf("list backups failed: %w", err)
		}
		printTitle(stdout, fmt.Sprintf("Backups for %s", tool.DisplayName))
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

	if deleteMode {
		entries, err := manager.List(tool.Name)
		if err != nil {
			return fmt.Errorf("list backups failed: %w", err)
		}
		if len(entries) == 0 {
			printInfo(stdout, "No backups found to delete.")
			return nil
		}

		selected, err := chooseBackupsToDeleteInteractive(entries, os.Stdin, stdout)
		if err != nil {
			return fmt.Errorf("backup selection failed: %w", err)
		}

		deleted := 0
		for _, backupName := range selected {
			if err := manager.Delete(backupName); err != nil {
				return fmt.Errorf("delete backup failed (%s): %w", backupName, err)
			}
			deleted++
			printSuccess(stdout, fmt.Sprintf("Deleted: %s", backupName))
		}
		printInfo(stdout, fmt.Sprintf("Total deleted backups: %d", deleted))
		return nil
	}

	name, err := manager.BackupToolSnapshot(tool)
	if err != nil {
		return fmt.Errorf("backup snapshot failed: %w", err)
	}
	if name == "" {
		printInfo(stdout, "Backup skipped: no changes detected or no files to backup.")
		return nil
	}
	printSuccess(stdout, fmt.Sprintf("Backup created: %s", name))
	return nil
}
