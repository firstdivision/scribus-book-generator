package images

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const convertHEICScriptPath = "scripts/convert-heic.sh"

// ConvertHEIC runs the committed HEIC-to-JPG converter over a book's chapters/
// tree. It must run before chapters are loaded so the generated .jpg files are
// discovered as chapter images.
func ConvertHEIC(bookDir string) error {
	cmd := exec.Command("bash", convertHEICScriptPath, bookDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}
