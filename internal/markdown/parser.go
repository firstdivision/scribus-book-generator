package markdown

import (
	"fmt"
	"os"
	"strings"
)

type Chapter struct {
	Title      string
	Paragraphs []string
	Blocks     []Block
}

type Block struct {
	Kind string
	Text string
}

func ParseFile(path string) (Chapter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Chapter{}, err
	}

	lines := strings.Split(string(content), "\n")
	var title string
	var paragraphs []string
	var blocks []Block
	var callout []string
	calloutLine := 0

	for lineNumber, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if calloutLine != 0 {
			switch trimmed {
			case "```callout":
				return Chapter{}, fmt.Errorf("nested callout on line %d", lineNumber+1)
			case "```":
				text := strings.Join(callout, "\n")
				if strings.TrimSpace(text) == "" {
					return Chapter{}, fmt.Errorf("empty callout on line %d", calloutLine)
				}
				blocks = append(blocks, Block{Kind: "callout", Text: text})
				callout = nil
				calloutLine = 0
			default:
				callout = append(callout, line)
			}
			continue
		}
		if trimmed == "```callout" {
			calloutLine = lineNumber + 1
			continue
		}
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
			blocks = append(blocks, Block{Kind: "heading", Text: title})
			continue
		}
		paragraphs = append(paragraphs, trimmed)
		blocks = append(blocks, Block{Kind: "paragraph", Text: trimmed})
	}

	if calloutLine != 0 {
		return Chapter{}, fmt.Errorf("unclosed callout on line %d", calloutLine)
	}
	if title == "" {
		title = "Untitled Chapter"
	}

	return Chapter{Title: title, Paragraphs: paragraphs, Blocks: blocks}, nil
}
