# scribus-book-generator

## Table of Contents

- [Overview](#overview)
- [Book Structure](#book-structure)
  - [Chapter Markdown](#chapter-markdown)
- [Running The Generator](#running-the-generator)
  - [Converting HEIC Images](#converting-heic-images)
- [Creating A Template](#creating-a-template)
- [Template Fields](#template-fields)
  - [document](#document)
  - [page](#page)
  - [bleed](#bleed)
  - [safety_margin](#safety_margin)
  - [chapter_headings](#chapter_headings)
  - [images](#images)
  - [book.yaml layout](#bookyaml-layout)
  - [page_numbers](#page_numbers)
- [Defaults](#defaults)
- [Current Scope](#current-scope)

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

The command loads and validates the book folder (`book.yaml`, chapter markdown, and image paths), writes a Scribus job JSON under `books/<book>/out/scribus-job.json`, and runs the committed adapter in `scripts/scribus_generate.py`. It then writes `.sla` and `.pdf` files under `books/<book>/out/`. The file stem is the optional `layout.title` in `book.yaml`, or the book directory name if `title` is omitted.

```bash
go run ./cmd/bookgen -v books/sample-book/
```

### Converting HEIC Images

Chapter images in HEIC/HEIF format are not placed by the generator. Pass `--convert-heic` to run `scripts/convert-heic.sh` over the book's `chapters/` tree before the book is loaded, so every `*.heic`/`*.heif` file gets a sibling `.jpg` that is picked up as a chapter image. The original HEIC files are left in place and existing `.jpg` files are not overwritten.

```bash
go run ./cmd/bookgen --convert-heic books/sample-book/
```

The script requires either `heif-convert` (Debian/Ubuntu package `libheif-examples`) or ImageMagick (`magick`/`convert`) with HEIC support. It can also be run on its own:

```bash
./scripts/convert-heic.sh books/sample-book/ [--quality N] [--force]
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
	sorting: none

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
		edge_selection: preferred
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

#### `chapter_headings.background_color_rgb` and `chapter_headings.borders`

`background_color_rgb` fills the entire chapter-title text frame (the content width, not just the text). Supply exactly three integers from `0` through `255`; omitted or `null` means no fill.

`borders` accepts physical `top`, `bottom`, `left`, and `right` sides independently. Each side accepts `color_rgb` (exactly three integers from `0` through `255`, defaults to the heading text color) and `width_pt` (finite and `>= 0`, defaults to `1`). Omitted sides, `null` sides, and zero widths draw no border. Unknown sides are rejected.

```yaml
chapter_headings:
  background_color_rgb: [240, 230, 220]
  borders:
    bottom: {color_rgb: [40, 40, 40], width_pt: 2}
    left: {color_rgb: [120, 80, 40], width_pt: 1}
    top: {width_pt: 1}
    right: {width_pt: 0}
```

Borders are solid, separately editable rectangles inside the title-frame edges, drawn in top, bottom, left, right order. Their thickness is capped at the frame dimension. They do not add text padding or change heading spacing; thick borders can overlap title text. Backgrounds and borders are disabled by default.

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

#### `images.sorting`

- `none` (default): chapter images are ordered by file extension group, then file name
- `date-ascending`: oldest capture date first
- `date-descending`: newest capture date first

Date sorting reads the EXIF capture timestamp (`DateTimeOriginal`, falling back to `DateTimeDigitized` then `DateTime`) from JPEG files. It is best effort: files with no readable camera timestamp — including formats without EXIF, or files whose metadata was stripped — are moved to the end of the chapter and ordered by file name.

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
- `edge_selection`: `preferred` (default) chooses the first preferred edge; `random` chooses randomly from the preferred allowed edges
- `edge_gap_mm`: inward gap from selected snap edge (`>= 0`)

For facing pages:

- left page: `outside=left`, `inside=right`
- right page: `outside=right`, `inside=left`

Text wrap spacing remains separate from edge snap.

#### `images.leftovers`

- `gallery_columns`: maximum columns in the end-of-chapter leftover gallery (`>= 1`, default `2`)

Gallery pages fill the content area (margins), not bleed. Spacing between cells comes from `images.spacing_mm`. When a page holds fewer images than `gallery_columns`, the grid collapses to the image count so the row stays centered with equal blank space on all sides; a lone image is promoted to a dedicated full-page (no bleed) page instead of a gallery cell unless explicitly assigned `placement: gallery`. A single row that can form a more square grid is converted automatically (4 → 2x2, 6 → 2x3, 9 → 3x3) whenever the square grid does not shrink the cells.

### `book.yaml` layout

The optional `layout.title` names the generated `.sla` and `.pdf` files. If it is empty or omitted, the book directory name is used.

Store the layout alongside the template selection in each book's `book.yaml`:

```yaml
template: a4-landscape.yaml
layout:
  title: My Book
  images:
    - file: chapters/1-intro/photo.jpg
      placement: full_page
      bleed: true
      border:
        width_pt: 0
```

The `layout` section is optional; omitted or `null` means no image overrides. Separate `layout.json` files are no longer read. To migrate another book, copy its JSON object's `title` and `images` into `layout` in `book.yaml`, then remove the old file. Generated `out/scribus-job.json` remains an internal renderer artifact.

#### Image overrides

Each entry requires a non-empty `file` (or `src`); paths are relative to the book directory, and referenced files must exist. `file` takes precedence over `src`. Dimensions must be greater than zero. `snap_edge` accepts `outside`, `inside`, `top`, or `bottom` and must be allowed by the template when used. Optional `border.color_rgb` requires three integers from 0 through 255; `border.width_pt` must be nonnegative, with zero disabling the border.

Book-level defaults come from template YAML, but individual images in `book.yaml` under `layout.images` can override:

- `snap_edge`
- `width_mm`
- `height_mm`
- `placement` (`inline`, `full_page`, `ignore`, or `gallery`)
- `bleed`

`bleed: true` proportionally cover-fills the page and its top, bottom, and outside bleed, cropping excess image content evenly within the frame. On facing pages, the image frame stops at the trim edge on the inside so it ends exactly at the spine and cannot extend onto the facing page. Without `bleed`, full-page images are contain-fit inside the margins.

`placement: ignore` keeps the file on disk and valid in `layout.images`, but the generator does not place it. Ignore wins over `bleed` and size overrides.

```yaml
- file: chapters/1-the-road/outtake.png
  placement: ignore
```

Precedence is:

1. explicit `book.yaml` layout instruction
2. YAML image defaults
3. built-in defaults

If both `width_mm` and `height_mm` are set for an image, they are treated as a contain-fit bounding box (still preserving aspect ratio).

#### Force an image into the leftovers gallery

Set `placement: gallery` on an image in `book.yaml`:

```yaml
layout:
  images:
    - file: chapters/1-intro/photo.jpg
      placement: gallery
```

The image skips all body-text pages and joins the end-of-chapter leftovers gallery, preserving the configured image order among gallery images. Even a single explicitly assigned gallery image stays on a gallery page. Gallery placement takes precedence over `bleed: true`; gallery cells use contain-fit inside the margins, with the template's gallery columns and spacing. Per-image borders still apply; inline dimensions and snap edges do not apply to gallery cells. Omitted placement retains the existing automatic behavior.

#### Leftover images

In-flow images are placed one per page only while body text still overflows. After the text chain fits (or images run out):

- leftover images with `placement: full_page` or `bleed: true` each get a dedicated page, in leftover order, before the gallery
- a single remaining leftover without `placement: gallery` is treated as `full_page` without bleed (contain-fit and centered inside the margins)
- remaining leftovers pack into an end-of-chapter gallery (page role `chapter_gallery`)

If every leftover is full-page, there is no gallery. Gallery pages center the occupied cells vertically and horizontally: the last page may hold a short row, and the grid adapts its column count so the row is centered rather than stretched. Single-row results that factor into a square-like grid (2x2, 2x3, 3x3, …) are converted automatically; on height-constrained pages the single row is kept when it would yield larger cells.

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
