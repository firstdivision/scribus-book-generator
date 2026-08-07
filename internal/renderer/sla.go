package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/markdown"
)

const generatedScribusScriptPath = "scripts/scribus_generate.py"

const generatedScribusScriptTemplate = `#!/usr/bin/env python3
import sys
from pathlib import Path


def _new_document_compat(scribus, page_size, margins, layout_mode, first_page_mode, paper_size_constant) -> None:
	"""Create a new document across Scribus API variants."""
	orientation_name = "LANDSCAPE" if page_size[0] > page_size[1] else "PORTRAIT"
	orientation = getattr(scribus, orientation_name, 0)
	unit_points = getattr(scribus, "UNIT_POINTS", 0)
	paper_size = getattr(scribus, paper_size_constant, page_size)
	page_types = {
		"facing_pages": getattr(scribus, "PAGE_2", 1),
		"single_page": getattr(scribus, "PAGE_1", 0),
	}
	page_type = page_types.get(layout_mode, getattr(scribus, "PAGE_1", 0))
	first_page_order = 1 if layout_mode == "facing_pages" and first_page_mode == "right" else 0
	first_page_number = 1
	num_pages = 1

	try:
		scribus.newDocument(paper_size, margins, orientation, first_page_number, unit_points, page_type, first_page_order, num_pages)
		return
	except TypeError as exc:
		raise RuntimeError("Unable to create Scribus document with documented signature") from exc


def _save_document_compat(scribus, output_path: Path) -> None:
	"""Save the active Scribus document across API variants."""
	path_text = str(output_path)

	if hasattr(scribus, "saveDocAs"):
		scribus.saveDocAs(path_text)
		return

	if hasattr(scribus, "saveDoc"):
		try:
			scribus.saveDoc(path_text)
			return
		except TypeError:
			scribus.saveDoc()
			return

	raise RuntimeError("Scribus save API is unavailable in this build")


def _chapter_directories(chapters_dir: Path):
	return sorted([entry for entry in chapters_dir.iterdir() if entry.is_dir()])


def _first_markdown_file(chapter_dir: Path):
	files = sorted(chapter_dir.glob("*.md"))
	if not files:
		return None
	return files[0]


def _image_files(chapter_dir: Path):
	extensions = ("*.png", "*.jpg", "*.jpeg", "*.webp", "*.gif", "*.svg")
	results = []
	for pattern in extensions:
		for candidate in sorted(chapter_dir.glob(pattern)):
			if candidate.name.startswith("."):
				continue
			results.append(candidate)
	return results


def _image_dimensions_compat(image_path: Path):
	try:
		data = image_path.read_bytes()
	except Exception:
		return 0, 0

	if image_path.suffix.lower() == ".png" and len(data) >= 24 and data[12:16] == b"IHDR":
		return int.from_bytes(data[16:20], "big"), int.from_bytes(data[20:24], "big")

	if image_path.suffix.lower() in (".jpg", ".jpeg") and len(data) >= 4 and data[:2] == b"\xff\xd8":
		index = 2
		while index + 9 < len(data):
			if data[index] != 0xFF:
				index += 1
				continue
			marker = data[index + 1]
			index += 2
			if marker in (0xC0, 0xC1, 0xC2, 0xC3, 0xC5, 0xC6, 0xC7, 0xC9, 0xCA, 0xCB, 0xCD, 0xCE, 0xCF):
				height = int.from_bytes(data[index + 3:index + 5], "big")
				width = int.from_bytes(data[index + 5:index + 7], "big")
				return width, height
			segment_length = int.from_bytes(data[index:index + 2], "big")
			index += segment_length

	return 0, 0


def _parse_chapter_markdown(chapter_path: Path):
	title = "Untitled Chapter"
	paragraphs = []

	for raw_line in chapter_path.read_text(encoding="utf-8").splitlines():
		line = raw_line.strip()
		if not line:
			continue
		if title == "Untitled Chapter" and line.startswith("# "):
			title = line[2:].strip() or title
			continue
		paragraphs.append(line)

	body_text = "\n\n".join(paragraphs)
	if not body_text:
		body_text = " "
	return title, body_text


def _create_text_frame_compat(scribus, x, y, width, height, name):
	try:
		return scribus.createText(x, y, width, height, name)
	except TypeError:
		return scribus.createText(x, y, width, height)


def _set_frame_text_compat(scribus, frame_name, text):
	if hasattr(scribus, "setText"):
		scribus.setText(text, frame_name)
		return
	if hasattr(scribus, "insertText"):
		scribus.insertText(text, 0, frame_name)
		return
	raise RuntimeError("No compatible Scribus text insertion API found")


def _create_image_frame_compat(scribus, x, y, width, height, name):
	try:
		return scribus.createImage(x, y, width, height, name)
	except TypeError:
		return scribus.createImage(x, y, width, height)


def _load_image_compat(scribus, image_path: Path, frame_name):
	path_text = str(image_path)
	if hasattr(scribus, "loadImage"):
		scribus.loadImage(path_text, frame_name)
		return
	raise RuntimeError("No compatible Scribus image loading API found")


def _set_scale_image_to_frame_compat(scribus, frame_name):
	if not hasattr(scribus, "setScaleImageToFrame"):
		return
	try:
		scribus.setScaleImageToFrame(1, 1, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setScaleImageToFrame(1, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setScaleImageToFrame(frame_name)
	except TypeError:
		return


def _set_text_flow_mode_compat(scribus, frame_name):
	if not hasattr(scribus, "setTextFlowMode"):
		return

	modes = []
	for constant_name in (
		"TEXTFLOW_BOUNDINGBOX",
		"TEXTFLOW_CONTOURLINE",
		"TEXTFLOW_FRAME",
	):
		if hasattr(scribus, constant_name):
			modes.append(getattr(scribus, constant_name))

	if not modes:
		modes = [2, 3, 1]

	for mode in modes:
		try:
			scribus.setTextFlowMode(mode, frame_name)
			return
		except TypeError:
			try:
				scribus.setTextFlowMode(frame_name, mode)
				return
			except TypeError:
				continue


def _set_text_distances_compat(scribus, frame_name, distance):
	if not hasattr(scribus, "setTextDistances"):
		return

	try:
		scribus.setTextDistances(distance, distance, distance, distance, frame_name)
		return
	except Exception:
		pass

	try:
		scribus.setTextDistances(frame_name, distance, distance, distance, distance)
	except Exception:
		return


def _set_text_distances_sides_compat(scribus, frame_name, left, right, top, bottom):
	if not hasattr(scribus, "setTextDistances"):
		return

	try:
		scribus.setTextDistances(left, right, top, bottom, frame_name)
		return
	except Exception:
		pass

	try:
		scribus.setTextDistances(frame_name, left, right, top, bottom)
	except Exception:
		return


def _create_rect_compat(scribus, x, y, width, height, name):
	if not hasattr(scribus, "createRect"):
		raise RuntimeError("No compatible Scribus rectangle API found")
	try:
		return scribus.createRect(x, y, width, height, name)
	except TypeError:
		return scribus.createRect(x, y, width, height)


def _document_page_size_compat(scribus, fallback_page_size):
	if hasattr(scribus, "getPageSize"):
		try:
			page_size = scribus.getPageSize()
			if isinstance(page_size, tuple) and len(page_size) >= 2:
				return float(page_size[0]), float(page_size[1])
		except Exception:
			pass

	return fallback_page_size


def _rgb_color_name(rgb_values):
	return f"image_border_{rgb_values[0]}_{rgb_values[1]}_{rgb_values[2]}"


def _ensure_rgb_color_compat(scribus, rgb_values):
	color_name = _rgb_color_name(rgb_values)
	if hasattr(scribus, "getColorNames"):
		try:
			if color_name in scribus.getColorNames():
				return color_name
		except Exception:
			pass

	if hasattr(scribus, "defineColorRGB"):
		try:
			scribus.defineColorRGB(color_name, rgb_values[0], rgb_values[1], rgb_values[2])
			return color_name
		except Exception:
			pass

	return color_name


def _apply_image_frame_style_compat(scribus, frame_name, border_rgb, border_width_pt):
	if border_width_pt <= 0:
		return

	color_name = _ensure_rgb_color_compat(scribus, border_rgb)
	if hasattr(scribus, "setLineColor"):
		try:
			scribus.setLineColor(color_name, frame_name)
		except Exception:
			pass

	if hasattr(scribus, "setLineWidth"):
		try:
			scribus.setLineWidth(border_width_pt, frame_name)
		except Exception:
			pass


def _page_is_right_compat(layout_mode, first_page_mode, page_number):
	if layout_mode != "facing_pages":
		return True
	first_page_is_right = first_page_mode == "right"
	if first_page_is_right:
		return page_number % 2 == 1
	return page_number % 2 == 0


def _master_page_name_for_page(layout_mode, first_page_mode, page_number):
	if layout_mode != "facing_pages":
		return "Body"
	if _page_is_right_compat(layout_mode, first_page_mode, page_number):
		return "Body Right"
	return "Body Left"


def _apply_master_page_compat(scribus, master_page_name, page_number):
	if not master_page_name or not hasattr(scribus, "applyMasterPage"):
		return
	try:
		scribus.applyMasterPage(master_page_name, page_number)
	except Exception:
		return


def _create_background_master_compat(scribus, master_page_name, background_rgb, left_bleed, right_bleed, top_bleed, bottom_bleed, fallback_page_size):
	if background_rgb is None:
		return
	if not hasattr(scribus, "createMasterPage") or not hasattr(scribus, "closeMasterPage"):
		return

	scribus.createMasterPage(master_page_name)
	page_width, page_height = _document_page_size_compat(scribus, fallback_page_size)
	color_name = _ensure_rgb_color_compat(scribus, background_rgb)
	background_name = f"{master_page_name}_background"
	background = _create_rect_compat(
		scribus,
		-left_bleed,
		-top_bleed,
		page_width + left_bleed + right_bleed,
		page_height + top_bleed + bottom_bleed,
		background_name,
	)
	if hasattr(scribus, "setFillColor"):
		try:
			scribus.setFillColor(color_name, background)
		except Exception:
			pass
	if hasattr(scribus, "setLineColor"):
		try:
			scribus.setLineColor("None", background)
		except Exception:
			pass
	if hasattr(scribus, "setLineWidth"):
		try:
			scribus.setLineWidth(0, background)
		except Exception:
			pass
	scribus.closeMasterPage()


def _page_horizontal_bleeds(layout_mode, first_page_mode, page_number, bleed_inside, bleed_outside):
	if layout_mode != "facing_pages":
		return bleed_outside, bleed_outside
	if _page_is_right_compat(layout_mode, first_page_mode, page_number):
		return bleed_inside, bleed_outside
	return bleed_outside, bleed_inside


def _create_page_background_compat(scribus, page_number, layout_mode, first_page_mode, background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, fallback_page_size):
	if background_rgb is None:
		return

	left_bleed, right_bleed = _page_horizontal_bleeds(layout_mode, first_page_mode, page_number, bleed_inside, bleed_outside)
	page_width, page_height = _document_page_size_compat(scribus, fallback_page_size)
	color_name = _ensure_rgb_color_compat(scribus, background_rgb)
	background_name = f"page_{page_number}_background"
	background = _create_rect_compat(
		scribus,
		-left_bleed,
		-bleed_top,
		page_width + left_bleed + right_bleed,
		page_height + bleed_top + bleed_bottom,
		background_name,
	)
	if hasattr(scribus, "setFillColor"):
		try:
			scribus.setFillColor(color_name, background)
		except Exception:
			pass
	if hasattr(scribus, "setLineColor"):
		try:
			scribus.setLineColor("None", background)
		except Exception:
			pass
	if hasattr(scribus, "setLineWidth"):
		try:
			scribus.setLineWidth(0, background)
		except Exception:
			pass


def _logical_page_number_compat(physical_page, start_on_page, start_number):
	if physical_page < start_on_page:
		return None
	return start_number + (physical_page - start_on_page)


def _roman_numeral_compat(number):
	values = (
		(1000, "M"),
		(900, "CM"),
		(500, "D"),
		(400, "CD"),
		(100, "C"),
		(90, "XC"),
		(50, "L"),
		(40, "XL"),
		(10, "X"),
		(9, "IX"),
		(5, "V"),
		(4, "IV"),
		(1, "I"),
	)
	parts = []
	remaining = number
	for value, symbol in values:
		while remaining >= value:
			parts.append(symbol)
			remaining -= value
	return "".join(parts)


def _format_page_number_compat(number_format, number):
	if number_format == "arabic":
		return str(number)
	roman = _roman_numeral_compat(number)
	if number_format == "roman_lower":
		return roman.lower()
	return roman


def _page_number_placement_compat(position, is_right_page):
	if position == "bottom_outside":
		if is_right_page:
			return "bottom", "right", "right", "outside"
		return "bottom", "left", "left", "outside"
	if position == "bottom_inside":
		if is_right_page:
			return "bottom", "left", "left", "inside"
		return "bottom", "right", "right", "inside"
	if position == "bottom_center":
		return "bottom", "center", "center", "center"
	if position == "top_outside":
		if is_right_page:
			return "top", "right", "right", "outside"
		return "top", "left", "left", "outside"
	if position == "top_inside":
		if is_right_page:
			return "top", "left", "left", "inside"
		return "top", "right", "right", "inside"
	return "top", "center", "center", "center"


def _set_text_color_compat(scribus, frame_name, color_name):
	if not hasattr(scribus, "setTextColor"):
		return
	try:
		scribus.setTextColor(color_name, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setTextColor(frame_name, color_name)
	except TypeError:
		return


def _set_font_compat(scribus, frame_name, font_name, fallback_font_name):
	if not hasattr(scribus, "setFont"):
		return
	for candidate in (font_name, fallback_font_name):
		if not candidate:
			continue
		try:
			scribus.setFont(candidate, frame_name)
			return
		except Exception:
			continue


def _set_font_size_compat(scribus, frame_name, font_size_pt):
	if not hasattr(scribus, "setFontSize"):
		return
	try:
		scribus.setFontSize(font_size_pt, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setFontSize(frame_name, font_size_pt)
	except TypeError:
		return


def _set_fill_none_compat(scribus, frame_name):
	if not hasattr(scribus, "setFillColor"):
		return
	try:
		scribus.setFillColor("None", frame_name)
	except Exception:
		return


def _set_line_none_compat(scribus, frame_name):
	if hasattr(scribus, "setLineColor"):
		try:
			scribus.setLineColor("None", frame_name)
		except Exception:
			pass
	if hasattr(scribus, "setLineWidth"):
		try:
			scribus.setLineWidth(0, frame_name)
		except Exception:
			return


def _set_text_alignment_compat(scribus, frame_name, alignment_name):
	if not hasattr(scribus, "setTextAlignment"):
		return
	alignment_map = {
		"left": getattr(scribus, "ALIGN_LEFT", 0),
		"right": getattr(scribus, "ALIGN_RIGHT", 2),
		"center": getattr(scribus, "ALIGN_CENTERED", 1),
	}
	alignment = alignment_map.get(alignment_name, alignment_map["left"])
	try:
		scribus.setTextAlignment(alignment, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setTextAlignment(frame_name, alignment)
	except TypeError:
		return


def _render_page_number_frame_compat(scribus, page_number, page_role, page_size, layout_mode, first_page_mode, page_number_start_on_page, page_number_start_number, page_number_format, page_number_position, page_number_font_name, page_number_font_family, page_number_font_size_pt, page_number_color_rgb, page_number_offset_top, page_number_offset_bottom, page_number_offset_inside, page_number_offset_outside, page_number_hide_on):
	if page_role in page_number_hide_on:
		return

	logical_number = _logical_page_number_compat(page_number, page_number_start_on_page, page_number_start_number)
	if logical_number is None:
		return

	page_width, page_height = _document_page_size_compat(scribus, page_size)
	vertical, horizontal, alignment_name, offset_key = _page_number_placement_compat(
		page_number_position,
		_page_is_right_compat(layout_mode, first_page_mode, page_number),
	)
	frame_width = max(page_number_font_size_pt * 8.0, 72.0)
	frame_height = max(page_number_font_size_pt * 2.0, 18.0)

	if offset_key == "inside":
		horizontal_offset = page_number_offset_inside
	elif offset_key == "outside":
		horizontal_offset = page_number_offset_outside
	else:
		horizontal_offset = 0.0

	if horizontal == "left":
		x = horizontal_offset
	elif horizontal == "right":
		x = page_width - horizontal_offset - frame_width
	else:
		x = (page_width - frame_width) / 2.0

	if vertical == "top":
		y = page_number_offset_top
	else:
		y = page_height - page_number_offset_bottom - frame_height

	frame_name = f"page_{page_number}_number"
	frame = _create_text_frame_compat(scribus, x, y, frame_width, frame_height, frame_name)
	_set_frame_text_compat(scribus, frame, _format_page_number_compat(page_number_format, logical_number))
	_set_font_compat(scribus, frame, page_number_font_name, page_number_font_family)
	_set_font_size_compat(scribus, frame, page_number_font_size_pt)
	_set_text_alignment_compat(scribus, frame, alignment_name)
	_set_fill_none_compat(scribus, frame)
	_set_line_none_compat(scribus, frame)
	_set_text_color_compat(scribus, frame, _ensure_rgb_color_compat(scribus, page_number_color_rgb))


def _render_page_numbers_compat(scribus, total_pages, page_roles, page_size, layout_mode, first_page_mode, page_numbers_enabled, page_number_start_on_page, page_number_start_number, page_number_format, page_number_position, page_number_font_name, page_number_font_family, page_number_font_size_pt, page_number_color_rgb, page_number_offset_top, page_number_offset_bottom, page_number_offset_inside, page_number_offset_outside, page_number_hide_on):
	if not page_numbers_enabled:
		return

	for page_number in range(1, total_pages + 1):
		_goto_page_compat(scribus, page_number)
		_render_page_number_frame_compat(
			scribus,
			page_number,
			page_roles.get(page_number, "body"),
			page_size,
			layout_mode,
			first_page_mode,
			page_number_start_on_page,
			page_number_start_number,
			page_number_format,
			page_number_position,
			page_number_font_name,
			page_number_font_family,
			page_number_font_size_pt,
			page_number_color_rgb,
			page_number_offset_top,
			page_number_offset_bottom,
			page_number_offset_inside,
			page_number_offset_outside,
			page_number_hide_on,
		)


def _append_body_page_compat(scribus, current_page, page_role, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles):
	_append_page_compat(scribus)
	current_page += 1
	page_roles[current_page] = page_role
	_create_page_background_compat(scribus, current_page, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size)
	return current_page


def _estimate_body_pages(body_text):
	chars_per_page = 1700
	if not body_text:
		return 1
	estimated = (len(body_text) + chars_per_page - 1) // chars_per_page
	return max(1, min(40, estimated))


def _append_page_compat(scribus, master_page_name=None):
	if hasattr(scribus, "newPage"):
		page_args = []
		if master_page_name:
			page_args.extend(((-1, master_page_name), (1, master_page_name)))
		page_args.extend(((-1,), (1,), tuple()))
		for args in page_args:
			try:
				scribus.newPage(*args)
				return
			except TypeError:
				continue
	if hasattr(scribus, "createPage"):
		page_args = []
		if master_page_name:
			page_args.extend(((-1, master_page_name), (1, master_page_name)))
		page_args.extend(((-1,), (1,), tuple()))
		for args in page_args:
			try:
				scribus.createPage(*args)
				return
			except TypeError:
				continue
	raise RuntimeError("No compatible Scribus page-creation API found")


def _goto_page_compat(scribus, page_number):
	if hasattr(scribus, "gotoPage"):
		scribus.gotoPage(page_number)


def _link_text_frames_compat(scribus, source_frame, target_frame):
	if not hasattr(scribus, "linkTextFrames"):
		raise RuntimeError("No compatible Scribus text-link API found")

	try:
		scribus.linkTextFrames(source_frame, target_frame)
		return
	except TypeError:
		pass

	try:
		scribus.linkTextFrames(target_frame, source_frame)
	except TypeError as exc:
		raise RuntimeError("Unable to link Scribus text frames") from exc


def _text_overflows_compat(scribus, frame_name):
	if not hasattr(scribus, "textOverflows"):
		return False

	try:
		return bool(scribus.textOverflows(frame_name))
	except Exception:
		return False


def _start_chapter_on_right_page_compat(scribus, current_page, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles):
	if current_page % 2 == 1:
		current_page = _append_body_page_compat(scribus, current_page, "blank", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)

	current_page = _append_body_page_compat(scribus, current_page, "chapter_opening", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
	return current_page



def _render_basic_content(scribus, title_text, body_text, image_paths, chapter_index, start_page, page_size, margins, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, image_border_rgb, image_border_width_pt, image_spacing_top, image_spacing_bottom, image_spacing_inside, image_spacing_outside, page_roles):
	page_width, page_height = _document_page_size_compat(scribus, page_size)
	margin_top, margin_left, margin_right, margin_bottom = margins

	content_width = page_width - margin_left - margin_right
	title_height = 64.0
	chapter_opening_body_top = margin_top + title_height + 12.0
	chapter_opening_body_height = page_height - chapter_opening_body_top - margin_bottom
	continuation_body_top = margin_top
	continuation_body_height = page_height - continuation_body_top - margin_bottom
	chapter_image_cursor = start_page

	title_frame = _create_text_frame_compat(
		scribus,
		margin_left,
		margin_top,
		content_width,
		title_height,
		f"chapter_{chapter_index}_title",
	)
	body_frame = _create_text_frame_compat(
		scribus,
		margin_left,
		chapter_opening_body_top,
		content_width,
		chapter_opening_body_height,
		f"chapter_{chapter_index}_body",
	)
	body_frames = [body_frame]

	for image_index, image_path in enumerate(image_paths, start=1):
		if image_index > 1:
			chapter_image_cursor = _append_body_page_compat(scribus, chapter_image_cursor, "body", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
			_goto_page_compat(scribus, chapter_image_cursor)

		if layout_mode == "facing_pages" and _page_is_right_compat(layout_mode, first_page_mode, chapter_image_cursor):
			image_spacing_left = image_spacing_inside
			image_spacing_right = image_spacing_outside
		else:
			image_spacing_left = image_spacing_outside
			image_spacing_right = image_spacing_inside

		if chapter_image_cursor == start_page:
			image_body_top = chapter_opening_body_top
			image_body_height = chapter_opening_body_height
		else:
			image_body_top = continuation_body_top
			image_body_height = continuation_body_height

		image_width, image_height = _image_dimensions_compat(image_path)
		available_width = content_width - image_spacing_left - image_spacing_right
		available_height = image_body_height - image_spacing_top - image_spacing_bottom
		if image_width > 0 and image_height > 0:
			frame_width = available_width
			frame_height = frame_width * (float(image_height) / float(image_width))
			if frame_height > available_height:
				frame_height = available_height
				frame_width = frame_height * (float(image_width) / float(image_height))
		else:
			frame_width = available_width
			frame_height = available_height

		image_x = margin_left + image_spacing_left + ((available_width - frame_width) / 2.0)
		image_y = image_body_top + image_spacing_top + ((available_height - frame_height) / 2.0)

		image_frame = _create_image_frame_compat(
			scribus,
			image_x,
			image_y,
			frame_width,
			frame_height,
			f"chapter_{chapter_index}_image_{image_index}",
		)
		_load_image_compat(scribus, image_path, image_frame)
		_set_scale_image_to_frame_compat(scribus, image_frame)
		_set_text_flow_mode_compat(scribus, image_frame)
		_apply_image_frame_style_compat(scribus, image_frame, image_border_rgb, image_border_width_pt)
		_set_text_distances_sides_compat(scribus, image_frame, image_spacing_left, image_spacing_right, image_spacing_top, image_spacing_bottom)

	_set_frame_text_compat(scribus, title_frame, title_text)
	_set_frame_text_compat(scribus, body_frame, body_text)

	current_page = start_page

	# If Scribus reports overflow, grow the chain incrementally.
	max_extra_pages = 20
	while _text_overflows_compat(scribus, body_frames[-1]) and max_extra_pages > 0:
		current_page = _append_body_page_compat(scribus, current_page, "body", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
		_goto_page_compat(scribus, current_page)
		next_page_number = len(body_frames) + 1
		next_frame = _create_text_frame_compat(
			scribus,
			margin_left,
			continuation_body_top,
			content_width,
			continuation_body_height,
			f"chapter_{chapter_index}_body_{next_page_number}",
		)
		_link_text_frames_compat(scribus, body_frames[-1], next_frame)
		body_frames.append(next_frame)
		max_extra_pages -= 1

	return current_page


def main() -> int:
	if len(sys.argv) < 2:
		print("usage: scribus_generate.py <book-dir>", file=sys.stderr)
		return 2

	book_dir = Path(sys.argv[1]).resolve()
	chapters_dir = book_dir / "chapters"
	sla_path = book_dir / "out" / "example.sla"
	pdf_path = book_dir / "out" / "example.pdf"

	page_size = (__PAGE_WIDTH_POINTS__, __PAGE_HEIGHT_POINTS__)
	margins = (
		__MARGIN_TOP_POINTS__,
		__MARGIN_LEFT_POINTS__,
		__MARGIN_RIGHT_POINTS__,
		__MARGIN_BOTTOM_POINTS__,
	)
	page_size_constant = "__PAGE_SIZE_CONSTANT__"
	page_background_rgb = __PAGE_BACKGROUND_RGB__
	page_numbers_enabled = __PAGE_NUMBERS_ENABLED__
	page_number_start_on_page = __PAGE_NUMBER_START_ON_PAGE__
	page_number_start_number = __PAGE_NUMBER_START_NUMBER__
	page_number_format = __PAGE_NUMBER_FORMAT__
	page_number_position = __PAGE_NUMBER_POSITION__
	page_number_font_name = __PAGE_NUMBER_FONT_NAME__
	page_number_font_family = __PAGE_NUMBER_FONT_FAMILY__
	page_number_font_size_pt = __PAGE_NUMBER_FONT_SIZE_PT__
	page_number_color_rgb = (__PAGE_NUMBER_COLOR_RED__, __PAGE_NUMBER_COLOR_GREEN__, __PAGE_NUMBER_COLOR_BLUE__)
	page_number_offset_top = __PAGE_NUMBER_OFFSET_TOP_POINTS__
	page_number_offset_bottom = __PAGE_NUMBER_OFFSET_BOTTOM_POINTS__
	page_number_offset_inside = __PAGE_NUMBER_OFFSET_INSIDE_POINTS__
	page_number_offset_outside = __PAGE_NUMBER_OFFSET_OUTSIDE_POINTS__
	page_number_hide_on = __PAGE_NUMBER_HIDE_ON__
	bleed_top = __BLEED_TOP_POINTS__
	bleed_bottom = __BLEED_BOTTOM_POINTS__
	bleed_inside = __BLEED_INSIDE_POINTS__
	bleed_outside = __BLEED_OUTSIDE_POINTS__
	image_border_rgb = (__IMAGE_BORDER_RED__, __IMAGE_BORDER_GREEN__, __IMAGE_BORDER_BLUE__)
	image_border_width_pt = __IMAGE_BORDER_WIDTH_PT__
	image_spacing_top = __IMAGE_SPACING_TOP_POINTS__
	image_spacing_bottom = __IMAGE_SPACING_BOTTOM_POINTS__
	image_spacing_inside = __IMAGE_SPACING_INSIDE_POINTS__
	image_spacing_outside = __IMAGE_SPACING_OUTSIDE_POINTS__
	layout_mode = "__LAYOUT_MODE__"
	first_page_mode = "__FIRST_PAGE_MODE__"
	if not chapters_dir.exists():
		print(f"chapters dir not found: {chapters_dir}", file=sys.stderr)
		return 2

	chapter_dirs = _chapter_directories(chapters_dir)
	if not chapter_dirs:
		print(f"no chapter directories found in: {chapters_dir}", file=sys.stderr)
		return 2

	chapters = []
	for chapter_dir in chapter_dirs:
		chapter_path = _first_markdown_file(chapter_dir)
		if chapter_path is None:
			continue
		chapter_title, chapter_body = _parse_chapter_markdown(chapter_path)
		chapter_images = _image_files(chapter_dir)
		chapters.append((chapter_title, chapter_body, chapter_images))

	if not chapters:
		print(f"no markdown chapter files found under: {chapters_dir}", file=sys.stderr)
		return 2

	sla_path.parent.mkdir(parents=True, exist_ok=True)

	try:
		import scribus
	except ImportError as exc:
		print(
			"Scribus Python module is not available in this environment; "
			"use the Scribus CLI to run this script from a real Scribus-enabled session.",
			file=sys.stderr,
		)
		print(str(exc), file=sys.stderr)
		return 1

	try:
		_new_document_compat(
			scribus,
			page_size,
			margins,
			layout_mode,
			first_page_mode,
			page_size_constant,
		)
		_create_page_background_compat(scribus, 1, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size)
		title = chapters[0][0] or book_dir.name
		if hasattr(scribus, "setDocTitle"):
			scribus.setDocTitle(title)

		current_page = 1
		page_roles = {1: "chapter_opening"}
		for index, chapter_data in enumerate(chapters, start=1):
			chapter_title, chapter_body, chapter_images = chapter_data
			if index > 1:
				current_page = _start_chapter_on_right_page_compat(scribus, current_page, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
				_goto_page_compat(scribus, current_page)

			current_page = _render_basic_content(
				scribus,
				chapter_title,
				chapter_body,
				chapter_images,
				index,
				current_page,
				page_size,
				margins,
				layout_mode,
				first_page_mode,
				page_background_rgb,
				bleed_inside,
				bleed_outside,
				bleed_top,
				bleed_bottom,
				image_border_rgb,
				image_border_width_pt,
				image_spacing_top,
				image_spacing_bottom,
				image_spacing_inside,
				image_spacing_outside,
				page_roles,
			)

		_render_page_numbers_compat(
			scribus,
			current_page,
			page_roles,
			page_size,
			layout_mode,
			first_page_mode,
			page_numbers_enabled,
			page_number_start_on_page,
			page_number_start_number,
			page_number_format,
			page_number_position,
			page_number_font_name,
			page_number_font_family,
			page_number_font_size_pt,
			page_number_color_rgb,
			page_number_offset_top,
			page_number_offset_bottom,
			page_number_offset_inside,
			page_number_offset_outside,
			page_number_hide_on,
		)

		_goto_page_compat(scribus, 1)
		_save_document_compat(scribus, sla_path)

		if hasattr(scribus, "PDFfile"):
			pdf = scribus.PDFfile()
			pdf.file = str(pdf_path)
			pdf.save()
		else:
			print("Scribus PDF export API is unavailable in this build.", file=sys.stderr)
			return 1
	except Exception as exc:  # pragma: no cover - runtime integration path
		print(f"Scribus API call failed: {exc}", file=sys.stderr)
		return 1

	print(f"wrote Scribus document: {sla_path}")
	print(f"wrote PDF: {pdf_path}")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())
`

