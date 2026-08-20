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
		"def _image_dimensions_compat",
		"def _apply_image_frame_style_compat",
		"def _set_text_distances_sides_compat",
		"createText",
		"chapter_images = _image_files(chapter_dir)",
		"for image_index, image_path in enumerate(image_paths, start=1):",
		"continuation_frame = _create_text_frame_compat(",
		"_link_text_frames_compat(scribus, body_frames[-1], continuation_frame)",
		"current_page = chapter_image_cursor",
		"def _ensure_chapter_heading_styles_compat",
		"Configured chapter heading font '{font_name}' is not available in Scribus",
		"def _resolve_chapter_heading_alignment",
		"title_top = margin_top + chapter_heading_spacing_top",
		"if is_full_page:\n\t\t\tpage_roles[chapter_image_cursor] = \"full_page_image\"",
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
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("committed script missing %q", fragment)
		}
	}
	if strings.Contains(text, "current_page = start_page") {
		t.Fatal("script should not reset overflow cursor back to start_page")
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
