package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/config"
)

const (
	prefRootDir  = "rootDir"
	prefLastBook = "lastBook"
)

// MainWindow is the single application window.
type MainWindow struct {
	app    fyne.App
	window fyne.Window
	thumbs *thumbnailCache

	rootDir string
	state   *State

	// suppress > 0 while widgets are being (re)loaded from the model, so
	// their change callbacks do not mark the book dirty.
	suppress int

	rootLabel *widget.Label
	saveBtn   *widget.Button
	body      *fyne.Container

	overrides *overrideTabs
	layout    *layoutImagesTab
	chapters  *chaptersTab
}

// NewMainWindow builds the window. initialBook, when non-empty, is opened immediately.
func NewMainWindow(a fyne.App, initialBook string) *MainWindow {
	m := &MainWindow{app: a, thumbs: newThumbnailCache()}
	m.window = a.NewWindow("Book Generator")
	m.window.Resize(fyne.NewSize(1100, 750))

	m.rootDir = a.Preferences().String(prefRootDir)
	if m.rootDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			if root, err := config.FindProjectRoot(cwd); err == nil {
				m.rootDir = root
			}
		}
	}

	m.rootLabel = widget.NewLabel("")
	m.saveBtn = widget.NewButton("Save", m.save)
	m.saveBtn.Disable()
	toolbar := container.NewHBox(
		widget.NewButton("Open book…", m.openBookDialog),
		widget.NewButton("New book…", m.newBookDialog),
		m.saveBtn,
		widget.NewSeparator(),
		widget.NewButton("Project root…", m.chooseRootDialog),
		m.rootLabel,
	)
	m.updateRootLabel()

	m.body = container.NewStack(m.welcome())
	m.window.SetContent(container.NewBorder(toolbar, nil, nil, nil, m.body))
	m.window.SetCloseIntercept(m.confirmClose)

	if initialBook == "" {
		initialBook = a.Preferences().String(prefLastBook)
	}
	if initialBook != "" {
		if err := m.openBook(initialBook); err != nil {
			m.body.Objects = []fyne.CanvasObject{m.welcome()}
			m.body.Refresh()
		}
	}
	return m
}

// Window exposes the underlying fyne.Window (for Show/ShowAndRun).
func (m *MainWindow) Window() fyne.Window { return m.window }

func (m *MainWindow) updateRootLabel() {
	if m.rootDir == "" {
		m.rootLabel.SetText("Project root: (not set — templates and scripts cannot be found)")
		return
	}
	m.rootLabel.SetText("Project root: " + m.rootDir)
}

func (m *MainWindow) welcome() fyne.CanvasObject {
	text := widget.NewLabel("Open an existing book directory (containing book.yaml and chapters/) or create a new one.")
	text.Wrapping = fyne.TextWrapWord
	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle("Scribus Book Generator", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		text,
		container.NewHBox(
			widget.NewButton("Open book…", m.openBookDialog),
			widget.NewButton("New book…", m.newBookDialog),
		),
	))
}

func (m *MainWindow) markDirty() {
	if m.suppress > 0 || m.state == nil {
		return
	}
	m.state.Dirty = true
	m.saveBtn.Enable()
	m.window.SetTitle(m.state.Title())
}

func (m *MainWindow) withSuppressed(fn func()) {
	m.suppress++
	defer func() { m.suppress-- }()
	fn()
}

func (m *MainWindow) openBook(bookDir string) error {
	state, err := Open(m.rootDir, bookDir)
	if err != nil {
		dialog.ShowError(fmt.Errorf("open %s: %w", bookDir, err), m.window)
		return err
	}
	m.state = state
	if m.rootDir == "" {
		m.rootDir = state.RootDir
		m.app.Preferences().SetString(prefRootDir, m.rootDir)
		m.updateRootLabel()
	}
	m.app.Preferences().SetString(prefLastBook, state.BookDir)

	m.withSuppressed(func() {
		m.body.Objects = []fyne.CanvasObject{m.bookView()}
		m.body.Refresh()
	})
	m.saveBtn.Disable()
	m.window.SetTitle(state.Title())
	return nil
}

func (m *MainWindow) bookView() fyne.CanvasObject {
	state := m.state

	options, selected := state.TemplateOptions()
	templateSelect := widget.NewSelect(options, nil)
	templateSelect.PlaceHolder = "(none — built-in defaults)"
	templateSelect.SetSelected(selected)

	m.overrides = buildOverrideTabs(state, m.markDirty)
	m.layout = newLayoutImagesTab(state, m.thumbs, m.markDirty)
	m.chapters = newChaptersTab(state, m.window, m.thumbs,
		func() { m.layout.refresh() },
		func() {
			m.markDirty()
			m.layout.refresh()
		})
	generate := newGenerateTab(state, m.window, m.saveReturningError)

	templateSelect.OnChanged = func(name string) {
		if m.suppress > 0 {
			return
		}
		if err := state.SetTemplate(name); err != nil {
			dialog.ShowError(err, m.window)
			m.withSuppressed(func() { templateSelect.SetSelected(state.File.Template) })
			return
		}
		m.withSuppressed(m.overrides.refresh)
		m.markDirty()
	}
	m.withSuppressed(m.overrides.refresh)

	header := widget.NewForm(
		widget.NewFormItem("Book directory", widget.NewLabel(state.BookDir)),
		widget.NewFormItem("Template", templateSelect),
	)

	items := []*container.TabItem{
		container.NewTabItem("Chapters", m.chapters.content),
		container.NewTabItem("Layout Images", m.layout.content),
	}
	items = append(items, m.overrides.items...)
	items = append(items, container.NewTabItem("Generate", generate.content))
	tabs := container.NewAppTabs(items...)
	tabs.SetTabLocation(container.TabLocationTop)

	return container.NewBorder(header, nil, nil, nil, tabs)
}

