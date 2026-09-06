package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/layout/pagenumbering"
)

func TestLoadForBookFromTemplate(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	bookConfig := []byte("template: a4-landscape.yaml\n")
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), bookConfig, 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}

	template := []byte(`document:
  units: mm
  layout: facing_pages
  first_page: right
page:
  size: A4
  orientation: landscape
  background_color_rgb: [245, 235, 220]
bleed:
  top: 3.18
  bottom: 3.18
  inside: 3.18
  outside: 3.18
safety_margin:
  top: 12.7
  bottom: 12.7
  inside: 12.7
  outside: 12.7
chapter_headings:
  font:
    family: URW Bookman
    style: Demi
    size_pt: 28
  color_rgb: [40, 40, 40]
  background_color_rgb: [240, 230, 220]
  borders:
    bottom: {color_rgb: [1, 2, 3], width_pt: 2}
    left: {}
    top: null
    right: {width_pt: 0}
  alignment: left
  spacing_mm:
    top: 20
    bottom: 10
images:
  sizing:
    max_width_mm: 110
    max_height_mm: 100
  placement:
    snap_to_edge: true
    snap_target: content_area
  leftovers:
    gallery_columns: 4
page_numbers:
  enabled: true
  start_on_page: 1
  start_number: 1
  format: arabic
  position: bottom_outside
  font:
    family: Source Serif 4
    style: Regular
    size_pt: 9
  color_rgb: [80, 80, 80]
  offset_mm:
    top: 7
    bottom: 7
    inside: 10
    outside: 10
  hide_on:
    - chapter_opening
    - full_page_image
    - blank
`)
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	cfg, err := LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

	if cfg.PageWidth != 297 {
		t.Fatalf("expected page width 297mm, got %.2f", cfg.PageWidth)
	}
	if cfg.PageHeight != 210 {
		t.Fatalf("expected page height 210mm, got %.2f", cfg.PageHeight)
	}
	if cfg.PageLayout != "facing_pages" {
		t.Fatalf("expected facing_pages layout, got %q", cfg.PageLayout)
	}
	if cfg.FirstPage != "right" {
		t.Fatalf("expected first_page right, got %q", cfg.FirstPage)
	}
	if cfg.DocumentUnits != "mm" {
		t.Fatalf("expected document units mm, got %q", cfg.DocumentUnits)
	}
	if cfg.PageSize != "A4" {
		t.Fatalf("expected page size A4, got %q", cfg.PageSize)
	}
	if cfg.PageOrientation != "landscape" {
		t.Fatalf("expected page orientation landscape, got %q", cfg.PageOrientation)
	}
	if cfg.PageBackgroundRGB == nil {
		t.Fatalf("expected page background color to be loaded")
	}
	if *cfg.PageBackgroundRGB != [3]int{245, 235, 220} {
		t.Fatalf("expected page background rgb [245 235 220], got %v", *cfg.PageBackgroundRGB)
	}
	if !cfg.PageNumbers.Enabled {
		t.Fatalf("expected page numbers to be enabled")
	}
	if cfg.PageNumbers.StartOnPage != 1 || cfg.PageNumbers.StartNumber != 1 {
		t.Fatalf("expected page numbers to start at page 1/number 1, got page=%d number=%d", cfg.PageNumbers.StartOnPage, cfg.PageNumbers.StartNumber)
	}
	if cfg.PageNumbers.Format != pagenumbering.FormatArabic {
		t.Fatalf("expected arabic page number format, got %q", cfg.PageNumbers.Format)
	}
	if cfg.PageNumbers.Position != pagenumbering.PositionBottomOutside {
		t.Fatalf("expected bottom_outside page number position, got %q", cfg.PageNumbers.Position)
	}
	if cfg.PageNumbers.Font.Family != "Source Serif 4" || cfg.PageNumbers.Font.Style != "Regular" || cfg.PageNumbers.Font.SizePt != 9 {
		t.Fatalf("unexpected page number font settings: %+v", cfg.PageNumbers.Font)
	}
	if cfg.PageNumbers.ColorRGB != [3]int{80, 80, 80} {
		t.Fatalf("expected page number color [80 80 80], got %v", cfg.PageNumbers.ColorRGB)
	}
	if cfg.PageNumbers.OffsetMM.Top != 7 || cfg.PageNumbers.OffsetMM.Bottom != 7 || cfg.PageNumbers.OffsetMM.Inside != 10 || cfg.PageNumbers.OffsetMM.Outside != 10 {
		t.Fatalf("unexpected page number offsets: %+v", cfg.PageNumbers.OffsetMM)
	}
	if cfg.ChapterHeadings.BackgroundColorRGB == nil || *cfg.ChapterHeadings.BackgroundColorRGB != [3]int{240, 230, 220} {
		t.Fatalf("unexpected heading background: %v", cfg.ChapterHeadings.BackgroundColorRGB)
	}
	if b := cfg.ChapterHeadings.Borders["bottom"]; b.ColorRGB != [3]int{1, 2, 3} || b.WidthPt != 2 {
		t.Fatalf("unexpected bottom border: %+v", b)
	}
	if b := cfg.ChapterHeadings.Borders["left"]; b.ColorRGB != cfg.ChapterHeadings.ColorRGB || b.WidthPt != 1 {
		t.Fatalf("unexpected default border: %+v", b)
	}
	if cfg.ChapterHeadings.Borders["top"].WidthPt != 0 || cfg.ChapterHeadings.Borders["right"].WidthPt != 0 {
		t.Fatal("disabled borders must have zero width")
	}
	if cfg.ChapterHeadings.Font.Family != "URW Bookman" || cfg.ChapterHeadings.Font.Style != "Demi" || cfg.ChapterHeadings.Font.SizePt != 28 {
		t.Fatalf("unexpected chapter heading font: %+v", cfg.ChapterHeadings.Font)
	}
	if cfg.ChapterHeadings.ColorRGB != [3]int{40, 40, 40} || cfg.ChapterHeadings.Alignment != "left" {
		t.Fatalf("unexpected chapter heading style: %+v", cfg.ChapterHeadings)
	}
	if cfg.ChapterHeadings.SpacingMM.Top != 20 || cfg.ChapterHeadings.SpacingMM.Bottom != 10 {
		t.Fatalf("unexpected chapter heading spacing: %+v", cfg.ChapterHeadings.SpacingMM)
	}
	if len(cfg.PageNumbers.HideOn) != 3 {
		t.Fatalf("expected 3 page-number hidden roles, got %d", len(cfg.PageNumbers.HideOn))
	}
	if cfg.MarginTop != 12.7 || cfg.MarginBottom != 12.7 || cfg.MarginLeft != 12.7 || cfg.MarginRight != 12.7 {
		t.Fatalf("expected 12.7mm safety margins, got top=%.2f left=%.2f right=%.2f bottom=%.2f", cfg.MarginTop, cfg.MarginLeft, cfg.MarginRight, cfg.MarginBottom)
	}
	if cfg.Images.Sizing.MaxWidthMM != 110 || cfg.Images.Sizing.MaxHeightMM != 100 {
		t.Fatalf("expected image sizing defaults 110x100mm, got %+v", cfg.Images.Sizing)
	}
	if !cfg.Images.Placement.SnapToEdge {
		t.Fatalf("expected images.placement.snap_to_edge to be true")
	}
	if cfg.Images.Placement.SnapTarget != ImageSnapTargetContentArea {
		t.Fatalf("expected snap_target content_area, got %q", cfg.Images.Placement.SnapTarget)
	}
	if cfg.Images.Leftovers.GalleryColumns != 4 {
		t.Fatalf("expected leftover gallery_columns 4 from template, got %d", cfg.Images.Leftovers.GalleryColumns)
	}
}

