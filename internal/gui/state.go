package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scribus-book-generator/internal/book"
	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
)

// State is the in-memory model the GUI edits: one open book plus the project
// context (root directory and available templates) it lives in.
type State struct {
	RootDir   string
	BookDir   string
	File      book.File
	Chapters  []book.Chapter
	Templates []config.TemplateRef
	Dirty     bool

	// PendingText holds edited chapter markdown, keyed by chapter name, until Save.
	PendingText map[string]string

	// TemplateConfig is defaults + template only; the GUI shows it as the
	// inherited value next to each override.
	TemplateConfig config.Config
}

// Open loads bookDir for editing. rootDir may be empty, in which case it is
// discovered by walking up from bookDir.
func Open(rootDir, bookDir string) (*State, error) {
	bookDir = filepath.Clean(bookDir)
	if rootDir == "" {
		found, err := config.FindProjectRoot(bookDir)
		if err != nil {
			return nil, err
		}
		rootDir = found
	}

	file, err := book.ReadFile(bookDir)
	if err != nil {
		return nil, err
	}
	if file.Overrides == nil {
		file.Overrides = &config.TemplateFile{}
	}

	templates, err := config.ListTemplates(rootDir)
	if err != nil {
		return nil, err
	}

	s := &State{
		RootDir:     rootDir,
		BookDir:     bookDir,
		File:        file,
		Templates:   templates,
		PendingText: map[string]string{},
	}
	if err := s.refreshTemplateConfig(); err != nil {
		return nil, err
	}
	if err := s.ReloadChapters(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) refreshTemplateConfig() error {
	cfg, err := config.Resolve(s.BookDir, s.File.Template, nil)
	if err != nil {
		return err
	}
	s.TemplateConfig = cfg
	return nil
}

// SetTemplate changes the base template and recomputes the inherited values.
func (s *State) SetTemplate(name string) error {
	previous := s.File.Template
	s.File.Template = strings.TrimSpace(name)
	if err := s.refreshTemplateConfig(); err != nil {
		s.File.Template = previous
		return err
	}
	s.Dirty = true
	return nil
}

// TemplateOptions lists selectable template names, including the current
// value when it does not match a discovered template.
func (s *State) TemplateOptions() (options []string, selected string) {
	currentPath := ""
	if s.File.Template != "" {
		if resolved, err := config.ResolveTemplatePath(s.BookDir, s.File.Template); err == nil {
			currentPath = filepath.Clean(resolved)
		}
	}
	for _, ref := range s.Templates {
		options = append(options, ref.Name)
		if currentPath != "" && filepath.Clean(ref.Path) == currentPath {
			selected = ref.Name
		}
	}
	if selected == "" && s.File.Template != "" {
		options = append(options, s.File.Template)
		selected = s.File.Template
	}
	return options, selected
}

// Effective resolves defaults + template + current overrides.
func (s *State) Effective() (config.Config, error) {
	return config.Resolve(s.BookDir, s.File.Template, s.File.Overrides)
}

// Validate reports the first problem that would prevent saving.
func (s *State) Validate() error {
	return s.File.Validate(s.BookDir)
}

// Save writes book.yaml and any edited chapter markdown, then clears the dirty flag.
func (s *State) Save() error {
	if err := book.WriteFile(s.BookDir, s.File); err != nil {
		return err
	}
	for name, text := range s.PendingText {
		chapter, ok := s.ChapterByName(name)
		if !ok {
			return fmt.Errorf("chapter %s no longer exists", name)
		}
		if err := book.WriteChapterMarkdown(s.BookDir, s.MarkdownPath(chapter), text); err != nil {
			return err
		}
		delete(s.PendingText, name)
	}
	s.Dirty = false
	return s.ReloadChapters()
}

// ReloadChapters re-scans chapters/ using the effective image sorting.
func (s *State) ReloadChapters() error {
	sorting := s.TemplateConfig.Images.Sorting
	if effective, err := s.Effective(); err == nil {
		sorting = effective.Images.Sorting
	}
	chapters, err := book.ListChapters(s.BookDir, sorting)
	if err != nil {
		return err
	}
	s.Chapters = chapters
	return nil
}

// ChapterImages returns every discovered image path, book-relative.
func (s *State) ChapterImages() []string {
	var paths []string
	for _, chapter := range s.Chapters {
		paths = append(paths, chapter.Images...)
	}
	return paths
}

// ChapterNames returns the chapter directory names in order.
func (s *State) ChapterNames() []string {
	names := make([]string, 0, len(s.Chapters))
	for _, chapter := range s.Chapters {
		names = append(names, chapter.Name)
	}
	return names
}

// ChapterByName finds a chapter by directory name.
func (s *State) ChapterByName(name string) (book.Chapter, bool) {
	for _, chapter := range s.Chapters {
		if chapter.Name == name {
			return chapter, true
		}
	}
	return book.Chapter{}, false
}

// MarkdownPath returns the book-relative markdown path for c, defaulting to
// text.md for chapters that have no markdown file yet.
func (s *State) MarkdownPath(c book.Chapter) string {
	if c.Markdown != "" {
		return c.Markdown
	}
	return filepath.Join(c.Dir, "text.md")
}

// ChapterText returns the pending edit for c, or its markdown from disk.
// A chapter without a markdown file yields an empty string.
func (s *State) ChapterText(c book.Chapter) (string, error) {
	if text, ok := s.PendingText[c.Name]; ok {
		return text, nil
	}
	data, err := os.ReadFile(filepath.Join(s.BookDir, s.MarkdownPath(c)))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SetChapterText records an edit to c's markdown to be written on Save.
func (s *State) SetChapterText(c book.Chapter, text string) {
	s.PendingText[c.Name] = text
	s.Dirty = true
}

// ImageInstructionIndex returns the layout.images index for source, or -1.
func (s *State) ImageInstructionIndex(source string) int {
	for i, inst := range s.File.Layout.Images {
		if inst.Source() == source {
			return i
		}
	}
	return -1
}

// UpsertImageInstruction replaces the entry for inst's source or appends it.
func (s *State) UpsertImageInstruction(inst layoutplan.ImageInstruction) {
	if i := s.ImageInstructionIndex(inst.Source()); i >= 0 {
		s.File.Layout.Images[i] = inst
	} else {
		s.File.Layout.Images = append(s.File.Layout.Images, inst)
	}
	s.Dirty = true
}

// RemoveImageInstruction drops the entry for source, if any.
func (s *State) RemoveImageInstruction(source string) {
	i := s.ImageInstructionIndex(source)
	if i < 0 {
		return
	}
	images := s.File.Layout.Images
	s.File.Layout.Images = append(images[:i:i], images[i+1:]...)
	s.Dirty = true
}

// Title is a short window title for the open book.
func (s *State) Title() string {
	name := filepath.Base(s.BookDir)
	if strings.TrimSpace(s.File.Layout.Title) != "" {
		name = s.File.Layout.Title
	}
	if s.Dirty {
		return fmt.Sprintf("%s * — Book Generator", name)
	}
	return fmt.Sprintf("%s — Book Generator", name)
}
