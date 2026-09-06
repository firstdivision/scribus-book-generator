package gui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/chapterheadings"
	"scribus-book-generator/internal/layout/pagenumbering"
)

var (
	pageLayoutOptions      = []string{"single_page", "facing_pages"}
	firstPageOptions       = []string{"right", "left"}
	pageSizeOptions        = []string{"A4", "LETTER"}
	pageOrientationOptions = []string{"portrait", "landscape"}
	alignmentOptions       = []string{string(chapterheadings.AlignmentLeft), string(chapterheadings.AlignmentCenter), string(chapterheadings.AlignmentRight), string(chapterheadings.AlignmentInside), string(chapterheadings.AlignmentOutside)}
	borderSides            = []string{"top", "bottom", "left", "right"}
	imageSortingOptions    = []string{string(config.ImageSortingNone), string(config.ImageSortingDateAscending), string(config.ImageSortingDateDescending)}
	snapTargetOptions      = []string{string(config.ImageSnapTargetContentArea), string(config.ImageSnapTargetTrim), string(config.ImageSnapTargetBleed)}
	edgeOptions            = []string{string(config.ImageEdgeOutside), string(config.ImageEdgeInside), string(config.ImageEdgeTop), string(config.ImageEdgeBottom)}
	edgeSelectionOptions   = []string{string(config.ImageEdgeSelectionPreferred), string(config.ImageEdgeSelectionRandom)}
	numberFormatOptions    = []string{string(pagenumbering.FormatArabic), string(pagenumbering.FormatRomanLower), string(pagenumbering.FormatRomanUpper)}
	positionOptions        = []string{string(pagenumbering.PositionBottomOutside), string(pagenumbering.PositionBottomInside), string(pagenumbering.PositionBottomCenter), string(pagenumbering.PositionTopOutside), string(pagenumbering.PositionTopInside), string(pagenumbering.PositionTopCenter)}
	pageRoleOptions        = []string{string(pagenumbering.RoleBody), string(pagenumbering.RoleChapterOpening), string(pagenumbering.RoleFullPageImage), string(pagenumbering.RoleChapterGallery), string(pagenumbering.RoleBlank)}
)

// overrideTabs builds every template-override tab. Each returned section must be
// refreshed whenever the template (and thus the inherited values) changes.
type overrideTabs struct {
	sections []*section
	items    []*container.TabItem
}

func (o *overrideTabs) refresh() {
	for _, s := range o.sections {
		s.refresh()
	}
}

func buildOverrideTabs(state *State, onChange func()) *overrideTabs {
	o := &overrideTabs{}
	add := func(title string, content fyne.CanvasObject, sections ...*section) {
		o.sections = append(o.sections, sections...)
		o.items = append(o.items, container.NewTabItem(title, container.NewVScroll(content)))
	}

	page := pageSection(state, onChange)
	add("Page", page.form(), page)

	bleed := sidesSection(&state.File.Overrides.Bleed, func() config.Config { return state.TemplateConfig }, "bleed", onChange)
	margin := sidesSection(&state.File.Overrides.SafetyMargin, func() config.Config { return state.TemplateConfig }, "safety_margin", onChange)
	add("Bleed & Margins", container.NewVBox(
		widget.NewCard("Bleed (mm)", "", bleed.form()),
		widget.NewCard("Safety margin (mm)", "Becomes the page margins.", margin.form()),
	), bleed, margin)

	headings, headingBorders := chapterHeadingSection(state, onChange)
	add("Chapter Headings", container.NewVBox(
		headings.form(),
		widget.NewCard("Borders", "Overriding any side replaces all template borders.", headingBorders),
	), headings)

	imgs := imageSection(state, onChange)
	add("Image Defaults", imgs.form(), imgs)

	nums := pageNumberSection(state, onChange)
	add("Page Numbers", nums.form(), nums)

	return o
}

func pageSection(state *State, onChange func()) *section {
	s := &section{}
	ov := state.File.Overrides
	cfg := func() config.Config { return state.TemplateConfig }
	s.add(
		stringRow("Units", &ov.Document.Units, func() string { return cfg().DocumentUnits }, onChange),
		enumRow("Layout", pageLayoutOptions, &ov.Document.Layout, func() string { return cfg().PageLayout }, onChange),
		enumRow("First page", firstPageOptions, &ov.Document.FirstPage, func() string { return cfg().FirstPage }, onChange),
		enumRow("Page size", pageSizeOptions, &ov.Page.Size, func() string { return cfg().PageSize }, onChange),
		enumRow("Orientation", pageOrientationOptions, &ov.Page.Orientation, func() string { return cfg().PageOrientation }, onChange),
		floatRow("Width (mm)", &ov.Page.WidthMM, func() float64 { return cfg().PageWidth }, onChange),
		floatRow("Height (mm)", &ov.Page.HeightMM, func() float64 { return cfg().PageHeight }, onChange),
		rgbPtrRow("Background colour", &ov.Page.BackgroundColorRGB, func() *[3]int { return cfg().PageBackgroundRGB }, onChange),
	)
	return s
}

