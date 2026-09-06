package gui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// overrideRow is one "inherit or override" control: a checkbox that enables
// editing, an input, and a label showing the value inherited from the template.
type overrideRow struct {
	label     string
	check     *widget.Check
	input     fyne.CanvasObject
	inherited *widget.Label
	// load pulls the current pointer value into the widgets.
	load func()
}

func (r *overrideRow) formItem() *widget.FormItem {
	body := container.NewBorder(nil, nil, r.check, r.inherited, r.input)
	return widget.NewFormItem(r.label, body)
}

func inheritedText(v interface{}) string {
	return fmt.Sprintf("template: %v", v)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// floatRow edits a *float64 override.
func floatRow(label string, ptr **float64, inherited func() float64, onChange func()) *overrideRow {
	entry := widget.NewEntry()
	row := &overrideRow{label: label, input: entry, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			entry.Disable()
			return
		}
		entry.Enable()
		v, err := strconv.ParseFloat(strings.TrimSpace(entry.Text), 64)
		if err != nil {
			return
		}
		*ptr = &v
	}
	row.check.OnChanged = func(checked bool) {
		if checked && strings.TrimSpace(entry.Text) == "" {
			entry.SetText(formatFloat(inherited()))
		}
		apply()
		onChange()
	}
	entry.OnChanged = func(string) { apply(); onChange() }
	entry.Validator = func(s string) error {
		if !row.check.Checked {
			return nil
		}
		_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return err
	}
	row.load = func() {
		row.inherited.SetText(inheritedText(formatFloat(inherited())))
		if *ptr != nil {
			entry.SetText(formatFloat(**ptr))
			entry.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			entry.SetText("")
			entry.Disable()
		}
	}
	return row
}

// intRow edits a *int override.
func intRow(label string, ptr **int, inherited func() int, onChange func()) *overrideRow {
	entry := widget.NewEntry()
	row := &overrideRow{label: label, input: entry, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			entry.Disable()
			return
		}
		entry.Enable()
		v, err := strconv.Atoi(strings.TrimSpace(entry.Text))
		if err != nil {
			return
		}
		*ptr = &v
	}
	row.check.OnChanged = func(checked bool) {
		if checked && strings.TrimSpace(entry.Text) == "" {
			entry.SetText(strconv.Itoa(inherited()))
		}
		apply()
		onChange()
	}
	entry.OnChanged = func(string) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText(inherited()))
		if *ptr != nil {
			entry.SetText(strconv.Itoa(**ptr))
			entry.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			entry.SetText("")
			entry.Disable()
		}
	}
	return row
}

// stringRow edits a free-text *string override.
func stringRow(label string, ptr **string, inherited func() string, onChange func()) *overrideRow {
	entry := widget.NewEntry()
	row := &overrideRow{label: label, input: entry, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			entry.Disable()
			return
		}
		entry.Enable()
		v := entry.Text
		*ptr = &v
	}
	row.check.OnChanged = func(checked bool) {
		if checked && entry.Text == "" {
			entry.SetText(inherited())
		}
		apply()
		onChange()
	}
	entry.OnChanged = func(string) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText(inherited()))
		if *ptr != nil {
			entry.SetText(**ptr)
			entry.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			entry.SetText("")
			entry.Disable()
		}
	}
	return row
}

// enumRow edits a *string override restricted to options.
func enumRow(label string, options []string, ptr **string, inherited func() string, onChange func()) *overrideRow {
	sel := widget.NewSelect(options, nil)
	row := &overrideRow{label: label, input: sel, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			sel.Disable()
			return
		}
		sel.Enable()
		if sel.Selected == "" {
			return
		}
		v := sel.Selected
		*ptr = &v
	}
	row.check.OnChanged = func(checked bool) {
		if checked && sel.Selected == "" {
			sel.SetSelected(inherited())
		}
		apply()
		onChange()
	}
	sel.OnChanged = func(string) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText(inherited()))
		if *ptr != nil {
			sel.SetSelected(**ptr)
			sel.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			sel.ClearSelected()
			sel.Disable()
		}
	}
	return row
}

// boolRow edits a *bool override.
func boolRow(label string, ptr **bool, inherited func() bool, onChange func()) *overrideRow {
	value := widget.NewCheck("Enabled", nil)
	row := &overrideRow{label: label, input: value, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			value.Disable()
			return
		}
		value.Enable()
		v := value.Checked
		*ptr = &v
	}
	row.check.OnChanged = func(checked bool) {
		if checked && *ptr == nil {
			value.SetChecked(inherited())
		}
		apply()
		onChange()
	}
	value.OnChanged = func(bool) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText(inherited()))
		if *ptr != nil {
			value.SetChecked(**ptr)
			value.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			value.SetChecked(false)
			value.Disable()
		}
	}
	return row
}

