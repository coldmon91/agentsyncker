package detector

import (
	"os"

	"agentsyncker/internal/config"
)

type Status struct {
	Tool      config.Tool
	Installed bool
}

func Detect(tools []config.Tool) ([]Status, error) {
	statuses := make([]Status, 0, len(tools))
	for _, tool := range tools {
		info, err := os.Stat(tool.HomeDir)
		installed := err == nil && info.IsDir()
		statuses = append(statuses, Status{Tool: tool, Installed: installed})
	}
	return statuses, nil
}

func InstalledTools(tools []config.Tool) ([]config.Tool, error) {
	statuses, err := Detect(tools)
	if err != nil {
		return nil, err
	}

	out := make([]config.Tool, 0, len(statuses))
	for _, status := range statuses {
		if status.Installed {
			out = append(out, status.Tool)
		}
	}
	return out, nil
}