func generateScribusScript(cfg config.Config) string {
	pageBackgroundRGB := "None"
	if cfg.PageBackgroundRGB != nil {
		pageBackgroundRGB = fmt.Sprintf("(%d, %d, %d)", cfg.PageBackgroundRGB[0], cfg.PageBackgroundRGB[1], cfg.PageBackgroundRGB[2])
	}
	var err error
	pageNumberHideOnJSON := []byte("[]")
	if cfg.PageNumbers.HideOn != nil {
		pageNumberHideOnJSON, err = json.Marshal(cfg.PageNumbers.HideOn)
	}
	if err != nil {
		pageNumberHideOnJSON = []byte("[]")
	}
	pageNumberFontName := scribusFontName(cfg.PageNumbers.Font.Family, cfg.PageNumbers.Font.Style)

	replacer := strings.NewReplacer(
		"__PAGE_WIDTH_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageWidth)),
		"__PAGE_HEIGHT_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageHeight)),
		"__MARGIN_TOP_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.MarginTop)),
		"__MARGIN_LEFT_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.MarginLeft)),
		"__MARGIN_RIGHT_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.MarginRight)),
		"__MARGIN_BOTTOM_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.MarginBottom)),
		"__PAGE_BACKGROUND_RGB__", pageBackgroundRGB,
		"__PAGE_NUMBERS_ENABLED__", pythonBool(cfg.PageNumbers.Enabled),
		"__PAGE_NUMBER_START_ON_PAGE__", fmt.Sprintf("%d", cfg.PageNumbers.StartOnPage),
		"__PAGE_NUMBER_START_NUMBER__", fmt.Sprintf("%d", cfg.PageNumbers.StartNumber),
		"__PAGE_NUMBER_FORMAT__", fmt.Sprintf("%q", string(cfg.PageNumbers.Format)),
		"__PAGE_NUMBER_POSITION__", fmt.Sprintf("%q", string(cfg.PageNumbers.Position)),
		"__PAGE_NUMBER_FONT_NAME__", fmt.Sprintf("%q", pageNumberFontName),
		"__PAGE_NUMBER_FONT_FAMILY__", fmt.Sprintf("%q", cfg.PageNumbers.Font.Family),
		"__PAGE_NUMBER_FONT_SIZE_PT__", fmt.Sprintf("%.4f", cfg.PageNumbers.Font.SizePt),
		"__PAGE_NUMBER_COLOR_RED__", fmt.Sprintf("%d", cfg.PageNumbers.ColorRGB[0]),
		"__PAGE_NUMBER_COLOR_GREEN__", fmt.Sprintf("%d", cfg.PageNumbers.ColorRGB[1]),
		"__PAGE_NUMBER_COLOR_BLUE__", fmt.Sprintf("%d", cfg.PageNumbers.ColorRGB[2]),
		"__PAGE_NUMBER_OFFSET_TOP_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageNumbers.OffsetMM.Top)),
		"__PAGE_NUMBER_OFFSET_BOTTOM_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageNumbers.OffsetMM.Bottom)),
		"__PAGE_NUMBER_OFFSET_INSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageNumbers.OffsetMM.Inside)),
		"__PAGE_NUMBER_OFFSET_OUTSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.PageNumbers.OffsetMM.Outside)),
		"__PAGE_NUMBER_HIDE_ON__", string(pageNumberHideOnJSON),
		"__BLEED_TOP_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.BleedTop)),
		"__BLEED_BOTTOM_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.BleedBottom)),
		"__BLEED_INSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.BleedInside)),
		"__BLEED_OUTSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.BleedOutside)),
		"__IMAGE_BORDER_RED__", fmt.Sprintf("%d", cfg.ImageBorderRGB[0]),
		"__IMAGE_BORDER_GREEN__", fmt.Sprintf("%d", cfg.ImageBorderRGB[1]),
		"__IMAGE_BORDER_BLUE__", fmt.Sprintf("%d", cfg.ImageBorderRGB[2]),
		"__IMAGE_BORDER_WIDTH_PT__", fmt.Sprintf("%.4f", cfg.ImageBorderPt),
		"__IMAGE_SPACING_TOP_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.ImageSpaceTop)),
		"__IMAGE_SPACING_BOTTOM_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.ImageSpaceBottom)),
		"__IMAGE_SPACING_INSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.ImageSpaceInside)),
		"__IMAGE_SPACING_OUTSIDE_POINTS__", fmt.Sprintf("%.4f", mmToPoints(cfg.ImageSpaceOutside)),
		"__PAGE_SIZE_CONSTANT__", fmt.Sprintf("PAPER_%s", strings.ToUpper(strings.ReplaceAll(cfg.PageSize, " ", "_"))),
		"__LAYOUT_MODE__", cfg.PageLayout,
		"__FIRST_PAGE_MODE__", cfg.FirstPage,
	)

	return replacer.Replace(generatedScribusScriptTemplate)
}

func mmToPoints(mm float64) float64 {
	return mm * 72.0 / 25.4
}

func scribusFontName(family, style string) string {
	family = strings.TrimSpace(family)
	style = strings.TrimSpace(style)
	if family == "" {
		return style
	}
	if style == "" {
		return family
	}
	return family + " " + style
}

func pythonBool(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

func GenerateSLA(cfg config.Config, outputPath string, chapter markdown.Chapter, imagePath string) error {
	_ = cfg
	_ = chapter
	_ = imagePath
	if err := writeGeneratedScribusScript(generatedScribusScriptPath, cfg); err != nil {
		return err
	}
	bookDir := filepath.Dir(filepath.Dir(outputPath))
	cmd := buildScribusInvocation(bookDir)
	return runCommand(cmd)
}

func buildScribusInvocation(bookDir string) []string {
	return []string{"xvfb-run", "-a", "scribus", "-g", "-py", generatedScribusScriptPath, bookDir}
}

func runCommand(cmd []string) error {
	command := exec.Command(cmd[0], cmd[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func writeGeneratedScribusScript(path string, cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(generateScribusScript(cfg)), 0o755)
}
