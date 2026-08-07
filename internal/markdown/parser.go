package markdown

import (
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

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if title == "" && strings.HasPrefix(trimmed, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			continue
		}
		if title != "" {
			paragraphs = append(paragraphs, trimmed)
		}
	}

	if title == "" {
		title = "Untitled Chapter"
	}

	return Chapter{Title: title, Paragraphs: paragraphs}, nil
}
