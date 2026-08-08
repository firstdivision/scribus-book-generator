package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

func TestBuildScribusInvocation(t *testing.T) {
	invocation := buildScribusInvocation("/tmp/book")
	if len(invocation) != 7 {
		t.Fatalf("expected 7 parts, got %d", len(invocation))
	}
	if invocation[0] != "xvfb-run" {
		t.Fatalf("expected xvfb-run, got %q", invocation[0])
	}
	if invocation[1] != "-a" {
		t.Fatalf("expected -a, got %q", invocation[1])
	}
	if invocation[2] != "scribus" {
		t.Fatalf("expected scribus, got %q", invocation[2])
	}
	if invocation[3] != "-g" {
		t.Fatalf("expected -g, got %q", invocation[3])
	}
	if invocation[4] != "-py" {
		t.Fatalf("expected -py, got %q", invocation[4])
	}
	if invocation[5] != "scripts/scribus_generate.py" {
		t.Fatalf("expected script path, got %q", invocation[5])
	}
	if invocation[6] != "/tmp/book" {
		t.Fatalf("expected book dir argument, got %q", invocation[6])
	}
}

func TestWriteGeneratedScribusScript(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "scripts", "scribus_generate.py")
	bookDir := filepath.Clean(filepath.Join("..", "..", "books", "sample-book"))
	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

	if err := writeGeneratedScribusScript(scriptPath, cfg, layoutplan.Plan{}); err != nil {
		t.Fatalf("writeGeneratedScribusScript returned error: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "def _new_document_compat") {
		t.Fatalf("generated script missing new document compatibility helper")
	}
	if !strings.Contains(text, "page_size_constant = \"PAPER_A4\"") {
		t.Fatalf("generated script missing paper size constant")
	}
	if !strings.Contains(text, "def _start_chapter_on_right_page_compat") {
		t.Fatalf("generated script missing chapter right-page helper")
	}
	if !strings.Contains(text, "def _create_background_master_compat") {
		t.Fatalf("generated script missing background master helper")
	}
	if !strings.Contains(text, "def _apply_master_page_compat") {
		t.Fatalf("generated script missing master page application helper")
	}
	if !strings.Contains(text, "def _render_page_numbers_compat") {
		t.Fatalf("generated script missing page number rendering helper")
	}
	if !strings.Contains(text, "def _logical_page_number_compat") {
		t.Fatalf("generated script missing logical page number helper")
	}
	if !strings.Contains(text, "def _page_number_placement_compat") {
		t.Fatalf("generated script missing page number placement helper")
	}
	if !strings.Contains(text, "current_page = _start_chapter_on_right_page_compat(scribus, current_page, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)") {
		t.Fatalf("generated script missing chapter right-page call")
	}
	if !strings.Contains(text, "scribus.newDocument(paper_size, margins, orientation, first_page_number, unit_points, page_type, first_page_order, num_pages)") {
		t.Fatalf("generated script missing documented newDocument call")
	}
	if !strings.Contains(text, "saveDocAs") {
		t.Fatalf("generated script missing saveDocAs compatibility path")
	}
	if !strings.Contains(text, "def _render_basic_content") {
		t.Fatalf("generated script missing content rendering helper")
	}
	if !strings.Contains(text, "def _image_dimensions_compat") {
		t.Fatalf("generated script missing image dimension helper")
	}
	if !strings.Contains(text, "def _apply_image_frame_style_compat") {
		t.Fatalf("generated script missing image frame style helper")
	}
	if !strings.Contains(text, "def _set_text_distances_sides_compat") {
		t.Fatalf("generated script missing side-specific text distance helper")
	}
	if !strings.Contains(text, "createText") {
		t.Fatalf("generated script missing Scribus text frame creation")
	}
	if !strings.Contains(text, "chapter_images = _image_files(chapter_dir)") {
		t.Fatalf("generated script missing chapter image discovery")
	}
	if !strings.Contains(text, "for image_index, image_path in enumerate(image_paths, start=1):") {
		t.Fatalf("generated script missing per-image rendering loop")
	}
	if !strings.Contains(text, "continuation_frame = _create_text_frame_compat(") {
		t.Fatalf("generated script should create continuation text frames on image pages")
	}
	if !strings.Contains(text, "_link_text_frames_compat(scribus, body_frames[-1], continuation_frame)") {
		t.Fatalf("generated script should link image-page continuation text frames")
	}
	if !strings.Contains(text, "current_page = chapter_image_cursor") {
		t.Fatalf("generated script should continue overflow from chapter_image_cursor")
	}
	if strings.Contains(text, "current_page = start_page") {
		t.Fatalf("generated script should not reset overflow cursor back to start_page")
	}
	if !strings.Contains(text, "image_border_rgb = (255, 255, 255)") {
		t.Fatalf("generated script missing image border rgb settings")
	}
	if !strings.Contains(text, "page_background_rgb = (248, 244, 232)") {
		t.Fatalf("generated script missing page background rgb settings")
	}
	if !strings.Contains(text, "page_numbers_enabled = True") {
		t.Fatalf("generated script missing page number enabled setting")
	}
	if !strings.Contains(text, "page_number_position = \"bottom_outside\"") {
		t.Fatalf("generated script missing page number position setting")
	}
	if !strings.Contains(text, "page_number_font_name = \"Source Serif 4 Regular\"") {
		t.Fatalf("generated script missing page number font setting")
	}
	if !strings.Contains(text, "page_number_hide_on = [\"chapter_opening\",\"full_page_image\",\"blank\"]") {
		t.Fatalf("generated script missing page number hide_on setting")
	}
	if !strings.Contains(text, "if is_full_page:\n\t\t\tpage_roles[chapter_image_cursor] = \"full_page_image\"") {
		t.Fatalf("generated script should classify full-page image pages for page number suppression")
	}
	if !strings.Contains(text, "scribus.createMasterPage(master_page_name)") {
		t.Fatalf("generated script missing master page creation")
	}
	if !strings.Contains(text, "scribus.applyMasterPage(master_page_name, page_number)") {
		t.Fatalf("generated script missing master page application")
	}
	expectedBorderWidth := fmt.Sprintf("image_border_width_pt = %.4f", cfg.Images.Border.WidthPt)
	if !strings.Contains(text, expectedBorderWidth) {
		t.Fatalf("generated script missing image border width setting")
	}
	expectedSpacingTop := fmt.Sprintf("image_spacing_top = %.4f", mmToPoints(cfg.Images.SpacingMM.Top))
	if !strings.Contains(text, expectedSpacingTop) {
		t.Fatalf("generated script missing image spacing top setting")
	}
	expectedMaxWidth := fmt.Sprintf("image_max_width = %.4f", mmToPoints(cfg.Images.Sizing.MaxWidthMM))
	if !strings.Contains(text, expectedMaxWidth) {
		t.Fatalf("generated script missing image max width setting")
	}
	if !strings.Contains(text, "image_snap_to_edge = True") {
		t.Fatalf("generated script missing snap_to_edge setting")
	}
	if !strings.Contains(text, "layout_plan = json.loads(\"{\\\"images\\\":[]}\")") {
		t.Fatalf("generated script missing layout plan payload")
	}
	if !strings.Contains(text, "output_stem = _output_filename_stem(layout_plan, book_dir)") {
		t.Fatalf("generated script missing output filename resolution")
	}
	if !strings.Contains(text, "sla_path = book_dir / \"out\" / f\"{output_stem}.sla\"") {
		t.Fatalf("generated script missing resolved SLA filename")
	}
	if !strings.Contains(text, "pdf_path = book_dir / \"out\" / f\"{output_stem}.pdf\"") {
		t.Fatalf("generated script missing resolved PDF filename")
	}
	if !strings.Contains(text, "def _choose_snap_edge") {
		t.Fatalf("generated script missing edge selection helper")
	}
	if !strings.Contains(text, "def _fit_contain_dimensions") {
		t.Fatalf("generated script missing contain sizing helper")
	}
	if !strings.Contains(text, "def _snap_frame_to_edge(snap_rect, frame_width, frame_height, physical_edge, edge_gap, is_right_page, spacing_left, spacing_right, spacing_top, spacing_bottom):") {
		t.Fatalf("generated script missing spacing-aware snap helper signature")
	}
	if !strings.Contains(text, "available_width = max(1.0, snap_rect[2] - image_spacing_left - image_spacing_right)") {
		t.Fatalf("generated script missing wrap-aware width bound calculation")
	}
	if !strings.Contains(text, "available_height = max(1.0, snap_rect[3] - image_spacing_top_used - image_spacing_bottom_used)") {
		t.Fatalf("generated script missing wrap-aware height bound calculation")
	}
	if !strings.Contains(text, "def _create_wrap_frame_compat") {
		t.Fatalf("generated script missing wrap frame helper")
	}
	if !strings.Contains(text, "createImage") {
		t.Fatalf("generated script missing image frame creation")
	}
	if !strings.Contains(text, "loadImage") {
		t.Fatalf("generated script missing image loading call")
	}
	if !strings.Contains(text, "def _set_text_flow_mode_compat") {
		t.Fatalf("generated script missing text flow compatibility helper")
	}
	if !strings.Contains(text, "setTextFlowMode") {
		t.Fatalf("generated script missing setTextFlowMode usage")
	}
	if !strings.Contains(text, "def _append_page_compat") {
		t.Fatalf("generated script missing page creation helper")
	}
	if !strings.Contains(text, "linkTextFrames") {
		t.Fatalf("generated script missing text frame linking")
	}
	if !strings.Contains(text, "def _estimate_body_pages") {
		t.Fatalf("generated script missing body page estimation helper")
	}
	if strings.Contains(text, "__PAGE_WIDTH_POINTS__") {
		t.Fatalf("generated script still contains unresolved config placeholders")
	}
	if !strings.Contains(text, "layout_mode = \"facing_pages\"") {
		t.Fatalf("generated script missing configured layout mode")
	}
	if strings.Contains(text, "automatic_text_frames") {
		t.Fatalf("generated script still references automatic_text_frames")
	}
}

