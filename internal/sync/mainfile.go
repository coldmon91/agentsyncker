package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"agentsyncker/internal/config"
)

func syncMainFile(source config.Tool, target config.Tool) error {
	sourceContent, err := os.ReadFile(source.MainFilePath())
	if err != nil {
		return fmt.Errorf("read source main file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target.MainFilePath()), 0o755); err != nil {
		return fmt.Errorf("create target main dir: %w", err)
	}
	if err := os.WriteFile(target.MainFilePath(), sourceContent, 0o644); err != nil {
		return fmt.Errorf("write target main file: %w", err)
	}
	return nil
}
