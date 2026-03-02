package syncblock

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	startPrefix = "<!-- PROMAN-SYNC-START source=%s -->"
	endMarker   = "<!-- PROMAN-SYNC-END -->"
)

var blockPattern = regexp.MustCompile(`(?s)<!--\s*PROMAN-SYNC-START\s+source=([^\n]+?)\s*-->.*?<!--\s*PROMAN-SYNC-END\s*-->`)

var blockContentPattern = regexp.MustCompile(`(?s)^<!--\s*PROMAN-SYNC-START\s+source=([^\n]+?)\s*-->\n?(.*?)\n?<!--\s*PROMAN-SYNC-END\s*-->$`)

type Block struct {
	Source  string
	Content string
}

func Render(sourcePath string, sourceContent string) string {
	trimmed := strings.TrimRight(sourceContent, "\n")
	return fmt.Sprintf(startPrefix, sourcePath) + "\n" + trimmed + "\n" + endMarker
}

func Upsert(existing string, sourcePath string, sourceContent string) (string, bool) {
	block := Render(sourcePath, sourceContent)
	if blockPattern.MatchString(existing) {
		return blockPattern.ReplaceAllString(existing, block), true
	}

	if strings.TrimSpace(existing) == "" {
		return block + "\n", false
	}

	existing = strings.TrimRight(existing, "\n")
	return existing + "\n\n" + block + "\n", false
}

func Extract(input string) (Block, bool) {
	full := blockPattern.FindString(input)
	if full == "" {
		return Block{}, false
	}

	matches := blockContentPattern.FindStringSubmatch(full)
	if len(matches) != 3 {
		return Block{}, false
	}

	return Block{Source: strings.TrimSpace(matches[1]), Content: matches[2]}, true
}
