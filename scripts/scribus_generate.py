#!/usr/bin/env python3
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


def _document_page_size_compat(scribus, fallback_page_size):
	if hasattr(scribus, "getPageSize"):
		try:
			page_size = scribus.getPageSize()
			if isinstance(page_size, tuple) and len(page_size) >= 2:
				return float(page_size[0]), float(page_size[1])
		except Exception:
			pass

	return fallback_page_size


def _estimate_body_pages(body_text):
	chars_per_page = 1700
	if not body_text:
		return 1
	estimated = (len(body_text) + chars_per_page - 1) // chars_per_page
	return max(1, min(40, estimated))


def _append_page_compat(scribus):
	if hasattr(scribus, "newPage"):
		for args in ((-1,), (1,), tuple()):
			try:
				scribus.newPage(*args)
				return
			except TypeError:
				continue
	if hasattr(scribus, "createPage"):
		for args in ((-1,), (1,), tuple()):
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


def _start_chapter_on_right_page_compat(scribus, current_page):
	if current_page % 2 == 1:
		_append_page_compat(scribus)
		current_page += 1

	_append_page_compat(scribus)
	current_page += 1
	return current_page



def _render_basic_content(scribus, title_text, body_text, image_paths, chapter_index, start_page, page_size, margins):
	page_width, page_height = _document_page_size_compat(scribus, page_size)
	margin_top, margin_left, margin_right, margin_bottom = margins

	content_width = page_width - margin_left - margin_right
	title_height = 64.0
	body_top = margin_top + title_height + 12.0
	body_height = page_height - body_top - margin_bottom
	chapter_image_width = content_width * 0.72
	chapter_image_top = body_top + 20.0
	chapter_image_gap = 20.0
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
		body_top,
		content_width,
		body_height,
		f"chapter_{chapter_index}_body",
	)
	body_frames = [body_frame]

	for image_index, image_path in enumerate(image_paths, start=1):
		if image_index > 1:
			_append_page_compat(scribus)
			chapter_image_cursor += 1
			_goto_page_compat(scribus, chapter_image_cursor)

		image_width, image_height = _image_dimensions_compat(image_path)
		if image_width > 0 and image_height > 0:
			frame_height = min(260.0, body_height * 0.45)
			frame_width = frame_height * (float(image_width) / float(image_height))
			if frame_width > chapter_image_width:
				frame_width = chapter_image_width
				frame_height = frame_width * (float(image_height) / float(image_width))
		else:
			frame_width = chapter_image_width
			frame_height = min(260.0, body_height * 0.45)

		image_x = margin_left + ((content_width - frame_width) / 2.0)
		image_y = chapter_image_top + ((body_height - frame_height) / 2.0)

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
		_set_text_distances_compat(scribus, image_frame, 12.0)

	_set_frame_text_compat(scribus, title_frame, title_text)
	_set_frame_text_compat(scribus, body_frame, body_text)

	current_page = start_page

	# If Scribus reports overflow, grow the chain incrementally.
	max_extra_pages = 20
	while _text_overflows_compat(scribus, body_frames[-1]) and max_extra_pages > 0:
		_append_page_compat(scribus)
		current_page += 1
		_goto_page_compat(scribus, current_page)
		next_page_number = len(body_frames) + 1
		next_frame = _create_text_frame_compat(
			scribus,
			margin_left,
			body_top,
			content_width,
			body_height,
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

	page_size = (841.8898, 595.2756)
	margins = (
		36.0000,
		36.0000,
		36.0000,
		36.0000,
	)
	page_size_constant = "PAPER_A4"
	layout_mode = "facing_pages"
	first_page_mode = "right"
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
		title = chapters[0][0] or book_dir.name
		if hasattr(scribus, "setDocTitle"):
			scribus.setDocTitle(title)

		current_page = 1
		for index, chapter_data in enumerate(chapters, start=1):
			chapter_title, chapter_body, chapter_images = chapter_data
			if index > 1:
				current_page = _start_chapter_on_right_page_compat(scribus, current_page)
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
