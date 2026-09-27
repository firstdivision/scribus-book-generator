import os
import sys
import scribus


def _fonts_used_in_document():
    fonts = set()
    if not hasattr(scribus, "getAllObjects") or not hasattr(scribus, "getObjectType") or not hasattr(scribus, "getFont"):
        return fonts

    try:
        object_names = scribus.getAllObjects()
    except Exception:
        return fonts

    for object_name in object_names:
        try:
            if scribus.getObjectType(object_name) != "TextFrame":
                continue
        except Exception:
            continue

        try:
            text_length = scribus.getTextLength(object_name) if hasattr(scribus, "getTextLength") else 0
        except Exception:
            text_length = 0

        if text_length > 0:
            for index in range(text_length):
                try:
                    font_name = scribus.getFont(object_name, index)
                except TypeError:
                    try:
                        font_name = scribus.getFont(object_name)
                    except Exception:
                        font_name = ""
                except Exception:
                    font_name = ""
                if font_name:
                    fonts.add(font_name)
        else:
            try:
                font_name = scribus.getFont(object_name)
            except Exception:
                font_name = ""
            if font_name:
                fonts.add(font_name)

    return fonts


# Scribus PDFOptions::EmbedFonts; the scripter exposes no named constant for fontEmbedding.
PDF_FONT_EMBEDDING_EMBED = 0


def _type1_file_has_binary_eexec(font_path):
    """Scribus hex-decodes the eexec section of every '%!' Type 1 file, corrupting binary eexec data."""
    try:
        with open(font_path, "rb") as handle:
            data = handle.read(1 << 20)
    except OSError:
        return False
    if not data.startswith(b"%!"):
        return False
    marker = data.find(b"eexec")
    if marker < 0:
        return False
    payload = data[marker + len(b"eexec"):].lstrip(b" \t\r\n")
    if not payload:
        return False
    # Adobe Type 1 spec: the eexec section is hex only if its first four bytes are all hex digits.
    return any(byte not in b"0123456789abcdefABCDEF" for byte in payload[:4])


def _require_embeddable_fonts(font_names):
    font_files = {}
    if hasattr(scribus, "getXFontNames"):
        for row in scribus.getXFontNames():
            if len(row) >= 6 and row[0] and row[5]:
                font_files[row[0]] = row[5]
    broken = [
        f"'{font_name}' ({font_files[font_name]})"
        for font_name in sorted(font_names)
        if font_name in font_files and _type1_file_has_binary_eexec(font_files[font_name])
    ]
    if broken:
        raise RuntimeError(
            "Scribus cannot embed these Type 1 fonts because their files store eexec data in binary, "
            "which would produce a corrupt embedded font that printers reject: "
            + ", ".join(broken)
            + ". Install an OpenType, TrueType, or .pfb copy of the font, or choose a different font."
        )


def _configure_pdf_export(pdf):
    if hasattr(pdf, "fontEmbedding"):
        pdf.fontEmbedding = PDF_FONT_EMBEDDING_EMBED
    fonts = _fonts_used_in_document()
    _require_embeddable_fonts(fonts)
    if hasattr(pdf, "fonts"):
        pdf.fonts = sorted(fonts)


print("Python args:", sys.argv)

if len(sys.argv) < 2:
    print("ERROR: No SLA file specified.")
    sys.exit(1)

input_file = os.path.abspath(sys.argv[-1])

if not os.path.exists(input_file):
    print("ERROR: File not found:", input_file)
    sys.exit(1)

print("Opening:", input_file)

scribus.openDoc(input_file)

output_file = os.path.splitext(input_file)[0] + ".pdf"

pdf = scribus.PDFfile()
pdf.file = output_file
_configure_pdf_export(pdf)
pdf.save()

print("Created:", output_file)

scribus.closeDoc()