package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileReadsTitleAndParagraphs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chapter.md")
	content := "# Example Chapter\n\nThis is the first paragraph.\n\nThis is the second paragraph.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	chapter, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if chapter.Title != "Example Chapter" {
		t.Fatalf("expected title Example Chapter, got %q", chapter.Title)
	}
	if len(chapter.Paragraphs) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(chapter.Paragraphs))
	}
	if chapter.Paragraphs[0] != "This is the first paragraph." {
		t.Fatalf("unexpected first paragraph: %q", chapter.Paragraphs[0])
	}
}
