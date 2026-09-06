package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func writeProject(t *testing.T) (root, bookDir string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates", "lulu"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	templates := map[string]string{
		"templates/lulu/base.yaml":  "document:\n  units: mm\n  layout: facing_pages\npage:\n  size: A4\n  orientation: landscape\nbleed:\n  top: 3.18\nimages:\n  border:\n    width_pt: 11\n",
		"templates/lulu/other.yaml": "document:\n  units: mm\npage:\n  size: LETTER\n  orientation: portrait\nbleed:\n  top: 1\n",
	}
	for name, content := range templates {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	bookDir = filepath.Join(root, "books", "demo")
	chapterDir := filepath.Join(bookDir, "chapters", "1-first")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "text.md"), []byte("# First\n\nHello.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "a.png"), []byte("not really a png"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	bookYAML := "template: base.yaml\nlayout:\n  title: Demo\n  images:\n  - file: chapters/1-first/a.png\n    placement: inline\n"
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return root, bookDir
}

func TestOpenDiscoversRootAndTemplates(t *testing.T) {
	root, bookDir := writeProject(t)

	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if state.RootDir != root {
		t.Fatalf("expected root %s, got %s", root, state.RootDir)
	}
	if state.File.Overrides == nil {
		t.Fatal("expected Overrides to be initialised for editing")
	}
	if len(state.Chapters) != 1 || len(state.ChapterImages()) != 1 {
		t.Fatalf("unexpected chapters: %+v", state.Chapters)
	}
	if state.TemplateConfig.BleedTop != 3.18 || state.TemplateConfig.PageLayout != "facing_pages" {
		t.Fatalf("unexpected template config: %+v", state.TemplateConfig)
	}

	options, selected := state.TemplateOptions()
	if selected != "lulu/base.yaml" {
		t.Fatalf("expected the book's template to be matched by path, got %q (options %v)", selected, options)
	}
	if len(options) != 2 {
		t.Fatalf("expected 2 template options, got %v", options)
	}
}

func TestSetTemplateAndSave(t *testing.T) {
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	if err := state.SetTemplate("lulu/other.yaml"); err != nil {
		t.Fatalf("SetTemplate returned error: %v", err)
	}
	if !state.Dirty || state.TemplateConfig.BleedTop != 1 || state.TemplateConfig.PageSize != "LETTER" {
		t.Fatalf("expected template config to follow the new template, got %+v", state.TemplateConfig)
	}
	if err := state.SetTemplate("missing.yaml"); err == nil {
		t.Fatal("expected SetTemplate to fail for an unknown template")
	}
	if state.File.Template != "lulu/other.yaml" {
		t.Fatalf("expected failed SetTemplate to leave template unchanged, got %q", state.File.Template)
	}

	zero := 0.0
	state.File.Overrides.Bleed.Top = &zero
	effective, err := state.Effective()
	if err != nil {
		t.Fatalf("Effective returned error: %v", err)
	}
	if effective.BleedTop != 0 {
		t.Fatalf("expected zero override to apply, got %v", effective.BleedTop)
	}

	if err := state.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if state.Dirty {
		t.Fatal("expected Save to clear Dirty")
	}
	reread, err := book.ReadFile(bookDir)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if reread.Template != "lulu/other.yaml" || reread.Overrides == nil || reread.Overrides.Bleed.Top == nil || *reread.Overrides.Bleed.Top != 0 {
		t.Fatalf("unexpected saved file: %+v", reread)
	}
	if reread.Layout.Title != "Demo" || len(reread.Layout.Images) != 1 {
		t.Fatalf("expected layout to be preserved, got %+v", reread.Layout)
	}
}

