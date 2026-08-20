package layoutplan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/layout/imageplacement"
)

type Placement string

const (
	PlacementInline   Placement = "inline"
	PlacementFullPage Placement = "full_page"
	PlacementIgnore   Placement = "ignore"
)

type Plan struct {
	Title  string             `json:"title,omitempty"`
	Images []ImageInstruction `json:"images"`
}

type ImageInstruction struct {
	Src       string    `json:"src,omitempty"`
	File      string    `json:"file,omitempty"`
	Placement Placement `json:"placement,omitempty"`
	Bleed     bool      `json:"bleed,omitempty"`
	SnapEdge  string    `json:"snap_edge,omitempty"`
	WidthMM   *float64  `json:"width_mm,omitempty"`
	HeightMM  *float64  `json:"height_mm,omitempty"`
	Border    *Border   `json:"border,omitempty"`
}

type Border struct {
	ColorRGB []int    `json:"color_rgb,omitempty"`
	WidthPt  *float64 `json:"width_pt,omitempty"`
}

type InlineOverride struct {
	WidthMM  *float64
	HeightMM *float64
	SnapEdge *imageplacement.Edge
}

func LoadFromBookDir(bookDir string) (Plan, error) {
	layoutPath := filepath.Join(bookDir, "layout.json")
	data, err := os.ReadFile(layoutPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Plan{}, nil
		}
		return Plan{}, err
	}

	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, err
	}

	if err := plan.Validate(); err != nil {
		return Plan{}, err
	}

	return plan, nil
}

func (p Plan) Validate() error {
	for i, image := range p.Images {
		path := image.Source()
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("layout.images[%d].src or .file must be non-empty", i)
		}
		if image.WidthMM != nil && *image.WidthMM <= 0 {
			return fmt.Errorf("layout.images[%d].width_mm must be > 0", i)
		}
		if image.HeightMM != nil && *image.HeightMM <= 0 {
			return fmt.Errorf("layout.images[%d].height_mm must be > 0", i)
		}
		if image.Border != nil {
			if image.Border.WidthPt != nil && *image.Border.WidthPt < 0 {
				return fmt.Errorf("layout.images[%d].border.width_pt must be >= 0", i)
			}
			if image.Border.ColorRGB != nil {
				if len(image.Border.ColorRGB) != 3 {
					return fmt.Errorf("layout.images[%d].border.color_rgb must contain exactly 3 integers", i)
				}
				for _, component := range image.Border.ColorRGB {
					if component < 0 || component > 255 {
						return fmt.Errorf("layout.images[%d].border.color_rgb values must be between 0 and 255", i)
					}
				}
			}
		}
		if image.Placement != "" && image.Placement != PlacementInline && image.Placement != PlacementFullPage && image.Placement != PlacementIgnore {
			return fmt.Errorf("layout.images[%d].placement must be one of inline, full_page, ignore", i)
		}
		if strings.TrimSpace(image.SnapEdge) != "" {
			edge := imageplacement.Edge(strings.TrimSpace(image.SnapEdge))
			if !imageplacement.IsValidEdge(edge) {
				return fmt.Errorf("layout.images[%d].snap_edge must be one of outside, inside, top, bottom", i)
			}
		}
	}
	return nil
}

func (p Plan) JSON() string {
	if p.Images == nil {
		p.Images = []ImageInstruction{}
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return "{\"images\":[]}"
	}
	return string(encoded)
}

func (i ImageInstruction) Source() string {
	if strings.TrimSpace(i.File) != "" {
		return i.File
	}
	return i.Src
}

func (i ImageInstruction) InlineSettings() (InlineOverride, error) {
	override := InlineOverride{
		WidthMM:  i.WidthMM,
		HeightMM: i.HeightMM,
	}
	if strings.TrimSpace(i.SnapEdge) == "" {
		return override, nil
	}
	edge := imageplacement.Edge(strings.TrimSpace(i.SnapEdge))
	if !imageplacement.IsValidEdge(edge) {
		return override, fmt.Errorf("unsupported snap_edge %q", i.SnapEdge)
	}
	override.SnapEdge = &edge
	return override, nil
}
