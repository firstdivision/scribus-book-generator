package config

import (
	"strings"
	"testing"
)

func TestParseImageDefaultsSorting(t *testing.T) {
	defaults := DefaultImageDefaults()
	if defaults.Sorting != ImageSortingNone {
		t.Fatalf("expected default sorting none, got %q", defaults.Sorting)
	}

	var raw ImageTemplate
	raw.Sorting = strPtr("date-ascending")
	parsed, err := parseImageDefaults(raw, defaults)
	if err != nil {
		t.Fatalf("parseImageDefaults returned error: %v", err)
	}
	if parsed.Sorting != ImageSortingDateAscending {
		t.Fatalf("expected date-ascending sorting, got %q", parsed.Sorting)
	}

	raw.Sorting = strPtr("date-descending")
	parsed, err = parseImageDefaults(raw, defaults)
	if err != nil {
		t.Fatalf("parseImageDefaults returned error: %v", err)
	}
	if parsed.Sorting != ImageSortingDateDescending {
		t.Fatalf("expected date-descending sorting, got %q", parsed.Sorting)
	}
}

func TestParseImageDefaultsRejectsUnknownSorting(t *testing.T) {
	var raw ImageTemplate
	raw.Sorting = strPtr("alphabetical")

	_, err := parseImageDefaults(raw, DefaultImageDefaults())
	if err == nil || !strings.Contains(err.Error(), "images.sorting") {
		t.Fatalf("expected images.sorting validation error, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
