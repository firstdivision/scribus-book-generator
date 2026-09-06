package renderer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func committedScriptPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "scripts", "scribus_generate.py")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("committed Scribus script not found: %v", err)
	}
	return path
}

func TestBuildScribusInvocation(t *testing.T) {
	invocation := buildScribusInvocation("/tmp/book", "/tmp/book/out/scribus-job.json")
	if len(invocation) != 8 {
		t.Fatalf("expected 8 parts, got %d", len(invocation))
	}
	if invocation[0] != "xvfb-run" || invocation[2] != "scribus" || invocation[4] != "-py" {
		t.Fatalf("unexpected invocation prefix: %#v", invocation[:5])
	}
	if invocation[5] != scribusScriptPath {
		t.Fatalf("expected script path, got %q", invocation[5])
	}
	if invocation[6] != "/tmp/book" {
		t.Fatalf("expected book dir argument, got %q", invocation[6])
	}
	if invocation[7] != "/tmp/book/out/scribus-job.json" {
		t.Fatalf("expected job path argument, got %q", invocation[7])
	}
}

func TestCommittedScribusScriptContainsRendererHelpers(t *testing.T) {
	content, err := os.ReadFile(committedScriptPath(t))
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	text := string(content)
	for _, fragment := range []string{
		"def _new_document_compat",
		"def _start_chapter_on_right_page_compat",
		"def _create_background_master_compat",
		"def _apply_master_page_compat",
		"def _render_page_numbers_compat",
		"def _logical_page_number_compat",
		"def _page_number_placement_compat",
		"scribus.newDocument(paper_size, margins, orientation, first_page_number, unit_points, page_type, first_page_order, num_pages)",
		"saveDocAs",
		"def _render_basic_content",
		"def _placeable_images",
		"def _gallery_cell_rects",
		"def _place_gallery_pages",
		"def _image_dimensions_compat",
		"def _apply_image_frame_style_compat",
		"def _set_text_distances_sides_compat",
		"createText",
		"chapter_images = _image_files(chapter_dir)",
		"placeable_images = _placeable_images(image_paths, layout_index, book_dir)",
		"_set_frame_text_compat(scribus, body_frame, body_text)",
		"while _text_overflows_compat(scribus, body_frames[-1]) and in_flow_index < len(placeable_images):",
		"continuation_frame = _create_text_frame_compat(",
		"_link_text_frames_compat(scribus, body_frames[-1], continuation_frame)",
		"leftover_full_page = []",
		"gallery_images = []",
		"def _gallery_effective_columns",
		"columns = _gallery_effective_columns(image_count, configured_columns)",
		"def _gallery_balanced_columns",
		"def _gallery_fitted_geometry",
		"def _place_full_page_image",
		"single_gallery_page = len(gallery_images) == 1",
		"current_page, placed_count = _place_gallery_pages(",
		"def _ensure_chapter_heading_styles_compat",
		"Configured chapter heading font '{font_name}' is not available in Scribus",
		"def _resolve_chapter_heading_alignment",
		"title_top = margin_top + chapter_heading_spacing_top",
		"if is_full_page:\n\t\tpage_roles[page_number] = \"full_page_image\"",
		"current_page = _append_body_page_compat(scribus, current_page, \"chapter_gallery\"",
		"scribus.createMasterPage(master_page_name)",
		"scribus.applyMasterPage(master_page_name, page_number)",
		"def _choose_snap_edge",
		"def _fit_contain_dimensions",
		"def _snap_frame_to_edge(snap_rect, frame_width, frame_height, physical_edge, edge_gap, is_right_page, spacing_left, spacing_right, spacing_top, spacing_bottom):",
		"available_width = max(1.0, snap_rect[2] - image_spacing_left - image_spacing_right)",
		"available_height = max(1.0, snap_rect[3] - image_spacing_top_used - image_spacing_bottom_used)",
		"def _create_wrap_frame_compat",
		"createImage",
		"loadImage",
		"def _set_text_flow_mode_compat",
		"setTextFlowMode",
		"def _append_page_compat",
		"linkTextFrames",
		"def _estimate_body_pages",
		"def _resolve_border_override",
		"border_override = image_instruction.get(\"border\")",
		"def _load_job",
		"job = _load_job(job_path)",
		"gallery_columns = images.get(\"gallery_columns\") or 2",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("committed script missing %q", fragment)
		}
	}
	setTextIdx := strings.Index(text, `_set_frame_text_compat(scribus, body_frame, body_text)`)
	inFlowIdx := strings.Index(text, "while _text_overflows_compat(scribus, body_frames[-1]) and in_flow_index < len(placeable_images):")
	overflowIdx := strings.Index(text, "max_extra_pages = 20")
	leftoverIdx := strings.Index(text, "leftover_full_page = []")
	galleryIdx := strings.Index(text, "current_page, placed_count = _place_gallery_pages(")
	if setTextIdx < 0 || inFlowIdx < setTextIdx {
		t.Fatal("body text must be set before in-flow images continue for overflow")
	}
	if overflowIdx < 0 || leftoverIdx < overflowIdx {
		t.Fatal("leftover full-page pages must be appended after the text overflow loop")
	}
	if galleryIdx < leftoverIdx {
		t.Fatal("gallery pages must come after leftover full-page pages")
	}
	if strings.Contains(text, "automatic_text_frames") {
		t.Fatal("script still references automatic_text_frames")
	}
}

