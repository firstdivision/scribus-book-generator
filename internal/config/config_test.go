package config

import (
	"os"
	"path/filepath"
	"testing"
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
	if *cfg.PageBackgroundRGB != [3]int{200, 200, 200} {
		t.Fatalf("expected page background rgb [200 200 200], got %v", *cfg.PageBackgroundRGB)
	}
	if cfg.MarginTop != 12.7 || cfg.MarginBottom != 12.7 || cfg.MarginLeft != 12.7 || cfg.MarginRight != 12.7 {
		t.Fatalf("expected 12.7mm safety margins, got top=%.2f left=%.2f right=%.2f bottom=%.2f", cfg.MarginTop, cfg.MarginLeft, cfg.MarginRight, cfg.MarginBottom)
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
	if cfg != defaults {
		t.Fatalf("expected defaults %+v, got %+v", defaults, cfg)
	}
}
