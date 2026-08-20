#!/usr/bin/env python3
import json
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
	patterns = ["*.png", "*.jpg", "*.jpeg", "*.webp", "*.gif", "*.svg"]
	results = []
	seen = set()
	for pattern in patterns:
		for candidate in sorted(chapter_dir.glob(pattern)):
			if candidate.name.startswith("."):
				continue
			candidate_key = candidate.resolve().as_posix() if candidate.exists() else candidate.as_posix()
			if candidate_key in seen:
				continue
			seen.add(candidate_key)
			results.append(candidate)
	for pattern in patterns:
		for candidate in sorted(chapter_dir.glob(pattern.upper())):
			if candidate.name.startswith("."):
				continue
			candidate_key = candidate.resolve().as_posix() if candidate.exists() else candidate.as_posix()
			if candidate_key in seen:
				continue
			seen.add(candidate_key)
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
	title_found = False
	paragraphs = []

	for line_number, raw_line in enumerate(chapter_path.read_text(encoding="utf-8").splitlines(), start=1):
		line = raw_line.strip()
		if not line:
			continue
		if line.startswith("# "):
			if title_found:
				raise RuntimeError(f"{chapter_path}: additional H1 heading on line {line_number}; only the first H1 may be a chapter title")
			parsed_title = line[2:].strip()
			if not parsed_title:
				raise RuntimeError(f"{chapter_path}: chapter title on line {line_number} must be non-empty")
			title = parsed_title
			title_found = True
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


def _mm_to_points(value_mm):
	return float(value_mm) * 72.0 / 25.4


def _normalize_path_key(path_value):
	return str(Path(path_value)).replace("\\", "/")


def _build_layout_index(layout_plan):
	index = {}
	for entry in layout_plan.get("images", []):
		source = entry.get("file") or entry.get("src")
		if not source:
			continue
		index[_normalize_path_key(source)] = entry
	return index


def _output_filename_stem(layout_plan, book_dir):
	title = str(layout_plan.get("title") or "").strip()
	stem = title or book_dir.name
	stem = stem.replace("/", "-").replace("\\", "-").strip()
	return stem or book_dir.name


def _resolve_image_instruction(layout_index, book_dir, image_path):
	image_resolved = image_path.resolve()
	candidates = [_normalize_path_key(image_resolved)]
	try:
		candidates.append(_normalize_path_key(image_resolved.relative_to(book_dir)))
	except ValueError:
		pass
	candidates.append(_normalize_path_key(image_path))

	for candidate in candidates:
		if candidate in layout_index:
			return layout_index[candidate]

	best = None
	best_len = -1
	for key, value in layout_index.items():
		for candidate in candidates:
			if candidate.endswith(key) and len(key) > best_len:
				best = value
				best_len = len(key)
	return best


def _fit_contain_dimensions(source_width, source_height, max_width, max_height):
	if source_width <= 0 or source_height <= 0 or max_width <= 0 or max_height <= 0:
		return max(max_width, 1.0), max(max_height, 1.0)
	scale = min(max_width / float(source_width), max_height / float(source_height))
	return float(source_width) * scale, float(source_height) * scale


def _resolve_wrap_spacing(is_right_page, spacing_inside, spacing_outside, spacing_top, spacing_bottom):
	if is_right_page:
		return spacing_inside, spacing_outside, spacing_top, spacing_bottom
	return spacing_outside, spacing_inside, spacing_top, spacing_bottom


def _resolve_snap_rect(snap_target, content_rect, trim_rect, bleed_rect):
	if snap_target == "trim":
		return trim_rect
	if snap_target == "bleed":
		return bleed_rect
	return content_rect


def _resolve_semantic_edge(edge_name, is_right_page):
	if edge_name == "outside":
		return "right" if is_right_page else "left"
	if edge_name == "inside":
		return "left" if is_right_page else "right"
	if edge_name == "top":
		return "top"
	if edge_name == "bottom":
		return "bottom"
	raise RuntimeError(f"unsupported image edge: {edge_name}")


