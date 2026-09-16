package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/pdfexport"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("printpdf", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dpi := fs.Int("dpi", 300, "raster resolution in DPI (300–1200; use 600 for finer text)")
	output := fs.String("o", "", "new output PDF path (default: <input>-print.pdf; never overwritten)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: printpdf [-dpi 300] [-o output.pdf] <input.pdf>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 || *dpi < 300 || *dpi > 1200 {
		fs.Usage()
		return 2
	}
	input := fs.Arg(0)
	if !strings.EqualFold(filepath.Ext(input), ".pdf") {
		fmt.Fprintln(os.Stderr, "printpdf: input must be a .pdf file")
		return 2
	}
	if *output == "" {
		*output = strings.TrimSuffix(input, filepath.Ext(input)) + "-print.pdf"
	}
	if err := pdfexport.Flatten(input, *output, *dpi, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "printpdf: %v\n", err)
		return 1
	}
	fmt.Printf("wrote flattened print PDF: %s\n", *output)
	return 0
}