func TestFloatRowTogglesPointer(t *testing.T) {
	test.NewTempApp(t)

	var ptr *float64
	changes := 0
	row := floatRow("Top", &ptr, func() float64 { return 3.18 }, func() { changes++ })
	row.load()
	if ptr != nil || row.check.Checked {
		t.Fatal("expected unset pointer to load as not overridden")
	}
	if !strings.Contains(row.inherited.Text, "3.18") {
		t.Fatalf("expected inherited label to show the template value, got %q", row.inherited.Text)
	}

	row.check.SetChecked(true)
	if ptr == nil || *ptr != 3.18 {
		t.Fatalf("expected enabling override to seed the inherited value, got %v", ptr)
	}

	entry := row.input.(interface{ SetText(string) })
	entry.SetText("0")
	if ptr == nil || *ptr != 0 {
		t.Fatalf("expected explicit zero to be stored, got %v", ptr)
	}
	entry.SetText("abc")
	if ptr == nil || *ptr != 0 {
		t.Fatalf("expected invalid text to leave the last valid value, got %v", ptr)
	}

	row.check.SetChecked(false)
	if ptr != nil {
		t.Fatalf("expected disabling override to clear pointer, got %v", *ptr)
	}
	if changes == 0 {
		t.Fatal("expected change callbacks")
	}

	v := 7.5
	ptr = &v
	row.load()
	if !row.check.Checked {
		t.Fatal("expected load to reflect an existing override")
	}
}

func TestRGBAndListRows(t *testing.T) {
	test.NewTempApp(t)

	var rgb []int
	row := rgbRow("Colour", &rgb, func() [3]int { return [3]int{1, 2, 3} }, func() {})
	row.load()
	row.check.SetChecked(true)
	if len(rgb) != 3 || rgb[2] != 3 {
		t.Fatalf("expected inherited colour to seed override, got %v", rgb)
	}
	row.input.(interface{ SetText(string) }).SetText("[10, 20, 30]")
	if len(rgb) != 3 || rgb[0] != 10 || rgb[1] != 20 || rgb[2] != 30 {
		t.Fatalf("expected bracketed colour to parse, got %v", rgb)
	}
	if _, err := parseRGB("1, 2"); err == nil {
		t.Fatal("expected two components to be rejected")
	}
	if _, err := parseRGB("1, 2, 300"); err == nil {
		t.Fatal("expected out-of-range component to be rejected")
	}

	var edges []string
	list := listRow("Edges", edgeOptions, &edges, func() []string { return []string{"outside", "top"} }, func() {})
	list.load()
	list.check.SetChecked(true)
	if len(edges) != 2 {
		t.Fatalf("expected inherited edges to seed override, got %v", edges)
	}
	list.check.SetChecked(false)
	if edges != nil {
		t.Fatalf("expected nil after disabling, got %v", edges)
	}
}

func TestBuildOverrideTabsRoundTrip(t *testing.T) {
	test.NewTempApp(t)
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	width := 0.0
	state.File.Overrides.Images.Border.WidthPt = &width
	state.File.Overrides.ChapterHeadings.Borders = map[string]*config.ChapterHeadingBorderTemplate{
		"bottom": {WidthPt: &width},
	}

	tabs := buildOverrideTabs(state, func() {})
	if len(tabs.items) != 5 {
		t.Fatalf("expected 5 override tabs, got %d", len(tabs.items))
	}
	tabs.refresh()

	if state.File.Overrides.Images.Border.WidthPt == nil || *state.File.Overrides.Images.Border.WidthPt != 0 {
		t.Fatalf("expected refresh to preserve the zero border override, got %v", state.File.Overrides.Images.Border.WidthPt)
	}
	if _, ok := state.File.Overrides.ChapterHeadings.Borders["bottom"]; !ok {
		t.Fatalf("expected refresh to preserve border map, got %v", state.File.Overrides.ChapterHeadings.Borders)
	}
	if state.File.Overrides.Bleed.Top != nil {
		t.Fatal("expected refresh not to invent overrides")
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("expected state to validate after refresh: %v", err)
	}
}