def _choose_snap_edge(explicit_edge, allowed_edges, preferred_edges):
	allowed_set = set(allowed_edges)
	if explicit_edge is not None:
		if explicit_edge not in allowed_set:
			raise RuntimeError(f"layout.json snap_edge '{explicit_edge}' is not allowed by images.placement.allowed_edges")
		return explicit_edge

	for edge in preferred_edges:
		if edge in allowed_set:
			return edge

	if allowed_edges:
		return allowed_edges[0]
	raise RuntimeError("images.placement.allowed_edges must not be empty")


def _snap_frame_to_edge(snap_rect, frame_width, frame_height, physical_edge, edge_gap, is_right_page, spacing_left, spacing_right, spacing_top, spacing_bottom):
	left, top, rect_width, rect_height = snap_rect
	if physical_edge == "left":
		return left + edge_gap + spacing_left, top + spacing_top
	if physical_edge == "right":
		return left + rect_width - frame_width - edge_gap - spacing_right, top + spacing_top
	if physical_edge == "top":
		x = left + rect_width - frame_width - spacing_right if is_right_page else left + spacing_left
		return x, top + edge_gap + spacing_top
	if physical_edge == "bottom":
		x = left + rect_width - frame_width - spacing_right if is_right_page else left + spacing_left
		return x, top + rect_height - frame_height - edge_gap - spacing_bottom
	raise RuntimeError(f"unsupported physical edge: {physical_edge}")


def _create_wrap_frame_compat(scribus, frame_name, x, y, width, height):
	wrap_name = f"{frame_name}_wrap"
	wrap_frame = _create_rect_compat(scribus, x, y, width, height, wrap_name)
	_set_fill_none_compat(scribus, wrap_frame)
	_set_line_none_compat(scribus, wrap_frame)
	_set_text_flow_mode_compat(scribus, wrap_frame)
	_set_text_distances_compat(scribus, wrap_frame, 0)
	return wrap_frame


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


def _ensure_chapter_heading_color_compat(scribus, rgb_values):
	color_name = f"chapter_heading_{rgb_values[0]}_{rgb_values[1]}_{rgb_values[2]}"
	if hasattr(scribus, "getColorNames"):
		try:
			if color_name in scribus.getColorNames():
				return color_name
		except Exception:
			pass
	if not hasattr(scribus, "defineColorRGB"):
		raise RuntimeError("Scribus color creation API is unavailable")
	scribus.defineColorRGB(color_name, rgb_values[0], rgb_values[1], rgb_values[2])
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


def _resolve_border_override(image_instruction, default_border_rgb, default_border_width_pt):
	border_rgb = default_border_rgb
	border_width_pt = default_border_width_pt

	if not image_instruction:
		return border_rgb, border_width_pt

	border_override = image_instruction.get("border")
	if not isinstance(border_override, dict):
		return border_rgb, border_width_pt

	color_rgb = border_override.get("color_rgb")
	if isinstance(color_rgb, list) and len(color_rgb) == 3:
		try:
			border_rgb = (int(color_rgb[0]), int(color_rgb[1]), int(color_rgb[2]))
		except Exception:
			pass

	width_pt = border_override.get("width_pt")
	if width_pt is not None:
		try:
			border_width_pt = float(width_pt)
		except Exception:
			pass

	return border_rgb, border_width_pt


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


def _require_font_compat(scribus, font_name):
	if not hasattr(scribus, "getFontNames"):
		return
	try:
		available_fonts = scribus.getFontNames()
	except Exception as exc:
		raise RuntimeError(f"Unable to inspect Scribus fonts while validating '{font_name}'") from exc
	if font_name not in available_fonts:
		raise RuntimeError(f"Configured chapter heading font '{font_name}' is not available in Scribus")


