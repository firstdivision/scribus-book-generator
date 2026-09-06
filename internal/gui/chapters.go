package gui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/book"
)

// chapterRow is a list item that selects on tap and opens the editor on double-tap.
type chapterRow struct {
	widget.BaseWidget
	name, detail *widget.Label
	onTap        func()
	onDoubleTap  func()
}

func newChapterRow() *chapterRow {
	r := &chapterRow{
		name:   widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		detail: widget.NewLabel(""),
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *chapterRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewVBox(r.name, r.detail))
}

func (r *chapterRow) Tapped(*fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap()
	}
}

func (r *chapterRow) DoubleTapped(*fyne.PointEvent) {
	if r.onDoubleTap != nil {
		r.onDoubleTap()
	}
}

// chaptersTab lists chapter directories and hosts the in-place chapter editor.
type chaptersTab struct {
	state  *State
	window fyne.Window
	thumbs *thumbnailCache

	// onChange reports directory changes (new chapter, imported images);
	// onDirty reports edits to book.yaml or chapter text.
	onChange func()
	onDirty  func()

	list     *widget.List
	selected int
	listView fyne.CanvasObject
	stack    *fyne.Container
	editor   *chapterEditor
	content  fyne.CanvasObject
}

func newChaptersTab(state *State, window fyne.Window, thumbs *thumbnailCache, onChange, onDirty func()) *chaptersTab {
	t := &chaptersTab{state: state, window: window, thumbs: thumbs, onChange: onChange, onDirty: onDirty, selected: -1}

	t.list = widget.NewList(
		func() int { return len(state.Chapters) },
		func() fyne.CanvasObject { return newChapterRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(state.Chapters) {
				return
			}
			chapter := state.Chapters[id]
			row := obj.(*chapterRow)
			row.name.SetText(chapter.Name)
			row.detail.SetText(describeChapter(chapter))
			row.onTap = func() { t.list.Select(id) }
			row.onDoubleTap = func() { t.openEditor(id) }
		},
	)
	t.list.OnSelected = func(id widget.ListItemID) { t.selected = id }
	t.list.OnUnselected = func(widget.ListItemID) { t.selected = -1 }

	editButton := widget.NewButton("Edit chapter", func() { t.openEditor(t.selected) })
	newButton := widget.NewButton("New chapter…", t.newChapter)
	importButton := widget.NewButton("Import images…", t.importImages)
	openButton := widget.NewButton("Open text.md", t.openMarkdown)
	openDirButton := widget.NewButton("Open folder", t.openFolder)
	refreshButton := widget.NewButton("Rescan", func() {
		if err := state.ReloadChapters(); err != nil {
			dialog.ShowError(err, window)
		}
		t.refresh()
	})
	buttons := container.NewHBox(editButton, newButton, importButton, openButton, openDirButton, refreshButton)

	hint := widget.NewLabel("Chapters are the directories under chapters/, sorted by name. Double-click a chapter to edit its text and images.")
	hint.Wrapping = fyne.TextWrapWord
	t.listView = container.NewBorder(hint, buttons, nil, nil, t.list)
	t.stack = container.NewStack(t.listView)
	t.content = t.stack
	return t
}

// openEditor replaces the chapter list with an editor for chapter id.
func (t *chaptersTab) openEditor(id int) {
	if id < 0 || id >= len(t.state.Chapters) {
		dialog.ShowInformation("No chapter selected", "Select a chapter first.", t.window)
		return
	}
	editor, err := newChapterEditor(t.state, t.thumbs, t.window, t.state.Chapters[id], t.onDirty, t.closeEditor, t.importImagesInto)
	if err != nil {
		dialog.ShowError(err, t.window)
		return
	}
	t.editor = editor
	t.stack.Objects = []fyne.CanvasObject{editor.content}
	t.stack.Refresh()
}

func (t *chaptersTab) closeEditor() {
	t.editor = nil
	t.stack.Objects = []fyne.CanvasObject{t.listView}
	t.stack.Refresh()
	t.refresh()
}

func describeChapter(c book.Chapter) string {
	title := c.Title
	if strings.TrimSpace(title) == "" {
		if c.Markdown == "" {
			title = "(no markdown file)"
		} else {
			title = "(untitled)"
		}
	}
	return fmt.Sprintf("%s — %d image(s)", title, len(c.Images))
}

func (t *chaptersTab) refresh() {
	t.list.Refresh()
	if t.editor != nil {
		t.editor.refresh()
	}
}

func (t *chaptersTab) newChapter() {
	title := widget.NewEntry()
	title.SetPlaceHolder("Chapter title")
	items := []*widget.FormItem{widget.NewFormItem("Title", title)}
	dialog.ShowForm("New chapter", "Create", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		if _, err := book.CreateChapter(t.state.BookDir, title.Text); err != nil {
			dialog.ShowError(err, t.window)
			return
		}
		if err := t.state.ReloadChapters(); err != nil {
			dialog.ShowError(err, t.window)
		}
		t.onChange()
		t.refresh()
	}, t.window)
}

func (t *chaptersTab) selectedChapter() (book.Chapter, bool) {
	if t.selected < 0 || t.selected >= len(t.state.Chapters) {
		dialog.ShowInformation("No chapter selected", "Select a chapter first.", t.window)
		return book.Chapter{}, false
	}
	return t.state.Chapters[t.selected], true
}

func (t *chaptersTab) importImages() {
	chapter, ok := t.selectedChapter()
	if !ok {
		return
	}
	t.importImagesInto(chapter, t.refresh)
}

// importImagesInto copies a chosen file into chapter's directory, rescans, then calls after.
func (t *chaptersTab) importImagesInto(chapter book.Chapter, after func()) {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, t.window)
			return
		}
		if reader == nil {
			return
		}
		src := reader.URI().Path()
		reader.Close()
		if _, err := book.ImportImages(t.state.BookDir, chapter.Name, []string{src}); err != nil {
			dialog.ShowError(err, t.window)
			return
		}
		if err := t.state.ReloadChapters(); err != nil {
			dialog.ShowError(err, t.window)
		}
		t.onChange()
		after()
	}, t.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".PNG", ".JPG", ".JPEG"}))
	fd.Resize(fyne.NewSize(900, 600))
	fd.Show()
}

func (t *chaptersTab) openMarkdown() {
	chapter, ok := t.selectedChapter()
	if !ok {
		return
	}
	if chapter.Markdown == "" {
		dialog.ShowInformation("No markdown file", "This chapter has no .md file yet.", t.window)
		return
	}
	xdgOpen(filepath.Join(t.state.BookDir, chapter.Markdown), t.window)
}

func (t *chaptersTab) openFolder() {
	chapter, ok := t.selectedChapter()
	if !ok {
		return
	}
	xdgOpen(filepath.Join(t.state.BookDir, chapter.Dir), t.window)
}

func xdgOpen(path string, window fyne.Window) {
	if err := exec.Command("xdg-open", path).Start(); err != nil {
		dialog.ShowError(fmt.Errorf("open %s: %w", path, err), window)
	}
}
