package renderer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/layout/layoutplan"
)

const scribusScriptPath = "scripts/scribus_generate.py"

const scribusJobFileName = "scribus-job.json"

// Result is the artifact paths Scribus writes for a book directory.
type Result struct {
	SLAPath string
	PDFPath string
}

// Generate loads and validates a book directory, then runs Scribus.
func Generate(bookDir string) (Result, error) {
	loaded, err := book.Load(bookDir)
	if err != nil {
		return Result{}, err
	}
	return GenerateFromBook(loaded)
}

// GenerateFromBook writes a Scribus job JSON and runs the committed Python adapter.
func GenerateFromBook(loaded book.Book) (Result, error) {
	jobPath := filepath.Join(loaded.Dir, "out", scribusJobFileName)
	if err := writeScribusJob(jobPath, loaded.Config, loaded.Plan); err != nil {
		return Result{}, fmt.Errorf("write Scribus job: %w", err)
	}

	cmd := buildScribusInvocation(loaded.Dir, jobPath)
	if err := runCommand(cmd); err != nil {
		return Result{}, err
	}

	return outputPaths(loaded.Dir, loaded.Plan), nil
}

func outputPaths(bookDir string, plan layoutplan.Plan) Result {
	stem := outputFilenameStem(plan.Title, filepath.Base(filepath.Clean(bookDir)))
	outDir := filepath.Join(bookDir, "out")
	return Result{
		SLAPath: filepath.Join(outDir, stem+".sla"),
		PDFPath: filepath.Join(outDir, stem+".pdf"),
	}
}

func outputFilenameStem(title, bookDirName string) string {
	stem := strings.TrimSpace(title)
	if stem == "" {
		stem = bookDirName
	}
	stem = strings.ReplaceAll(stem, "/", "-")
	stem = strings.ReplaceAll(stem, "\\", "-")
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return bookDirName
	}
	return stem
}

func buildScribusInvocation(bookDir, jobPath string) []string {
	return []string{"xvfb-run", "-a", "scribus", "-g", "-py", scribusScriptPath, bookDir, jobPath}
}

func runCommand(cmd []string) error {
	command := exec.Command(cmd[0], cmd[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", strings.Join(cmd, " "), err)
	}
	return nil
}
