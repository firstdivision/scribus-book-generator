package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/renderer"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("bookgen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	verbose := fs.Bool("v", false, "print resolved configuration and chapter inventory")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-v] <book-dir>\n", filepath.Base(os.Args[0]))
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	bookDir := fs.Arg(0)
	fmt.Printf("Loading book from %s\n", bookDir)
	loaded, err := book.Load(bookDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bookgen: %v\n", err)
		return 1
	}
	if *verbose {
		fmt.Println("Using resolved configuration:")
		configJSON, err := json.MarshalIndent(loaded.Config, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "bookgen: encode configuration: %v\n", err)
			return 1
		}
		fmt.Println(string(configJSON))
		fmt.Println("Chapters:")
		for _, chapter := range loaded.Chapters {
			fmt.Printf("  %s (%q, %d images)\n", chapter.Name, chapter.Title, len(chapter.Images))
		}
	}

	fmt.Printf("Generating Scribus artifacts from %d chapters\n", len(loaded.Chapters))
	result, err := renderer.GenerateFromBook(loaded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bookgen: %v\n", err)
		return 1
	}

	fmt.Printf("wrote Scribus document: %s\n", result.SLAPath)
	fmt.Printf("wrote PDF: %s\n", result.PDFPath)
	return 0
}
