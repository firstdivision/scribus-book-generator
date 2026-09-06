package book

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func writeProject(t *testing.T) (root, bookDir string) {
	t.Helper()
	root = t.TempDir()
	templateDir := filepath.Join(root, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	template := "document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\nbleed:\n  top: 3\n"
	if err := os.WriteFile(filepath.Join(templateDir, "base.yaml"), []byte(template), 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}
	bookDir = filepath.Join(root, "books", "demo")
	if err := os.MkdirAll(filepath.Join(bookDir, "chapters"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	return root, bookDir
}

func floatPtr(v float64) *float64 { return &v }

func TestFileRoundTrip(t *testing.T) {
	_, bookDir := writeProject(t)

	original := File{
		Template:  "base.yaml",
		Overrides: &config.TemplateFile{},
		Layout: layoutplan.Plan{
			Title: "Demo",
			Images: []layoutplan.ImageInstruction{
				{File: "chapters/1-a/one.png", Placement: layoutplan.PlacementInline, SnapEdge: "top", WidthMM: floatPtr(120)},
				{File: "chapters/1-a/two.png", Placement: layoutplan.PlacementFullPage, Bleed: true, Border: &layoutplan.Border{WidthPt: floatPtr(0)}},
			},
		},
	}
	original.Overrides.Bleed.Top = floatPtr(0)
	enabled := false
	original.Overrides.PageNumbers.Enabled = &enabled

	if err := os.MkdirAll(filepath.Join(bookDir, "chapters", "1-a"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := WriteFile(bookDir, original); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(bookDir, "book.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	for _, want := range []string{"template: base.yaml", "overrides:", "bleed:", "top: 0", "enabled: false", "layout:", "title: Demo", "snap_edge: top"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected written yaml to contain %q, got:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"safety_margin", "chapter_headings", "sorting", "src:", "height_mm"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("expected written yaml not to contain %q, got:\n%s", unwanted, text)
		}
	}

	reread, err := ReadFile(bookDir)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !reflect.DeepEqual(reread, original) {
		t.Fatalf("round trip mismatch:\n got %#v\nwant %#v", reread, original)
	}

	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}
	if cfg.BleedTop != 0 || cfg.PageNumbers.Enabled {
		t.Fatalf("expected overrides to apply, got bleed.top=%v page_numbers.enabled=%v", cfg.BleedTop, cfg.PageNumbers.Enabled)
	}
}

func TestMarshalDropsEmptyOverrides(t *testing.T) {
	file := File{Template: "base.yaml", Overrides: &config.TemplateFile{}}
	data, err := file.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if strings.Contains(string(data), "overrides") {
		t.Fatalf("expected empty overrides to be omitted, got:\n%s", data)
	}
	if !strings.Contains(string(data), "images: []") {
		t.Fatalf("expected empty image list to be written explicitly, got:\n%s", data)
	}
}

func TestWriteFileRejectsInvalidContent(t *testing.T) {
	_, bookDir := writeProject(t)

	bad := File{Template: "base.yaml", Layout: layoutplan.Plan{Images: []layoutplan.ImageInstruction{{File: "x.png", Placement: "sideways"}}}}
	if err := WriteFile(bookDir, bad); err == nil || !strings.Contains(err.Error(), "placement") {
		t.Fatalf("expected placement validation error, got %v", err)
	}

	bad = File{Template: "base.yaml", Overrides: &config.TemplateFile{}}
	bad.Overrides.Bleed.Top = floatPtr(-1)
	if err := WriteFile(bookDir, bad); err == nil || !strings.Contains(err.Error(), "bleed") {
		t.Fatalf("expected bleed validation error, got %v", err)
	}

	bad = File{Template: "missing.yaml"}
	if err := WriteFile(bookDir, bad); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected template resolution error, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(bookDir, "book.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no book.yaml to be written after failures, stat err=%v", err)
	}
}

func TestCreateAndCreateChapter(t *testing.T) {
	root, _ := writeProject(t)
	bookDir := filepath.Join(root, "books", "new-book")

	if err := Create(bookDir, "base.yaml", "My Book"); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := Create(bookDir, "base.yaml", "Again"); err == nil {
		t.Fatal("expected Create to refuse an existing book")
	}

	file, err := ReadFile(bookDir)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if file.Template != "base.yaml" || file.Layout.Title != "My Book" {
		t.Fatalf("unexpected new book file: %+v", file)
	}

	chapters, err := ListChapters(bookDir, config.ImageSortingNone)
	if err != nil || len(chapters) != 0 {
		t.Fatalf("expected empty chapter list, got %v %v", chapters, err)
	}

	first, err := CreateChapter(bookDir, "The Road: Part One!")
	if err != nil {
		t.Fatalf("CreateChapter returned error: %v", err)
	}
	if first.Name != "1-the-road-part-one" {
		t.Fatalf("unexpected chapter name %q", first.Name)
	}
	second, err := CreateChapter(bookDir, "Second")
	if err != nil {
		t.Fatalf("CreateChapter returned error: %v", err)
	}
	if second.Name != "2-second" {
		t.Fatalf("unexpected chapter name %q", second.Name)
	}
	if _, err := CreateChapter(bookDir, "!!!"); err == nil {
		t.Fatal("expected CreateChapter to reject a title with no slug characters")
	}

	chapters, err = ListChapters(bookDir, config.ImageSortingNone)
	if err != nil {
		t.Fatalf("ListChapters returned error: %v", err)
	}
	if len(chapters) != 2 || chapters[0].Title != "The Road: Part One!" || chapters[1].Markdown != filepath.Join("chapters", "2-second", "text.md") {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}

	loaded, err := Load(bookDir)
	if err != nil {
		t.Fatalf("Load returned error for created book: %v", err)
	}
	if len(loaded.Chapters) != 2 {
		t.Fatalf("expected Load to see 2 chapters, got %d", len(loaded.Chapters))
	}
}

func TestImportImages(t *testing.T) {
	_, bookDir := writeProject(t)
	chapter, err := CreateChapter(bookDir, "Photos")
	if err != nil {
		t.Fatalf("CreateChapter returned error: %v", err)
	}

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "photo.jpg")
	if err := os.WriteFile(src, []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	notImage := filepath.Join(srcDir, "notes.txt")
	if err := os.WriteFile(notImage, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	written, err := ImportImages(bookDir, chapter.Name, []string{src})
	if err != nil {
		t.Fatalf("ImportImages returned error: %v", err)
	}
	if len(written) != 1 || written[0] != filepath.Join("chapters", chapter.Name, "photo.jpg") {
		t.Fatalf("unexpected written paths: %v", written)
	}
	if data, err := os.ReadFile(filepath.Join(bookDir, written[0])); err != nil || string(data) != "jpeg-bytes" {
		t.Fatalf("copied file mismatch: %q %v", data, err)
	}

	if _, err := ImportImages(bookDir, chapter.Name, []string{src}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected no-clobber error, got %v", err)
	}
	if _, err := ImportImages(bookDir, chapter.Name, []string{notImage}); err == nil || !strings.Contains(err.Error(), "not a supported image") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
	if _, err := ImportImages(bookDir, "9-missing", []string{src}); err == nil {
		t.Fatal("expected missing chapter error")
	}

	chapters, err := ListChapters(bookDir, config.ImageSortingNone)
	if err != nil {
		t.Fatalf("ListChapters returned error: %v", err)
	}
	if len(chapters) != 1 || len(chapters[0].Images) != 1 {
		t.Fatalf("expected imported image to be listed, got %+v", chapters)
	}
}

func TestListChaptersToleratesMissingMarkdown(t *testing.T) {
	_, bookDir := writeProject(t)
	if err := os.MkdirAll(filepath.Join(bookDir, "chapters", "1-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	chapters, err := ListChapters(bookDir, config.ImageSortingNone)
	if err != nil {
		t.Fatalf("ListChapters returned error: %v", err)
	}
	if len(chapters) != 1 || chapters[0].Title != "" || chapters[0].Markdown != "" {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
}

func TestWriteChapterMarkdown(t *testing.T) {
	_, bookDir := writeProject(t)
	rel := filepath.Join("chapters", "2-new", "text.md")
	if err := WriteChapterMarkdown(bookDir, rel, "# New\n\nBody.\n"); err != nil {
		t.Fatalf("WriteChapterMarkdown returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(bookDir, rel))
	if err != nil || string(data) != "# New\n\nBody.\n" {
		t.Fatalf("unexpected file contents %q, err %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Join(bookDir, "chapters", "2-new"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only text.md to remain (no temp files), got %v err %v", entries, err)
	}
	if err := WriteChapterMarkdown(bookDir, "../escape.md", "x"); err == nil {
		t.Fatal("expected paths outside the book to be rejected")
	}
	if err := WriteChapterMarkdown(bookDir, "/abs.md", "x"); err == nil {
		t.Fatal("expected absolute paths to be rejected")
	}
}
