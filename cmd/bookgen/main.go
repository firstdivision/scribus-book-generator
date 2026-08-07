package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/markdown"
	"scribus-book-generator/internal/renderer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <book-dir>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	bookDir := os.Args[1]
	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		panic(err)
	}
	chapterPath := firstChapterPath(bookDir)
	chapter, err := markdown.ParseFile(chapterPath)
	if err != nil {
		panic(err)
	}

	outputPath := filepath.Join(bookDir, "out", "example.sla")
	if err := renderer.GenerateSLA(cfg, outputPath, chapter, "images/placeholder.svg"); err != nil {
		panic(err)
	}

	fmt.Printf("book generation scaffold initialized for %s\n", bookDir)
}

func firstChapterPath(bookDir string) string {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return filepath.Join(chaptersDir, "example.md")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		chapterEntries, readErr := os.ReadDir(filepath.Join(chaptersDir, entry.Name()))
		if readErr != nil {
			continue
		}

		for _, chapterEntry := range chapterEntries {
			if chapterEntry.IsDir() || !strings.HasSuffix(chapterEntry.Name(), ".md") {
				continue
			}
			return filepath.Join(chaptersDir, entry.Name(), chapterEntry.Name())
		}
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		return filepath.Join(chaptersDir, entry.Name())
	}

	return filepath.Join(chaptersDir, "example.md")
}