func TestLayoutImagesTabAddRemove(t *testing.T) {
	test.NewTempApp(t)
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "chapters", "1-first", "b.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := state.ReloadChapters(); err != nil {
		t.Fatalf("ReloadChapters returned error: %v", err)
	}

	changes := 0
	tab := newLayoutImagesTab(state, newThumbnailCache(), func() { changes++ })
	test.NewTempWindow(t, tab.content)

	tab.list.Select(0)
	if tab.selected != 0 || changes != 0 {
		t.Fatalf("selecting an image must populate the editor without marking dirty (selected=%d changes=%d)", tab.selected, changes)
	}

	tab.addImage()
	if len(state.File.Layout.Images) != 2 {
		t.Fatalf("expected 2 images after add, got %d", len(state.File.Layout.Images))
	}
	added := state.File.Layout.Images[1]
	if added.File != filepath.Join("chapters", "1-first", "b.png") || added.Placement != layoutplan.PlacementInline {
		t.Fatalf("expected add to pick the first unused chapter image, got %+v", added)
	}
	if tab.selected != 1 {
		t.Fatalf("expected new image to be selected, got %d", tab.selected)
	}

	tab.moveSelected(-1)
	if state.File.Layout.Images[0].File != filepath.Join("chapters", "1-first", "b.png") || tab.selected != 0 {
		t.Fatalf("expected move up to reorder and follow selection, got %+v selected=%d", state.File.Layout.Images, tab.selected)
	}

	tab.removeSelected()
	if len(state.File.Layout.Images) != 1 || state.File.Layout.Images[0].File != "chapters/1-first/a.png" {
		t.Fatalf("unexpected images after remove: %+v", state.File.Layout.Images)
	}
	if changes == 0 {
		t.Fatal("expected change callbacks")
	}

	tab.titleBox.SetText("New Title")
	if state.File.Layout.Title != "New Title" {
		t.Fatalf("expected title edit to update model, got %q", state.File.Layout.Title)
	}
}

func TestChapterRowTapAndDoubleTap(t *testing.T) {
	test.NewTempApp(t)
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	tab := newChaptersTab(state, test.NewTempWindow(t, widget.NewLabel("")), newThumbnailCache(), func() {}, func() {})
	test.NewTempWindow(t, tab.content)

	row := newChapterRow()
	tab.list.UpdateItem(0, row)
	if row.name.Text != "1-first" || !strings.Contains(row.detail.Text, "First") {
		t.Fatalf("unexpected row texts %q / %q", row.name.Text, row.detail.Text)
	}

	test.Tap(row)
	if tab.selected != 0 {
		t.Fatalf("expected tap to select the chapter, got %d", tab.selected)
	}
	if tab.stack.Objects[0] != tab.listView {
		t.Fatal("expected a single tap to keep the list visible")
	}

	test.DoubleTap(row)
	if tab.editor == nil || tab.stack.Objects[0] != tab.editor.content {
		t.Fatal("expected double tap to open the chapter editor in place")
	}
	if tab.editor.chapterName != "1-first" || tab.editor.text.Text != "# First\n\nHello.\n" {
		t.Fatalf("unexpected editor state: %q %q", tab.editor.chapterName, tab.editor.text.Text)
	}

	tab.closeEditor()
	if tab.editor != nil || tab.stack.Objects[0] != tab.listView {
		t.Fatal("expected Back to restore the chapter list")
	}
}