func TestLoadForBookSupportsNullPageBackground(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	bookConfig := []byte("template: a4-landscape.yaml\n")
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), bookConfig, 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}

	template := []byte("document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\n  background_color_rgb: null\n")
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	cfg, err := LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

	if cfg.PageBackgroundRGB != nil {
		t.Fatalf("expected nil page background rgb, got %v", *cfg.PageBackgroundRGB)
	}
}

func TestLoadForBookDefaultsWhenBookConfigMissing(t *testing.T) {
	cfg, err := LoadForBook(t.TempDir())
	if err != nil {
		t.Fatalf("LoadForBook returned error for temp dir: %v", err)
	}

	defaults := Default()
	if cfg.PageWidth != defaults.PageWidth || cfg.PageHeight != defaults.PageHeight || cfg.PageLayout != defaults.PageLayout || cfg.FirstPage != defaults.FirstPage || cfg.DocumentUnits != defaults.DocumentUnits || cfg.PageSize != defaults.PageSize || cfg.PageOrientation != defaults.PageOrientation {
		t.Fatalf("expected defaults %+v, got %+v", defaults, cfg)
	}
	if cfg.PageNumbers.Enabled != defaults.PageNumbers.Enabled || cfg.PageNumbers.StartOnPage != defaults.PageNumbers.StartOnPage || cfg.PageNumbers.StartNumber != defaults.PageNumbers.StartNumber || cfg.PageNumbers.Format != defaults.PageNumbers.Format || cfg.PageNumbers.Position != defaults.PageNumbers.Position {
		t.Fatalf("expected page number defaults %+v, got %+v", defaults.PageNumbers, cfg.PageNumbers)
	}
}

