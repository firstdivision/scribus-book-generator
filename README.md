# scribus-book-generator

## Overview

`scribus-book-generator` turns a structured book directory into an editable Scribus `.sla` document and a PDF preview.

The generator currently:

- loads a book template from YAML
- reads chapter folders from `books/<book>/chapters/`
- parses chapter markdown
- styles chapter titles from the first Markdown H1 in each chapter
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

### Chapter Markdown

The first Markdown H1 in a chapter file is its chapter title. The H1 marker is not included in the generated document, and following paragraphs continue as body text.

```markdown
# The Road to San Rosario

There is a point, about forty miles after the last gas station...
```

A chapter without an H1 uses `Untitled Chapter` as its title. Additional H1 headings in the same file are rejected with a validation error; they are not treated as additional chapter titles.

Example `book.yaml`:

```yaml
template: a4-landscape.yaml
```

## Running The Generator

```bash
go run ./cmd/bookgen books/sample-book/
```

Use `-v` to print the fully resolved configuration and the chapter inventory after the book directory is loaded.

The command loads and validates the book folder (`book.yaml`, `layout.json`, chapter markdown, and image paths), writes a Scribus job JSON under `books/<book>/out/scribus-job.json`, and runs the committed adapter in `scripts/scribus_generate.py`. It then writes `.sla` and `.pdf` files under `books/<book>/out/`. The file stem is the optional `title` in `layout.json`, or the book directory name if `title` is omitted.

```bash
go run ./cmd/bookgen -v books/sample-book/
```

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

chapter_headings:
	font:
		family: Source Serif 4
		style: Semibold
		size_pt: 28

	color_rgb: [40, 40, 40]
	alignment: left

	spacing_mm:
		top: 20
		bottom: 10

images:
	border:
		color_rgb: [255, 255, 255]
		width_pt: 3

	spacing_mm:
		top: 5
		bottom: 5
		inside: 5
		outside: 5

	sizing:
		max_width_mm: 110
		max_height_mm: 100

	placement:
		snap_to_edge: true
		snap_target: content_area
		allowed_edges:
			- outside
			- inside
			- top
			- bottom
		preferred_edges:
			- outside
			- top
		edge_gap_mm: 0

	leftovers:
		gallery_columns: 2

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

### `chapter_headings`

Controls the reusable Scribus paragraph style applied to the first H1 in each chapter file. Chapter-heading styling is separate from body text, captions, page numbers, and other text styles.

#### `chapter_headings.font`

- `family`: non-empty font family name
- `style`: non-empty font style name
- `size_pt`: font size in points, must be `> 0`

The renderer combines `family` and `style` into an exact Scribus font name, such as `Source Serif 4 Semibold`. Generation fails with an error if that font is not available in Scribus; the renderer does not silently substitute another font.

#### `chapter_headings.color_rgb`

- Chapter-title text color as `[r, g, b]`
- Must contain exactly three integers from `0` through `255`
- A named Scribus color is created once and reused by the chapter-heading style

#### `chapter_headings.alignment`

Valid values:

- `left`
- `center`
- `right`
- `inside`
- `outside`

`inside` and `outside` resolve from the chapter-opening page side:

- left page: `outside=left`, `inside=right`
- right page: `inside=left`, `outside=right`

#### `chapter_headings.spacing_mm`

- `top`: vertical space before the chapter title, in millimeters; must be `>= 0`
- `bottom`: vertical space between the title and first body paragraph, in millimeters; must be `>= 0`

The renderer applies these values through text-frame geometry rather than inserting blank lines.

### `images`

Controls image frame styling and the text wrap spacing around image frames.

#### `images.border`

- `color_rgb`: `[r, g, b]`
- `width_pt`: border width in points

#### `images.spacing_mm`

- `top`, `bottom`, `inside`, `outside`: image-to-text spacing in millimeters

#### `images.sizing`

- `max_width_mm`: default maximum inline-image width (`> 0`)
- `max_height_mm`: default maximum inline-image height (`> 0`)

Sizing is contain-fit by default and always preserves source aspect ratio.

#### `images.placement`

- `snap_to_edge`: `true` or `false`
- `snap_target`: `content_area`, `trim`, or `bleed`
- `allowed_edges`: list of `outside`, `inside`, `top`, `bottom`
- `preferred_edges`: ordered subset of `allowed_edges`
- `edge_gap_mm`: inward gap from selected snap edge (`>= 0`)

For facing pages:

- left page: `outside=left`, `inside=right`
- right page: `outside=right`, `inside=left`

Text wrap spacing remains separate from edge snap.

#### `images.leftovers`

- `gallery_columns`: columns in the end-of-chapter leftover gallery (`>= 1`, default `2`)

Gallery pages fill the content area (margins), not bleed. Spacing between cells comes from `images.spacing_mm`.

### `layout.json`

The optional top-level `title` names the generated `.sla` and `.pdf` files. If it is empty or omitted, the book directory name is used.

#### Image overrides

Book-level defaults come from template YAML, but individual images in `layout.json` can override:

- `snap_edge`
- `width_mm`
- `height_mm`
- `placement` (`inline`, `full_page`, or `ignore`)
- `bleed`

`placement: ignore` keeps the file on disk and valid in `layout.json`, but the generator does not place it. Ignore wins over `bleed` and size overrides.

```json
{ "file": "chapters/1-the-road/outtake.png", "placement": "ignore" }
```

Precedence is:

1. explicit `layout.json` instruction
2. YAML image defaults
3. built-in defaults

If both `width_mm` and `height_mm` are set for an image, they are treated as a contain-fit bounding box (still preserving aspect ratio).

#### Leftover images

In-flow images are placed one per page only while body text still overflows. After the text chain fits (or images run out):

- leftover images with `placement: full_page` or `bleed: true` each get a dedicated page, in leftover order, before the gallery
- remaining leftovers pack into an end-of-chapter gallery (page role `chapter_gallery`)

If every leftover is full-page, there is no gallery. The last gallery page may have a short row; cells are not stretched to fill the page.

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
- `chapter_gallery`
- `blank`

Notes:

- Hidden page numbers still participate in the numbering sequence.
- The generator uses `chapter_opening`, `body`, `full_page_image`, `chapter_gallery`, and `blank` roles.
- `chapter_gallery` is assigned to leftover gallery pages so `hide_on` can target them.

## Defaults

If `chapter_headings` is omitted entirely, the generator uses these defaults:

```yaml
chapter_headings:
	font:
		family: Source Serif 4
		style: Semibold
		size_pt: 28
	color_rgb: [40, 40, 40]
	alignment: left
	spacing_mm:
		top: 20
		bottom: 10
```

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
- configurable chapter-title typography, color, alignment, and spacing
- flowing body text
- chapter-local image placement
- page backgrounds
- page numbering

Planned extensions such as richer semantic page roles, `full_page_image`, and more advanced layout planning can be added on top of the existing configuration and rendering structure.