func TestChapterEditorTextSavesToDisk(t *testing.T) {
	test.NewTempApp(t)
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	dirty := 0
	editor, err := newChapterEditor(state, newThumbnailCache(), test.NewTempWindow(t, widget.NewLabel("")), state.Chapters[0], func() { dirty++ }, func() {}, nil)
	if err != nil {
		t.Fatalf("newChapterEditor returned error: %v", err)
	}
	test.NewTempWindow(t, editor.content)
	if dirty != 0 || state.Dirty {
		t.Fatal("building the editor must not mark the book dirty")
	}

	editor.text.SetText("# Renamed\n\nNew body.\n")
	if dirty == 0 || !state.Dirty || state.PendingText["1-first"] != "# Renamed\n\nNew body.\n" {
		t.Fatalf("expected edit to be pending and dirty, got dirty=%d pending=%q", dirty, state.PendingText)
	}

	if err := state.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(bookDir, "chapters", "1-first", "text.md"))
	if err != nil || string(data) != "# Renamed\n\nNew body.\n" {
		t.Fatalf("expected text.md to be written, got %q err %v", data, err)
	}
	if len(state.PendingText) != 0 || state.Dirty {
		t.Fatalf("expected Save to clear pending text and dirty flag, got %v %v", state.PendingText, state.Dirty)
	}
	if state.Chapters[0].Title != "Renamed" {
		t.Fatalf("expected Save to reload chapter titles, got %q", state.Chapters[0].Title)
	}

	// A chapter with no markdown yet gets text.md created on save.
	if err := os.MkdirAll(filepath.Join(bookDir, "chapters", "2-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := state.ReloadChapters(); err != nil {
		t.Fatalf("ReloadChapters returned error: %v", err)
	}
	empty, err := newChapterEditor(state, newThumbnailCache(), test.NewTempWindow(t, widget.NewLabel("")), state.Chapters[1], func() {}, func() {}, nil)
	if err != nil {
		t.Fatalf("newChapterEditor returned error: %v", err)
	}
	if empty.text.Text != "" {
		t.Fatalf("expected empty text for a chapter without markdown, got %q", empty.text.Text)
	}
	empty.text.SetText("# Empty\n")
	if err := state.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if state.Chapters[1].Markdown != filepath.Join("chapters", "2-empty", "text.md") || state.Chapters[1].Title != "Empty" {
		t.Fatalf("expected text.md to be created for the empty chapter, got %+v", state.Chapters[1])
	}
}

func TestChapterEditorUpsertsAndResetsInstruction(t *testing.T) {
	test.NewTempApp(t)
	_, bookDir := writeProject(t)
	state, err := Open("", bookDir)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "chapters", "1-first", "b.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := state.ReloadChapters(); err != nil {
		t.Fatalf("ReloadChapters returned error: %v", err)
	}

	dirty := 0
	editor, err := newChapterEditor(state, newThumbnailCache(), test.NewTempWindow(t, widget.NewLabel("")), state.Chapters[0], func() { dirty++ }, func() {}, nil)
	if err != nil {
		t.Fatalf("newChapterEditor returned error: %v", err)
	}
	test.NewTempWindow(t, editor.content)
	if len(editor.images()) != 2 {
		t.Fatalf("expected both chapter images to be listed, got %v", editor.images())
	}

	bSource := filepath.Join("chapters", "1-first", "b.png")
	if editor.describe(bSource) != "template defaults" {
		t.Fatalf("expected unlisted image to show template defaults, got %q", editor.describe(bSource))
	}

	// Select b.png (not in book.yaml) and change its placement: an entry is appended.
	editor.grid.Select(1)
	if dirty != 0 {
		t.Fatal("selecting an image must not mark dirty")
	}
	form := editor.form.Objects[3].(*widget.Form)
	form.Items[0].Widget.(*widget.Select).SetSelected(string(layoutplan.PlacementFullPage))
	if len(state.File.Layout.Images) != 2 || state.File.Layout.Images[1].File != bSource || state.File.Layout.Images[1].Placement != layoutplan.PlacementFullPage {
		t.Fatalf("expected b.png to be added with full_page placement, got %+v", state.File.Layout.Images)
	}
	if dirty == 0 || !state.Dirty {
		t.Fatal("expected change to mark dirty")
	}
	form.Items[3].Widget.(*widget.Entry).SetText("80")
	if len(state.File.Layout.Images) != 2 || state.File.Layout.Images[1].WidthMM == nil || *state.File.Layout.Images[1].WidthMM != 80 {
		t.Fatalf("expected second edit to update the same entry, got %+v", state.File.Layout.Images)
	}
	if !strings.Contains(editor.describe(bSource), "full_page") {
		t.Fatalf("expected grid description to reflect the entry, got %q", editor.describe(bSource))
	}

	// Editing a.png (already listed) must not duplicate it.
	editor.grid.Select(0)
	form = editor.form.Objects[3].(*widget.Form)
	form.Items[1].Widget.(*widget.Check).SetChecked(true)
	if len(state.File.Layout.Images) != 2 || !state.File.Layout.Images[0].Bleed {
		t.Fatalf("expected a.png entry to be updated in place, got %+v", state.File.Layout.Images)
	}

	state.RemoveImageInstruction(bSource)
	if len(state.File.Layout.Images) != 1 || state.File.Layout.Images[0].File != "chapters/1-first/a.png" {
		t.Fatalf("expected reset to drop b.png only, got %+v", state.File.Layout.Images)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("expected state to validate: %v", err)
	}
}

