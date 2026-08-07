package layoutplan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromBookDirSupportsFileField(t *testing.T) {
	bookDir := t.TempDir()
	content := []byte("{\"images\":[{\"file\":\"images/ch01/a.jpg\",\"placement\":\"inline\",\"snap_edge\":\"outside\",\"width_mm\":140}]}")
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
	if plan.Images[0].Source() != "images/ch01/a.jpg" {
		t.Fatalf("expected source from file field, got %q", plan.Images[0].Source())
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
