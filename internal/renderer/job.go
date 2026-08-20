package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

// scribusJob is the JSON payload written for scripts/scribus_generate.py.
// Go owns book config and layout.json; the Python adapter only applies this job
// inside Scribus. Lengths are in PostScript points because that is what the
// Scribus API uses.
type scribusJob struct {
	Page            scribusJobPage            `json:"page"`
	BleedPoints     scribusJobSides           `json:"bleed_points"`
	PageNumbers     scribusJobPageNumbers     `json:"page_numbers"`
	ChapterHeadings scribusJobChapterHeadings `json:"chapter_headings"`
	Images          scribusJobImages          `json:"images"`
	// Layout is the layout.json object (title plus per-image overrides).
	Layout json.RawMessage `json:"layout"`
}

// scribusJobPage is trim size, margins, and facing-page setup for newDocument.
type scribusJobPage struct {
	SizePoints [2]float64 `json:"size_points"` // width, height
	// MarginsPoints is top, left, right, bottom — the Scribus newDocument order.
	MarginsPoints [4]float64 `json:"margins_points"`
	// SizeConstant is a Scribus paper name such as PAPER_A4.
	SizeConstant string `json:"size_constant"`
	// BackgroundRGB is nil when the page should stay the Scribus default fill.
	BackgroundRGB *[3]int `json:"background_rgb"`
	Layout        string  `json:"layout"`     // single_page or facing_pages
	FirstPage     string  `json:"first_page"` // right for a typical book start
}

// scribusJobSides is inside/outside-aware spacing in points (bleed, wrap, offsets).
type scribusJobSides struct {
	Top     float64 `json:"top"`
	Bottom  float64 `json:"bottom"`
	Inside  float64 `json:"inside"`
	Outside float64 `json:"outside"`
}

// scribusJobSpacing is vertical-only spacing, used for chapter-title geometry.
type scribusJobSpacing struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

// scribusJobPageNumbers is logical numbering, type, placement, and which page
// roles hide the folio. Hidden numbers still advance the sequence.
type scribusJobPageNumbers struct {
	Enabled     bool   `json:"enabled"`
	StartOnPage int    `json:"start_on_page"`
	StartNumber int    `json:"start_number"`
	Format      string `json:"format"`
	Position    string `json:"position"`
	// FontName is the exact Scribus face ("Family Style") that must be installed.
	FontName     string          `json:"font_name"`
	FontFamily   string          `json:"font_family"`
	FontSizePt   float64         `json:"font_size_pt"`
	ColorRGB     [3]int          `json:"color_rgb"`
	OffsetPoints scribusJobSides `json:"offset_points"`
	HideOn       []string        `json:"hide_on"`
}

// scribusJobChapterHeadings is the reusable chapter-title paragraph style and
// the extra space above/below the title frame (not blank lines in the text).
type scribusJobChapterHeadings struct {
	FontName      string            `json:"font_name"`
	FontSizePt    float64           `json:"font_size_pt"`
	ColorRGB      [3]int            `json:"color_rgb"`
	Alignment     string            `json:"alignment"`
	SpacingPoints scribusJobSpacing `json:"spacing_points"`
}

// scribusJobImages is template defaults for frames: border, wrap inset, contain
// fit, and which edges images may snap to. layout.json can override per file.
type scribusJobImages struct {
	BorderRGB       [3]int          `json:"border_rgb"`
	BorderWidthPt   float64         `json:"border_width_pt"`
	SpacingPoints   scribusJobSides `json:"spacing_points"`
	MaxWidthPoints  float64         `json:"max_width_points"`
	MaxHeightPoints float64         `json:"max_height_points"`
	SnapToEdge      bool            `json:"snap_to_edge"`
	SnapTarget      string          `json:"snap_target"`
	AllowedEdges    []string        `json:"allowed_edges"`
	PreferredEdges  []string        `json:"preferred_edges"`
	EdgeGapPoints   float64         `json:"edge_gap_points"`
}