def _chapter_heading_style_name(alignment_name):
	return f"Chapter Heading {alignment_name.title()}"


def _ensure_chapter_heading_styles_compat(scribus, font_name, font_size_pt, color_rgb):
	_require_font_compat(scribus, font_name)
	color_name = _ensure_chapter_heading_color_compat(scribus, color_rgb)
	character_style_name = "Chapter Heading Characters"
	if not hasattr(scribus, "createCharStyle") or not hasattr(scribus, "createParagraphStyle"):
		raise RuntimeError("Scribus paragraph/character style creation API is unavailable")

	existing_character_styles = []
	if hasattr(scribus, "getCharStyles"):
		existing_character_styles = scribus.getCharStyles()
	if character_style_name not in existing_character_styles:
		try:
			scribus.createCharStyle(
				name=character_style_name,
				font=font_name,
				fontsize=font_size_pt,
				fillcolor=color_name,
			)
		except TypeError:
			scribus.createCharStyle(character_style_name, font_name, font_size_pt, "", color_name)

	existing_paragraph_styles = []
	if hasattr(scribus, "getParagraphStyles"):
		existing_paragraph_styles = scribus.getParagraphStyles()
	for alignment_name in ("left", "center", "right"):
		style_name = _chapter_heading_style_name(alignment_name)
		if style_name in existing_paragraph_styles:
			continue
		alignment_map = {
			"left": getattr(scribus, "ALIGN_LEFT", 0),
			"center": getattr(scribus, "ALIGN_CENTERED", 1),
			"right": getattr(scribus, "ALIGN_RIGHT", 2),
		}
		try:
			scribus.createParagraphStyle(
				name=style_name,
				alignment=alignment_map[alignment_name],
				charstyle=character_style_name,
			)
		except TypeError:
			scribus.createParagraphStyle(style_name, 0, 15, alignment_map[alignment_name], 0, 0, 0, 0, 0, 0, 2, 0, character_style_name)


def _set_paragraph_style_compat(scribus, frame_name, style_name):
	if not hasattr(scribus, "setParagraphStyle"):
		raise RuntimeError("Scribus paragraph style application API is unavailable")
	try:
		scribus.setParagraphStyle(style_name, frame_name)
		return
	except TypeError:
		pass
	try:
		scribus.setParagraphStyle(frame_name, style_name)
	except Exception as exc:
		raise RuntimeError(f"Unable to apply Scribus paragraph style '{style_name}'") from exc


def _resolve_chapter_heading_alignment(alignment_name, is_right_page):
	if alignment_name in ("left", "center", "right"):
		return alignment_name
	if alignment_name == "inside":
		return "left" if is_right_page else "right"
	if alignment_name == "outside":
		return "right" if is_right_page else "left"
	raise RuntimeError(f"unsupported chapter heading alignment: {alignment_name}")


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


def _image_is_ignored(image_instruction):
	return bool(image_instruction and image_instruction.get("placement") == "ignore")


def _image_is_full_page(image_instruction):
	return bool(image_instruction and (image_instruction.get("placement") == "full_page" or image_instruction.get("bleed") is True))


def _placeable_images(image_paths, layout_index, book_dir):
	results = []
	for image_path in image_paths:
		instruction = _resolve_image_instruction(layout_index, book_dir, image_path)
		if _image_is_ignored(instruction):
			continue
		results.append(image_path)
	return results


