import importlib.util
import os
from pathlib import Path
import traceback

import scribus


def main():
	spec = importlib.util.spec_from_file_location("renderer", os.environ["BOOKGEN_SCRIPT"])
	renderer = importlib.util.module_from_spec(spec)
	spec.loader.exec_module(renderer)
	root = Path(os.environ["BOOKGEN_SCRIPT"]).resolve().parent.parent
	book = root / "books" / "sample-book"
	image = book / "chapters" / "1-the-road-to-san-rosario" / "the-road.png"
	scribus.newDocument((320, 240), (20, 20, 20, 20), scribus.PORTRAIT, 1, scribus.UNIT_POINTS, scribus.NOFACINGPAGES, 0, 1)
	renderer._ensure_chapter_heading_styles_compat(scribus, "DejaVu Sans Book", 14, (0, 0, 0))
	job = {
		"page": {"size_points": [320, 240], "margins_points": [20, 20, 20, 20], "layout": "single_page", "first_page": "right"},
		"bleed_points": {"top": 0, "bottom": 0, "inside": 0, "outside": 0},
		"chapter_headings": {"font_size_pt": 14, "alignment": "left", "spacing_points": {"top": 2, "bottom": 4}},
		"callouts": {
			"font_name": "DejaVu Sans Book", "font_size_pt": 10, "color_rgb": [0, 0, 0],
			"background_color_rgb": [220, 240, 220], "border_color_rgb": [60, 100, 50],
			"border_width_pt": 1, "padding_points": 4, "indent_points": 3,
			"spacing_before_points": 3, "spacing_after_points": 3,
		},
		"images": {
			"border_rgb": [0, 0, 0], "border_width_pt": 2,
			"spacing_points": {"top": 5, "bottom": 5, "inside": 5, "outside": 5},
			"max_width_points": 100, "max_height_points": 100, "snap_to_edge": True,
			"snap_target": "content_area", "allowed_edges": ["outside"], "preferred_edges": ["outside"],
			"edge_selection": "preferred", "edge_gap_points": 0,
		},
	}
	long_text = ("Long callout line with words and unicode é.\n\n" * 40) + "Final callout line."
	blocks = [
		("callout", "Before title."),
		("heading", "Chapter heading"),
		("paragraph", "Opening narrative."),
		("callout", long_text),
		("callout", "Adjacent callout."),
		("paragraph", "Ending narrative."),
	]
	roles = {1: "chapter_opening"}
	last, count, flow_images, consumed_images, all_images = renderer._render_ordered_flow(
		scribus, blocks, [image], 1, 1, job, book, {}, roles,
	)
	assert last > 2 and count == 1 and consumed_images == 1
	assert flow_images == all_images == [image]
	segments = []
	for page_number in range(1, last + 1):
		scribus.gotoPage(page_number)
		for name, _, _ in scribus.getPageItems():
			if name.startswith("chapter_1_callout_") or name.startswith("chapter_1_paragraph_"):
				assert not scribus.textOverflows(name)
				segments.append((int(name.rsplit("_", 1)[1]), name, scribus.getFrameText(name).replace("\r", "\n")))
	segments.sort()
	callout_text = "".join(text for _, name, text in segments if "_callout_" in name)
	prose_text = "".join(text for _, name, text in segments if "_paragraph_" in name)
	assert callout_text == "Before title." + long_text + "Adjacent callout.", repr(callout_text)
	assert prose_text == "Opening narrative.Ending narrative.", repr(prose_text)
	assert len([name for _, name, _ in segments if "_callout_" in name]) > 3
	assert scribus.objectExists("chapter_1_title")
	with open(os.environ["BOOKGEN_PROBE_RESULT"], "w", encoding="utf-8") as result:
		result.write("CALLOUT_FLOW_INTEGRATION_PASS\n")


try:
	main()
except Exception:
	with open(os.environ["BOOKGEN_PROBE_RESULT"], "w", encoding="utf-8") as result:
		traceback.print_exc(file=result)
