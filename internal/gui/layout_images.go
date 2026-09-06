package gui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"scribus-book-generator/internal/layout/layoutplan"
)

// layoutImagesTab edits layout.title and the layout.images list.
type layoutImagesTab struct {
	state    *State
	thumbs   *thumbnailCache
	onChange func()

	list     *widget.List
	selected int
	editor   *fyne.Container
	content  fyne.CanvasObject
	titleBox *widget.Entry
}

func newLayoutImagesTab(state *State, thumbs *thumbnailCache, onChange func()) *layoutImagesTab {
	t := &layoutImagesTab{state: state, thumbs: thumbs, onChange: onChange, selected: -1}

	t.titleBox = widget.NewEntry()
	t.titleBox.SetPlaceHolder("Book title (used for output file names)")
	t.titleBox.SetText(state.File.Layout.Title)
	t.titleBox.OnChanged = func(s string) {
		state.File.Layout.Title = s
		onChange()
	}

	t.list = widget.NewList(
		func() int { return len(t.images()) },
		func() fyne.CanvasObject {
			img := canvas.NewImageFromImage(nil)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(thumbnailSize, thumbnailSize*0.75))
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis
			detail := widget.NewLabel("")
			detail.TextStyle = fyne.TextStyle{Italic: true}
			return container.NewBorder(nil, nil, img, nil, container.NewVBox(name, detail))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			images := t.images()
			if id < 0 || id >= len(images) {
				return
			}
			instruction := images[id]
			row := obj.(*fyne.Container)
			img := row.Objects[1].(*canvas.Image)
			labels := row.Objects[0].(*fyne.Container)
			labels.Objects[0].(*widget.Label).SetText(instruction.Source())
			labels.Objects[1].(*widget.Label).SetText(describeInstruction(instruction))
			t.thumbs.fill(img, filepath.Join(state.BookDir, instruction.Source()), func() { t.list.RefreshItem(id) })
		},
	)
	t.list.OnSelected = func(id widget.ListItemID) {
		t.selected = id
		t.rebuildEditor()
	}
	t.list.OnUnselected = func(widget.ListItemID) {
		t.selected = -1
		t.rebuildEditor()
	}

	addButton := widget.NewButton("Add image", t.addImage)
	removeButton := widget.NewButton("Remove", t.removeSelected)
	upButton := widget.NewButton("Move up", func() { t.moveSelected(-1) })
	downButton := widget.NewButton("Move down", func() { t.moveSelected(1) })
	buttons := container.NewHBox(addButton, removeButton, upButton, downButton)

	t.editor = container.NewVBox()
	t.rebuildEditor()

	left := container.NewBorder(nil, buttons, nil, nil, t.list)
	split := container.NewHSplit(left, container.NewVScroll(t.editor))
	split.SetOffset(0.45)

	header := widget.NewForm(widget.NewFormItem("Title", t.titleBox))
	t.content = container.NewBorder(header, nil, nil, nil, split)
	return t
}

func (t *layoutImagesTab) images() []layoutplan.ImageInstruction {
	return t.state.File.Layout.Images
}

func describeInstruction(i layoutplan.ImageInstruction) string {
	parts := []string{string(i.Placement)}
	if i.Placement == "" {
		parts[0] = "(no placement)"
	}
	if i.Bleed {
		parts = append(parts, "bleed")
	}
	if i.SnapEdge != "" {
		parts = append(parts, "snap "+i.SnapEdge)
	}
	if i.WidthMM != nil {
		parts = append(parts, formatFloat(*i.WidthMM)+"mm wide")
	}
	if i.HeightMM != nil {
		parts = append(parts, formatFloat(*i.HeightMM)+"mm tall")
	}
	if i.Border != nil && i.Border.WidthPt != nil {
		parts = append(parts, "border "+formatFloat(*i.Border.WidthPt)+"pt")
	}
	return strings.Join(parts, " · ")
}

func (t *layoutImagesTab) refresh() {
	t.titleBox.SetText(t.state.File.Layout.Title)
	t.list.Refresh()
	t.rebuildEditor()
}

func (t *layoutImagesTab) addImage() {
	used := map[string]bool{}
	for _, img := range t.images() {
		used[img.Source()] = true
	}
	file := ""
	for _, candidate := range t.state.ChapterImages() {
		if !used[candidate] {
			file = candidate
			break
		}
	}
	t.state.File.Layout.Images = append(t.state.File.Layout.Images, layoutplan.ImageInstruction{
		File:      file,
		Placement: layoutplan.PlacementInline,
	})
	t.onChange()
	t.list.Refresh()
	t.list.Select(len(t.images()) - 1)
}

func (t *layoutImagesTab) removeSelected() {
	id := t.selected
	if id < 0 || id >= len(t.images()) {
		return
	}
	t.state.File.Layout.Images = append(t.images()[:id], t.images()[id+1:]...)
	t.list.UnselectAll()
	t.selected = -1
	t.onChange()
	t.list.Refresh()
	t.rebuildEditor()
}

func (t *layoutImagesTab) moveSelected(delta int) {
	id := t.selected
	target := id + delta
	if id < 0 || target < 0 || target >= len(t.images()) {
		return
	}
	images := t.images()
	images[id], images[target] = images[target], images[id]
	t.onChange()
	t.list.Refresh()
	t.list.Select(target)
}

func (t *layoutImagesTab) rebuildEditor() {
	t.editor.Objects = nil
	if t.selected < 0 || t.selected >= len(t.images()) {
		t.editor.Add(widget.NewLabel("Select an image to edit its placement, or add one."))
		t.editor.Refresh()
		return
	}
	inst := &t.state.File.Layout.Images[t.selected]
	changed := func() {
		t.onChange()
		t.list.RefreshItem(t.selected)
	}

	// Callbacks are attached after seeding so populating the editor does not mark the book dirty.
	options := t.state.ChapterImages()
	if inst.Source() != "" && !contains(options, inst.Source()) {
		options = append(options, inst.Source())
	}
	fileSelect := widget.NewSelect(options, nil)
	fileSelect.SetSelected(inst.Source())
	fileSelect.OnChanged = func(s string) {
		inst.File = s
		inst.Src = ""
		changed()
	}

	preview := canvas.NewImageFromImage(t.thumbs.Get(filepath.Join(t.state.BookDir, inst.Source())))
	preview.FillMode = canvas.ImageFillContain
	preview.SetMinSize(fyne.NewSize(240, 180))

	items := append([]*widget.FormItem{widget.NewFormItem("File", fileSelect)}, instructionFormItems(inst, changed)...)
	t.editor.Add(preview)
	t.editor.Add(widget.NewForm(items...))
	t.editor.Add(widget.NewLabel(fmt.Sprintf("Image %d of %d", t.selected+1, len(t.images()))))
	t.editor.Refresh()
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