// buildScribusJob converts resolved book config (millimetres, YAML enums) into
// the point-based job the Python adapter reads.
func buildScribusJob(cfg config.Config, plan layoutplan.Plan) scribusJob {
	hideOn := make([]string, 0, len(cfg.PageNumbers.HideOn))
	for _, role := range cfg.PageNumbers.HideOn {
		hideOn = append(hideOn, string(role))
	}

	return scribusJob{
		Page: scribusJobPage{
			SizePoints:    [2]float64{mmToPoints(cfg.PageWidth), mmToPoints(cfg.PageHeight)},
			MarginsPoints: [4]float64{mmToPoints(cfg.MarginTop), mmToPoints(cfg.MarginLeft), mmToPoints(cfg.MarginRight), mmToPoints(cfg.MarginBottom)},
			SizeConstant:  fmt.Sprintf("PAPER_%s", strings.ToUpper(strings.ReplaceAll(cfg.PageSize, " ", "_"))),
			BackgroundRGB: cfg.PageBackgroundRGB,
			Layout:        cfg.PageLayout,
			FirstPage:     cfg.FirstPage,
		},
		BleedPoints: scribusJobSides{
			Top:     mmToPoints(cfg.BleedTop),
			Bottom:  mmToPoints(cfg.BleedBottom),
			Inside:  mmToPoints(cfg.BleedInside),
			Outside: mmToPoints(cfg.BleedOutside),
		},
		PageNumbers: scribusJobPageNumbers{
			Enabled:     cfg.PageNumbers.Enabled,
			StartOnPage: cfg.PageNumbers.StartOnPage,
			StartNumber: cfg.PageNumbers.StartNumber,
			Format:      string(cfg.PageNumbers.Format),
			Position:    string(cfg.PageNumbers.Position),
			FontName:    scribusFontName(cfg.PageNumbers.Font.Family, cfg.PageNumbers.Font.Style),
			FontFamily:  cfg.PageNumbers.Font.Family,
			FontSizePt:  cfg.PageNumbers.Font.SizePt,
			ColorRGB:    cfg.PageNumbers.ColorRGB,
			OffsetPoints: scribusJobSides{
				Top:     mmToPoints(cfg.PageNumbers.OffsetMM.Top),
				Bottom:  mmToPoints(cfg.PageNumbers.OffsetMM.Bottom),
				Inside:  mmToPoints(cfg.PageNumbers.OffsetMM.Inside),
				Outside: mmToPoints(cfg.PageNumbers.OffsetMM.Outside),
			},
			HideOn: hideOn,
		},
		ChapterHeadings: scribusJobChapterHeadings{
			FontName:   scribusFontName(cfg.ChapterHeadings.Font.Family, cfg.ChapterHeadings.Font.Style),
			FontSizePt: cfg.ChapterHeadings.Font.SizePt,
			ColorRGB:   cfg.ChapterHeadings.ColorRGB,
			Alignment:  string(cfg.ChapterHeadings.Alignment),
			SpacingPoints: scribusJobSpacing{
				Top:    mmToPoints(cfg.ChapterHeadings.SpacingMM.Top),
				Bottom: mmToPoints(cfg.ChapterHeadings.SpacingMM.Bottom),
			},
		},
		Images: scribusJobImages{
			BorderRGB:     cfg.Images.Border.ColorRGB,
			BorderWidthPt: cfg.Images.Border.WidthPt,
			SpacingPoints: scribusJobSides{
				Top:     mmToPoints(cfg.Images.SpacingMM.Top),
				Bottom:  mmToPoints(cfg.Images.SpacingMM.Bottom),
				Inside:  mmToPoints(cfg.Images.SpacingMM.Inside),
				Outside: mmToPoints(cfg.Images.SpacingMM.Outside),
			},
			MaxWidthPoints:  mmToPoints(cfg.Images.Sizing.MaxWidthMM),
			MaxHeightPoints: mmToPoints(cfg.Images.Sizing.MaxHeightMM),
			SnapToEdge:      cfg.Images.Placement.SnapToEdge,
			SnapTarget:      string(cfg.Images.Placement.SnapTarget),
			AllowedEdges:    imageEdges(cfg.Images.Placement.AllowedEdges),
			PreferredEdges:  imageEdges(cfg.Images.Placement.Preferred),
			EdgeGapPoints:   mmToPoints(cfg.Images.Placement.EdgeGapMM),
		},
		Layout: json.RawMessage(plan.JSON()),
	}
}

// writeScribusJob marshals the job to path, creating the parent directory
// (typically books/<book>/out/).
func writeScribusJob(path string, cfg config.Config, plan layoutplan.Plan) error {
	data, err := json.MarshalIndent(buildScribusJob(cfg, plan), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// imageEdges copies snap-edge enums into JSON strings (outside, inside, top, bottom).
func imageEdges(edges []config.ImageEdge) []string {
	values := make([]string, 0, len(edges))
	for _, edge := range edges {
		values = append(values, string(edge))
	}
	return values
}

// mmToPoints converts millimetres from templates into PostScript points (72/inch).
func mmToPoints(mm float64) float64 {
	return mm * 72.0 / 25.4
}

// scribusFontName joins family and style into the exact face Scribus must provide,
// for example "Source Serif 4 Regular". The renderer does not substitute fonts.
func scribusFontName(family, style string) string {
	family = strings.TrimSpace(family)
	style = strings.TrimSpace(style)
	if family == "" {
		return style
	}
	if style == "" {
		return family
	}
	return family + " " + style
}
