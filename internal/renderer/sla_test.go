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
	if job.Images.GalleryColumns != 2 {
		t.Fatalf("expected leftover gallery_columns 2, got %d", job.Images.GalleryColumns)
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
	if parsed.Short[0][2] != parsed.Full[0][2] || parsed.Short[0][3] != parsed.Full[0][3] {
		t.Fatalf("short-row cell must keep full-grid size, full=%v short=%v", parsed.Full[0], parsed.Short[0])
	}
	if parsed.Short[0][2] >= 400 {
		t.Fatalf("short-row cell should not stretch to content width, got %v", parsed.Short[0])
	}
	if parsed.Full[1][0] <= parsed.Full[0][0] || parsed.Full[1][1] != parsed.Full[0][1] {
		t.Fatalf("expected row-major second cell to the right, got first=%v second=%v", parsed.Full[0], parsed.Full[1])
	}
	if parsed.Full[2][1] <= parsed.Full[0][1] || parsed.Full[2][0] != parsed.Full[0][0] {
		t.Fatalf("expected third cell on the next row, got first=%v third=%v", parsed.Full[0], parsed.Full[2])
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
