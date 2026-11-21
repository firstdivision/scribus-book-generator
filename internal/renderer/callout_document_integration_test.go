package renderer

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func TestScribusCalloutDocumentIntegration(t *testing.T) {
	if os.Getenv("BOOKGEN_SCRIBUS_INTEGRATION") != "1" {
		t.Skip("set BOOKGEN_SCRIBUS_INTEGRATION=1 to run Scribus integration tests")
	}
	dir := t.TempDir()
	chapter := filepath.Join(dir, "chapters", "01-test")
	if err := os.MkdirAll(chapter, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "```callout\nPreludeunique\n```\n# Headingunique\nProsebeforeunique\n```callout\n" +
		strings.Repeat("Longcalloutunique line with words.\n", 120) +
		"Finalcalloutunique\n```\n```callout\nAdjacentunique\n```\nProseafterunique\n"
	if err := os.WriteFile(filepath.Join(chapter, "text.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(filepath.Join("..", "..", "books", "sample-book", "chapters", "1-the-road-to-san-rosario", "the-road.png"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.png", "b.png"} {
		if err := os.WriteFile(filepath.Join(chapter, name), image, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	cfg.ChapterHeadings.Font.Family = "DejaVu Sans"
	cfg.ChapterHeadings.Font.Style = "Book"
	cfg.ChapterHeadings.Font.SizePt = 16
	cfg.PageNumbers.Font.Family = "DejaVu Sans"
	cfg.PageNumbers.Font.Style = "Book"
	cfg.Callouts.Font.Style = "Oblique"
	cfg.Callouts.BorderWidthPt = 1
	green := [3]int{220, 240, 220}
	cfg.Callouts.BackgroundColorRGB = &green
	plan := layoutplan.Plan{
		Title: "Integration",
		Images: []layoutplan.ImageInstruction{{
			File: "chapters/01-test/b.png", Placement: layoutplan.PlacementGallery,
		}},
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	loaded := book.Book{Dir: dir, Config: cfg, Plan: plan}
	var output bytes.Buffer
	opts := Options{RootDir: root, Stdout: &output, Stderr: &output}
	_, err = GenerateFromBookWithOptions(loaded, opts)
	pdf := filepath.Join(dir, "out", "Integration.pdf")
	sla := filepath.Join(dir, "out", "Integration.sla")
	for _, path := range []string{pdf, sla} {
		info, statErr := os.Stat(path)
		if err != nil || statErr != nil || info.Size() == 0 {
			t.Fatalf("generation failed: %v, %v\n%s", err, statErr, output.String())
		}
	}
	extracted, err := exec.Command("pdftotext", "-layout", pdf, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("extract PDF: %v\n%s", err, extracted)
	}
	pdfText := string(extracted)
	lastIndex := -1
	for _, word := range []string{"Preludeunique", "Headingunique", "Prosebeforeunique", "Longcalloutunique", "Finalcalloutunique", "Adjacentunique", "Proseafterunique"} {
		index := strings.Index(pdfText, word)
		if index <= lastIndex {
			t.Fatalf("%q missing or out of order:\n%s", word, pdfText)
		}
		lastIndex = index
	}
	if count := strings.Count(pdfText, "Longcalloutunique"); count != 120 {
		t.Fatalf("wanted 120 long-callout lines, got %d", count)
	}
	fonts, err := exec.Command("pdffonts", pdf).CombinedOutput()
	if err != nil || !strings.Contains(string(fonts), "DejaVuSans-Oblique") {
		t.Fatalf("callout font not embedded: %v\n%s", err, fonts)
	}
	document, err := os.ReadFile(sla)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(document), "chapter_1_callout_") || !strings.Contains(string(document), "chapter_1_image_2") {
		t.Fatal("document did not retain callout and gallery image frames")
	}
	loaded.Config.Callouts.Font.Family = "Missing callout font"
	_, err = GenerateFromBookWithOptions(loaded, opts)
	if err == nil || !strings.Contains(err.Error(), "Configured callout font") {
		t.Fatalf("expected a reported missing-font error even with old output present, got %v\n%s", err, output.String())
	}
	loaded.Config.Callouts.Font.Family = "DejaVu Sans"
	loaded.Config.Callouts.PaddingMM = 500
	_, err = GenerateFromBookWithOptions(loaded, opts)
	if err == nil || !strings.Contains(err.Error(), "too narrow") {
		t.Fatalf("expected an impossible callout layout error, got %v\n%s", err, output.String())
	}
}
