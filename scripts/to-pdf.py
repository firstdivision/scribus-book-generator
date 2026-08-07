import os
import sys
import scribus

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
pdf.save()

print("Created:", output_file)

scribus.closeDoc()