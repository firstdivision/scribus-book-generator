package markdown

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseFileWithoutH1KeepsBodyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chapter.md")
	if err := os.WriteFile(path, []byte("First paragraph.\n\nSecond paragraph.\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	chapter, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if chapter.Title != "Untitled Chapter" {
		t.Fatalf("expected fallback title, got %q", chapter.Title)
	}
	if len(chapter.Paragraphs) != 2 || chapter.Paragraphs[0] != "First paragraph." {
		t.Fatalf("unexpected body paragraphs: %#v", chapter.Paragraphs)
	}
}

func TestParseFileRejectsAdditionalH1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chapter.md")
	content := "# Chapter Title\n\nBody paragraph.\n\n# Another H1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected additional H1 to return an error")
	}
	if !strings.Contains(err.Error(), "additional H1 heading on line 5") {
		t.Fatalf("unexpected error: %v", err)
	}
}
