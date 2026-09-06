package renderer

import (
	"encoding/json"
	"os/exec"
	"testing"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/chapterheadings"
	"scribus-book-generator/internal/layout/layoutplan"
)

func TestChapterHeadingDecoration(t *testing.T) {
	cfg := config.Default()
	empty, err := json.Marshal(buildScribusJob(cfg, layoutplan.Plan{}).ChapterHeadings)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ChapterHeadings.BackgroundColorRGB = &[3]int{240, 230, 220}
	cfg.ChapterHeadings.Borders = map[string]chapterheadings.Border{
		"top":    {ColorRGB: [3]int{1, 2, 3}, WidthPt: 1},
		"bottom": {ColorRGB: [3]int{4, 5, 6}, WidthPt: 2},
		"left":   {ColorRGB: [3]int{7, 8, 9}, WidthPt: 3},
		"right":  {ColorRGB: [3]int{10, 11, 12}, WidthPt: 4},
	}
	decorated, err := json.Marshal(buildScribusJob(cfg, layoutplan.Plan{}).ChapterHeadings)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-c", `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("renderer", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
class Scribus:
    def __init__(self):
        self.rects = {}
        self.fills = {}
        self.colors = {}
    def getColorNames(self): return list(self.colors)
    def defineColorRGB(self, name, *rgb): self.colors[name] = rgb
    def createRect(self, x, y, w, h, name):
        self.rects[name] = (x, y, w, h)
        return name
    def setFillColor(self, color, name): self.fills[name] = self.colors[color]
    def setLineColor(self, color, name): assert color == "None"
    def setLineWidth(self, width, name): assert width == 0
s = Scribus()
module._decorate_chapter_heading(s, "title", 10, 20, 100, 40, json.loads(sys.argv[2]))
assert not s.rects and not s.fills
settings = json.loads(sys.argv[3])
module._decorate_chapter_heading(s, "title", 10, 20, 100, 40, settings)
assert s.fills["title"] == (240, 230, 220)
assert list(s.rects.values()) == [(10, 20, 100, 1), (10, 58, 100, 2), (10, 20, 3, 40), (106, 20, 4, 40)], s.rects
assert [s.fills[name] for name in s.rects] == [(1, 2, 3), (4, 5, 6), (7, 8, 9), (10, 11, 12)]
s = Scribus()
settings["borders"] = {"top": {"width_pt": 0}, "left": None}
settings["background_color_rgb"] = None
module._decorate_chapter_heading(s, "title", 10, 20, 100, 40, settings)
assert not s.rects and not s.fills
`, committedScriptPath(t), string(empty), string(decorated))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("decoration failed: %v\n%s", err, output)
	}
}
