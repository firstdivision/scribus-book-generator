package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scribus-book-generator/internal/config"
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

	if err := writeGeneratedScribusScript(scriptPath, cfg); err != nil {
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
	if !strings.Contains(text, "scribus.createMasterPage(master_page_name)") {
		t.Fatalf("generated script missing master page creation")
	}
	if !strings.Contains(text, "scribus.applyMasterPage(master_page_name, page_number)") {
		t.Fatalf("generated script missing master page application")
	}
	if !strings.Contains(text, "image_border_width_pt = 3.0000") {
		t.Fatalf("generated script missing image border width setting")
	}
	if !strings.Contains(text, "image_spacing_top = 14.1732") {
		t.Fatalf("generated script missing image spacing top setting")
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

	if err := writeGeneratedScribusScript(scriptPath, cfg); err != nil {
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