func sidesSection(sides *config.SidesTemplate, cfg func() config.Config, kind string, onChange func()) *section {
	s := &section{}
	var top, bottom, inside, outside func() float64
	switch kind {
	case "bleed":
		top = func() float64 { return cfg().BleedTop }
		bottom = func() float64 { return cfg().BleedBottom }
		inside = func() float64 { return cfg().BleedInside }
		outside = func() float64 { return cfg().BleedOutside }
	default:
		top = func() float64 { return cfg().MarginTop }
		bottom = func() float64 { return cfg().MarginBottom }
		inside = func() float64 { return cfg().MarginLeft }
		outside = func() float64 { return cfg().MarginRight }
	}
	s.add(
		floatRow("Top", &sides.Top, top, onChange),
		floatRow("Bottom", &sides.Bottom, bottom, onChange),
		floatRow("Inside", &sides.Inside, inside, onChange),
		floatRow("Outside", &sides.Outside, outside, onChange),
	)
	return s
}

func chapterHeadingSection(state *State, onChange func()) (*section, fyne.CanvasObject) {
	s := &section{}
	ov := &state.File.Overrides.ChapterHeadings
	ch := func() chapterheadings.Settings { return state.TemplateConfig.ChapterHeadings }
	s.add(
		stringRow("Font family", &ov.Font.Family, func() string { return ch().Font.Family }, onChange),
		stringRow("Font style", &ov.Font.Style, func() string { return ch().Font.Style }, onChange),
		floatRow("Font size (pt)", &ov.Font.SizePt, func() float64 { return ch().Font.SizePt }, onChange),
		rgbRow("Text colour", &ov.ColorRGB, func() [3]int { return ch().ColorRGB }, onChange),
		rgbRow("Background colour", &ov.BackgroundColorRGB, func() [3]int {
			if bg := ch().BackgroundColorRGB; bg != nil {
				return *bg
			}
			return [3]int{255, 255, 255}
		}, onChange),
		enumRow("Alignment", alignmentOptions, &ov.Alignment, func() string { return string(ch().Alignment) }, onChange),
		floatRow("Space above (mm)", &ov.SpacingMM.Top, func() float64 { return ch().SpacingMM.Top }, onChange),
		floatRow("Space below (mm)", &ov.SpacingMM.Bottom, func() float64 { return ch().SpacingMM.Bottom }, onChange),
	)

	borders := widget.NewForm()
	for _, side := range borderSides {
		row := borderRow(side, ov, ch, onChange)
		s.rows = append(s.rows, row)
		borders.AppendItem(row.formItem())
	}
	return s, borders
}

// borderRow edits one side of chapter_headings.borders; the side is present in
// the override map only while its checkbox is on.
func borderRow(side string, ov *config.ChapterHeadingTemplate, ch func() chapterheadings.Settings, onChange func()) *overrideRow {
	width := widget.NewEntry()
	width.SetPlaceHolder("width pt")
	colour := widget.NewEntry()
	colour.SetPlaceHolder("r, g, b (optional)")
	input := container.NewGridWithColumns(2, width, colour)
	row := &overrideRow{label: strings.ToUpper(side[:1]) + side[1:], input: input, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	inheritedText := func() string {
		if b, ok := ch().Borders[side]; ok && b.WidthPt > 0 {
			return fmt.Sprintf("template: %s pt %s", formatFloat(b.WidthPt), formatRGB(b.ColorRGB))
		}
		return "template: none"
	}
	apply := func() {
		if !row.check.Checked {
			if ov.Borders != nil {
				delete(ov.Borders, side)
				if len(ov.Borders) == 0 {
					ov.Borders = nil
				}
			}
			width.Disable()
			colour.Disable()
			return
		}
		width.Enable()
		colour.Enable()
		if ov.Borders == nil {
			ov.Borders = map[string]*config.ChapterHeadingBorderTemplate{}
		}
		entry := &config.ChapterHeadingBorderTemplate{}
		if w, err := strconv.ParseFloat(strings.TrimSpace(width.Text), 64); err == nil {
			entry.WidthPt = &w
		}
		if strings.TrimSpace(colour.Text) != "" {
			if rgb, err := parseRGB(colour.Text); err == nil {
				entry.ColorRGB = rgb
			}
		}
		ov.Borders[side] = entry
	}
	row.check.OnChanged = func(checked bool) {
		if checked && strings.TrimSpace(width.Text) == "" {
			if b, ok := ch().Borders[side]; ok {
				width.SetText(formatFloat(b.WidthPt))
			} else {
				width.SetText("1")
			}
		}
		apply()
		onChange()
	}
	width.OnChanged = func(string) { apply(); onChange() }
	colour.OnChanged = func(string) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText())
		if b, ok := ov.Borders[side]; ok {
			width.Enable()
			colour.Enable()
			if b != nil && b.WidthPt != nil {
				width.SetText(formatFloat(*b.WidthPt))
			} else {
				width.SetText("1")
			}
			if b != nil && len(b.ColorRGB) == 3 {
				colour.SetText(formatRGB([3]int{b.ColorRGB[0], b.ColorRGB[1], b.ColorRGB[2]}))
			} else {
				colour.SetText("")
			}
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			width.SetText("")
			colour.SetText("")
			width.Disable()
			colour.Disable()
		}
	}
	return row
}

