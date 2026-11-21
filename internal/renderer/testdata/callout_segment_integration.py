import importlib.util
import os
from pathlib import Path
import traceback

import scribus


def main():
	spec = importlib.util.spec_from_file_location("renderer", os.environ["BOOKGEN_SCRIPT"])
	renderer = importlib.util.module_from_spec(spec)
	spec.loader.exec_module(renderer)
	scribus.newDocument((600, 800), (20, 20, 20, 20), scribus.PORTRAIT, 1, scribus.UNIT_POINTS, scribus.NOFACINGPAGES, 0, 1)
	style = {
		"padding_points": 5,
		"font_name": "Lora Italic",
		"font_size_pt": 15,
		"color_rgb": [12, 45, 23],
		"background_color_rgb": [220, 240, 220],
		"border_color_rgb": [60, 100, 50],
		"border_width_pt": 1,
	}
	# Only require a font included in the repository's normal Scribus setup.
	style["font_name"] = "DejaVu Sans Oblique"
	text = "  First line 😀\n\nSecond line, longer than the frame and with a few words. \n" * 5 + "last\n"
	remaining = text
	parts = []
	for i in range(50):
		name = f"segment_{i}"
		part, height = renderer._layout_callout_segment(scribus, remaining, (20, 20, 120, 55), style, name)
		assert part and 0 < height <= 55, (part, height)
		assert scribus.getFillColor(name) == "callout_220_240_220"
		assert scribus.getLineColor(name) == "callout_60_100_50"
		assert scribus.getLineWidth(name) == 1
		assert scribus.getFont(name) == style["font_name"]
		assert scribus.getFontSize(name) == style["font_size_pt"]
		assert not scribus.textOverflows(name)
		parts.append(part)
		remaining = remaining[len(part):]
		if not remaining:
			break
	assert not remaining
	assert len(parts) > 1
	assert "".join(parts) == text

	part, height = renderer._layout_callout_segment(scribus, "x", (20, 20, 120, 11), style, "no_progress")
	assert (part, height) == ("", 0), (part, height)
	assert "no_progress" not in [item[0] for item in scribus.getPageItems()]
	assert renderer._layout_callout_segment(scribus, "x", (20, 20, 120, 10), style, "no_space") == ("", 0)
	try:
		renderer._layout_callout_segment(scribus, "x", (20, 20, 10, 100), style, "too_narrow")
	except RuntimeError as exc:
		assert "too narrow" in str(exc)
	else:
		raise AssertionError("an impossibly narrow callout should fail")

	transparent = dict(style, background_color_rgb=None, border_width_pt=0)
	part, height = renderer._layout_callout_segment(scribus, "Short text", (20, 20, 120, 100), transparent, "transparent")
	assert part == "Short text" and height < 100
	assert scribus.getFillColor("transparent") == "None"
	assert scribus.getLineColor("transparent") == "None"

	scribus.newPage(-1)
	scribus.gotoPage(2)
	prose = "one two three four five six seven eight nine ten " * 25
	bounds = (20, 20, 150, 100)
	unwrapped, _ = renderer._layout_prose_segment(scribus, prose, bounds, "unwrapped")
	scribus.deleteObject("unwrapped")
	renderer._create_wrap_frame_compat(scribus, "image", 70, 20, 60, 100)
	wrapped, _ = renderer._layout_prose_segment(scribus, prose, bounds, "wrapped", "image_wrap")
	assert 0 < len(wrapped) < len(unwrapped), (len(wrapped), len(unwrapped))
	assert prose.startswith(wrapped)
	assert not scribus.textOverflows("wrapped")
	scribus.deleteObject("wrapped")
	remaining = prose[len(wrapped):]
	wrapped_again, _ = renderer._layout_prose_segment(scribus, remaining, bounds, "wrapped_again", "image_wrap")
	assert remaining.startswith(wrapped_again) and wrapped_again
	assert not scribus.textOverflows("wrapped_again")

	image = (70, 20, 60, 100)
	area = renderer._callout_available_area(bounds, 20, image, 0, style["padding_points"], style["border_width_pt"])
	assert area is not None
	text, height = renderer._layout_callout_segment(scribus, "A highlighted note", area, style, "beside_image")
	assert text == "A highlighted note"
	x, y = scribus.getPosition("beside_image")
	width, actual_height = scribus.getSize("beside_image")
	stroke = style["border_width_pt"] / 2
	assert actual_height == height
	assert x + width + stroke <= image[0] or x - stroke >= image[0] + image[2]
	assert y >= bounds[1] and y + actual_height <= bounds[1] + bounds[3]

	scribus.newPage(-1)
	scribus.gotoPage(3)
	root = Path(os.environ["BOOKGEN_SCRIPT"]).resolve().parent.parent
	book = root / "books" / "sample-book"
	image_path = book / "chapters" / "1-the-road-to-san-rosario" / "the-road.png"
	exclusion = renderer._place_chapter_image(
		scribus, image_path, 1, 1, 3, (600, 800), (20, 20, 20, 20),
		"single_page", "right", 0, 0, 0, 0, 20, 760,
		(0, 0, 0), 0, 5, 6, 7, 8, 200, 200,
		True, "content_area", ["inside"], ["inside"], "preferred", 0,
		{}, book, {3: "chapter_opening"},
	)
	wrap_position = scribus.getPosition("chapter_1_image_1_wrap")
	wrap_size = scribus.getSize("chapter_1_image_1_wrap")
	for expected, actual in zip(exclusion, wrap_position + wrap_size):
		assert abs(expected - actual) < 0.001, (exclusion, wrap_position, wrap_size)
	scribus.newPage(-1)
	scribus.gotoPage(4)
	bordered = renderer._place_chapter_image(
		scribus, image_path, 2, 1, 4, (600, 800), (20, 20, 20, 20),
		"single_page", "right", 0, 0, 0, 0, 20, 760,
		(0, 0, 0), 12, 0, 0, 0, 0, 200, 200,
		True, "content_area", ["inside"], ["inside"], "preferred", 0,
		{}, book, {4: "body"},
	)
	x, y = scribus.getPosition("chapter_1_image_2")
	w, h = scribus.getSize("chapter_1_image_2")
	for expected, actual in zip(bordered, (x - 6, y - 6, w + 12, h + 12)):
		assert abs(expected - actual) < 0.001

	scribus.newPage(-1)
	scribus.gotoPage(5)
	roles = {5: "body"}
	full_page = renderer._place_chapter_image(
		scribus, image_path, 3, 1, 5, (600, 800), (20, 20, 20, 20),
		"single_page", "right", 0, 0, 0, 0, 20, 760,
		(0, 0, 0), 0, 5, 6, 7, 8, 200, 200,
		True, "content_area", ["inside"], ["inside"], "preferred", 0,
		{str(image_path): {"placement": "full_page"}}, book, roles,
	)
	assert full_page == (0, 0, 600, 800)
	assert roles[5] == "full_page_image"
	with open(os.environ["BOOKGEN_PROBE_RESULT"], "w", encoding="utf-8") as result:
		result.write("CALLOUT_SEGMENT_INTEGRATION_PASS\n")


try:
	main()
except Exception:
	with open(os.environ["BOOKGEN_PROBE_RESULT"], "w", encoding="utf-8") as result:
		traceback.print_exc(file=result)
