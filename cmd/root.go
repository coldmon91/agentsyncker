package cmd

import (
	"fmt"
	"io"
	"os"

	"agentsyncker/internal/config"
	"agentsyncker/internal/detector"

	"github.com/spf13/cobra"
)

const Version = "v0.0.1"

func Execute(args []string, stdout io.Writer, stderr io.Writer) int {
	rootCmd := newRootCmd(stdout, stderr)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func newRootCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "agentsyncker",
		Short:         "Prompt sync manager for AI coding tools",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefaultInteractive(stdout, stderr)
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.AddCommand(
		newDetectCmd(stdout, stderr),
		newSyncCmd(stdout, stderr),
		newBackupCmd(stdout, stderr),
		newRestoreCmd(stdout, stderr),
		newViewCmd(stdout, stderr),
	)

	return rootCmd
}

func runDefaultInteractive(stdout io.Writer, stderr io.Writer) error {
	mode, err := chooseMainModeInteractive(os.Stdin, stdout)
	if err != nil {
		return fmt.Errorf("mode selection failed: %w", err)
	}

	if mode == mainModeSync {
		return runSyncCommand(stdout, stderr, "", "", "")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory failed: %w", err)
	}
	installed, err := detector.InstalledTools(config.DefaultTools(home))
	if err != nil {
		return fmt.Errorf("detect installed tools failed: %w", err)
	}

	tool, err := chooseToolInteractive(installed, os.Stdin, stdout, "Tool number: ")
	if err != nil {
		return fmt.Errorf("tool selection failed: %w", err)
	}

	switch mode {
	case mainModeRestore:
		return runRestoreCommand(stdout, stderr, tool.Name, "", "")
	case mainModeView:
		return runViewCommand(stdout, stderr, tool.Name, "")
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}