func TestWriteGeneratedScribusScriptSupportsNilPageBackground(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "scripts", "scribus_generate.py")
	cfg := config.Default()

	if err := writeGeneratedScribusScript(scriptPath, cfg, layoutplan.Plan{}); err != nil {
		t.Fatalf("writeGeneratedScribusScript returned error: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "page_background_rgb = None") {
		t.Fatalf("generated script missing nil page background setting")
	}
	if !strings.Contains(text, "page_numbers_enabled = False") {
		t.Fatalf("generated script missing disabled page number setting")
	}
}

func TestWriteGeneratedScribusScriptIncludesImageBorderOverrideSupport(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "scripts", "scribus_generate.py")
	bookDir := filepath.Clean(filepath.Join("..", "..", "books", "sample-book"))
	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		t.Fatalf("LoadForBook returned error: %v", err)
	}

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

	if err := writeGeneratedScribusScript(scriptPath, cfg, plan); err != nil {
		t.Fatalf("writeGeneratedScribusScript returned error: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "def _resolve_border_override") {
		t.Fatalf("generated script missing border override helper")
	}
	if !strings.Contains(text, "border_override = image_instruction.get(\"border\")") {
		t.Fatalf("generated script missing border override lookup")
	}
	if !strings.Contains(text, "\\\"width_pt\\\":0") {
		t.Fatalf("generated script missing width_pt override in embedded layout plan")
	}
}
