package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/renderer"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("bookgen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	verbose := fs.Bool("v", false, "print resolved configuration")
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
	fmt.Printf("Loading configuration from %s\n", filepath.Join(bookDir, "book.yaml"))
	if *verbose {
		cfg, err := config.LoadForBook(bookDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bookgen: load configuration: %v\n", err)
			return 1
		}
		fmt.Println("Using resolved configuration:")
		configJSON, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "bookgen: encode configuration: %v\n", err)
			return 1
		}
		fmt.Println(string(configJSON))
	}

	fmt.Printf("Generating Scribus artifacts from %s\n", bookDir)
	result, err := renderer.Generate(bookDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bookgen: %v\n", err)
		return 1
	}

	fmt.Printf("wrote Scribus document: %s\n", result.SLAPath)
	fmt.Printf("wrote PDF: %s\n", result.PDFPath)
	return 0
}
