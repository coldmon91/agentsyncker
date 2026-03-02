package cmd

import (
	"fmt"
	"io"
	"os"

	"proman/internal/config"
	"proman/internal/detector"

	"github.com/spf13/cobra"
)

func newDetectCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Detect installed tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetectCommand(stdout, stderr)
		},
	}
}

func runDetectCommand(stdout io.Writer, stderr io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory failed: %w", err)
	}

	statuses, err := detector.Detect(config.DefaultTools(home))
	if err != nil {
		return fmt.Errorf("detect failed: %w", err)
	}

	printTitle(stdout, "Detected Tools")
	for _, status := range statuses {
		if status.Installed {
			printSuccess(stdout, fmt.Sprintf("%s (%s)", status.Tool.DisplayName, styleDim.Render(status.Tool.HomeDir)))
		} else {
			printError(stdout, fmt.Sprintf("%s (%s - not found)", status.Tool.DisplayName, styleDim.Render(status.Tool.HomeDir)))
		}
	}
	return nil
}
