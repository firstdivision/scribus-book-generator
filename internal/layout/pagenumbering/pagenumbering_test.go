package pagenumbering

import "testing"

func TestDisplayNumber(t *testing.T) {
	settings := DefaultSettings()
	settings.Enabled = true
	settings.StartOnPage = 5
	settings.StartNumber = 1

	if _, ok := settings.DisplayNumber(4); ok {
		t.Fatalf("expected no displayed number on physical page 4")
	}

	if value, ok := settings.DisplayNumber(5); !ok || value != 1 {
		t.Fatalf("expected page 5 to display 1, got %d ok=%v", value, ok)
	}

	if value, ok := settings.DisplayNumber(6); !ok || value != 2 {
		t.Fatalf("expected page 6 to display 2, got %d ok=%v", value, ok)
	}
}

func TestFormatNumberRoman(t *testing.T) {
	testCases := []struct {
		format NumberFormat
		number int
		want   string
	}{
		{FormatRomanLower, 1, "i"},
		{FormatRomanLower, 4, "iv"},
		{FormatRomanLower, 9, "ix"},
		{FormatRomanLower, 14, "xiv"},
		{FormatRomanLower, 49, "xlix"},
		{FormatRomanUpper, 1, "I"},
		{FormatRomanUpper, 4, "IV"},
		{FormatRomanUpper, 9, "IX"},
		{FormatRomanUpper, 14, "XIV"},
		{FormatRomanUpper, 49, "XLIX"},
	}

	for _, tc := range testCases {
		got, err := FormatNumber(tc.format, tc.number)
		if err != nil {
			t.Fatalf("FormatNumber(%q, %d) returned error: %v", tc.format, tc.number, err)
		}
		if got != tc.want {
			t.Fatalf("FormatNumber(%q, %d) = %q, want %q", tc.format, tc.number, got, tc.want)
		}
	}
}

func TestResolvePlacementFacingPages(t *testing.T) {
	left, err := ResolvePlacement(PositionBottomOutside, false)
	if err != nil {
		t.Fatalf("ResolvePlacement returned error: %v", err)
	}
	if left.Vertical != "bottom" || left.Horizontal != "left" || left.Alignment != "left" {
		t.Fatalf("expected left page bottom_outside -> bottom/left/left, got %+v", left)
	}

	right, err := ResolvePlacement(PositionBottomOutside, true)
	if err != nil {
		t.Fatalf("ResolvePlacement returned error: %v", err)
	}
	if right.Vertical != "bottom" || right.Horizontal != "right" || right.Alignment != "right" {
		t.Fatalf("expected right page bottom_outside -> bottom/right/right, got %+v", right)
	}

	insideLeft, err := ResolvePlacement(PositionBottomInside, false)
	if err != nil {
		t.Fatalf("ResolvePlacement returned error: %v", err)
	}
	if insideLeft.Horizontal != "right" || insideLeft.Alignment != "right" {
		t.Fatalf("expected left page bottom_inside -> right/right, got %+v", insideLeft)
	}

	insideRight, err := ResolvePlacement(PositionBottomInside, true)
	if err != nil {
		t.Fatalf("ResolvePlacement returned error: %v", err)
	}
	if insideRight.Horizontal != "left" || insideRight.Alignment != "left" {
		t.Fatalf("expected right page bottom_inside -> left/left, got %+v", insideRight)
	}
}

func TestHideOnDoesNotChangeSequence(t *testing.T) {
	settings := DefaultSettings()
	settings.Enabled = true
	settings.HideOn = []PageRole{RoleChapterOpening}

	if !settings.HidesRole(RoleChapterOpening) {
		t.Fatalf("expected chapter opening role to be hidden")
	}
	if settings.HidesRole(RoleBody) {
		t.Fatalf("did not expect body role to be hidden")
	}

	value, ok := settings.DisplayNumber(9)
	if !ok || value != 9 {
		t.Fatalf("expected physical page 9 to keep logical number 9, got %d ok=%v", value, ok)
	}
}

func TestHideOnAcceptsChapterGalleryRole(t *testing.T) {
	settings := DefaultSettings()
	settings.HideOn = []PageRole{RoleChapterGallery}
	if err := settings.Validate(); err != nil {
		t.Fatalf("expected chapter_gallery hide_on role to be valid, got %v", err)
	}
	if !settings.HidesRole(RoleChapterGallery) {
		t.Fatalf("expected chapter_gallery role to be hidden")
	}
}
