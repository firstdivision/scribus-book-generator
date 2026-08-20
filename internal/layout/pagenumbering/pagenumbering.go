package pagenumbering

import (
	"fmt"
	"strings"
)

type NumberFormat string

const (
	FormatArabic     NumberFormat = "arabic"
	FormatRomanLower NumberFormat = "roman_lower"
	FormatRomanUpper NumberFormat = "roman_upper"
)

type Position string

const (
	PositionBottomOutside Position = "bottom_outside"
	PositionBottomInside  Position = "bottom_inside"
	PositionBottomCenter  Position = "bottom_center"
	PositionTopOutside    Position = "top_outside"
	PositionTopInside     Position = "top_inside"
	PositionTopCenter     Position = "top_center"
)

type PageRole string

const (
	RoleBody           PageRole = "body"
	RoleChapterOpening PageRole = "chapter_opening"
	RoleFullPageImage  PageRole = "full_page_image"
	RoleChapterGallery PageRole = "chapter_gallery"
	RoleBlank          PageRole = "blank"
)

type Font struct {
	Family string
	Style  string
	SizePt float64
}

type Offsets struct {
	Top     float64
	Bottom  float64
	Inside  float64
	Outside float64
}

type Settings struct {
	Enabled     bool
	StartOnPage int
	StartNumber int
	Format      NumberFormat
	Position    Position
	Font        Font
	ColorRGB    [3]int
	OffsetMM    Offsets
	HideOn      []PageRole
}

type Placement struct {
	Vertical   string
	Horizontal string
	Alignment  string
	OffsetKey  string
}

func DefaultSettings() Settings {
	return Settings{
		Enabled:     false,
		StartOnPage: 1,
		StartNumber: 1,
		Format:      FormatArabic,
		Position:    PositionBottomOutside,
		Font: Font{
			Family: "Source Serif 4",
			Style:  "Regular",
			SizePt: 9,
		},
		ColorRGB: [3]int{80, 80, 80},
		OffsetMM: Offsets{
			Top:     7,
			Bottom:  7,
			Inside:  10,
			Outside: 10,
		},
		HideOn: nil,
	}
}

func (s Settings) Validate() error {
	if s.StartOnPage < 1 {
		return fmt.Errorf("page_numbers.start_on_page must be >= 1")
	}
	if s.StartNumber < 1 {
		return fmt.Errorf("page_numbers.start_number must be >= 1")
	}
	if !isValidFormat(s.Format) {
		return fmt.Errorf("page_numbers.format must be one of arabic, roman_lower, roman_upper")
	}
	if !isValidPosition(s.Position) {
		return fmt.Errorf("page_numbers.position must be one of bottom_outside, bottom_inside, bottom_center, top_outside, top_inside, top_center")
	}
	if strings.TrimSpace(s.Font.Family) == "" {
		return fmt.Errorf("page_numbers.font.family must be non-empty")
	}
	if strings.TrimSpace(s.Font.Style) == "" {
		return fmt.Errorf("page_numbers.font.style must be non-empty")
	}
	if s.Font.SizePt <= 0 {
		return fmt.Errorf("page_numbers.font.size_pt must be > 0")
	}
	for _, component := range s.ColorRGB {
		if component < 0 || component > 255 {
			return fmt.Errorf("page_numbers.color_rgb values must be between 0 and 255")
		}
	}
	if s.OffsetMM.Top < 0 || s.OffsetMM.Bottom < 0 || s.OffsetMM.Inside < 0 || s.OffsetMM.Outside < 0 {
		return fmt.Errorf("page_numbers.offset_mm values must be non-negative")
	}
	for _, role := range s.HideOn {
		if !isValidRole(role) {
			return fmt.Errorf("page_numbers.hide_on contains unsupported role %q", role)
		}
	}
	return nil
}

func (s Settings) DisplayNumber(physicalPage int) (int, bool) {
	if !s.Enabled || physicalPage < s.StartOnPage {
		return 0, false
	}
	return s.StartNumber + (physicalPage - s.StartOnPage), true
}

func (s Settings) FormattedNumber(physicalPage int) (string, bool, error) {
	number, ok := s.DisplayNumber(physicalPage)
	if !ok {
		return "", false, nil
	}
	formatted, err := FormatNumber(s.Format, number)
	if err != nil {
		return "", false, err
	}
	return formatted, true, nil
}

func (s Settings) HidesRole(role PageRole) bool {
	for _, hiddenRole := range s.HideOn {
		if hiddenRole == role {
			return true
		}
	}
	return false
}

func FormatNumber(format NumberFormat, number int) (string, error) {
	if number < 1 {
		return "", fmt.Errorf("page number must be >= 1")
	}

	switch format {
	case FormatArabic:
		return fmt.Sprintf("%d", number), nil
	case FormatRomanLower:
		return strings.ToLower(toRoman(number)), nil
	case FormatRomanUpper:
		return toRoman(number), nil
	default:
		return "", fmt.Errorf("unsupported page number format %q", format)
	}
}

func ResolvePlacement(position Position, isRightPage bool) (Placement, error) {
	switch position {
	case PositionBottomOutside:
		if isRightPage {
			return Placement{Vertical: "bottom", Horizontal: "right", Alignment: "right", OffsetKey: "outside"}, nil
		}
		return Placement{Vertical: "bottom", Horizontal: "left", Alignment: "left", OffsetKey: "outside"}, nil
	case PositionBottomInside:
		if isRightPage {
			return Placement{Vertical: "bottom", Horizontal: "left", Alignment: "left", OffsetKey: "inside"}, nil
		}
		return Placement{Vertical: "bottom", Horizontal: "right", Alignment: "right", OffsetKey: "inside"}, nil
	case PositionBottomCenter:
		return Placement{Vertical: "bottom", Horizontal: "center", Alignment: "center", OffsetKey: "center"}, nil
	case PositionTopOutside:
		if isRightPage {
			return Placement{Vertical: "top", Horizontal: "right", Alignment: "right", OffsetKey: "outside"}, nil
		}
		return Placement{Vertical: "top", Horizontal: "left", Alignment: "left", OffsetKey: "outside"}, nil
	case PositionTopInside:
		if isRightPage {
			return Placement{Vertical: "top", Horizontal: "left", Alignment: "left", OffsetKey: "inside"}, nil
		}
		return Placement{Vertical: "top", Horizontal: "right", Alignment: "right", OffsetKey: "inside"}, nil
	case PositionTopCenter:
		return Placement{Vertical: "top", Horizontal: "center", Alignment: "center", OffsetKey: "center"}, nil
	default:
		return Placement{}, fmt.Errorf("unsupported page number position %q", position)
	}
}

func toRoman(number int) string {
	values := []struct {
		Value  int
		Symbol string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	var builder strings.Builder
	remaining := number
	for _, entry := range values {
		for remaining >= entry.Value {
			builder.WriteString(entry.Symbol)
			remaining -= entry.Value
		}
	}
	return builder.String()
}

func isValidFormat(format NumberFormat) bool {
	switch format {
	case FormatArabic, FormatRomanLower, FormatRomanUpper:
		return true
	default:
		return false
	}
}

func isValidPosition(position Position) bool {
	switch position {
	case PositionBottomOutside, PositionBottomInside, PositionBottomCenter, PositionTopOutside, PositionTopInside, PositionTopCenter:
		return true
	default:
		return false
	}
}

func isValidRole(role PageRole) bool {
	switch role {
	case RoleBody, RoleChapterOpening, RoleFullPageImage, RoleChapterGallery, RoleBlank:
		return true
	default:
		return false
	}
}