func TestBuildScribusJobFromSampleBook(t *testing.T) {
	bookDir := filepath.Clean(filepath.Join("..", "..", "books", "sample-book"))
	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

	job := buildScribusJob(cfg, layoutplan.Plan{})
	if job.Page.SizeConstant != "PAPER_A4" {
		t.Fatalf("unexpected paper constant: %q", job.Page.SizeConstant)
	}
	if job.Page.Layout != "facing_pages" {
		t.Fatalf("unexpected layout: %q", job.Page.Layout)
	}
	if job.Page.BackgroundRGB == nil {
		t.Fatal("expected page background rgb")
	}
	if !job.PageNumbers.Enabled {
		t.Fatal("expected page numbers enabled")
	}
	if job.PageNumbers.Position != "bottom_outside" {
		t.Fatalf("unexpected page number position: %q", job.PageNumbers.Position)
	}
	if job.PageNumbers.FontName != "Source Serif 4 Regular" {
		t.Fatalf("unexpected page number font: %q", job.PageNumbers.FontName)
	}
	if job.ChapterHeadings.FontName != "URW Bookman Demi" {
		t.Fatalf("unexpected chapter heading font: %q", job.ChapterHeadings.FontName)
	}
	if job.ChapterHeadings.FontSizePt != 28 {
		t.Fatalf("unexpected heading size: %v", job.ChapterHeadings.FontSizePt)
	}
	if !job.Images.SnapToEdge {
		t.Fatal("expected snap_to_edge")
	}
	if job.Images.GalleryColumns != cfg.Images.Leftovers.GalleryColumns {
		t.Fatalf("expected job gallery_columns to mirror config value %d, got %d", cfg.Images.Leftovers.GalleryColumns, job.Images.GalleryColumns)
	}

	encoded, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if !strings.Contains(string(encoded), `"hide_on":["chapter_opening","full_page_image","blank"]`) &&
		!strings.Contains(string(encoded), `"hide_on":["chapter_opening", "full_page_image", "blank"]`) {
		t.Fatalf("job missing hide_on: %s", encoded)
	}
}

