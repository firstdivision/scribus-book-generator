package gui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/layout/layoutplan"
)

const gridThumbSize = 160

// chapterEditor edits one chapter's markdown and the layout instructions for
// every image found in its directory.
type chapterEditor struct {
	state       *State
	thumbs      *thumbnailCache
	window      fyne.Window
	chapterName string

	onDirty  func()
	onBack   func()
	onImport func(chapter book.Chapter, after func())

	title    *widget.Label
	text     *widget.Entry
	grid     *widget.GridWrap
	selected int
	form     *fyne.Container
	content  fyne.CanvasObject
}

func newChapterEditor(state *State, thumbs *thumbnailCache, window fyne.Window, chapter book.Chapter, onDirty, onBack func(), onImport func(book.Chapter, func())) (*chapterEditor, error) {
	e := &chapterEditor{
		state:       state,
		thumbs:      thumbs,
		window:      window,
		chapterName: chapter.Name,
		onDirty:     onDirty,
		onBack:      onBack,
		onImport:    onImport,
		selected:    -1,
	}

	initial, err := state.ChapterText(chapter)
	if err != nil {
		return nil, err
	}
	e.text = widget.NewMultiLineEntry()
	e.text.Wrapping = fyne.TextWrapWord
	e.text.TextStyle = fyne.TextStyle{Monospace: true}
	e.text.SetPlaceHolder("# Chapter title\n\nChapter text in Markdown…")
	e.text.SetText(initial)
	e.text.OnChanged = func(s string) {
		if c, ok := e.chapter(); ok {
			state.SetChapterText(c, s)
			onDirty()
		}
	}

	e.grid = widget.NewGridWrap(
		func() int { return len(e.images()) },
		func() fyne.CanvasObject {
			img := canvas.NewImageFromImage(nil)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(gridThumbSize, gridThumbSize*0.75))
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis
			name.Alignment = fyne.TextAlignCenter
			status := widget.NewLabel("")
			status.Truncation = fyne.TextTruncateEllipsis
			status.Alignment = fyne.TextAlignCenter
			status.TextStyle = fyne.TextStyle{Italic: true}
			return container.NewBorder(nil, container.NewVBox(name, status), nil, nil, img)
		},
		func(id widget.GridWrapItemID, obj fyne.CanvasObject) {
			images := e.images()
			if id < 0 || id >= len(images) {
				return
			}
			source := images[id]
			cell := obj.(*fyne.Container)
			img := cell.Objects[0].(*canvas.Image)
			labels := cell.Objects[1].(*fyne.Container)
			labels.Objects[0].(*widget.Label).SetText(filepath.Base(source))
			labels.Objects[1].(*widget.Label).SetText(e.describe(source))
			thumbs.fill(img, filepath.Join(state.BookDir, source), func() { e.grid.RefreshItem(id) })
		},
	)
	e.grid.OnSelected = func(id widget.GridWrapItemID) {
		e.selected = id
		e.rebuildForm()
	}
	e.grid.OnUnselected = func(widget.GridWrapItemID) {
		e.selected = -1
		e.rebuildForm()
	}

	e.form = container.NewVBox()
	e.rebuildForm()

	e.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	e.title.Truncation = fyne.TextTruncateEllipsis
	e.updateTitle()

	back := widget.NewButton("← Back to chapters", onBack)
	importBtn := widget.NewButton("Import images…", func() {
		if c, ok := e.chapter(); ok {
			onImport(c, e.refresh)
		}
	})
	openDir := widget.NewButton("Open folder", func() {
		if c, ok := e.chapter(); ok {
			xdgOpen(filepath.Join(state.BookDir, c.Dir), window)
		}
	})
	toolbar := container.NewBorder(nil, nil, back, container.NewHBox(importBtn, openDir), e.title)

	images := container.NewVSplit(e.grid, container.NewVScroll(e.form))
	images.SetOffset(0.55)
	split := container.NewHSplit(e.text, images)
	split.SetOffset(0.5)

	e.content = container.NewBorder(toolbar, nil, nil, nil, split)
	return e, nil
}

func (e *chapterEditor) chapter() (book.Chapter, bool) {
	return e.state.ChapterByName(e.chapterName)
}

func (e *chapterEditor) images() []string {
	c, ok := e.chapter()
	if !ok {
		return nil
	}
	return c.Images
}

func (e *chapterEditor) describe(source string) string {
	if i := e.state.ImageInstructionIndex(source); i >= 0 {
		return describeInstruction(e.state.File.Layout.Images[i])
	}
	return "template defaults"
}

func (e *chapterEditor) updateTitle() {
	c, ok := e.chapter()
	if !ok {
		e.title.SetText(e.chapterName)
		return
	}
	title := strings.TrimSpace(c.Title)
	if title == "" {
		e.title.SetText(c.Name)
		return
	}
	e.title.SetText(c.Name + " — " + title)
}

// refresh re-reads chapter metadata after the directory changed.
func (e *chapterEditor) refresh() {
	e.updateTitle()
	e.grid.Refresh()
	e.rebuildForm()
}

func (e *chapterEditor) rebuildForm() {
	e.form.Objects = nil
	images := e.images()
	if e.selected < 0 || e.selected >= len(images) {
		e.form.Add(widget.NewLabel("Select an image to override its placement, size, or border. Unlisted images use the template defaults."))
		e.form.Refresh()
		return
	}
	source := images[e.selected]
	id := e.selected

	// Edit a draft; the first change adds it to layout.images.
	draft := layoutplan.ImageInstruction{File: source}
	listed := e.state.ImageInstructionIndex(source) >= 0
	if listed {
		draft = e.state.File.Layout.Images[e.state.ImageInstructionIndex(source)]
	}

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	reset := widget.NewButton("Reset to defaults", func() {
		if e.state.ImageInstructionIndex(source) < 0 {
			return
		}
		dialog.ShowConfirm("Reset image", "Remove this image's entry from book.yaml so it uses the template defaults?", func(ok bool) {
			if !ok {
				return
			}
			e.state.RemoveImageInstruction(source)
			e.onDirty()
			e.grid.RefreshItem(id)
			e.rebuildForm()
		}, e.window)
	})
	showListed := func(listed bool) {
		if listed {
			status.SetText("Listed in book.yaml.")
			reset.Enable()
		} else {
			status.SetText("Not in book.yaml — uses template defaults. Changing a field adds it.")
			reset.Disable()
		}
	}
	showListed(listed)

	changed := func() {
		e.state.UpsertImageInstruction(draft)
		e.onDirty()
		e.grid.RefreshItem(id)
		showListed(true)
	}

	preview := canvas.NewImageFromImage(e.thumbs.Get(filepath.Join(e.state.BookDir, source)))
	preview.FillMode = canvas.ImageFillContain
	preview.SetMinSize(fyne.NewSize(240, 180))

	e.form.Add(preview)
	e.form.Add(widget.NewLabelWithStyle(source, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	e.form.Add(status)
	e.form.Add(widget.NewForm(instructionFormItems(&draft, changed)...))
	e.form.Add(reset)
	e.form.Refresh()
}
