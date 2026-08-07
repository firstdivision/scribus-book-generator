# scribus-book-generator

## Overview

`scribus-book-generator` turns a structured book directory into an editable Scribus `.sla` document and a PDF preview.

The generator currently:

- loads a book template from YAML
- reads chapter folders from `books/<book>/chapters/`
- parses chapter markdown
- finds chapter-local images
- creates flowing text frames across multiple pages
- places chapter images with configurable frame styling and spacing
- applies per-page background color
- renders configurable page numbers
- writes an editable Scribus document plus a PDF export

The intended workflow is:

1. Create a book directory with `book.yaml` and chapter folders.
2. Point `book.yaml` at a template file.
3. Run `go run ./cmd/bookgen books/<book>/`.
4. Open the generated `.sla` in Scribus for manual review and final production work.

## Book Structure

The generator expects a structure like this:

```text
books/
	sample-book/
		book.yaml
		chapters/
			1-chapter-name/
				text.md
				image-1.png
				image-2.jpg
			2-another-chapter/
				chapter.md
				photo.png
```

Rules:

- `book.yaml` selects the template.
- Each chapter lives in its own folder.
- Each chapter folder must contain at least one `.md` file.
- Images are discovered from the same chapter folder.

Example `book.yaml`:

```yaml
template: a4-landscape.yaml
```

## Running The Generator

```bash
go run ./cmd/bookgen books/sample-book/
```

The command writes output under `books/sample-book/out/`.

## Creating A Template

Templates are YAML files stored under `templates/`. A book selects one template by filename.

Example:

```yaml
document:
	units: mm
	layout: facing_pages
	first_page: right

page:
	size: A4
	orientation: landscape
	background_color_rgb: [248, 244, 232]

bleed:
	top: 3.18
	bottom: 3.18
	inside: 3.18
	outside: 3.18

safety_margin:
	top: 12.7
	bottom: 12.7
	inside: 12.7
	outside: 12.7

images:
	border:
		color_rgb: [255, 255, 255]
		width_pt: 3

	spacing_mm:
		top: 5
		bottom: 5
		inside: 5
		outside: 5

page_numbers:
	enabled: true
	start_on_page: 1
	start_number: 1
	format: arabic
	position: bottom_outside

	font:
		family: Source Serif 4
		style: Regular
		size_pt: 9

	color_rgb: [80, 80, 80]

	offset_mm:
		top: 7
		bottom: 7
		inside: 10
		outside: 10

	hide_on:
		- chapter_opening
		- full_page_image
		- blank
```

## Template Fields

### `document`

- `units`: document units.
	Valid values:
	`mm`
- `layout`: page layout mode.
	Valid values:
	`single_page`, `facing_pages`
- `first_page`: semantic side of the first page in facing-page layout.
	Valid values:
	`right`

### `page`

- `size`: named page size.
	Valid values:
	`A4`, `LETTER`
- `orientation`: page orientation.
	Valid values:
	`portrait`, `landscape`
- `width_mm`, `height_mm`: optional explicit page size override in millimeters.
- `background_color_rgb`: page background color as `[r, g, b]` or `null`.

Example:

```yaml
page:
	size: A4
	orientation: landscape
	background_color_rgb: null
```

### `bleed`

- `top`, `bottom`, `inside`, `outside`: bleed in millimeters.

### `safety_margin`

- `top`, `bottom`, `inside`, `outside`: text-safe margins in millimeters.

### `images`

Controls image frame styling and the text wrap spacing around image frames.

#### `images.border`

- `color_rgb`: `[r, g, b]`
- `width_pt`: border width in points

#### `images.spacing_mm`

- `top`, `bottom`, `inside`, `outside`: image-to-text spacing in millimeters

### `page_numbers`

Controls logical numbering, formatting, position, styling, and when page numbers are hidden.

#### `page_numbers.enabled`

- `true` or `false`

#### `page_numbers.start_on_page`

- Physical Scribus page on which numbering begins.
- Must be `>= 1`.

#### `page_numbers.start_number`

- Displayed number on `start_on_page`.
- Must be `>= 1`.

#### `page_numbers.format`

Valid values:

- `arabic`
- `roman_lower`
- `roman_upper`

#### `page_numbers.position`

Valid values:

- `bottom_outside`
- `bottom_inside`
- `bottom_center`
- `top_outside`
- `top_inside`
- `top_center`

These are semantic positions. In facing-page layout, `inside` and `outside` are resolved from left/right page side automatically.

#### `page_numbers.font`

- `family`: font family name, non-empty
- `style`: font style name, non-empty
- `size_pt`: font size in points, must be `> 0`

The current Scribus renderer combines `family` and `style` into a Scribus font name such as `Source Serif 4 Regular`.

#### `page_numbers.color_rgb`

- `[r, g, b]`
- Must contain exactly three integers from `0` through `255`

#### `page_numbers.offset_mm`

- `top`, `bottom`, `inside`, `outside`
- All values are millimeters
- All values must be non-negative

#### `page_numbers.hide_on`

Valid values:

- `body`
- `chapter_opening`
- `full_page_image`
- `blank`

Notes:

- Hidden page numbers still participate in the numbering sequence.
- The current generator actively uses `chapter_opening`, `body`, and `blank` roles.
- `full_page_image` is recognized and validated now so templates can use it, and it is reserved for fuller layout support.

## Defaults

If `page_numbers` is omitted entirely, the generator uses these defaults:

```yaml
page_numbers:
	enabled: false
	start_on_page: 1
	start_number: 1
	format: arabic
	position: bottom_outside
	font:
		family: Source Serif 4
		style: Regular
		size_pt: 9
	color_rgb: [80, 80, 80]
	offset_mm:
		top: 7
		bottom: 7
		inside: 10
		outside: 10
	hide_on: []
```

## Current Scope

The current generator is deterministic and template-driven. It is focused on:

- chapter opening pages
- flowing body text
- chapter-local image placement
- page backgrounds
- page numbering

Planned extensions such as richer semantic page roles, `full_page_image`, and more advanced layout planning can be added on top of the existing configuration and rendering structure.