package imagesizing

import "testing"

func almostEqual(a, b float64) bool {
	delta := a - b
	if delta < 0 {
		delta = -delta
	}
	return delta < 1e-6
}

func TestFitContainLandscapeImage(t *testing.T) {
	result, err := FitContain(1600, 900, 110, 100)
	if err != nil {
		t.Fatalf("FitContain returned error: %v", err)
	}

	if !almostEqual(result.WidthMM, 110) {
		t.Fatalf("expected width 110, got %.6f", result.WidthMM)
	}
	if !almostEqual(result.HeightMM, 61.875) {
		t.Fatalf("expected height 61.875, got %.6f", result.HeightMM)
	}
	if !PreservesAspectRatio(1600, 900, result.WidthMM, result.HeightMM) {
		t.Fatalf("expected aspect ratio to be preserved")
	}
}

func TestFitContainPortraitImage(t *testing.T) {
	result, err := FitContain(900, 1600, 110, 100)
	if err != nil {
		t.Fatalf("FitContain returned error: %v", err)
	}

	if !almostEqual(result.WidthMM, 56.25) {
		t.Fatalf("expected width 56.25, got %.6f", result.WidthMM)
	}
	if !almostEqual(result.HeightMM, 100) {
		t.Fatalf("expected height 100, got %.6f", result.HeightMM)
	}
	if !PreservesAspectRatio(900, 1600, result.WidthMM, result.HeightMM) {
		t.Fatalf("expected aspect ratio to be preserved")
	}
}

func TestExplicitWidthOverridePreservesAspectRatio(t *testing.T) {
	width := 140.0
	result, err := FitInline(1600, 1200, BoundsMM{MaxWidthMM: 110, MaxHeightMM: 100}, Overrides{WidthMM: &width})
	if err != nil {
		t.Fatalf("FitInline returned error: %v", err)
	}

	if !almostEqual(result.WidthMM, 140) {
		t.Fatalf("expected width 140, got %.6f", result.WidthMM)
	}
	if !almostEqual(result.HeightMM, 105) {
		t.Fatalf("expected height 105, got %.6f", result.HeightMM)
	}
	if !PreservesAspectRatio(1600, 1200, result.WidthMM, result.HeightMM) {
		t.Fatalf("expected aspect ratio to be preserved")
	}
}

func TestConflictingWidthHeightOverridesActAsBoundingBox(t *testing.T) {
	width := 140.0
	height := 80.0
	result, err := FitInline(1600, 1200, BoundsMM{MaxWidthMM: 110, MaxHeightMM: 100}, Overrides{WidthMM: &width, HeightMM: &height})
	if err != nil {
		t.Fatalf("FitInline returned error: %v", err)
	}

	if !almostEqual(result.WidthMM, 106.66666666666667) {
		t.Fatalf("expected width 106.6667, got %.10f", result.WidthMM)
	}
	if !almostEqual(result.HeightMM, 80) {
		t.Fatalf("expected height 80, got %.6f", result.HeightMM)
	}
	if !PreservesAspectRatio(1600, 1200, result.WidthMM, result.HeightMM) {
		t.Fatalf("expected aspect ratio to be preserved")
	}
}

func TestAspectRatioInvariant(t *testing.T) {
	cases := []struct {
		sourceW float64
		sourceH float64
		maxW    float64
		maxH    float64
	}{
		{1600, 900, 110, 100},
		{900, 1600, 110, 100},
		{4000, 3000, 50, 80},
		{3000, 4000, 120, 60},
	}

	for _, tc := range cases {
		result, err := FitContain(tc.sourceW, tc.sourceH, tc.maxW, tc.maxH)
		if err != nil {
			t.Fatalf("FitContain returned error: %v", err)
		}
		if !PreservesAspectRatio(tc.sourceW, tc.sourceH, result.WidthMM, result.HeightMM) {
			t.Fatalf("aspect ratio was not preserved for source=%fx%f result=%fx%f", tc.sourceW, tc.sourceH, result.WidthMM, result.HeightMM)
		}
	}
}
