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
	if err := os.WriteFile(filepath.Join(bookDir, "layout.json"), content, 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(bookDir, "layout.json"), content, 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(bookDir, "layout.json"), content, 0o644); err != nil {
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
