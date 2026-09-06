//go:build gui

// Command bookgen-gui is the desktop editor. Build with `go build -tags gui ./cmd/bookgen-gui`;
// the tag keeps the cgo/X11 dependency out of the default `go build ./...`.
package main

import (
	"flag"
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"

	"scribus-book-generator/internal/gui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bookgen-gui [book-dir]")
		flag.PrintDefaults()
	}
	flag.Parse()

	bookDir := ""
	if flag.NArg() > 0 {
		bookDir = flag.Arg(0)
	}

	a := app.NewWithID("scribus-book-generator.bookgen-gui")
	window := gui.NewMainWindow(a, bookDir)
	window.Window().ShowAndRun()
}
