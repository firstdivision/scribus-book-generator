package layoutplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromBookDirSupportsFileField(t *testing.T) {
	bookDir := t.TempDir()
	content := []byte("{\"title\":\"The Road to San Rosario\",\"images\":[{\"file\":\"images/ch01/a.jpg\",\"placement\":\"inline\",\"snap_edge\":\"outside\",\"width_mm\":140}]}")
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), append([]byte("template: example.yaml\nlayout: "), content...), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	plan, err := LoadFromBookDir(bookDir)
	if err != nil {
		t.Fatalf("LoadFromBookDir returned error: %v", err)
	}

	if len(plan.Images) != 1 {
		t.Fatalf("expected 1 image override, got %d", len(plan.Images))
	}
	if plan.Title != "The Road to San Rosario" {
		t.Fatalf("expected layout title to be loaded, got %q", plan.Title)
	}
	if plan.Images[0].Source() != "images/ch01/a.jpg" {
		t.Fatalf("expected source from file field, got %q", plan.Images[0].Source())
	}
}

func TestValidateAcceptsIgnorePlacement(t *testing.T) {
	width := 140.0
	plan := Plan{Images: []ImageInstruction{{
		File:      "chapters/1-the-road/outtake.png",
		Placement: PlacementIgnore,
		Bleed:     true,
		WidthMM:   &width,
	}}}
	if err := plan.Validate(); err != nil {
		t.Fatalf("expected ignore placement to be valid even with extra fields, got %v", err)
	}
}

func TestValidateRejectsUnknownPlacement(t *testing.T) {
	plan := Plan{Images: []ImageInstruction{{File: "images/ch01/a.jpg", Placement: "spread"}}}
	err := plan.Validate()
	if err == nil {
		t.Fatalf("expected unknown placement error")
	}
	if !strings.Contains(err.Error(), "inline, full_page, ignore") {
		t.Fatalf("expected placement error to list ignore, got %v", err)
	}
}

func TestLoadFromBookDirSupportsIgnorePlacement(t *testing.T) {
	bookDir := t.TempDir()
	content := []byte("{\"images\":[{\"file\":\"chapters/1-the-road/outtake.png\",\"placement\":\"ignore\"}]}")
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), append([]byte("template: example.yaml\nlayout: "), content...), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	plan, err := LoadFromBookDir(bookDir)
	if err != nil {
		t.Fatalf("LoadFromBookDir returned error: %v", err)
	}
	if len(plan.Images) != 1 || plan.Images[0].Placement != PlacementIgnore {
		t.Fatalf("expected ignore placement, got %+v", plan.Images)
	}
}

func TestValidateRejectsInvalidEdge(t *testing.T) {
	plan := Plan{Images: []ImageInstruction{{File: "images/ch01/a.jpg", SnapEdge: "left"}}}
	if err := plan.Validate(); err == nil {
		t.Fatalf("expected invalid edge error")
	}
}

func TestValidateRejectsNegativeDimensions(t *testing.T) {
	width := -1.0
	plan := Plan{Images: []ImageInstruction{{File: "images/ch01/a.jpg", WidthMM: &width}}}
	if err := plan.Validate(); err == nil {
		t.Fatalf("expected negative width error")
	}
}

func TestInlineSettingsPrecedenceShape(t *testing.T) {
	width := 140.0
	instruction := ImageInstruction{File: "images/ch01/a.jpg", WidthMM: &width, SnapEdge: "outside"}
	override, err := instruction.InlineSettings()
	if err != nil {
		t.Fatalf("InlineSettings returned error: %v", err)
	}
	if override.WidthMM == nil || *override.WidthMM != 140 {
		t.Fatalf("expected width override 140, got %+v", override.WidthMM)
	}
	if override.SnapEdge == nil || *override.SnapEdge != "outside" {
		t.Fatalf("expected snap edge outside, got %+v", override.SnapEdge)
	}
}

