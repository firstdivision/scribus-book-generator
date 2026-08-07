package config

import "fmt"

type ImageEdge string

type ImageSnapTarget string

const (
	ImageEdgeOutside ImageEdge = "outside"
	ImageEdgeInside  ImageEdge = "inside"
	ImageEdgeTop     ImageEdge = "top"
	ImageEdgeBottom  ImageEdge = "bottom"
)

const (
	ImageSnapTargetContentArea ImageSnapTarget = "content_area"
	ImageSnapTargetTrim        ImageSnapTarget = "trim"
	ImageSnapTargetBleed       ImageSnapTarget = "bleed"
)

type ImageBorder struct {
	ColorRGB [3]int
	WidthPt  float64
}

type ImageSpacing struct {
	Top     float64
	Bottom  float64
	Inside  float64
	Outside float64
}

type ImageSizing struct {
	MaxWidthMM  float64
	MaxHeightMM float64
}

type ImagePlacement struct {
	SnapToEdge   bool
	SnapTarget   ImageSnapTarget
	AllowedEdges []ImageEdge
	Preferred    []ImageEdge
	EdgeGapMM    float64
}

type ImageDefaults struct {
	Border    ImageBorder
	SpacingMM ImageSpacing
	Sizing    ImageSizing
	Placement ImagePlacement
}

type imageTemplateConfig struct {
	Border struct {
		ColorRGB []int    `yaml:"color_rgb"`
		WidthPt  *float64 `yaml:"width_pt"`
	} `yaml:"border"`
	SpacingMM struct {
		Top     *float64 `yaml:"top"`
		Bottom  *float64 `yaml:"bottom"`
		Inside  *float64 `yaml:"inside"`
		Outside *float64 `yaml:"outside"`
	} `yaml:"spacing_mm"`
	Sizing struct {
		MaxWidthMM  *float64 `yaml:"max_width_mm"`
		MaxHeightMM *float64 `yaml:"max_height_mm"`
	} `yaml:"sizing"`
	Placement struct {
		SnapToEdge   *bool    `yaml:"snap_to_edge"`
		SnapTarget   string   `yaml:"snap_target"`
		AllowedEdges []string `yaml:"allowed_edges"`
		Preferred    []string `yaml:"preferred_edges"`
		EdgeGapMM    *float64 `yaml:"edge_gap_mm"`
	} `yaml:"placement"`
}

func DefaultImageDefaults() ImageDefaults {
	return ImageDefaults{
		Border: ImageBorder{
			ColorRGB: [3]int{255, 255, 255},
			WidthPt:  0,
		},
		SpacingMM: ImageSpacing{
			Top:     5,
			Bottom:  5,
			Inside:  5,
			Outside: 5,
		},
		Sizing: ImageSizing{
			MaxWidthMM:  110,
			MaxHeightMM: 100,
		},
		Placement: ImagePlacement{
			SnapToEdge: true,
			SnapTarget: ImageSnapTargetContentArea,
			AllowedEdges: []ImageEdge{
				ImageEdgeOutside,
				ImageEdgeInside,
				ImageEdgeTop,
				ImageEdgeBottom,
			},
			Preferred: []ImageEdge{
				ImageEdgeOutside,
				ImageEdgeTop,
			},
			EdgeGapMM: 0,
		},
	}
}

