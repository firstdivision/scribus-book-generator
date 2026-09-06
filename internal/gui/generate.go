package gui

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/images"
	"scribus-book-generator/internal/renderer"
)

// generateTab runs the same pipeline as the bookgen CLI and streams its output.
type generateTab struct {
	state  *State
	window fyne.Window
	// save is called before generating when the book has unsaved changes.
	save func() error

	convertHEIC *widget.Check
	log         *widget.Entry
	run         *widget.Button
	openSLA     *widget.Button
	progress    *widget.ProgressBarInfinite
	content     fyne.CanvasObject

	mu      sync.Mutex
	running bool
	result  renderer.Result
}

func newGenerateTab(state *State, window fyne.Window, save func() error) *generateTab {
	t := &generateTab{state: state, window: window, save: save}

	t.convertHEIC = widget.NewCheck("Convert HEIC images first", nil)
	t.log = widget.NewMultiLineEntry()
	t.log.Wrapping = fyne.TextWrapWord
	t.log.TextStyle = fyne.TextStyle{Monospace: true}
	t.log.Disable()

	t.run = widget.NewButton("Generate Scribus document", t.generate)
	t.openSLA = widget.NewButton("Open .sla in Scribus", t.openInScribus)
	t.openSLA.Disable()
	t.progress = widget.NewProgressBarInfinite()
	t.progress.Hide()

	controls := container.NewVBox(
		t.convertHEIC,
		container.NewHBox(t.run, t.openSLA),
		t.progress,
	)
	t.content = container.NewBorder(controls, nil, nil, nil, t.log)
	return t
}

// logWriter appends command output to the log entry on the UI thread.
type logWriter struct {
	tab *generateTab
}

func (w logWriter) Write(p []byte) (int, error) {
	text := string(p)
	fyne.Do(func() { w.tab.append(text) })
	return len(p), nil
}

func (t *generateTab) append(text string) {
	t.log.SetText(t.log.Text + text)
	t.log.CursorRow = strings.Count(t.log.Text, "\n")
	t.log.Refresh()
}

func (t *generateTab) setRunning(running bool) {
	t.mu.Lock()
	t.running = running
	t.mu.Unlock()
	if running {
		t.run.Disable()
		t.progress.Show()
	} else {
		t.run.Enable()
		t.progress.Hide()
	}
}

func (t *generateTab) generate() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	if t.state.Dirty {
		if err := t.save(); err != nil {
			dialog.ShowError(err, t.window)
			return
		}
	}

	t.log.SetText("")
	t.openSLA.Disable()
	t.setRunning(true)

	convert := t.convertHEIC.Checked
	bookDir := t.state.BookDir
	rootDir := t.state.RootDir
	out := logWriter{tab: t}

	go func() {
		result, err := runPipeline(bookDir, rootDir, convert, out)
		fyne.Do(func() {
			t.setRunning(false)
			if err != nil {
				t.append("\nERROR: " + err.Error() + "\n")
				dialog.ShowError(err, t.window)
				return
			}
			t.result = result
			t.append(fmt.Sprintf("\nWrote %s\nWrote %s\n", result.SLAPath, result.PDFPath))
			t.openSLA.Enable()
		})
	}()
}

func runPipeline(bookDir, rootDir string, convertHEIC bool, out logWriter) (renderer.Result, error) {
	if convertHEIC {
		fmt.Fprintln(out, "Converting HEIC images...")
		if err := images.ConvertHEICWithOptions(bookDir, rootDir, out, out); err != nil {
			return renderer.Result{}, err
		}
	}
	fmt.Fprintln(out, "Loading book...")
	loaded, err := book.Load(bookDir)
	if err != nil {
		return renderer.Result{}, err
	}
	fmt.Fprintf(out, "Loaded %d chapter(s); running Scribus...\n", len(loaded.Chapters))
	return renderer.GenerateFromBookWithOptions(loaded, renderer.Options{RootDir: rootDir, Stdout: out, Stderr: out})
}

func (t *generateTab) openInScribus() {
	if t.result.SLAPath == "" {
		return
	}
	if err := exec.Command("scribus", t.result.SLAPath).Start(); err != nil {
		dialog.ShowError(fmt.Errorf("launch scribus: %w", err), t.window)
	}
}
