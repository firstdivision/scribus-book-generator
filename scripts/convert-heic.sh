#!/usr/bin/env bash
# Converts HEIC images under a book's chapters/ tree to JPG in place, keeping the originals.
set -euo pipefail

usage() {
	echo "usage: ${0##*/} <book-dir> [--quality N] [--force]" >&2
	echo "  <book-dir>    book directory containing a chapters/ folder" >&2
	echo "  --quality N   JPEG quality 1-100 (default: 92)" >&2
	echo "  --force       overwrite existing .jpg files" >&2
}

book_dir=""
quality=92
force=0

while [[ $# -gt 0 ]]; do
	case "$1" in
		-h|--help)
			usage
			exit 0
			;;
		--quality)
			[[ $# -ge 2 ]] || { echo "error: --quality requires a value" >&2; exit 2; }
			quality="$2"
			shift 2
			;;
		--force)
			force=1
			shift
			;;
		-*)
			echo "error: unknown option: $1" >&2
			usage
			exit 2
			;;
		*)
			[[ -z "$book_dir" ]] || { echo "error: unexpected argument: $1" >&2; exit 2; }
			book_dir="$1"
			shift
			;;
	esac
done

if [[ -z "$book_dir" ]]; then
	usage
	exit 2
fi

if [[ ! "$quality" =~ ^[0-9]+$ ]] || (( quality < 1 || quality > 100 )); then
	echo "error: --quality must be an integer between 1 and 100" >&2
	exit 2
fi

chapters_dir="$book_dir/chapters"
if [[ ! -d "$chapters_dir" ]]; then
	echo "error: chapters dir not found: $chapters_dir" >&2
	exit 2
fi

converter=""
if command -v heif-convert >/dev/null 2>&1; then
	converter="heif-convert"
elif command -v magick >/dev/null 2>&1; then
	converter="magick"
elif command -v convert >/dev/null 2>&1; then
	converter="convert"
else
	echo "error: no HEIC converter found; install libheif-examples (heif-convert) or ImageMagick" >&2
	exit 1
fi

convert_one() {
	local src="$1" dst="$2"
	case "$converter" in
		heif-convert) heif-convert -q "$quality" "$src" "$dst" >/dev/null ;;
		magick) magick "$src" -quality "$quality" "$dst" ;;
		convert) convert "$src" -quality "$quality" "$dst" ;;
	esac
}

converted=0
skipped=0
failed=0

echo "converting HEIC images in $chapters_dir (converter: $converter, quality: $quality)"

while IFS= read -r -d '' src; do
	dst="${src%.*}.jpg"
	if [[ -e "$dst" && $force -eq 0 ]]; then
		echo "skip (exists): $dst"
		skipped=$((skipped + 1))
		continue
	fi
	if convert_one "$src" "$dst"; then
		echo "converting $(basename "$src") to $(basename "$dst")"
		converted=$((converted + 1))
	else
		echo "error: failed to convert: $src" >&2
		failed=$((failed + 1))
	fi
done < <(find "$chapters_dir" -type f \( -iname '*.heic' -o -iname '*.heif' \) -print0 | sort -z)

echo "converted=$converted skipped=$skipped failed=$failed (converter: $converter)"

if (( failed > 0 )); then
	exit 1
fi