// rgbRow edits a []int{r,g,b} override; the slice is nil when not overridden.
func rgbRow(label string, ptr *[]int, inherited func() [3]int, onChange func()) *overrideRow {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("r, g, b")
	row := &overrideRow{label: label, input: entry, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			entry.Disable()
			return
		}
		entry.Enable()
		rgb, err := parseRGB(entry.Text)
		if err != nil {
			return
		}
		*ptr = rgb
	}
	row.check.OnChanged = func(checked bool) {
		if checked && strings.TrimSpace(entry.Text) == "" {
			entry.SetText(formatRGB(inherited()))
		}
		apply()
		onChange()
	}
	entry.OnChanged = func(string) { apply(); onChange() }
	entry.Validator = func(s string) error {
		if !row.check.Checked {
			return nil
		}
		_, err := parseRGB(s)
		return err
	}
	row.load = func() {
		row.inherited.SetText(inheritedText(formatRGB(inherited())))
		if *ptr != nil && len(*ptr) == 3 {
			entry.SetText(formatRGB([3]int{(*ptr)[0], (*ptr)[1], (*ptr)[2]}))
			entry.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			entry.SetText("")
			entry.Disable()
		}
	}
	return row
}

// rgbPtrRow edits a *[3]int override.
func rgbPtrRow(label string, ptr **[3]int, inherited func() *[3]int, onChange func()) *overrideRow {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("r, g, b")
	row := &overrideRow{label: label, input: entry, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	inheritedString := func() string {
		if v := inherited(); v != nil {
			return formatRGB(*v)
		}
		return "none"
	}
	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			entry.Disable()
			return
		}
		entry.Enable()
		rgb, err := parseRGB(entry.Text)
		if err != nil {
			return
		}
		*ptr = &[3]int{rgb[0], rgb[1], rgb[2]}
	}
	row.check.OnChanged = func(checked bool) {
		if checked && strings.TrimSpace(entry.Text) == "" {
			if v := inherited(); v != nil {
				entry.SetText(formatRGB(*v))
			} else {
				entry.SetText("255, 255, 255")
			}
		}
		apply()
		onChange()
	}
	entry.OnChanged = func(string) { apply(); onChange() }
	entry.Validator = func(s string) error {
		if !row.check.Checked {
			return nil
		}
		_, err := parseRGB(s)
		return err
	}
	row.load = func() {
		row.inherited.SetText(inheritedText(inheritedString()))
		if *ptr != nil {
			entry.SetText(formatRGB(**ptr))
			entry.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			entry.SetText("")
			entry.Disable()
		}
	}
	return row
}

// listRow edits a []string override as a set of checkboxes drawn from options.
func listRow(label string, options []string, ptr *[]string, inherited func() []string, onChange func()) *overrideRow {
	group := widget.NewCheckGroup(options, nil)
	group.Horizontal = true
	row := &overrideRow{label: label, input: group, inherited: widget.NewLabel("")}
	row.check = widget.NewCheck("Override", nil)

	apply := func() {
		if !row.check.Checked {
			*ptr = nil
			group.Disable()
			return
		}
		group.Enable()
		selected := append([]string{}, group.Selected...)
		*ptr = selected
	}
	row.check.OnChanged = func(checked bool) {
		if checked && *ptr == nil {
			group.SetSelected(append([]string{}, inherited()...))
		}
		apply()
		onChange()
	}
	group.OnChanged = func([]string) { apply(); onChange() }
	row.load = func() {
		row.inherited.SetText(inheritedText(strings.Join(inherited(), ", ")))
		if *ptr != nil {
			group.SetSelected(append([]string{}, (*ptr)...))
			group.Enable()
			row.check.SetChecked(true)
		} else {
			row.check.SetChecked(false)
			group.SetSelected(nil)
			group.Disable()
		}
	}
	return row
}

func parseRGB(text string) ([]int, error) {
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ' ' || r == '[' || r == ']' })
	if len(parts) != 3 {
		return nil, fmt.Errorf("enter three values: r, g, b")
	}
	rgb := make([]int, 3)
	for i, part := range parts {
		v, err := strconv.Atoi(part)
		if err != nil || v < 0 || v > 255 {
			return nil, fmt.Errorf("colour components must be integers 0-255")
		}
		rgb[i] = v
	}
	return rgb, nil
}

func formatRGB(rgb [3]int) string {
	return fmt.Sprintf("%d, %d, %d", rgb[0], rgb[1], rgb[2])
}

// section groups rows into a titled card; refresh reloads every row from the model.
type section struct {
	rows []*overrideRow
}

func (s *section) add(rows ...*overrideRow) {
	s.rows = append(s.rows, rows...)
}

func (s *section) refresh() {
	for _, row := range s.rows {
		row.load()
	}
}

func (s *section) form() *widget.Form {
	form := widget.NewForm()
	for _, row := range s.rows {
		form.AppendItem(row.formItem())
	}
	return form
}