func TestMainWindowOpensBookAndSaves(t *testing.T) {
	a := test.NewTempApp(t)
	_, bookDir := writeProject(t)

	m := NewMainWindow(a, bookDir)
	if m.state == nil || m.state.BookDir != bookDir {
		t.Fatalf("expected initial book to be opened, got %+v", m.state)
	}
	if m.chapters == nil || m.chapters.stack.Objects[0] != m.chapters.listView {
		t.Fatal("expected the chapters tab to start on the chapter list")
	}
	if m.rootDir == "" {
		t.Fatal("expected project root to be discovered from the book directory")
	}
	if m.state.Dirty {
		t.Fatal("expected a freshly opened book to be clean (widget loading must not mark dirty)")
	}
	if !strings.HasPrefix(m.window.Title(), "Demo") {
		t.Fatalf("unexpected window title %q", m.window.Title())
	}

	m.layout.titleBox.SetText("Renamed")
	if !m.state.Dirty || !strings.Contains(m.window.Title(), "*") {
		t.Fatalf("expected edit to mark dirty and update title, got dirty=%v title=%q", m.state.Dirty, m.window.Title())
	}

	m.save()
	if m.state.Dirty {
		t.Fatal("expected save to clear dirty flag")
	}
	reread, err := book.ReadFile(bookDir)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if reread.Layout.Title != "Renamed" {
		t.Fatalf("expected saved title, got %q", reread.Layout.Title)
	}
	if reread.Overrides != nil {
		t.Fatalf("expected no overrides to be written when none were set, got %+v", reread.Overrides)
	}
	if a.Preferences().String(prefLastBook) != bookDir {
		t.Fatal("expected last book to be remembered in preferences")
	}
}

func TestMainWindowWithoutBook(t *testing.T) {
	a := test.NewTempApp(t)
	a.Preferences().SetString(prefRootDir, t.TempDir())
	m := NewMainWindow(a, "")
	if m.state != nil {
		t.Fatal("expected no book to be open without an initial path or preference")
	}
	if err := m.saveReturningError(); err == nil {
		t.Fatal("expected save without a book to fail")
	}
}

func TestDescribeInstruction(t *testing.T) {
	w := 120.0
	b := 0.0
	got := describeInstruction(layoutplan.ImageInstruction{Placement: layoutplan.PlacementFullPage, Bleed: true, WidthMM: &w, Border: &layoutplan.Border{WidthPt: &b}})
	for _, want := range []string{"full_page", "bleed", "120mm wide", "border 0pt"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
	if describeInstruction(layoutplan.ImageInstruction{}) != "(no placement)" {
		t.Fatalf("unexpected description for empty instruction: %q", describeInstruction(layoutplan.ImageInstruction{}))
	}
}

func TestScaleToFit(t *testing.T) {
	if img := loadThumbnail(filepath.Join(t.TempDir(), "missing.png"), 96); img != nil {
		t.Fatal("expected nil thumbnail for missing file")
	}
	cache := newThumbnailCache()
	path := filepath.Join(t.TempDir(), "bad.png")
	if err := os.WriteFile(path, []byte("garbage"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if cache.Get(path) != nil {
		t.Fatal("expected nil thumbnail for undecodable file")
	}
	if _, ok := cache.items[path]; !ok {
		t.Fatal("expected failed decode to be cached")
	}
}
