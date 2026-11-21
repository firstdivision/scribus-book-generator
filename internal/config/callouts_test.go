package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCalloutTemplateInheritance(t *testing.T) {
	var base, override TemplateFile
	if err := yaml.Unmarshal([]byte("callouts:\n  font:\n    family: Lora\n    style: Italic\n    size_pt: 13\n  background_color_rgb: [235, 242, 229]\n  padding_mm: 4\n"), &base); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte("callouts:\n  padding_mm: 0\n  border_width_pt: 1\n"), &override); err != nil {
		t.Fatal(err)
	}
	cfg, err := ApplyTemplate(Default(), base)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = ApplyTemplate(cfg, override)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Callouts.Font.Family != "Lora" || cfg.Callouts.Font.Style != "Italic" ||
		cfg.Callouts.Font.SizePt != 13 || cfg.Callouts.PaddingMM != 0 ||
		cfg.Callouts.BorderWidthPt != 1 || cfg.Callouts.BackgroundColorRGB == nil ||
		*cfg.Callouts.BackgroundColorRGB != [3]int{235, 242, 229} {
		t.Fatalf("unexpected inherited callout settings: %+v", cfg.Callouts)
	}
}

func TestCalloutTemplateValidation(t *testing.T) {
	for _, input := range []string{
		"callouts:\n  font: {family: ''}",
		"callouts:\n  font: {size_pt: 0}",
		"callouts:\n  padding_mm: -1",
		"callouts:\n  border_width_pt: .nan",
		"callouts:\n  color_rgb: [0, 256, 1]",
		"callouts:\n  background_color_rgb: [1, 2]",
	} {
		var template TemplateFile
		if err := yaml.Unmarshal([]byte(input), &template); err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyTemplate(Default(), template); err == nil || !strings.Contains(err.Error(), "callouts.") {
			t.Fatalf("input %q: expected callouts validation error, got %v", input, err)
		}
	}
}
