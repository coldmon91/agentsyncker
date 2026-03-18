package converter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Command struct {
	Description string
	Prompt      string
}

var descriptionPattern = regexp.MustCompile(`(?m)^description\s*=\s*("(?:\\.|[^"])*")\s*$`)

var promptMultilinePattern = regexp.MustCompile(`(?s)prompt\s*=\s*"""\n?(.*?)"""`)

var promptMultilineLiteralPattern = regexp.MustCompile(`(?s)prompt\s*=\s*'''\n?(.*?)'''`)

var promptSingleLinePattern = regexp.MustCompile(`(?m)^prompt\s*=\s*("(?:\\.|[^"])*")\s*$`)

func ParseMarkdown(input []byte) (Command, error) {
	text := normalizeNewlines(string(input))
	description := ""
	body := text

	if strings.HasPrefix(text, "---\n") {
		rest := strings.TrimPrefix(text, "---\n")
		idx := strings.Index(rest, "\n---\n")
		if idx >= 0 {
			frontmatter := rest[:idx]
			body = rest[idx+5:]
			description = parseDescriptionFromFrontmatter(frontmatter)
		}
	}

	body = strings.TrimLeft(body, "\n")
	body = strings.TrimRight(body, "\n")
	return Command{Description: description, Prompt: body}, nil
}

func EncodeMarkdown(command Command) []byte {
	var b strings.Builder
	if command.Description != "" {
		b.WriteString("---\n")
		b.WriteString("description: ")
		b.WriteString(strconv.Quote(command.Description))
		b.WriteString("\n---\n")
	}

	prompt := strings.TrimRight(command.Prompt, "\n")
	if prompt != "" {
		b.WriteString(prompt)
		b.WriteString("\n")
	}

	return []byte(b.String())
}

func ParseTOML(input []byte) (Command, error) {
	text := normalizeNewlines(string(input))
	var cmd Command

	descriptionMatch := descriptionPattern.FindStringSubmatch(text)
	if len(descriptionMatch) == 2 {
		description, err := strconv.Unquote(descriptionMatch[1])
		if err != nil {
			return Command{}, fmt.Errorf("parse description: %w", err)
		}
		cmd.Description = description
	}

	promptMultilineMatch := promptMultilinePattern.FindStringSubmatch(text)
	if len(promptMultilineMatch) == 2 {
		cmd.Prompt = strings.ReplaceAll(promptMultilineMatch[1], `\\"\\"\\"`, `"""`)
		cmd.Prompt = strings.TrimRight(cmd.Prompt, "\n")
		return cmd, nil
	}

	promptMultilineLiteralMatch := promptMultilineLiteralPattern.FindStringSubmatch(text)
	if len(promptMultilineLiteralMatch) == 2 {
		cmd.Prompt = strings.TrimRight(promptMultilineLiteralMatch[1], "\n")
		return cmd, nil
	}

	promptSingleLineMatch := promptSingleLinePattern.FindStringSubmatch(text)
	if len(promptSingleLineMatch) == 2 {
		prompt, err := strconv.Unquote(promptSingleLineMatch[1])
		if err != nil {
			return Command{}, fmt.Errorf("parse prompt: %w", err)
		}
		cmd.Prompt = prompt
		return cmd, nil
	}

	return cmd, nil
}

func EncodeTOML(command Command) []byte {
	description := strconv.Quote(command.Description)
	prompt := strings.TrimRight(command.Prompt, "\n")

	if !strings.Contains(prompt, `'''`) {
		output := fmt.Sprintf("description = %s\nprompt = '''\n%s\n'''\n", description, prompt)
		return []byte(output)
	}

	output := fmt.Sprintf("description = %s\nprompt = %s\n", description, strconv.Quote(prompt))
	return []byte(output)
}

func MDToTOML(input []byte) ([]byte, error) {
	cmd, err := ParseMarkdown(input)
	if err != nil {
		return nil, err
	}
	return EncodeTOML(cmd), nil
}

func TOMLToMD(input []byte) ([]byte, error) {
	cmd, err := ParseTOML(input)
	if err != nil {
		return nil, err
	}
	return EncodeMarkdown(cmd), nil
}

func parseDescriptionFromFrontmatter(frontmatter string) string {
	for _, line := range strings.Split(frontmatter, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "description:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
		return strings.Trim(value, "'\"")
	}
	return ""
}

func normalizeNewlines(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}