func (m *MainWindow) saveReturningError() error {
	if m.state == nil {
		return fmt.Errorf("no book is open")
	}
	if err := m.state.Save(); err != nil {
		return err
	}
	m.saveBtn.Disable()
	m.window.SetTitle(m.state.Title())
	if m.chapters != nil {
		m.chapters.refresh()
	}
	return nil
}

func (m *MainWindow) save() {
	if err := m.saveReturningError(); err != nil {
		dialog.ShowError(err, m.window)
	}
}

func (m *MainWindow) confirmClose() {
	if m.state == nil || !m.state.Dirty {
		m.window.Close()
		return
	}
	dialog.ShowConfirm("Unsaved changes", "Discard unsaved changes to book.yaml and chapter text?", func(discard bool) {
		if discard {
			m.state.Dirty = false
			m.window.Close()
		}
	}, m.window)
}

func (m *MainWindow) openBookDialog() {
	fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, m.window)
			return
		}
		if uri == nil {
			return
		}
		if m.state != nil && m.state.Dirty {
			dialog.ShowConfirm("Unsaved changes", "Discard unsaved changes and open another book?", func(ok bool) {
				if ok {
					m.openBook(uri.Path())
				}
			}, m.window)
			return
		}
		m.openBook(uri.Path())
	}, m.window)
	m.setDialogLocation(fd, m.defaultBooksDir())
	fd.Resize(fyne.NewSize(900, 600))
	fd.Show()
}

func (m *MainWindow) defaultBooksDir() string {
	if m.state != nil {
		return filepath.Dir(m.state.BookDir)
	}
	if m.rootDir != "" {
		if info, err := os.Stat(filepath.Join(m.rootDir, "books")); err == nil && info.IsDir() {
			return filepath.Join(m.rootDir, "books")
		}
		return m.rootDir
	}
	return ""
}

func (m *MainWindow) setDialogLocation(fd *dialog.FileDialog, dir string) {
	if dir == "" {
		return
	}
	if lister, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
		fd.SetLocation(lister)
	}
}

func (m *MainWindow) chooseRootDialog() {
	fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, m.window)
			return
		}
		if uri == nil {
			return
		}
		root := uri.Path()
		if info, err := os.Stat(filepath.Join(root, "templates")); err != nil || !info.IsDir() {
			dialog.ShowError(fmt.Errorf("%s has no templates/ directory", root), m.window)
			return
		}
		m.rootDir = root
		m.app.Preferences().SetString(prefRootDir, root)
		m.updateRootLabel()
		if m.state != nil {
			m.openBook(m.state.BookDir)
		}
	}, m.window)
	m.setDialogLocation(fd, m.rootDir)
	fd.Resize(fyne.NewSize(900, 600))
	fd.Show()
}

func (m *MainWindow) newBookDialog() {
	if m.rootDir == "" {
		dialog.ShowInformation("Project root required", "Choose the project root (the directory containing templates/) first.", m.window)
		return
	}
	templates, err := config.ListTemplates(m.rootDir)
	if err != nil {
		dialog.ShowError(err, m.window)
		return
	}
	names := make([]string, 0, len(templates))
	for _, ref := range templates {
		names = append(names, ref.Name)
	}

	parent := widget.NewEntry()
	parent.SetText(m.defaultBooksDir())
	browse := widget.NewButton("Browse…", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				parent.SetText(uri.Path())
			}
		}, m.window)
		m.setDialogLocation(fd, parent.Text)
		fd.Show()
	})
	name := widget.NewEntry()
	name.SetPlaceHolder("directory name, e.g. summer-2026")
	title := widget.NewEntry()
	title.SetPlaceHolder("Book title")
	template := widget.NewSelect(names, nil)
	if len(names) > 0 {
		template.SetSelected(names[0])
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Parent directory", container.NewBorder(nil, nil, nil, browse, parent)),
		widget.NewFormItem("Directory name", name),
		widget.NewFormItem("Title", title),
		widget.NewFormItem("Template", template),
	}
	form := dialog.NewForm("New book", "Create", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		dirName := strings.TrimSpace(name.Text)
		if dirName == "" {
			dirName = book.Slugify(title.Text)
		}
		if dirName == "" {
			dialog.ShowError(fmt.Errorf("a directory name or title is required"), m.window)
			return
		}
		bookDir := filepath.Join(strings.TrimSpace(parent.Text), dirName)
		if err := book.Create(bookDir, template.Selected, title.Text); err != nil {
			dialog.ShowError(err, m.window)
			return
		}
		m.openBook(bookDir)
	}, m.window)
	form.Resize(fyne.NewSize(700, 350))
	form.Show()
}