func imageSection(state *State, onChange func()) *section {
	s := &section{}
	ov := &state.File.Overrides.Images
	im := func() config.ImageDefaults { return state.TemplateConfig.Images }
	edges := func(list []config.ImageEdge) []string {
		out := make([]string, 0, len(list))
		for _, e := range list {
			out = append(out, string(e))
		}
		return out
	}
	s.add(
		enumRow("Sorting", imageSortingOptions, &ov.Sorting, func() string { return string(im().Sorting) }, onChange),
		floatRow("Border width (pt)", &ov.Border.WidthPt, func() float64 { return im().Border.WidthPt }, onChange),
		rgbRow("Border colour", &ov.Border.ColorRGB, func() [3]int { return im().Border.ColorRGB }, onChange),
		floatRow("Spacing top (mm)", &ov.SpacingMM.Top, func() float64 { return im().SpacingMM.Top }, onChange),
		floatRow("Spacing bottom (mm)", &ov.SpacingMM.Bottom, func() float64 { return im().SpacingMM.Bottom }, onChange),
		floatRow("Spacing inside (mm)", &ov.SpacingMM.Inside, func() float64 { return im().SpacingMM.Inside }, onChange),
		floatRow("Spacing outside (mm)", &ov.SpacingMM.Outside, func() float64 { return im().SpacingMM.Outside }, onChange),
		floatRow("Max width (mm)", &ov.Sizing.MaxWidthMM, func() float64 { return im().Sizing.MaxWidthMM }, onChange),
		floatRow("Max height (mm)", &ov.Sizing.MaxHeightMM, func() float64 { return im().Sizing.MaxHeightMM }, onChange),
		boolRow("Snap to edge", &ov.Placement.SnapToEdge, func() bool { return im().Placement.SnapToEdge }, onChange),
		enumRow("Snap target", snapTargetOptions, &ov.Placement.SnapTarget, func() string { return string(im().Placement.SnapTarget) }, onChange),
		listRow("Allowed edges", edgeOptions, &ov.Placement.AllowedEdges, func() []string { return edges(im().Placement.AllowedEdges) }, onChange),
		listRow("Preferred edges", edgeOptions, &ov.Placement.Preferred, func() []string { return edges(im().Placement.Preferred) }, onChange),
		enumRow("Edge selection", edgeSelectionOptions, &ov.Placement.Selection, func() string { return string(im().Placement.Selection) }, onChange),
		floatRow("Edge gap (mm)", &ov.Placement.EdgeGapMM, func() float64 { return im().Placement.EdgeGapMM }, onChange),
		intRow("Gallery columns", &ov.Leftovers.GalleryColumns, func() int { return im().Leftovers.GalleryColumns }, onChange),
	)
	return s
}

func pageNumberSection(state *State, onChange func()) *section {
	s := &section{}
	ov := &state.File.Overrides.PageNumbers
	pn := func() pagenumbering.Settings { return state.TemplateConfig.PageNumbers }
	roles := func() []string {
		out := make([]string, 0, len(pn().HideOn))
		for _, r := range pn().HideOn {
			out = append(out, string(r))
		}
		sort.Strings(out)
		return out
	}
	s.add(
		boolRow("Page numbers", &ov.Enabled, func() bool { return pn().Enabled }, onChange),
		intRow("Start on page", &ov.StartOnPage, func() int { return pn().StartOnPage }, onChange),
		intRow("Start number", &ov.StartNumber, func() int { return pn().StartNumber }, onChange),
		enumRow("Format", numberFormatOptions, &ov.Format, func() string { return string(pn().Format) }, onChange),
		enumRow("Position", positionOptions, &ov.Position, func() string { return string(pn().Position) }, onChange),
		stringRow("Font family", &ov.Font.Family, func() string { return pn().Font.Family }, onChange),
		stringRow("Font style", &ov.Font.Style, func() string { return pn().Font.Style }, onChange),
		floatRow("Font size (pt)", &ov.Font.SizePt, func() float64 { return pn().Font.SizePt }, onChange),
		rgbRow("Colour", &ov.ColorRGB, func() [3]int { return pn().ColorRGB }, onChange),
		floatRow("Offset top (mm)", &ov.OffsetMM.Top, func() float64 { return pn().OffsetMM.Top }, onChange),
		floatRow("Offset bottom (mm)", &ov.OffsetMM.Bottom, func() float64 { return pn().OffsetMM.Bottom }, onChange),
		floatRow("Offset inside (mm)", &ov.OffsetMM.Inside, func() float64 { return pn().OffsetMM.Inside }, onChange),
		floatRow("Offset outside (mm)", &ov.OffsetMM.Outside, func() float64 { return pn().OffsetMM.Outside }, onChange),
		listRow("Hide on", pageRoleOptions, &ov.HideOn, roles, onChange),
	)
	return s
}
