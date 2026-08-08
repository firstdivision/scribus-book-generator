package chapterheadings

import (
	"fmt"
	"strings"
)

type Alignment string

const (
	AlignmentLeft    Alignment = "left"
	AlignmentCenter  Alignment = "center"
	AlignmentRight   Alignment = "right"
	AlignmentInside  Alignment = "inside"
	AlignmentOutside Alignment = "outside"
)

type Font struct {
	Family string
	Style  string
	SizePt float64
}

type Spacing struct {
	Top    float64
	Bottom float64
}

type Settings struct {
	Font      Font
	ColorRGB  [3]int
	Alignment Alignment
	SpacingMM Spacing
}

func DefaultSettings() Settings {
	return Settings{
		Font: Font{
			Family: "Source Serif 4",
			Style:  "Semibold",
			SizePt: 28,
		},
		ColorRGB:  [3]int{40, 40, 40},
		Alignment: AlignmentLeft,
		SpacingMM: Spacing{Top: 20, Bottom: 10},
	}
}

func (s Settings) Validate() error {
	if strings.TrimSpace(s.Font.Family) == "" {
		return fmt.Errorf("chapter_headings.font.family must be non-empty")
	}
	if strings.TrimSpace(s.Font.Style) == "" {
		return fmt.Errorf("chapter_headings.font.style must be non-empty")
	}
	if s.Font.SizePt <= 0 {
		return fmt.Errorf("chapter_headings.font.size_pt must be > 0")
	}
	for _, component := range s.ColorRGB {
		if component < 0 || component > 255 {
			return fmt.Errorf("chapter_headings.color_rgb values must be between 0 and 255")
		}
	}
	if !IsValidAlignment(s.Alignment) {
		return fmt.Errorf("chapter_headings.alignment must be one of left, center, right, inside, outside")
	}
	if s.SpacingMM.Top < 0 {
		return fmt.Errorf("chapter_headings.spacing_mm.top must be >= 0")
	}
	if s.SpacingMM.Bottom < 0 {
		return fmt.Errorf("chapter_headings.spacing_mm.bottom must be >= 0")
	}
	return nil
}

func IsValidAlignment(alignment Alignment) bool {
	switch alignment {
	case AlignmentLeft, AlignmentCenter, AlignmentRight, AlignmentInside, AlignmentOutside:
		return true
	default:
		return false
	}
}

func ResolveAlignment(alignment Alignment, isRightPage bool) (Alignment, error) {
	switch alignment {
	case AlignmentLeft, AlignmentCenter, AlignmentRight:
		return alignment, nil
	case AlignmentInside:
		if isRightPage {
			return AlignmentLeft, nil
		}
		return AlignmentRight, nil
	case AlignmentOutside:
		if isRightPage {
			return AlignmentRight, nil
		}
		return AlignmentLeft, nil
	default:
		return "", fmt.Errorf("unsupported chapter heading alignment %q", alignment)
	}
}
