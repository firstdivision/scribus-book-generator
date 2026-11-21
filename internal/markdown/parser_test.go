package markdown

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestParseCallouts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chapter.md")
	content := "```callout\n first line\n\n# inside\n```\n# Title\nBody\n```callout\nLast\n```\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	chapter, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []Block{
		{Kind: "callout", Text: " first line\n\n# inside"},
		{Kind: "heading", Text: "Title"},
		{Kind: "paragraph", Text: "Body"},
		{Kind: "callout", Text: "Last"},
	}
	if !reflect.DeepEqual(chapter.Blocks, want) {
		t.Fatalf("got blocks %#v; want %#v", chapter.Blocks, want)
	}
}

func TestParseInvalidCallouts(t *testing.T) {
	for _, tc := range []struct{ text, message string }{
		{"```callout\n\n```\n", "empty callout on line 1"},
		{"```callout\ntext\n", "unclosed callout on line 1"},
		{"```callout\n```callout\n", "nested callout on line 2"},
		{"\n\n```callout\ntext\n", "unclosed callout on line 3"},
	} {
		path := filepath.Join(t.TempDir(), "chapter.md")
		if err := os.WriteFile(path, []byte(tc.text), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ParseFile(path); err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Fatalf("got error %v, want %q", err, tc.message)
		}
	}
}
