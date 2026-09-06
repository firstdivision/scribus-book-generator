package book

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

// File is the editable content of book.yaml.
type File struct {
	Template  string               `yaml:"template,omitempty"`
	Overrides *config.TemplateFile `yaml:"overrides,omitempty"`
	Layout    layoutplan.Plan      `yaml:"layout"`
}

// ReadFile parses book.yaml from bookDir. A missing file yields an empty File.
func ReadFile(bookDir string) (File, error) {
	data, err := os.ReadFile(filepath.Join(bookDir, "book.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse book.yaml: %w", err)
	}
	return file, nil
}

// Validate checks the layout plan and resolves the effective configuration so
// invalid overrides are reported before anything is written.
func (f File) Validate(bookDir string) error {
	if err := f.Layout.Validate(); err != nil {
		return err
	}
	if _, err := config.Resolve(bookDir, f.Template, f.Overrides); err != nil {
		return err
	}
	return nil
}

// Marshal renders the file as YAML. Comments from a hand-edited book.yaml are not preserved.
func (f File) Marshal() ([]byte, error) {
	if f.Layout.Images == nil {
		f.Layout.Images = []layoutplan.ImageInstruction{}
	}
	if f.Overrides != nil && isZeroTemplate(*f.Overrides) {
		f.Overrides = nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile validates and atomically writes book.yaml into bookDir.
func WriteFile(bookDir string, file File) error {
	if err := file.Validate(bookDir); err != nil {
		return err
	}
	data, err := file.Marshal()
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(bookDir, "book.yaml"), data)
}

// writeAtomic writes data to target via a temp file in the same directory and a rename.
func writeAtomic(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func isZeroTemplate(t config.TemplateFile) bool {
	data, err := yaml.Marshal(t)
	if err != nil {
		return false
	}
	return bytes.Equal(bytes.TrimSpace(data), []byte("{}"))
}
