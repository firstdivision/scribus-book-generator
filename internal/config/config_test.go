package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/layout/pagenumbering"
)

func TestLoadForBookFromTemplate(t *testing.T) {
	bookDir := filepath.Clean(filepath.Join("..", "..", "books", "sample-book"))

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
	if *cfg.PageBackgroundRGB != [3]int{248, 244, 232} {
		t.Fatalf("expected page background rgb [248 244 232], got %v", *cfg.PageBackgroundRGB)
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
	if cfg.ChapterHeadings.Font.Family != "Source Serif 4" || cfg.ChapterHeadings.Font.Style != "Semibold" || cfg.ChapterHeadings.Font.SizePt != 28 {
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

func TestLoadForBookRejectsInvalidChapterHeadingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
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
