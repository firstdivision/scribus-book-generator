package markdown

import (
	"fmt"
	"os"
	"strings"
)

type Chapter struct {
	Title      string
	Paragraphs []string
}

func ParseFile(path string) (Chapter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Chapter{}, err
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var title string
	var paragraphs []string

	for lineNumber, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			if title != "" {
				return Chapter{}, fmt.Errorf("additional H1 heading on line %d; only the first H1 may be a chapter title", lineNumber+1)
			}
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title == "" {
				return Chapter{}, fmt.Errorf("chapter title on line %d must be non-empty", lineNumber+1)
			}
			continue
		}
		paragraphs = append(paragraphs, trimmed)
	}

	if title == "" {
		title = "Untitled Chapter"
	}

	return Chapter{Title: title, Paragraphs: paragraphs}, nil
}
