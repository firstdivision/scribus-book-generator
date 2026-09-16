package pdfexport

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestFlattenRejectsInvalidInputs(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dpi := range []int{-1, 0, 299, 1201} {
		if err := Flatten(input, filepath.Join(dir, "output.pdf"), dpi, io.Discard); err == nil {
			t.Errorf("accepted DPI %d", dpi)
		}
	}
	for _, path := range []string{filepath.Join(dir, "missing.pdf"), dir} {
		if err := Flatten(path, filepath.Join(dir, "output.pdf"), 300, io.Discard); err == nil {
			t.Errorf("accepted input %s", path)
		}
	}
	if err := os.WriteFile(input, []byte("not a PDF"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Flatten(input, filepath.Join(dir, "output.pdf"), 300, io.Discard); err == nil {
		t.Fatal("accepted a non-PDF")
	}
}

func TestFlattenNeverOverwritesFiles(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	content := []byte("%PDF-1.4\noriginal")
	if err := os.WriteFile(input, content, 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "existing.pdf")
	if err := os.WriteFile(output, content, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.pdf")
	if err := os.Symlink(input, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{input, output, link} {
		if err := Flatten(input, path, 300, io.Discard); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("expected existing-output error for %s, got %v", path, err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, content) {
			t.Fatalf("changed existing file %s: %v", path, err)
		}
	}
}

func TestFlattenMissingGhostscript(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	err := Flatten(input, filepath.Join(dir, "output.pdf"), 300, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "requires Ghostscript") {
		t.Fatalf("expected actionable dependency error, got %v", err)
	}
}

func TestFlattenFailureDoesNotPublishPartialOutput(t *testing.T) {
	for _, exitCode := range []int{0, 1} {
		t.Run(fmt.Sprint(exitCode), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "source.pdf")
			content := []byte("%PDF-1.4\noriginal")
			if err := os.WriteFile(input, content, 0o600); err != nil {
				t.Fatal(err)
			}
			stub := fmt.Sprintf(`#!/bin/sh
for arg do
    case "$arg" in -sOutputFile=*) output=${arg#*=};; esac
done
printf 'incomplete' > "$output"
exit %d
`, exitCode)
			if err := os.WriteFile(filepath.Join(dir, "gs"), []byte(stub), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			output := filepath.Join(dir, "output.pdf")
			if err := Flatten(input, output, 300, io.Discard); err == nil {
				t.Fatal("accepted failed or invalid conversion")
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("partial output exists: %v", err)
			}
			temps, err := filepath.Glob(filepath.Join(dir, ".print-pdf-*"))
			if err != nil || len(temps) != 0 {
				t.Fatalf("temporary files not cleaned: %v, %v", temps, err)
			}
			got, err := os.ReadFile(input)
			if err != nil || !bytes.Equal(got, content) {
				t.Fatalf("source changed: %v", err)
			}
		})
	}
}

func TestFlattenGhostscript(t *testing.T) {
	gs, err := exec.LookPath("gs")
	if err != nil {
		t.Skip("Ghostscript is not installed")
	}
	dir := filepath.Join(t.TempDir(), "book with spaces and 100% color")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "-source.pdf")
	source := transparencyPDF()
	if err := os.WriteFile(input, source, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dpi := range []int{300, 600} {
		output := filepath.Join(dir, fmt.Sprintf("print-%d.pdf", dpi))
		var log bytes.Buffer
		if err := Flatten(input, output, dpi, &log); err != nil {
			t.Fatalf("flatten: %v\n%s", err, &log)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"/SMask", "/ExtGState", "/Transparency", "/Font", "/OCProperties"} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Errorf("flattened PDF still contains %s", forbidden)
			}
		}
		if count := len(regexp.MustCompile(`/Subtype\s*/Image\b`).FindAll(data, -1)); count != 3 {
			t.Fatalf("expected one opaque raster per page, got %d", count)
		}
		for page, want := range []struct {
			width, height int
			r, g, b       uint32
		}{
			{72, 72, 128, 0, 127},
			{144, 72, 0, 128, 127},
			{72, 144, 0, 255, 255},
		} {
			cmd := exec.Command(gs, "-q", "-dSAFER", "-dBATCH", "-dNOPAUSE",
				"-sDEVICE=png16m", "-r72", fmt.Sprintf("-dFirstPage=%d", page+1),
				fmt.Sprintf("-dLastPage=%d", page+1), "-sOutputFile=-", "-f", output)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			raster, err := cmd.Output()
			if err != nil {
				t.Fatalf("render page %d: %v\n%s", page+1, err, &stderr)
			}
			image, err := png.Decode(bytes.NewReader(raster))
			if err != nil {
				t.Fatal(err)
			}
			if image.Bounds().Dx() != want.width || image.Bounds().Dy() != want.height {
				t.Fatalf("page %d dimensions changed: %v", page+1, image.Bounds())
			}
			r, g, b, a := image.At(want.width/2, want.height/2).RGBA()
			for i, channel := range []uint32{r >> 8, g >> 8, b >> 8} {
				expected := []uint32{want.r, want.g, want.b}[i]
				if delta := int(channel) - int(expected); delta < -2 || delta > 2 {
					t.Errorf("page %d composited color = (%d, %d, %d), want (%d, %d, %d)",
						page+1, r>>8, g>>8, b>>8, want.r, want.g, want.b)
				}
			}
			if a != 65535 {
				t.Errorf("page %d is not opaque", page+1)
			}
		}
	}
	got, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("source PDF changed: %v", err)
	}
}

// Three pages exercise vector opacity, an image soft mask, and opaque text/vectors.
func transparencyPDF() []byte {
	stream := func(data string) string {
		return fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(data), data)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 6 0 R 10 0 R] /Count 3 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 72 72] /Resources << /ExtGState << /Alpha 5 0 R >> >> /Contents 4 0 R >>",
		stream("0 0 1 rg 0 0 72 72 re f\n/Alpha gs 1 0 0 rg 0 0 72 72 re f\n"),
		"<< /Type /ExtGState /ca 0.5 /CA 0.5 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 144 72] /Resources << /XObject << /Image 8 0 R >> >> /Contents 7 0 R >>",
		stream("0 0 1 rg 0 0 144 72 re f\nq 144 0 0 72 0 0 cm /Image Do Q\n"),
		"<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB /BitsPerComponent 8 /SMask 9 0 R /Length 3 >>\nstream\n\x00\xff\x00\nendstream",
		"<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceGray /BitsPerComponent 8 /Length 1 >>\nstream\n\x80\nendstream",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 72 144] /Resources << /Font << /F1 12 0 R >> >> /Contents 11 0 R >>",
		stream("0 1 1 rg 0 0 72 144 re f\n0 0 0 rg BT /F1 10 Tf 5 5 Td (Text) Tj ET\n"),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return pdf.Bytes()
}
