package renderer

import (
	"fmt"
	"io"
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

// Options controls where the Scribus adapter script is found and where its output goes.
type Options struct {
	// RootDir is the project root containing scripts/; "." when empty.
	RootDir string
	Stdout  io.Writer
	Stderr  io.Writer
}

func (o Options) withDefaults() Options {
	if o.RootDir == "" {
		o.RootDir = "."
	}
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	if o.Stderr == nil {
		o.Stderr = os.Stderr
	}
	return o
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
	return GenerateFromBookWithOptions(loaded, Options{})
}

// GenerateFromBookWithOptions is GenerateFromBook with an explicit project root and output writers.
func GenerateFromBookWithOptions(loaded book.Book, opts Options) (Result, error) {
	opts = opts.withDefaults()
	jobPath := filepath.Join(loaded.Dir, "out", scribusJobFileName)
	if err := writeScribusJob(jobPath, loaded.Config, loaded.Plan); err != nil {
		return Result{}, fmt.Errorf("write Scribus job: %w", err)
	}

	env, cleanupFonts, err := scribusFontEnvironment(opts.Stderr)
	if err != nil {
		return Result{}, err
	}
	defer cleanupFonts()

	status, err := os.CreateTemp("", "bookgen-scribus-status-*")
	if err != nil {
		return Result{}, fmt.Errorf("create Scribus status file: %w", err)
	}
	statusPath := status.Name()
	defer os.Remove(statusPath)
	if err := status.Close(); err != nil {
		return Result{}, fmt.Errorf("close Scribus status file: %w", err)
	}
	if env == nil {
		env = os.Environ()
	}
	env = append(env, "BOOKGEN_SCRIBUS_STATUS="+statusPath)
	cmd := buildScribusInvocation(loaded.Dir, jobPath, opts.RootDir)
	if err := runCommand(cmd, env, opts.Stdout, opts.Stderr); err != nil {
		return Result{}, err
	}
	result, err := os.ReadFile(statusPath)
	if err != nil {
		return Result{}, fmt.Errorf("read Scribus generation status: %w", err)
	}
	if string(result) != "success" {
		message := strings.TrimSpace(string(result))
		if message == "" {
			message = "Scribus did not report a generation result"
		}
		return Result{}, fmt.Errorf("Scribus generation failed: %s", message)
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

func buildScribusInvocation(bookDir, jobPath, rootDir string) []string {
	script := scribusScriptPath
	if rootDir != "" && rootDir != "." {
		script = filepath.Join(rootDir, scribusScriptPath)
	}
	return []string{"xvfb-run", "-a", "scribus", "-g", "-py", script, bookDir, jobPath}
}

// runCommand runs cmd; a nil env inherits the current process environment.
func runCommand(cmd []string, env []string, stdout, stderr io.Writer) error {
	command := exec.Command(cmd[0], cmd[1:]...)
	command.Env = env
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", strings.Join(cmd, " "), err)
	}
	return nil
}

// OpenInScribus launches interactive Scribus on slaPath with the same font
// environment used during generation, so PDFs exported after manual review
// embed the same usable font files.
func OpenInScribus(slaPath string, stderr io.Writer) error {
	if info, err := os.Stat(slaPath); err != nil {
		return fmt.Errorf("open Scribus document: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("open Scribus document: %s is not a regular file", slaPath)
	}
	env, cleanup, err := scribusFontEnvironment(stderr)
	if err != nil {
		return err
	}
	defer cleanup()
	command := exec.Command("scribus", slaPath)
	command.Env = env
	command.Stdout = os.Stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("launch scribus: %w", err)
	}
	return nil
}
