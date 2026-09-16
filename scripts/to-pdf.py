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
                    font_name = scribus.getFont(index, object_name)
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


def _configure_pdf_export(pdf):
    if hasattr(pdf, "fontEmbedding"):
        pdf.fontEmbedding = 0
    if hasattr(pdf, "fonts"):
        pdf.fonts = sorted(_fonts_used_in_document())


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