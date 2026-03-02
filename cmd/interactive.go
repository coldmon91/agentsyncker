package cmd

import (
	"fmt"
	"io"

	"agentsyncker/internal/backup"
	"agentsyncker/internal/config"

	"github.com/charmbracelet/huh"
)

type mainMode string

const (
	mainModeSync    mainMode = "sync"
	mainModeRestore mainMode = "restore"
	mainModeView    mainMode = "view"
)

func chooseMainModeInteractive(in io.Reader, out io.Writer) (mainMode, error) {
	var selected mainMode
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[mainMode]().
				Title("Select Mode").
				Description("What would you like to do?").
				Options(
					huh.NewOption("Sync (Compare and update across tools)", mainModeSync),
					huh.NewOption("Restore (Rollback to a previous state)", mainModeRestore),
					huh.NewOption("View (Examine current prompts)", mainModeView),
				).
				Value(&selected),
		),
	).WithInput(in).WithOutput(out)

	err := form.Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

func chooseToolInteractive(tools []config.Tool, in io.Reader, out io.Writer, prompt string) (config.Tool, error) {
	if len(tools) == 0 {
		return config.Tool{}, fmt.Errorf("no tools found")
	}

	options := make([]huh.Option[int], len(tools))
	for idx, tool := range tools {
		options[idx] = huh.NewOption(fmt.Sprintf("%s (%s)", tool.DisplayName, tool.Name), idx)
	}

	var selectedIdx int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select Tool").
				Description(prompt).
				Options(options...).
				Value(&selectedIdx),
		),
	).WithInput(in).WithOutput(out)

	err := form.Run()
	if err != nil {
		return config.Tool{}, err
	}
	return tools[selectedIdx], nil
}

func chooseSourceAndTargetsInteractive(installed []config.Tool, in io.Reader, out io.Writer) (config.Tool, []config.Tool, error) {
	if len(installed) < 2 {
		return config.Tool{}, nil, fmt.Errorf("at least 2 installed tools are required for sync")
	}

	sourceOptions := make([]huh.Option[int], len(installed))
	for idx, tool := range installed {
		sourceOptions[idx] = huh.NewOption(fmt.Sprintf("%s (%s)", tool.DisplayName, tool.Name), idx)
	}

	var sourceIdx int
	sourceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select Source Tool").
				Description("Which tool's prompts should be the source?").
				Options(sourceOptions...).
				Value(&sourceIdx),
		),
	).WithInput(in).WithOutput(out)

	if err := sourceForm.Run(); err != nil {
		return config.Tool{}, nil, err
	}

	source := installed[sourceIdx]
	targetOptions := []huh.Option[int]{
		huh.NewOption("All (All targets)", -1),
	}
	for idx, tool := range installed {
		if tool.Name == source.Name {
			continue
		}
		targetOptions = append(targetOptions, huh.NewOption(fmt.Sprintf("%s (%s)", tool.DisplayName, tool.Name), idx))
	}

	var targetIndices []int
	targetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select Target Tools").
				Description("Press Space to select/deselect, Enter to confirm").
				Options(targetOptions...).
				Value(&targetIndices).
				Validate(func(v []int) error {
					if len(v) == 0 {
						return fmt.Errorf("please select at least one target")
					}
					return nil
				}),
		),
	).WithInput(in).WithOutput(out)

	if err := targetForm.Run(); err != nil {
		return config.Tool{}, nil, err
	}

	isAll := false
	for _, idx := range targetIndices {
		if idx == -1 {
			isAll = true
			break
		}
	}

	var targets []config.Tool
	if isAll {
		for _, tool := range installed {
			if tool.Name == source.Name {
				continue
			}
			targets = append(targets, tool)
		}
	} else {
		targets = make([]config.Tool, len(targetIndices))
		for i, idx := range targetIndices {
			targets[i] = installed[idx]
		}
	}

	return source, targets, nil
}

func chooseBackupInteractive(entries []backup.Entry, in io.Reader, out io.Writer) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("no backups found")
	}

	options := make([]huh.Option[int], len(entries))
	for idx, entry := range entries {
		label := fmt.Sprintf("%s (%s)", entry.Name, entry.Timestamp.Format("2006-01-02 15:04:05"))
		if entry.PreRestore {
			label += " [pre-restore]"
		}
		options[idx] = huh.NewOption(label, idx)
	}

	var selectedIdx int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select Backup to Restore").
				Description("Choose a backup version to roll back to.").
				Options(options...).
				Value(&selectedIdx),
		),
	).WithInput(in).WithOutput(out)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return entries[selectedIdx].Name, nil
}

func chooseBackupsToDeleteInteractive(entries []backup.Entry, in io.Reader, out io.Writer) ([]string, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no backups found")
	}

	options := []huh.Option[int]{
		huh.NewOption("All (All backups)", -1),
	}
	for idx, entry := range entries {
		label := fmt.Sprintf("%s (%s)", entry.Name, entry.Timestamp.Format("2006-01-02 15:04:05"))
		if entry.PreRestore {
			label += " [pre-restore]"
		}
		options = append(options, huh.NewOption(label, idx))
	}

	var selectedIndices []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select Backups to Delete").
				Description("Choose one or more backups to remove (Space to select)").
				Options(options...).
				Value(&selectedIndices).
				Validate(func(v []int) error {
					if len(v) == 0 {
						return fmt.Errorf("please select at least one backup")
					}
					return nil
				}),
		),
	).WithInput(in).WithOutput(out)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	isAll := false
	for _, idx := range selectedIndices {
		if idx == -1 {
			isAll = true
			break
		}
	}

	var names []string
	if isAll {
		names = make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name
		}
	} else {
		names = make([]string, len(selectedIndices))
		for i, idx := range selectedIndices {
			names[i] = entries[idx].Name
		}
	}

	return names, nil
}
