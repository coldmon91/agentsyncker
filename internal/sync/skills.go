package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"agentsyncker/internal/config"
)

func syncSkills(source config.Tool, target config.Tool) (int, error) {
	return mirrorDirectory(source.SkillDirPath(), target.SkillDirPath())
}

func mirrorDirectory(sourceDir string, targetDir string) (int, error) {
	if err := removeAndRecreateDir(targetDir); err != nil {
		return 0, fmt.Errorf("prepare target dir: %w", err)
	}

	count := 0
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
