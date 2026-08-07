package imagesizing

import "fmt"

type BoundsMM struct {
	MaxWidthMM  float64
	MaxHeightMM float64
}

type Overrides struct {
	WidthMM  *float64
	HeightMM *float64
}

type Result struct {
	WidthMM  float64
	HeightMM float64
	Scale    float64
}

func FitContain(sourceWidth, sourceHeight, maxWidth, maxHeight float64) (Result, error) {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return Result{}, fmt.Errorf("source dimensions must be > 0")
	}
	if maxWidth <= 0 || maxHeight <= 0 {
		return Result{}, fmt.Errorf("bounding dimensions must be > 0")
	}

	scale := min(maxWidth/sourceWidth, maxHeight/sourceHeight)
	return Result{
		WidthMM:  sourceWidth * scale,
		HeightMM: sourceHeight * scale,
		Scale:    scale,
	}, nil
}

func FitInline(sourceWidth, sourceHeight float64, defaults BoundsMM, overrides Overrides) (Result, error) {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return Result{}, fmt.Errorf("source dimensions must be > 0")
	}
	if defaults.MaxWidthMM <= 0 || defaults.MaxHeightMM <= 0 {
		return Result{}, fmt.Errorf("default bounds must be > 0")
	}

	if overrides.WidthMM != nil && *overrides.WidthMM <= 0 {
		return Result{}, fmt.Errorf("width override must be > 0")
	}
	if overrides.HeightMM != nil && *overrides.HeightMM <= 0 {
		return Result{}, fmt.Errorf("height override must be > 0")
	}

	if overrides.WidthMM != nil && overrides.HeightMM != nil {
		return FitContain(sourceWidth, sourceHeight, *overrides.WidthMM, *overrides.HeightMM)
	}

	if overrides.WidthMM != nil {
		height := *overrides.WidthMM * (sourceHeight / sourceWidth)
		return Result{
			WidthMM:  *overrides.WidthMM,
			HeightMM: height,
			Scale:    *overrides.WidthMM / sourceWidth,
		}, nil
	}

	if overrides.HeightMM != nil {
		width := *overrides.HeightMM * (sourceWidth / sourceHeight)
		return Result{
			WidthMM:  width,
			HeightMM: *overrides.HeightMM,
			Scale:    *overrides.HeightMM / sourceHeight,
		}, nil
	}

	return FitContain(sourceWidth, sourceHeight, defaults.MaxWidthMM, defaults.MaxHeightMM)
}

func PreservesAspectRatio(sourceWidth, sourceHeight, renderedWidth, renderedHeight float64) bool {
	if sourceWidth <= 0 || sourceHeight <= 0 || renderedWidth <= 0 || renderedHeight <= 0 {
		return false
	}
	sourceRatio := sourceWidth / sourceHeight
	renderedRatio := renderedWidth / renderedHeight
	delta := sourceRatio - renderedRatio
	if delta < 0 {
		delta = -delta
	}
	return delta < 1e-6
}