def _gallery_page_geometry(columns, content_width, content_height, gap_x, gap_y):
	columns = max(1, int(columns))
	gap_x = max(0.0, float(gap_x))
	gap_y = max(0.0, float(gap_y))
	content_width = max(1.0, float(content_width))
	content_height = max(1.0, float(content_height))

	cell_width = (content_width - gap_x * (columns - 1)) / float(columns)
	if cell_width <= 0:
		columns = 1
		cell_width = content_width

	cell_height = min(cell_width, content_height)
	stride_y = cell_height + gap_y
	if stride_y <= 0:
		rows = 1
	else:
		rows = max(1, int((content_height + gap_y) // stride_y))

	return columns, rows, cell_width, cell_height


def _gallery_cell_rects(image_count, columns, content_x, content_y, content_width, content_height, gap_x, gap_y):
	columns, _rows, cell_width, cell_height = _gallery_page_geometry(columns, content_width, content_height, gap_x, gap_y)
	rects = []
	for index in range(max(0, int(image_count))):
		row = index // columns
		col = index % columns
		x = content_x + col * (cell_width + gap_x)
		y = content_y + row * (cell_height + gap_y)
		rects.append((x, y, cell_width, cell_height))
	return rects


def _place_chapter_image(scribus, image_path, image_index, chapter_index, page_number, page_size, margins, layout_mode, first_page_mode, bleed_inside, bleed_outside, bleed_top, bleed_bottom, image_body_top, image_body_height, image_border_rgb, image_border_width_pt, image_spacing_top, image_spacing_bottom, image_spacing_inside, image_spacing_outside, image_max_width, image_max_height, image_snap_to_edge, image_snap_target, image_allowed_edges, image_preferred_edges, image_edge_gap, layout_index, book_dir, page_roles):
	page_width, page_height = _document_page_size_compat(scribus, page_size)
	margin_top, margin_left, margin_right, margin_bottom = margins
	content_width = page_width - margin_left - margin_right
	is_right_page = layout_mode == "facing_pages" and _page_is_right_compat(layout_mode, first_page_mode, page_number)
	image_spacing_left, image_spacing_right, image_spacing_top_used, image_spacing_bottom_used = _resolve_wrap_spacing(
		is_right_page,
		image_spacing_inside,
		image_spacing_outside,
		image_spacing_top,
		image_spacing_bottom,
	)

	image_instruction = _resolve_image_instruction(layout_index, book_dir, image_path)
	image_border_rgb_used, image_border_width_pt_used = _resolve_border_override(
		image_instruction,
		image_border_rgb,
		image_border_width_pt,
	)
	is_full_page = _image_is_full_page(image_instruction)
	image_width, image_height = _image_dimensions_compat(image_path)

	if is_full_page:
		page_roles[page_number] = "full_page_image"
		left_bleed, right_bleed = _page_horizontal_bleeds(layout_mode, first_page_mode, page_number, bleed_inside, bleed_outside)
		trim_rect = (0.0, 0.0, page_width, page_height)
		bleed_rect = (
			-left_bleed,
			-bleed_top,
			page_width + left_bleed + right_bleed,
			page_height + bleed_top + bleed_bottom,
		)
		target_rect = bleed_rect if image_instruction and image_instruction.get("bleed") else trim_rect
		image_x, image_y, frame_width, frame_height = target_rect
	else:
		content_rect = (margin_left, image_body_top, content_width, image_body_height)
		left_bleed, right_bleed = _page_horizontal_bleeds(layout_mode, first_page_mode, page_number, bleed_inside, bleed_outside)
		trim_rect = (0.0, 0.0, page_width, page_height)
		bleed_rect = (
			-left_bleed,
			-bleed_top,
			page_width + left_bleed + right_bleed,
			page_height + bleed_top + bleed_bottom,
		)
		snap_rect = _resolve_snap_rect(image_snap_target, content_rect, trim_rect, bleed_rect)

		override_width = None
		override_height = None
		explicit_edge = None
		if image_instruction:
			override_width_mm = image_instruction.get("width_mm")
			override_height_mm = image_instruction.get("height_mm")
			if override_width_mm is not None:
				override_width = _mm_to_points(override_width_mm)
			if override_height_mm is not None:
				override_height = _mm_to_points(override_height_mm)
			explicit_edge = image_instruction.get("snap_edge")

		chosen_edge = None
		physical_edge = None
		if image_snap_to_edge:
			chosen_edge = _choose_snap_edge(explicit_edge, image_allowed_edges, image_preferred_edges)
			physical_edge = _resolve_semantic_edge(chosen_edge, is_right_page)

		available_width = max(1.0, snap_rect[2] - image_spacing_left - image_spacing_right)
		available_height = max(1.0, snap_rect[3] - image_spacing_top_used - image_spacing_bottom_used)
		if physical_edge in ("left", "right"):
			available_width = max(1.0, available_width - image_edge_gap)
		if physical_edge in ("top", "bottom"):
			available_height = max(1.0, available_height - image_edge_gap)

		max_width = min(image_max_width, available_width)
		max_height = min(image_max_height, available_height)

		if override_width is not None and override_height is not None:
			frame_width, frame_height = _fit_contain_dimensions(image_width, image_height, override_width, override_height)
		elif override_width is not None and image_width > 0 and image_height > 0:
			frame_width = override_width
			frame_height = frame_width * (float(image_height) / float(image_width))
		elif override_height is not None and image_width > 0 and image_height > 0:
			frame_height = override_height
			frame_width = frame_height * (float(image_width) / float(image_height))
		else:
			frame_width, frame_height = _fit_contain_dimensions(image_width, image_height, max_width, max_height)

		if image_snap_to_edge:
			image_x, image_y = _snap_frame_to_edge(
				snap_rect,
				frame_width,
				frame_height,
				physical_edge,
				image_edge_gap,
				is_right_page,
				image_spacing_left,
				image_spacing_right,
				image_spacing_top_used,
				image_spacing_bottom_used,
			)
		else:
			image_x = margin_left + image_spacing_left
			image_y = image_body_top + image_spacing_top_used

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
	_apply_image_frame_style_compat(scribus, image_frame, image_border_rgb_used, image_border_width_pt_used)
	if not is_full_page:
		_set_text_flow_mode_compat(scribus, image_frame)
		_set_text_distances_sides_compat(
			scribus,
			image_frame,
			image_spacing_left,
			image_spacing_right,
			image_spacing_top_used,
			image_spacing_bottom_used,
		)
		_create_wrap_frame_compat(
			scribus,
			f"chapter_{chapter_index}_image_{image_index}",
			image_x - image_spacing_left,
			image_y - image_spacing_top_used,
			frame_width + image_spacing_left + image_spacing_right,
			frame_height + image_spacing_top_used + image_spacing_bottom_used,
		)
	else:
		_set_text_flow_mode_compat(scribus, image_frame)


def _place_gallery_pages(scribus, gallery_images, placed_count, chapter_index, current_page, page_size, margins, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, image_border_rgb, image_border_width_pt, image_spacing_top, image_spacing_bottom, image_spacing_inside, image_spacing_outside, gallery_columns, layout_index, book_dir, page_roles):
	if not gallery_images:
		return current_page, placed_count

	page_width, page_height = _document_page_size_compat(scribus, page_size)
	margin_top, margin_left, margin_right, margin_bottom = margins
	content_x = margin_left
	content_y = margin_top
	content_width = page_width - margin_left - margin_right
	content_height = page_height - margin_top - margin_bottom
	gap_x = max(image_spacing_inside, image_spacing_outside)
	gap_y = max(image_spacing_top, image_spacing_bottom)
	columns, rows, _cell_width, _cell_height = _gallery_page_geometry(gallery_columns, content_width, content_height, gap_x, gap_y)
	capacity = max(1, columns * rows)

	offset = 0
	while offset < len(gallery_images):
		current_page = _append_body_page_compat(scribus, current_page, "chapter_gallery", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
		_goto_page_compat(scribus, current_page)
		batch = gallery_images[offset:offset + capacity]
		rects = _gallery_cell_rects(len(batch), columns, content_x, content_y, content_width, content_height, gap_x, gap_y)
		for image_path, cell in zip(batch, rects):
			placed_count += 1
			cell_x, cell_y, cell_width, cell_height = cell
			image_instruction = _resolve_image_instruction(layout_index, book_dir, image_path)
			image_border_rgb_used, image_border_width_pt_used = _resolve_border_override(
				image_instruction,
				image_border_rgb,
				image_border_width_pt,
			)
			source_width, source_height = _image_dimensions_compat(image_path)
			frame_width, frame_height = _fit_contain_dimensions(source_width, source_height, cell_width, cell_height)
			image_x = cell_x + (cell_width - frame_width) / 2.0
			image_y = cell_y + (cell_height - frame_height) / 2.0
			image_frame = _create_image_frame_compat(
				scribus,
				image_x,
				image_y,
				frame_width,
				frame_height,
				f"chapter_{chapter_index}_image_{placed_count}",
			)
			_load_image_compat(scribus, image_path, image_frame)
			_set_scale_image_to_frame_compat(scribus, image_frame)
			_apply_image_frame_style_compat(scribus, image_frame, image_border_rgb_used, image_border_width_pt_used)
		offset += capacity

	return current_page, placed_count


def _render_basic_content(scribus, title_text, body_text, image_paths, chapter_index, start_page, page_size, margins, layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, chapter_heading_font_size_pt, chapter_heading_alignment, chapter_heading_spacing_top, chapter_heading_spacing_bottom, image_border_rgb, image_border_width_pt, image_spacing_top, image_spacing_bottom, image_spacing_inside, image_spacing_outside, image_max_width, image_max_height, image_snap_to_edge, image_snap_target, image_allowed_edges, image_preferred_edges, image_edge_gap, gallery_columns, layout_index, book_dir, page_roles):
	page_width, page_height = _document_page_size_compat(scribus, page_size)
	margin_top, margin_left, margin_right, margin_bottom = margins

	content_width = page_width - margin_left - margin_right
	title_height = max(chapter_heading_font_size_pt * 1.5, chapter_heading_font_size_pt + 4.0)
	title_top = margin_top + chapter_heading_spacing_top
	chapter_opening_body_top = title_top + title_height + chapter_heading_spacing_bottom
	chapter_opening_body_height = page_height - chapter_opening_body_top - margin_bottom
	continuation_body_top = margin_top
	continuation_body_height = page_height - continuation_body_top - margin_bottom

	title_frame = _create_text_frame_compat(
		scribus,
		margin_left,
		title_top,
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
	placeable_images = _placeable_images(image_paths, layout_index, book_dir)
	in_flow_index = 0
	placed_count = 0
	current_page = start_page

	if in_flow_index < len(placeable_images):
		placed_count += 1
		_place_chapter_image(
			scribus,
			placeable_images[in_flow_index],
			placed_count,
			chapter_index,
			current_page,
			page_size,
			margins,
			layout_mode,
			first_page_mode,
			bleed_inside,
			bleed_outside,
			bleed_top,
			bleed_bottom,
			chapter_opening_body_top,
			chapter_opening_body_height,
			image_border_rgb,
			image_border_width_pt,
			image_spacing_top,
			image_spacing_bottom,
			image_spacing_inside,
			image_spacing_outside,
			image_max_width,
			image_max_height,
			image_snap_to_edge,
			image_snap_target,
			image_allowed_edges,
			image_preferred_edges,
			image_edge_gap,
			layout_index,
			book_dir,
			page_roles,
		)
		in_flow_index += 1

	_set_frame_text_compat(scribus, title_frame, title_text)
	physical_heading_alignment = _resolve_chapter_heading_alignment(
		chapter_heading_alignment,
		_page_is_right_compat(layout_mode, first_page_mode, start_page),
	)
	_set_paragraph_style_compat(scribus, title_frame, _chapter_heading_style_name(physical_heading_alignment))
	_set_frame_text_compat(scribus, body_frame, body_text)

	while _text_overflows_compat(scribus, body_frames[-1]) and in_flow_index < len(placeable_images):
		current_page = _append_body_page_compat(scribus, current_page, "body", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
		_goto_page_compat(scribus, current_page)
		placed_count += 1
		continuation_frame = _create_text_frame_compat(
			scribus,
			margin_left,
			continuation_body_top,
			content_width,
			continuation_body_height,
			f"chapter_{chapter_index}_body_image_{placed_count}",
		)
		_link_text_frames_compat(scribus, body_frames[-1], continuation_frame)
		body_frames.append(continuation_frame)
		_place_chapter_image(
			scribus,
			placeable_images[in_flow_index],
			placed_count,
			chapter_index,
			current_page,
			page_size,
			margins,
			layout_mode,
			first_page_mode,
			bleed_inside,
			bleed_outside,
			bleed_top,
			bleed_bottom,
			continuation_body_top,
			continuation_body_height,
			image_border_rgb,
			image_border_width_pt,
			image_spacing_top,
			image_spacing_bottom,
			image_spacing_inside,
			image_spacing_outside,
			image_max_width,
			image_max_height,
			image_snap_to_edge,
			image_snap_target,
			image_allowed_edges,
			image_preferred_edges,
			image_edge_gap,
			layout_index,
			book_dir,
			page_roles,
		)
		in_flow_index += 1

	# If Scribus reports overflow after in-flow images are exhausted, grow the chain incrementally.
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

	leftover_images = placeable_images[in_flow_index:]
	leftover_full_page = []
	gallery_images = []
	for leftover_path in leftover_images:
		instruction = _resolve_image_instruction(layout_index, book_dir, leftover_path)
		if _image_is_full_page(instruction):
			leftover_full_page.append(leftover_path)
		else:
			gallery_images.append(leftover_path)

	for leftover_path in leftover_full_page:
		current_page = _append_body_page_compat(scribus, current_page, "full_page_image", layout_mode, first_page_mode, page_background_rgb, bleed_inside, bleed_outside, bleed_top, bleed_bottom, page_size, page_roles)
		_goto_page_compat(scribus, current_page)
		placed_count += 1
		_place_chapter_image(
			scribus,
			leftover_path,
			placed_count,
			chapter_index,
			current_page,
			page_size,
			margins,
			layout_mode,
			first_page_mode,
			bleed_inside,
			bleed_outside,
			bleed_top,
			bleed_bottom,
			continuation_body_top,
			continuation_body_height,
			image_border_rgb,
			image_border_width_pt,
			image_spacing_top,
			image_spacing_bottom,
			image_spacing_inside,
			image_spacing_outside,
			image_max_width,
			image_max_height,
			image_snap_to_edge,
			image_snap_target,
			image_allowed_edges,
			image_preferred_edges,
			image_edge_gap,
			layout_index,
			book_dir,
			page_roles,
		)

	current_page, placed_count = _place_gallery_pages(
		scribus,
		gallery_images,
		placed_count,
		chapter_index,
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
		gallery_columns,
		layout_index,
		book_dir,
		page_roles,
	)

	return current_page


def _as_rgb(value):
	if value is None:
		return None
	return tuple(value)


def _load_job(job_path: Path):
	return json.loads(job_path.read_text(encoding="utf-8"))


def main() -> int:
	if len(sys.argv) < 2:
		print("usage: scribus_generate.py <book-dir> [job-json]", file=sys.stderr)
		return 2

	book_dir = Path(sys.argv[1]).resolve()
	job_path = Path(sys.argv[2]).resolve() if len(sys.argv) > 2 else book_dir / "out" / "scribus-job.json"
	try:
		job = _load_job(job_path)
	except FileNotFoundError:
		print(f"scribus job file not found: {job_path}", file=sys.stderr)
		print("Run bookgen to write the job JSON, or pass the job path as the second argument.", file=sys.stderr)
		return 2
	except json.JSONDecodeError as exc:
		print(f"invalid scribus job JSON: {job_path}: {exc}", file=sys.stderr)
		return 2

	chapters_dir = book_dir / "chapters"
	layout_plan = job.get("layout") or {"images": []}
	output_stem = _output_filename_stem(layout_plan, book_dir)
	sla_path = book_dir / "out" / f"{output_stem}.sla"
	pdf_path = book_dir / "out" / f"{output_stem}.pdf"

	page = job["page"]
	page_size = tuple(page["size_points"])
	margins = tuple(page["margins_points"])
	page_size_constant = page["size_constant"]
	page_background_rgb = _as_rgb(page.get("background_rgb"))
	layout_mode = page["layout"]
	first_page_mode = page["first_page"]

	page_numbers = job["page_numbers"]
	page_numbers_enabled = page_numbers["enabled"]
	page_number_start_on_page = page_numbers["start_on_page"]
	page_number_start_number = page_numbers["start_number"]
	page_number_format = page_numbers["format"]
	page_number_position = page_numbers["position"]
	page_number_font_name = page_numbers["font_name"]
	page_number_font_family = page_numbers["font_family"]
	page_number_font_size_pt = page_numbers["font_size_pt"]
	page_number_color_rgb = tuple(page_numbers["color_rgb"])
	page_number_offsets = page_numbers["offset_points"]
	page_number_offset_top = page_number_offsets["top"]
	page_number_offset_bottom = page_number_offsets["bottom"]
	page_number_offset_inside = page_number_offsets["inside"]
	page_number_offset_outside = page_number_offsets["outside"]
	page_number_hide_on = page_numbers.get("hide_on") or []

	headings = job["chapter_headings"]
	chapter_heading_font_name = headings["font_name"]
	chapter_heading_font_size_pt = headings["font_size_pt"]
	chapter_heading_color_rgb = tuple(headings["color_rgb"])
	chapter_heading_alignment = headings["alignment"]
	heading_spacing = headings["spacing_points"]
	chapter_heading_spacing_top = heading_spacing["top"]
	chapter_heading_spacing_bottom = heading_spacing["bottom"]

	bleed = job["bleed_points"]
	bleed_top = bleed["top"]
	bleed_bottom = bleed["bottom"]
	bleed_inside = bleed["inside"]
	bleed_outside = bleed["outside"]

	images = job["images"]
	image_border_rgb = tuple(images["border_rgb"])
	image_border_width_pt = images["border_width_pt"]
	image_spacing = images["spacing_points"]
	image_spacing_top = image_spacing["top"]
	image_spacing_bottom = image_spacing["bottom"]
	image_spacing_inside = image_spacing["inside"]
	image_spacing_outside = image_spacing["outside"]
	image_max_width = images["max_width_points"]
	image_max_height = images["max_height_points"]
	image_snap_to_edge = images["snap_to_edge"]
	image_snap_target = images["snap_target"]
	image_allowed_edges = images.get("allowed_edges") or []
	image_preferred_edges = images.get("preferred_edges") or []
	image_edge_gap = images["edge_gap_points"]
	gallery_columns = images.get("gallery_columns") or 2
	layout_index = _build_layout_index(layout_plan)
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
		_ensure_chapter_heading_styles_compat(scribus, chapter_heading_font_name, chapter_heading_font_size_pt, chapter_heading_color_rgb)
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
				chapter_heading_font_size_pt,
				chapter_heading_alignment,
				chapter_heading_spacing_top,
				chapter_heading_spacing_bottom,
				image_border_rgb,
				image_border_width_pt,
				image_spacing_top,
				image_spacing_bottom,
				image_spacing_inside,
				image_spacing_outside,
				image_max_width,
				image_max_height,
				image_snap_to_edge,
				image_snap_target,
				image_allowed_edges,
				image_preferred_edges,
				image_edge_gap,
				gallery_columns,
				layout_index,
				book_dir,
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