func parseImageDefaults(raw imageTemplateConfig, defaults ImageDefaults) (ImageDefaults, error) {
	parsed := defaults

	if raw.Border.ColorRGB != nil {
		if len(raw.Border.ColorRGB) != 3 {
			return parsed, fmt.Errorf("images.border.color_rgb must contain exactly 3 integers")
		}
		parsed.Border.ColorRGB = [3]int{raw.Border.ColorRGB[0], raw.Border.ColorRGB[1], raw.Border.ColorRGB[2]}
	}
	if raw.Border.WidthPt != nil {
		parsed.Border.WidthPt = *raw.Border.WidthPt
	}

	if raw.SpacingMM.Top != nil {
		parsed.SpacingMM.Top = *raw.SpacingMM.Top
	}
	if raw.SpacingMM.Bottom != nil {
		parsed.SpacingMM.Bottom = *raw.SpacingMM.Bottom
	}
	if raw.SpacingMM.Inside != nil {
		parsed.SpacingMM.Inside = *raw.SpacingMM.Inside
	}
	if raw.SpacingMM.Outside != nil {
		parsed.SpacingMM.Outside = *raw.SpacingMM.Outside
	}

	if raw.Sizing.MaxWidthMM != nil {
		parsed.Sizing.MaxWidthMM = *raw.Sizing.MaxWidthMM
	}
	if raw.Sizing.MaxHeightMM != nil {
		parsed.Sizing.MaxHeightMM = *raw.Sizing.MaxHeightMM
	}

	if raw.Placement.SnapToEdge != nil {
		parsed.Placement.SnapToEdge = *raw.Placement.SnapToEdge
	}
	if raw.Placement.SnapTarget != "" {
		parsed.Placement.SnapTarget = ImageSnapTarget(raw.Placement.SnapTarget)
	}
	if raw.Placement.AllowedEdges != nil {
		parsedAllowed := make([]ImageEdge, 0, len(raw.Placement.AllowedEdges))
		for _, edge := range raw.Placement.AllowedEdges {
			parsedAllowed = append(parsedAllowed, ImageEdge(edge))
		}
		parsed.Placement.AllowedEdges = parsedAllowed
	}
	if raw.Placement.Preferred != nil {
		parsedPreferred := make([]ImageEdge, 0, len(raw.Placement.Preferred))
		for _, edge := range raw.Placement.Preferred {
			parsedPreferred = append(parsedPreferred, ImageEdge(edge))
		}
		parsed.Placement.Preferred = parsedPreferred
	}
	if raw.Placement.EdgeGapMM != nil {
		parsed.Placement.EdgeGapMM = *raw.Placement.EdgeGapMM
	}

	if err := parsed.Validate(); err != nil {
		return parsed, err
	}

	return parsed, nil
}

func (d ImageDefaults) Validate() error {
	for _, component := range d.Border.ColorRGB {
		if component < 0 || component > 255 {
			return fmt.Errorf("images.border.color_rgb values must be between 0 and 255")
		}
	}
	if d.Border.WidthPt < 0 {
		return fmt.Errorf("images.border.width_pt must be >= 0")
	}
	if d.SpacingMM.Top < 0 || d.SpacingMM.Bottom < 0 || d.SpacingMM.Inside < 0 || d.SpacingMM.Outside < 0 {
		return fmt.Errorf("images.spacing_mm values must be >= 0")
	}
	if d.Sizing.MaxWidthMM <= 0 {
		return fmt.Errorf("images.sizing.max_width_mm must be > 0")
	}
	if d.Sizing.MaxHeightMM <= 0 {
		return fmt.Errorf("images.sizing.max_height_mm must be > 0")
	}
	if !isValidImageSnapTarget(d.Placement.SnapTarget) {
		return fmt.Errorf("images.placement.snap_target must be one of content_area, trim, bleed")
	}
	if d.Placement.EdgeGapMM < 0 {
		return fmt.Errorf("images.placement.edge_gap_mm must be >= 0")
	}
	if len(d.Placement.AllowedEdges) == 0 {
		return fmt.Errorf("images.placement.allowed_edges must not be empty")
	}

	allowed := map[ImageEdge]bool{}
	for _, edge := range d.Placement.AllowedEdges {
		if !isValidImageEdge(edge) {
			return fmt.Errorf("images.placement.allowed_edges contains unsupported edge %q", edge)
		}
		allowed[edge] = true
	}

	for _, edge := range d.Placement.Preferred {
		if !isValidImageEdge(edge) {
			return fmt.Errorf("images.placement.preferred_edges contains unsupported edge %q", edge)
		}
		if !allowed[edge] {
			return fmt.Errorf("images.placement.preferred_edges contains %q which is not in allowed_edges", edge)
		}
	}

	return nil
}

func isValidImageSnapTarget(target ImageSnapTarget) bool {
	switch target {
	case ImageSnapTargetContentArea, ImageSnapTargetTrim, ImageSnapTargetBleed:
		return true
	default:
		return false
	}
}

func isValidImageEdge(edge ImageEdge) bool {
	switch edge {
	case ImageEdgeOutside, ImageEdgeInside, ImageEdgeTop, ImageEdgeBottom:
		return true
	default:
		return false
	}
}
