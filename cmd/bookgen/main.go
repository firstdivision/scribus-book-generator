package main

import (
	"encoding/json"
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
	bookConfigPath := filepath.Join(bookDir, "book.yaml")
	fmt.Printf("Loading configuration from %s\n", bookConfigPath)
	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		panic(err)
	}
	fmt.Println("Using resolved configuration:")
	configJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(configJSON))

	fmt.Printf("Finding chapter in %s\n", filepath.Join(bookDir, "chapters"))
	chapterPath := firstChapterPath(bookDir)
	fmt.Printf("Parsing chapter from %s\n", chapterPath)
	chapter, err := markdown.ParseFile(chapterPath)
	if err != nil {
		panic(err)
	}

	outputDir := filepath.Join(bookDir, "out")
	outputPath := filepath.Join(outputDir, "example.sla")
	fmt.Printf("Generating Scribus artifacts in %s\n", outputDir)
	if err := renderer.GenerateSLA(cfg, outputPath, chapter, "images/placeholder.svg"); err != nil {
		panic(err)
	}

	fmt.Printf("Generated Scribus artifacts in %s\n", outputDir)
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
