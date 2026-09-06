package gui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/layout/layoutplan"
)

const defaultOption = "(default)"

var placementOptions = []string{
	string(layoutplan.PlacementInline),
	string(layoutplan.PlacementFullPage),
	string(layoutplan.PlacementGallery),
	string(layoutplan.PlacementIgnore),
}

// instructionFormItems builds the per-image fields shared by the layout images
// tab and the chapter editor. Widgets are seeded from inst before callbacks are
// attached so building the form never reports a change.
func instructionFormItems(inst *layoutplan.ImageInstruction, changed func()) []*widget.FormItem {
	placement := widget.NewSelect(append([]string{defaultOption}, placementOptions...), nil)
	if inst.Placement == "" {
		placement.SetSelected(defaultOption)
	} else {
		placement.SetSelected(string(inst.Placement))
	}
	placement.OnChanged = func(s string) {
		if s == defaultOption {
			inst.Placement = ""
		} else {
			inst.Placement = layoutplan.Placement(s)
		}
		changed()
	}

	bleed := widget.NewCheck("Extend into bleed", nil)
	bleed.SetChecked(inst.Bleed)
	bleed.OnChanged = func(b bool) {
		inst.Bleed = b
		changed()
	}

	snap := widget.NewSelect(append([]string{defaultOption}, edgeOptions...), nil)
	if inst.SnapEdge == "" {
		snap.SetSelected(defaultOption)
	} else {
		snap.SetSelected(inst.SnapEdge)
	}
	snap.OnChanged = func(s string) {
		if s == defaultOption {
			inst.SnapEdge = ""
		} else {
			inst.SnapEdge = s
		}
		changed()
	}

	width := optionalFloatEntry(&inst.WidthMM, changed)
	height := optionalFloatEntry(&inst.HeightMM, changed)

	borderWidth := widget.NewEntry()
	borderWidth.SetPlaceHolder("template default")
	borderColour := widget.NewEntry()
	borderColour.SetPlaceHolder("r, g, b (template default)")
	if inst.Border != nil {
		if inst.Border.WidthPt != nil {
			borderWidth.SetText(formatFloat(*inst.Border.WidthPt))
		}
		if len(inst.Border.ColorRGB) == 3 {
			borderColour.SetText(formatRGB([3]int{inst.Border.ColorRGB[0], inst.Border.ColorRGB[1], inst.Border.ColorRGB[2]}))
		}
	}
	applyBorder := func() {
		border := &layoutplan.Border{}
		if v, err := strconv.ParseFloat(strings.TrimSpace(borderWidth.Text), 64); err == nil {
			border.WidthPt = &v
		}
		if rgb, err := parseRGB(borderColour.Text); err == nil {
			border.ColorRGB = rgb
		}
		if border.WidthPt == nil && border.ColorRGB == nil {
			inst.Border = nil
		} else {
			inst.Border = border
		}
		changed()
	}
	borderWidth.OnChanged = func(string) { applyBorder() }
	borderColour.OnChanged = func(string) { applyBorder() }

	return []*widget.FormItem{
		widget.NewFormItem("Placement", placement),
		widget.NewFormItem("Bleed", bleed),
		widget.NewFormItem("Snap edge", snap),
		widget.NewFormItem("Width (mm)", width),
		widget.NewFormItem("Height (mm)", height),
		widget.NewFormItem("Border width (pt)", borderWidth),
		widget.NewFormItem("Border colour", borderColour),
	}
}

// optionalFloatEntry binds an entry to a *float64 that is nil when blank.
func optionalFloatEntry(ptr **float64, changed func()) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("auto")
	if *ptr != nil {
		entry.SetText(formatFloat(**ptr))
	}
	entry.OnChanged = func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			*ptr = nil
			changed()
			return
		}
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			*ptr = &v
			changed()
		}
	}
	entry.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return err
	}
	return entry
}