func TestWriteScribusJobNilPageBackground(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scribus-job.json")
	if err := writeScribusJob(path, config.Default(), layoutplan.Plan{}); err != nil {
		t.Fatalf("writeScribusJob returned error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}
	var job scribusJob
	if err := json.Unmarshal(content, &job); err != nil {
		t.Fatalf("unmarshal job: %v", err)
	}
	if job.Page.BackgroundRGB != nil {
		t.Fatalf("expected nil background, got %#v", job.Page.BackgroundRGB)
	}
	if job.PageNumbers.Enabled {
		t.Fatal("expected page numbers disabled in defaults")
	}
}

func TestWriteScribusJobIncludesLayoutBorderOverride(t *testing.T) {
	width := 0.0
	plan := layoutplan.Plan{
		Images: []layoutplan.ImageInstruction{
			{
				File:      "chapters/2-the-people-who-stayed/sunset-gathering-at-hotel-rosario.png",
				Placement: layoutplan.PlacementFullPage,
				Border: &layoutplan.Border{
					WidthPt: &width,
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "scribus-job.json")
	if err := writeScribusJob(path, config.Default(), plan); err != nil {
		t.Fatalf("writeScribusJob returned error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}
	if !strings.Contains(string(content), `"width_pt": 0`) && !strings.Contains(string(content), `"width_pt":0`) {
		t.Fatalf("job missing width_pt override: %s", content)
	}
}

func TestScribusScriptDiscoversUppercaseImageExtensions(t *testing.T) {
	chapterDir := filepath.Join(t.TempDir(), "chapters", "case-test")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("failed to create chapter dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "IMG_1234.JPG"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write uppercase image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "img_5678.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write lowercase image: %v", err)
	}

	cmd := exec.Command("python3", "-c", `import importlib.util, pathlib, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
results = [path.name for path in module._image_files(pathlib.Path(sys.argv[2]))]
print("\n".join(results))`, committedScriptPath(t), chapterDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute script helper: %v\n%s", err, output)
	}

	text := strings.TrimSpace(string(output))
	if !strings.Contains(text, "IMG_1234.JPG") {
		t.Fatalf("expected helper to discover uppercase image, got output %q", text)
	}
	if !strings.Contains(text, "img_5678.jpg") {
		t.Fatalf("expected helper to discover lowercase image, got output %q", text)
	}
}

func TestScribusScriptPlaceableImagesSkipsIgnore(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, json, pathlib, sys, tempfile
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
book_dir = pathlib.Path(tempfile.mkdtemp())
keep = book_dir / "chapters" / "1-the-road" / "keep.png"
skip = book_dir / "chapters" / "1-the-road" / "outtake.png"
keep.parent.mkdir(parents=True)
keep.write_bytes(b"x")
skip.write_bytes(b"x")
layout_index = {
    "chapters/1-the-road/outtake.png": {"file": "chapters/1-the-road/outtake.png", "placement": "ignore"},
    "chapters/1-the-road/keep.png": {"file": "chapters/1-the-road/keep.png", "placement": "inline"},
}
names = [path.name for path in module._placeable_images([keep, skip], layout_index, book_dir)]
print(json.dumps(names))`, committedScriptPath(t))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute placeable helper: %v\n%s", err, output)
	}
	var names []string
	if err := json.Unmarshal(output, &names); err != nil {
		t.Fatalf("decode helper output %q: %v", output, err)
	}
	if len(names) != 1 || names[0] != "keep.png" {
		t.Fatalf("expected ignore to drop outtake.png, got %#v", names)
	}
}

func TestScribusScriptGalleryCellRectsDoNotStretchShortRow(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
full = module._gallery_cell_rects(4, 2, 10, 20, 400, 300, 10, 10)
short = module._gallery_cell_rects(1, 2, 10, 20, 400, 300, 10, 10)
print(json.dumps({"full": full, "short": short}))`, committedScriptPath(t))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute gallery helper: %v\n%s", err, output)
	}
	var parsed struct {
		Full  [][]float64 `json:"full"`
		Short [][]float64 `json:"short"`
	}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("decode gallery output %q: %v", output, err)
	}
	if len(parsed.Full) != 4 || len(parsed.Short) != 1 {
		t.Fatalf("unexpected cell counts full=%d short=%d", len(parsed.Full), len(parsed.Short))
	}
	// A lone image gets one centered cell spanning the content width (height stays square-capped).
	if parsed.Short[0][0] != 10 || parsed.Short[0][1] != 20 {
		t.Fatalf("single-cell gallery must start at the content origin, got %v", parsed.Short[0])
	}
	if parsed.Short[0][2] != 400 || parsed.Short[0][3] != 300 {
		t.Fatalf("single-cell gallery must span the content area, got %v", parsed.Short[0])
	}
	if parsed.Full[0][2] != 195 {
		t.Fatalf("full-grid cell width should be 195, got %v", parsed.Full[0])
	}
	if parsed.Full[1][0] <= parsed.Full[0][0] || parsed.Full[1][1] != parsed.Full[0][1] {
		t.Fatalf("expected row-major second cell to the right, got first=%v second=%v", parsed.Full[0], parsed.Full[1])
	}
	if parsed.Full[2][1] <= parsed.Full[0][1] || parsed.Full[2][0] != parsed.Full[0][0] {
		t.Fatalf("expected third cell on the next row, got first=%v third=%v", parsed.Full[0], parsed.Full[2])
	}
}

func TestScribusScriptGalleryCellRectsCentered(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
four_in_four = module._gallery_cell_rects(4, 4, 10, 20, 400, 300, 10, 10)
two_in_four = module._gallery_cell_rects(2, 4, 10, 20, 400, 300, 10, 10)
print(json.dumps({"four": four_in_four, "two": two_in_four}))`, committedScriptPath(t))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute gallery helper: %v\n%s", err, output)
	}
	var parsed struct {
		Four [][]float64 `json:"four"`
		Two  [][]float64 `json:"two"`
	}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("decode gallery output %q: %v", output, err)
	}

	// 4 images in 4 columns: converted to a 2x2 square grid.
	// cell_width = (400 - 10) / 2 = 195; cell_height = min(195, (300 - 10) / 2) = 145.
	// total height 2*145 + 10 = 300 fills the content area -> rows at y=20 and y=175.
	wantFour := [][]float64{
		{10, 20, 195, 145},
		{215, 20, 195, 145},
		{10, 175, 195, 145},
		{215, 175, 195, 145},
	}
	if len(parsed.Four) != len(wantFour) {
		t.Fatalf("expected %d rects, got %d", len(wantFour), len(parsed.Four))
	}
	for i, want := range wantFour {
		got := parsed.Four[i]
		if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
			t.Fatalf("rect %d: expected %v, got %v", i, want, got)
		}
	}

	// 2 images in 4 columns: effective columns collapse to 2, giving equal-width
	// cells that fill the content width (centered by construction).
	// cell_width = (400 - 10) / 2 = 195 -> first x = 10, second x = 215.
	if len(parsed.Two) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(parsed.Two))
	}
	if parsed.Two[0][0] != 10.0 {
		t.Fatalf("expected first rect x=10.0 (content edge), got %f", parsed.Two[0][0])
	}
	if parsed.Two[1][0] != 215.0 {
		t.Fatalf("expected second rect x=215.0, got %f", parsed.Two[1][0])
	}
	if parsed.Two[0][1] != 72.5 || parsed.Two[1][1] != 72.5 {
		t.Fatalf("expected y=72.5 (vertically centered), got %f, %f", parsed.Two[0][1], parsed.Two[1][1])
	}
}

func TestScribusScriptGalleryEffectiveColumns(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
cases = [
    (1, 4),
    (2, 4),
    (3, 4),
    (4, 4),
    (5, 4),
    (5, 5),
    (6, 4),
    (7, 4),
    (9, 4),
    (6, 6),
    (8, 8),
    (9, 9),
    (12, 12),
]
print(json.dumps([module._gallery_effective_columns(count, columns) for count, columns in cases]))`, committedScriptPath(t))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute gallery helper: %v\n%s", err, output)
	}
	var got []int
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode gallery output %q: %v", output, err)
	}
	want := []int{1, 2, 3, 2, 4, 5, 3, 4, 3, 3, 4, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("expected %d results, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("case %d: expected %d effective columns, got %d", i, want[i], got[i])
		}
	}
}

func TestScribusScriptGallerySquareConversionSkipsShortPages(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
print(json.dumps(module._gallery_cell_rects(9, 9, 0, 0, 400, 60, 10, 10)))`, committedScriptPath(t))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute gallery helper: %v\n%s", err, output)
	}
	var rects [][]float64
	if err := json.Unmarshal(output, &rects); err != nil {
		t.Fatalf("decode gallery output %q: %v", output, err)
	}

	// A 3x3 grid would shrink cells to (60 - 20) / 3 = 13.33 high, so the
	// height-constrained page must keep the single row of 9 instead.
	if len(rects) != 9 {
		t.Fatalf("expected 9 rects, got %d", len(rects))
	}
	wantCell := 320.0 / 9.0
	for i, r := range rects {
		if r[1] != rects[0][1] {
			t.Fatalf("rect %d: expected a single row, got y=%f vs first y=%f", i, r[1], rects[0][1])
		}
		if r[2] != wantCell || r[3] != wantCell {
			t.Fatalf("rect %d: expected single-row cell %f, got %v", i, wantCell, r)
		}
	}
}

func TestScribusScriptSingleLeftoverImageGetsFullPage(t *testing.T) {
	content, err := os.ReadFile(committedScriptPath(t))
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	text := string(content)
	singleIdx := strings.Index(text, "single_gallery_page = len(gallery_images) == 1")
	fullPageIdx := strings.Index(text, "\t\t_place_full_page_image(")
	galleryIdx := strings.Index(text, "current_page, placed_count = _place_gallery_pages(")
	if singleIdx < 0 || fullPageIdx < 0 || galleryIdx < 0 {
		t.Fatal("script must route a single leftover image to a dedicated full-page placement")
	}
	if !(singleIdx < fullPageIdx && fullPageIdx < galleryIdx) {
		t.Fatal("single leftover full-page placement must run before the gallery placement")
	}
}

func TestScribusScriptParsesChapterHeading(t *testing.T) {
	scriptPath := committedScriptPath(t)
	chapterPath := filepath.Join(t.TempDir(), "chapter.md")
	content := "# The Road to San Rosario\n\nFirst body paragraph.\n\nSecond body paragraph.\n"
	if err := os.WriteFile(chapterPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write chapter: %v", err)
	}

	cmd := exec.Command("python3", "-c", `import importlib.util, json, pathlib, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
print(json.dumps(module._parse_chapter_markdown(pathlib.Path(sys.argv[2]))))`, scriptPath, chapterPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute Markdown parser: %v\n%s", err, output)
	}

	var parsed []string
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("decode parser output %q: %v", output, err)
	}
	if len(parsed) != 2 || parsed[0] != "The Road to San Rosario" {
		t.Fatalf("unexpected parsed chapter: %#v", parsed)
	}
	if parsed[1] != "First body paragraph.\n\nSecond body paragraph." {
		t.Fatalf("unexpected parsed body: %q", parsed[1])
	}

	if err := os.WriteFile(chapterPath, []byte("# First\n\nBody.\n\n# Second\n"), 0o644); err != nil {
		t.Fatalf("failed to rewrite chapter: %v", err)
	}
	cmd = exec.Command("python3", "-c", `import importlib.util, pathlib, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
module._parse_chapter_markdown(pathlib.Path(sys.argv[2]))`, scriptPath, chapterPath)
	output, err = cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "additional H1 heading on line 5") {
		t.Fatalf("expected useful additional-H1 error, got err=%v output=%q", err, output)
	}
}

func TestOutputPathsUsesLayoutTitle(t *testing.T) {
	plan := layoutplan.Plan{Title: "The Roast to San Rosario"}
	result := outputPaths("books/sample-book", plan)
	if result.SLAPath != "books/sample-book/out/The Roast to San Rosario.sla" {
		t.Fatalf("unexpected SLA path: %q", result.SLAPath)
	}
	if result.PDFPath != "books/sample-book/out/The Roast to San Rosario.pdf" {
		t.Fatalf("unexpected PDF path: %q", result.PDFPath)
	}
}

func TestOutputPathsFallsBackToBookDirectoryName(t *testing.T) {
	result := outputPaths("/tmp/my-book", layoutplan.Plan{})
	if result.SLAPath != "/tmp/my-book/out/my-book.sla" {
		t.Fatalf("unexpected SLA path: %q", result.SLAPath)
	}
}

func TestGenerateRequiresBookDirectory(t *testing.T) {
	if _, err := Generate(""); err == nil {
		t.Fatal("expected error for empty book directory")
	}
}

func TestScribusScriptForcesGalleryPlacement(t *testing.T) {
	cmd := exec.Command("python3", "-c", `import importlib.util, inspect, pathlib, sys
spec = importlib.util.spec_from_file_location("renderer", sys.argv[1])
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
for name in ("_decorate_chapter_heading", "_set_frame_text_compat", "_set_paragraph_style_compat", "_goto_page_compat", "_link_text_frames_compat"):
    setattr(m, name, lambda *a, **k: None)
m._document_page_size_compat = lambda *a: (600, 800)
m._create_text_frame_compat = lambda *a: a[-1]
m._append_body_page_compat = lambda *a: a[1] + 1
root = pathlib.Path("/book")
forced, inline, auto, ignored = [root / name for name in ("forced.jpg", "inline.jpg", "auto.jpg", "ignored.jpg")]
for paths, overflow, expected_flow, expected_gallery, expected_full in [
    ([forced], False, [], [forced], []),
    ([forced, inline, auto, ignored], False, [inline], [forced, auto], []),
    ([forced, inline, auto, ignored], True, [inline, auto], [forced], []),
    ([inline, auto], False, [inline], [], [auto]),
]:
    flow, gallery, full = [], [], []
    m._text_overflows_compat = lambda *a: overflow
    m._place_chapter_image = lambda *a, **k: flow.append(a[1])
    m._place_full_page_image = lambda *a: full.append(a[1])
    def place_gallery(*a):
        gallery.extend(a[1])
        return a[4], a[2] + len(a[1])
    m._place_gallery_pages = place_gallery
    args = {name: 0 for name in inspect.signature(m._render_basic_content).parameters}
    args.update(scribus=object(), image_paths=paths, book_dir=root,
        page_size=(600, 800), margins=(20, 20, 20, 20),
        layout_mode="single_page", first_page_mode="right", chapter_heading_alignment="left",
        chapter_heading_decoration={}, page_roles={},
        layout_index={"forced.jpg": {"placement": "gallery", "bleed": True},
                      "ignored.jpg": {"placement": "ignore"}})
    m._render_basic_content(**args)
    assert (flow, gallery, full) == (expected_flow, expected_gallery, expected_full), (flow, gallery, full)
`, committedScriptPath(t))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gallery routing failed: %v\n%s", err, output)
	}
}