func TestLoadForBookRejectsInvalidPageNumberConfig(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte("template: a4-landscape.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}

	template := []byte("document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\npage_numbers:\n  enabled: true\n  start_on_page: 0\n")
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	if _, err := LoadForBook(bookDir); err == nil {
		t.Fatalf("expected invalid page number config to return an error")
	}
}

func TestLoadForBookRejectsInvalidImageConfig(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte("template: a4-landscape.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}

	template := []byte("document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\nimages:\n  sizing:\n    max_width_mm: -1\n  placement:\n    allowed_edges: [outside]\n    preferred_edges: [inside]\n")
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	if _, err := LoadForBook(bookDir); err == nil {
		t.Fatalf("expected invalid image config to return an error")
	}
}

func TestLoadForBookParsesLeftoverGalleryColumns(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte("template: a4-landscape.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}
	template := []byte("document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\nimages:\n  leftovers:\n    gallery_columns: 3\n")
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	cfg, err := LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}
	if cfg.Images.Leftovers.GalleryColumns != 3 {
		t.Fatalf("expected gallery_columns 3, got %d", cfg.Images.Leftovers.GalleryColumns)
	}
}

func TestLoadForBookRejectsInvalidLeftoverGalleryColumns(t *testing.T) {
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte("template: a4-landscape.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}
	template := []byte("document:\n  units: mm\npage:\n  size: A4\n  orientation: landscape\nimages:\n  leftovers:\n    gallery_columns: 0\n")
	if err := os.WriteFile(filepath.Join(templateDir, "a4-landscape.yaml"), template, 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}

	_, err := LoadForBook(bookDir)
	if err == nil || !strings.Contains(err.Error(), "gallery_columns") {
		t.Fatalf("expected gallery_columns validation error, got %v", err)
	}
}

func TestLoadForBookRejectsInvalidChapterHeadingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{name: "short background", config: "background_color_rgb: [1, 2]\n", wantErr: "background_color_rgb"},
		{name: "bad background", config: "background_color_rgb: [1, -1, 2]\n", wantErr: "background_color_rgb"},
		{name: "unknown side", config: "borders: {outside: {}}\n", wantErr: "unsupported side"},
		{name: "short border color", config: "borders: {top: {color_rgb: []}}\n", wantErr: "exactly 3"},
		{name: "bad border color", config: "borders: {left: {color_rgb: [256, 0, 0]}}\n", wantErr: "between 0 and 255"},
		{name: "negative border", config: "borders: {bottom: {width_pt: -1}}\n", wantErr: "width_pt"},
		{name: "nan border", config: "borders: {right: {width_pt: .nan}}\n", wantErr: "width_pt"},
		{name: "infinite border", config: "borders: {top: {width_pt: .inf}}\n", wantErr: "width_pt"},
		{name: "empty family", config: "font:\n  family: ''\n", wantErr: "font.family"},
		{name: "empty style", config: "font:\n  style: ' '\n", wantErr: "font.style"},
		{name: "zero size", config: "font:\n  size_pt: 0\n", wantErr: "font.size_pt"},
		{name: "short color", config: "color_rgb: [40, 40]\n", wantErr: "exactly 3"},
		{name: "color out of range", config: "color_rgb: [40, 40, 256]\n", wantErr: "between 0 and 255"},
		{name: "bad alignment", config: "alignment: justify\n", wantErr: "alignment"},
		{name: "negative top spacing", config: "spacing_mm:\n  top: -1\n", wantErr: "spacing_mm.top"},
		{name: "negative bottom spacing", config: "spacing_mm:\n  bottom: -1\n", wantErr: "spacing_mm.bottom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bookDir := t.TempDir()
			templateDir := filepath.Join(bookDir, "templates", "lulu")
			if err := os.MkdirAll(templateDir, 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte("template: test.yaml\n"), 0o644); err != nil {
				t.Fatalf("WriteFile book.yaml returned error: %v", err)
			}
			indentedConfig := "  " + strings.ReplaceAll(test.config, "\n", "\n  ")
			template := "document:\n  units: mm\nchapter_headings:\n" + indentedConfig
			if err := os.WriteFile(filepath.Join(templateDir, "test.yaml"), []byte(template), 0o644); err != nil {
				t.Fatalf("WriteFile template returned error: %v", err)
			}

			_, err := LoadForBook(bookDir)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func writeOverrideFixture(t *testing.T, bookYAML string) string {
	t.Helper()
	bookDir := t.TempDir()
	templateDir := filepath.Join(bookDir, "templates", "lulu")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	template := `document:
  units: mm
  layout: facing_pages
page:
  size: A4
  orientation: landscape
bleed:
  top: 3.18
  bottom: 3.18
  inside: 3.18
  outside: 3.18
safety_margin:
  top: 12.7
  bottom: 12.7
  inside: 12.7
  outside: 12.7
chapter_headings:
  font:
    size_pt: 28
  spacing_mm:
    top: 20
images:
  border:
    width_pt: 11
  sizing:
    max_width_mm: 120
page_numbers:
  enabled: true
  format: arabic
  offset_mm:
    top: 7
`
	if err := os.WriteFile(filepath.Join(templateDir, "base.yaml"), []byte(template), 0o644); err != nil {
		t.Fatalf("WriteFile template returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}
	return bookDir
}

func TestLoadForBookAppliesOverridesOverTemplate(t *testing.T) {
	bookDir := writeOverrideFixture(t, `template: base.yaml
overrides:
  page:
    orientation: portrait
  bleed:
    top: 0
    inside: 5
  safety_margin:
    outside: 0
  chapter_headings:
    font:
      size_pt: 40
  images:
    border:
      width_pt: 0
  page_numbers:
    enabled: false
    offset_mm:
      bottom: 9
`)

	cfg, err := LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

	if cfg.PageOrientation != "portrait" || cfg.PageWidth != 210 || cfg.PageHeight != 297 {
		t.Fatalf("expected portrait A4 from override, got %s %vx%v", cfg.PageOrientation, cfg.PageWidth, cfg.PageHeight)
	}
	if cfg.PageLayout != "facing_pages" {
		t.Fatalf("expected template page layout to be inherited, got %q", cfg.PageLayout)
	}
	if cfg.BleedTop != 0 {
		t.Fatalf("expected explicit zero bleed.top override, got %v", cfg.BleedTop)
	}
	if cfg.BleedInside != 5 || cfg.BleedBottom != 3.18 || cfg.BleedOutside != 3.18 {
		t.Fatalf("unexpected bleed values: %v %v %v", cfg.BleedInside, cfg.BleedBottom, cfg.BleedOutside)
	}
	if cfg.MarginRight != 0 || cfg.MarginLeft != 12.7 {
		t.Fatalf("expected safety_margin.outside=0 and inside inherited, got right=%v left=%v", cfg.MarginRight, cfg.MarginLeft)
	}
	if cfg.ChapterHeadings.Font.SizePt != 40 || cfg.ChapterHeadings.SpacingMM.Top != 20 {
		t.Fatalf("unexpected chapter heading settings: %+v", cfg.ChapterHeadings)
	}
	if cfg.Images.Border.WidthPt != 0 || cfg.Images.Sizing.MaxWidthMM != 120 {
		t.Fatalf("unexpected image defaults: %+v", cfg.Images)
	}
	if cfg.PageNumbers.Enabled || cfg.PageNumbers.OffsetMM.Bottom != 9 || cfg.PageNumbers.OffsetMM.Top != 7 {
		t.Fatalf("unexpected page number settings: %+v", cfg.PageNumbers)
	}
}

func TestLoadForBookOverridesWithoutTemplate(t *testing.T) {
	bookDir := t.TempDir()
	bookYAML := "overrides:\n  page:\n    width_mm: 150\n    height_mm: 150\n  bleed:\n    top: 2\n"
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("WriteFile book.yaml returned error: %v", err)
	}

	cfg, err := LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}
	if cfg.PageWidth != 150 || cfg.PageHeight != 150 || cfg.BleedTop != 2 {
		t.Fatalf("expected overrides applied over defaults, got %+v", cfg)
	}
}

func TestLoadForBookRejectsInvalidOverrides(t *testing.T) {
	tests := []struct {
		name      string
		overrides string
		wantErr   string
	}{
		{name: "bad page number format", overrides: "  page_numbers:\n    format: bogus\n", wantErr: "overrides"},
		{name: "negative bleed", overrides: "  bleed:\n    top: -1\n", wantErr: "bleed"},
		{name: "negative margin", overrides: "  safety_margin:\n    top: -1\n", wantErr: "safety_margin"},
		{name: "bad image sorting", overrides: "  images:\n    sorting: alphabetical\n", wantErr: "images.sorting"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bookDir := writeOverrideFixture(t, "template: base.yaml\noverrides:\n"+test.overrides)
			_, err := LoadForBook(bookDir)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestListTemplatesAndFindProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates", "lulu"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"templates/lulu/b.yaml", "templates/a.yml", "templates/lulu/notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("document:\n  units: mm\n"), 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}
	bookDir := filepath.Join(root, "books", "demo")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	found, err := FindProjectRoot(bookDir)
	if err != nil {
		t.Fatalf("FindProjectRoot returned error: %v", err)
	}
	if found != root {
		t.Fatalf("expected project root %s, got %s", root, found)
	}

	refs, err := ListTemplates(root)
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if len(refs) != 2 || refs[0].Name != "a.yml" || refs[1].Name != "lulu/b.yaml" {
		t.Fatalf("unexpected templates: %+v", refs)
	}
	for _, ref := range refs {
		if _, err := ResolveTemplatePath(bookDir, ref.Name); err != nil {
			t.Fatalf("template name %q from ListTemplates does not resolve: %v", ref.Name, err)
		}
	}

	if refs, err := ListTemplates(t.TempDir()); err != nil || len(refs) != 0 {
		t.Fatalf("expected no templates for empty root, got %v %v", refs, err)
	}
}
