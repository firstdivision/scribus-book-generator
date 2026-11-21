package renderer

import (
	"encoding/json"
	"strings"
	"testing"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func TestCalloutJobEncodesResolvedSettings(t *testing.T) {
	cfg := config.Default()
	cfg.Callouts.Font.Family = "Lora"
	cfg.Callouts.Font.Style = "Italic"
	cfg.Callouts.PaddingMM = 25.4
	data, err := json.Marshal(buildScribusJob(cfg, layoutplan.Plan{}))
	if err != nil {
		t.Fatal(err)
	}
	var job struct {
		Callouts struct {
			FontName      string  `json:"font_name"`
			PaddingPoints float64 `json:"padding_points"`
		} `json:"callouts"`
	}
	if err := json.Unmarshal(data, &job); err != nil {
		t.Fatal(err)
	}
	if job.Callouts.FontName != "Lora Italic" || job.Callouts.PaddingPoints != 72 {
		t.Fatalf("unexpected callout job: %s", data)
	}
	if !strings.Contains(string(data), `"background_color_rgb":null`) {
		t.Fatalf("expected nullable callout background: %s", data)
	}
}
