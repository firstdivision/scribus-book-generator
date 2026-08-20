package book

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSampleBook(t *testing.T) {
	loaded, err := Load(filepath.Join("..", "..", "books", "sample-book"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(loaded.Chapters))
	}
	if loaded.Chapters[0].Name != "1-the-road-to-san-rosario" {
		t.Fatalf("unexpected first chapter: %q", loaded.Chapters[0].Name)
	}
	if loaded.Chapters[0].Title != "Chapter 1 — The Road to San Rosario" {
		t.Fatalf("unexpected first title: %q", loaded.Chapters[0].Title)
	}
	if loaded.Chapters[0].Markdown != filepath.Join("chapters", "1-the-road-to-san-rosario", "01-intro.md") {
		t.Fatalf("unexpected markdown path: %q", loaded.Chapters[0].Markdown)
	}
	if len(loaded.Chapters[0].Images) != 2 || len(loaded.Chapters[1].Images) != 2 {
		t.Fatalf("unexpected image counts: %#v %#v", loaded.Chapters[0].Images, loaded.Chapters[1].Images)
	}
	if loaded.Plan.Title != "The Roast to San Rosario" {
		t.Fatalf("unexpected layout title: %q", loaded.Plan.Title)
	}
}

func TestLoadRequiresBookDirectory(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("expected error for empty book directory")
	}
}

func TestLoadRequiresChaptersDirectory(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "chapters directory not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresChapterDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "chapters"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "no chapter directories found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresMarkdownInEachChapter(t *testing.T) {
	dir := writeMiniBook(t)
	empty := filepath.Join(dir, "chapters", "2-empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(empty, "photo.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "chapter 2-empty: no markdown file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsMissingLayoutImage(t *testing.T) {
	dir := writeMiniBook(t)
	layout := `{"images":[{"file":"chapters/1-intro/missing.png"}]}`
	if err := os.WriteFile(filepath.Join(dir, "layout.json"), []byte(layout), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "layout.images[0]: file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidMarkdown(t *testing.T) {
	dir := writeMiniBook(t)
	path := filepath.Join(dir, "chapters", "1-intro", "text.md")
	if err := os.WriteFile(path, []byte("# One\n\nBody.\n\n# Two\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "chapters/1-intro/text.md") || !strings.Contains(err.Error(), "additional H1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadDiscoversImagesAndSkipsDotfiles(t *testing.T) {
	dir := writeMiniBook(t)
	chapterDir := filepath.Join(dir, "chapters", "1-intro")
	if err := os.WriteFile(filepath.Join(chapterDir, ".hidden.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write hidden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "photo.JPG"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write jpg: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Chapters[0].Images) != 2 {
		t.Fatalf("expected png then jpg, got %#v", loaded.Chapters[0].Images)
	}
	if !strings.HasSuffix(loaded.Chapters[0].Images[0], "cover.png") {
		t.Fatalf("expected cover.png first, got %#v", loaded.Chapters[0].Images)
	}
	if !strings.HasSuffix(loaded.Chapters[0].Images[1], "photo.JPG") {
		t.Fatalf("expected photo.JPG second, got %#v", loaded.Chapters[0].Images)
	}
}

func writeMiniBook(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	chapterDir := filepath.Join(dir, "chapters", "1-intro")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "text.md"), []byte("# Intro\n\nHello.\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "cover.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	return dir
}
