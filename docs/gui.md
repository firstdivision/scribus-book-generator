# Desktop GUI

`cmd/bookgen-gui` (build tag `gui`) is a Fyne desktop editor for book
directories. It is a client of the existing packages, not a new pipeline:

- `internal/book` reads and writes `book.yaml` (`book.File`), creates books
  and chapters, and imports images.
- `internal/config` resolves defaults → template → `overrides` and lists the
  templates under the project root.
- `internal/renderer` and `internal/images` run the same Scribus/HEIC steps as
  `bookgen`, with `Options` that redirect output into the GUI log and locate
  `scripts/` from an explicit project root instead of the working directory.

The GUI never constructs Scribus documents or layout decisions itself; it only
edits the `book.yaml` contract and calls the deterministic generator.

## Overrides

`book.yaml` gained an `overrides` block whose schema is identical to a template
file. To make "override with zero" expressible, every template leaf became a
pointer (`config.TemplateFile`), and merging is now "present wins" rather than
"non-zero wins". Templates that previously wrote `0` to mean "unset" now mean
`0`, which matches the built-in default in every such field.

## Package layout

- `internal/gui/state.go` — `State`: open book, template list, dirty flag,
  inherited (`TemplateConfig`) and effective configuration, pending chapter
  text (written on `Save`), and upsert/remove helpers for `layout.images`.
- `internal/gui/override.go` — one reusable "Override" row per leaf type
  (float, int, string, enum, bool, RGB, string list) that writes a pointer or
  clears it.
- `internal/gui/overrides.go` — the five override tabs built from those rows.
- `internal/gui/instruction_form.go` — the per-image `layout.images` fields,
  shared by the layout images tab and the chapter editor.
- `internal/gui/layout_images.go` — `layout.title` and the ordered
  `layout.images` list with thumbnails.
- `internal/gui/chapters.go` — the chapter list (first tab). Rows are a
  custom widget implementing both `Tappable` and `DoubleTappable`, because the
  Fyne driver dispatches to whichever the topmost object implements; a
  double-tap swaps the list for the editor inside a `Stack`.
- `internal/gui/chapter_editor.go` — Markdown text area plus a `GridWrap` of
  every image in the chapter directory. Images not in `book.yaml` show
  "template defaults"; the first field change upserts an entry.
- `internal/gui/generate.go`, `window.go` — generation with streamed log, and
  the main window.

Everything in `internal/gui` is tested headlessly with `fyne.io/fyne/v2/test`;
only `cmd/bookgen-gui` needs the cgo/X11 toolchain.
