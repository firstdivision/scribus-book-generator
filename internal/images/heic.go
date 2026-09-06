package images

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const convertHEICScriptPath = "scripts/convert-heic.sh"

// ConvertHEIC runs the committed HEIC-to-JPG converter over a book's chapters/
// tree. It must run before chapters are loaded so the generated .jpg files are
// discovered as chapter images.
func ConvertHEIC(bookDir string) error {
	return ConvertHEICWithOptions(bookDir, ".", os.Stdout, os.Stderr)
}

// ConvertHEICWithOptions is ConvertHEIC with the project root (containing scripts/) and output writers.
func ConvertHEICWithOptions(bookDir, rootDir string, stdout, stderr io.Writer) error {
	script := convertHEICScriptPath
	if rootDir != "" && rootDir != "." {
		script = filepath.Join(rootDir, convertHEICScriptPath)
	}
	cmd := exec.Command("bash", script, bookDir)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}
