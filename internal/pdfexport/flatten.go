package pdfexport

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Flatten renders every page to an opaque RGB image in a separate PDF.
// The source is never modified and an existing destination is never replaced.
func Flatten(input, output string, dpi int, log io.Writer) error {
	if dpi < 300 || dpi > 1200 {
		return fmt.Errorf("print resolution must be between 300 and 1200 DPI")
	}
	input, err := filepath.Abs(input)
	if err != nil {
		return err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := checkPDF(input); err != nil {
		return fmt.Errorf("input PDF: %w", err)
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("output already exists: %s (choose a new output path)", output)
	} else if !os.IsNotExist(err) {
		return err
	}
	gs, err := exec.LookPath("gs")
	if err != nil {
		return fmt.Errorf("print PDF conversion requires Ghostscript (gs) with the pdfimage24 device: %w", err)
	}

	temp, err := os.CreateTemp(filepath.Dir(output), ".print-pdf-*.pdf")
	if err != nil {
		return fmt.Errorf("create temporary print PDF: %w", err)
	}
	defer os.Remove(temp.Name())
	if err := temp.Close(); err != nil {
		return err
	}

	// pdfwrite/PDF 1.3 alone does not rasterize opaque vectors. pdfimage24
	// composites the entire page, including transparency, before writing it.
	cmd := exec.Command(gs,
		"-dSAFER", "-dBATCH", "-dNOPAUSE", "-dPDFSTOPONERROR",
		"-sDEVICE=pdfimage24", fmt.Sprintf("-r%d", dpi),
		"-sCompression=Flate",
		"-dTextAlphaBits=4", "-dGraphicsAlphaBits=4",
		"-sOutputFile="+strings.ReplaceAll(temp.Name(), "%", "%%"),
		"-f", input,
	)
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flatten PDF with Ghostscript: %w (source unchanged)", err)
	}
	if err := checkPDF(temp.Name()); err != nil {
		return fmt.Errorf("Ghostscript did not produce a PDF: %w", err)
	}
	// Publish only a completed conversion, without a check/rename overwrite race.
	if err := os.Link(temp.Name(), output); err != nil {
		return fmt.Errorf("publish print PDF without overwriting existing files: %w", err)
	}
	return nil
}

func checkPDF(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	var header [5]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return err
	}
	if string(header[:]) != "%PDF-" {
		return fmt.Errorf("%s does not have a PDF header", path)
	}
	return nil
}