func TestLoadFromBookDirSupportsBorderWidthZero(t *testing.T) {
	bookDir := t.TempDir()
	content := []byte("{\"images\":[{\"file\":\"images/ch01/a.jpg\",\"placement\":\"full_page\",\"border\":{\"width_pt\":0}}]}")
	if err := os.WriteFile(filepath.Join(bookDir, "book.yaml"), append([]byte("template: example.yaml\nlayout: "), content...), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	plan, err := LoadFromBookDir(bookDir)
	if err != nil {
		t.Fatalf("LoadFromBookDir returned error: %v", err)
	}

	if len(plan.Images) != 1 || plan.Images[0].Border == nil || plan.Images[0].Border.WidthPt == nil {
		t.Fatalf("expected border width override to be present, got %+v", plan.Images)
	}
	if *plan.Images[0].Border.WidthPt != 0 {
		t.Fatalf("expected border width override to be 0, got %v", *plan.Images[0].Border.WidthPt)
	}

	jsonText := plan.JSON()
	if !strings.Contains(jsonText, "\"width_pt\":0") {
		t.Fatalf("expected plan JSON to preserve zero border width, got %s", jsonText)
	}
}

func TestValidateRejectsNegativeBorderWidth(t *testing.T) {
	width := -0.5
	plan := Plan{Images: []ImageInstruction{{File: "images/ch01/a.jpg", Border: &Border{WidthPt: &width}}}}
	if err := plan.Validate(); err == nil {
		t.Fatalf("expected negative border width error")
	}
}

func TestLoadFromBookDirYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "no layout", content: "template: test.yaml\n"},
		{name: "null layout", content: "layout: null\n"},
		{name: "malformed yaml", content: "layout: [", wantErr: "yaml:"},
		{name: "invalid layout type", content: "layout: text", wantErr: "cannot unmarshal"},
		{name: "missing image source", content: "layout:\n  images:\n    - placement: inline", wantErr: "src or .file"},
		{name: "invalid width", content: "layout:\n  images:\n    - src: photo.jpg\n      width_mm: 0", wantErr: "width_mm"},
		{name: "invalid color", content: "layout:\n  images:\n    - src: photo.jpg\n      border:\n        color_rgb: [0, 256, 0]", wantErr: "color_rgb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			plan, err := LoadFromBookDir(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected %q, got %v", tt.wantErr, err)
				}
			} else if err != nil || len(plan.Images) != 0 || plan.Title != "" {
				t.Fatalf("expected empty plan, got %+v, %v", plan, err)
			}
		})
	}
}

func TestLoadFromBookDirAllYAMLFields(t *testing.T) {
	dir := t.TempDir()
	content := `template: example.yaml
layout:
  title: My Book
  images:
    - src: fallback.jpg
      file: photo.jpg
      placement: full_page
      bleed: true
      snap_edge: outside
      width_mm: 140
      height_mm: 90
      border:
        color_rgb: [10, 20, 30]
        width_pt: 0
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// An obsolete file must not override the book's YAML layout.
	if err := os.WriteFile(filepath.Join(dir, "layout.json"), []byte(`{"title":"obsolete"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := LoadFromBookDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"title":"My Book","images":[{"src":"fallback.jpg","file":"photo.jpg","placement":"full_page","bleed":true,"snap_edge":"outside","width_mm":140,"height_mm":90,"border":{"color_rgb":[10,20,30],"width_pt":0}}]}`
	if got := plan.JSON(); got != want {
		t.Fatalf("renderer contract mismatch:\ngot  %s\nwant %s", got, want)
	}
}

func TestLoadFromBookDirMissingBookConfig(t *testing.T) {
	plan, err := LoadFromBookDir(t.TempDir())
	if err != nil || plan.Title != "" || len(plan.Images) != 0 {
		t.Fatalf("expected default plan, got %+v, %v", plan, err)
	}
}

func TestLoadGalleryPlacement(t *testing.T) {
	dir := t.TempDir()
	content := "layout:\n  images:\n    - file: photo.jpg\n      placement: gallery\n      bleed: true\n"
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := LoadFromBookDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Images) != 1 || plan.Images[0].Placement != PlacementGallery || !strings.Contains(plan.JSON(), `"placement":"gallery"`) {
		t.Fatalf("unexpected gallery plan: %+v", plan)
	}
}
