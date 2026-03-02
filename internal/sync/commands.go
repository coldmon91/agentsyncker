package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"proman/internal/config"
	"proman/internal/converter"
)

func syncCommands(source config.Tool, target config.Tool) (int, error) {
	sourceDir := source.CommandDirPath()
	targetDir := target.CommandDirPath()

	if err := removeAndRecreateDir(targetDir); err != nil {
		return 0, fmt.Errorf("prepare target command dir: %w", err)
	}

	sourceExt := commandExtension(source.CmdFormat)
	targetExt := commandExtension(target.CmdFormat)
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

		if strings.EqualFold(filepath.Ext(path), sourceExt) {
			converted, err := convertCommand(content, source.CmdFormat, target.CmdFormat)
			if err != nil {
				return fmt.Errorf("convert command %s: %w", path, err)
			}
			content = converted
			targetPath = strings.TrimSuffix(targetPath, filepath.Ext(targetPath)) + targetExt
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

func convertCommand(content []byte, from config.CommandFormat, to config.CommandFormat) ([]byte, error) {
	if from == to {
		return content, nil
	}

	if from == config.CommandFormatMarkdown && to == config.CommandFormatTOML {
		return converter.MDToTOML(content)
	}
	if from == config.CommandFormatTOML && to == config.CommandFormatMarkdown {
		return converter.TOMLToMD(content)
	}
	return nil, fmt.Errorf("unsupported command conversion: %s -> %s", from, to)
}

func commandExtension(format config.CommandFormat) string {
	if format == config.CommandFormatTOML {
		return ".toml"
	}
	return ".md"
}

func removeAndRecreateDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}
