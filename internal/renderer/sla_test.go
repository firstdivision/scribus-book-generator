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
	cfg := config.Default()

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
	if !strings.Contains(text, "current_page = _start_chapter_on_right_page_compat(scribus, current_page)") {
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
	if !strings.Contains(text, "createText") {
		t.Fatalf("generated script missing Scribus text frame creation")
	}
	if !strings.Contains(text, "chapter_images = _image_files(chapter_dir)") {
		t.Fatalf("generated script missing chapter image discovery")
	}
	if !strings.Contains(text, "for image_index, image_path in enumerate(image_paths, start=1):") {
		t.Fatalf("generated script missing per-image rendering loop")
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
	if !strings.Contains(text, "layout_mode = \"single_page\"") {
		t.Fatalf("generated script missing configured layout mode")
	}
	if strings.Contains(text, "automatic_text_frames") {
		t.Fatalf("generated script still references automatic_text_frames")
	}
}
