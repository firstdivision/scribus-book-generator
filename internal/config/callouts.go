package config

import (
	"fmt"
	"math"
	"strings"
)

// CalloutSettings describes the appearance of highlighted text blocks.
type CalloutSettings struct {
	Font               CalloutFont
	ColorRGB           [3]int
	BackgroundColorRGB *[3]int
	BorderColorRGB     [3]int
	BorderWidthPt      float64
	PaddingMM          float64
	IndentMM           float64
	SpacingBeforeMM    float64
	SpacingAfterMM     float64
}

type CalloutFont struct {
	Family string
	Style  string
	SizePt float64
}

type CalloutTemplate struct {
	Font               FontTemplate `yaml:"font,omitempty"`
	ColorRGB           []int        `yaml:"color_rgb,omitempty"`
	BackgroundColorRGB []int        `yaml:"background_color_rgb,omitempty"`
	BorderColorRGB     []int        `yaml:"border_color_rgb,omitempty"`
	BorderWidthPt      *float64     `yaml:"border_width_pt,omitempty"`
	PaddingMM          *float64     `yaml:"padding_mm,omitempty"`
	IndentMM           *float64     `yaml:"indent_mm,omitempty"`
	SpacingBeforeMM    *float64     `yaml:"spacing_before_mm,omitempty"`
	SpacingAfterMM     *float64     `yaml:"spacing_after_mm,omitempty"`
}

func defaultCalloutSettings() CalloutSettings {
	return CalloutSettings{
		Font:            CalloutFont{Family: "DejaVu Sans", Style: "Book", SizePt: 12},
		ColorRGB:        [3]int{0, 0, 0},
		BorderColorRGB:  [3]int{0, 0, 0},
		PaddingMM:       3,
		SpacingBeforeMM: 3,
		SpacingAfterMM:  3,
	}
}

func parseCalloutSettings(raw CalloutTemplate, defaults CalloutSettings) (CalloutSettings, error) {
	s := defaults
	if raw.Font.Family != nil {
		s.Font.Family = strings.TrimSpace(*raw.Font.Family)
	}
	if raw.Font.Style != nil {
		s.Font.Style = strings.TrimSpace(*raw.Font.Style)
	}
	setFloat(&s.Font.SizePt, raw.Font.SizePt)
	for _, field := range []struct {
		name string
		src  []int
		dst  *[3]int
	}{
		{"color_rgb", raw.ColorRGB, &s.ColorRGB},
		{"border_color_rgb", raw.BorderColorRGB, &s.BorderColorRGB},
	} {
		if field.src != nil {
			rgb, err := calloutRGB(field.name, field.src)
			if err != nil {
				return s, err
			}
			*field.dst = rgb
		}
	}
	if raw.BackgroundColorRGB != nil {
		rgb, err := calloutRGB("background_color_rgb", raw.BackgroundColorRGB)
		if err != nil {
			return s, err
		}
		s.BackgroundColorRGB = &rgb
	}
	setFloat(&s.BorderWidthPt, raw.BorderWidthPt)
	setFloat(&s.PaddingMM, raw.PaddingMM)
	setFloat(&s.IndentMM, raw.IndentMM)
	setFloat(&s.SpacingBeforeMM, raw.SpacingBeforeMM)
	setFloat(&s.SpacingAfterMM, raw.SpacingAfterMM)
	if s.Font.Family == "" || s.Font.Style == "" || math.IsNaN(s.Font.SizePt) || math.IsInf(s.Font.SizePt, 0) || s.Font.SizePt <= 0 {
		return s, fmt.Errorf("callouts.font requires a nonempty family and style and a finite size_pt > 0")
	}
	for _, field := range []struct {
		name  string
		value float64
	}{
		{"border_width_pt", s.BorderWidthPt}, {"padding_mm", s.PaddingMM},
		{"indent_mm", s.IndentMM}, {"spacing_before_mm", s.SpacingBeforeMM},
		{"spacing_after_mm", s.SpacingAfterMM},
	} {
		if math.IsNaN(field.value) || math.IsInf(field.value, 0) || field.value < 0 {
			return s, fmt.Errorf("callouts.%s must be finite and >= 0", field.name)
		}
	}
	return s, nil
}

func calloutRGB(name string, values []int) ([3]int, error) {
	var rgb [3]int
	if len(values) != len(rgb) {
		return rgb, fmt.Errorf("callouts.%s must contain three integers", name)
	}
	for i, value := range values {
		if value < 0 || value > 255 {
			return rgb, fmt.Errorf("callouts.%s values must be between 0 and 255", name)
		}
		rgb[i] = value
	}
	return rgb, nil
}
