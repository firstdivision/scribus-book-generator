# Book layout storage

Each book stores its template selection and optional layout in `book.yaml`.
The `layout` mapping contains `title` and `images`, with the same fields and
validation previously used by the standalone JSON layout. Missing or null layout
uses the default empty plan. Legacy `layout.json` files are not loaded.

The layout remains the contract between planning and deterministic generation.
Go decodes and validates it into `layoutplan.Plan`, checks referenced image files,
and serializes the existing JSON shape into `out/scribus-job.json`. The Scribus
adapter consumes that generated job; it does not parse book YAML or plan layouts.

The existing Sample Book and Puerto Rico layouts were migrated into their
respective book YAML files without changing titles, image order, or overrides.
