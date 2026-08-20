package book

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/layout/layoutplan"
	"scribus-book-generator/internal/markdown"
)

var imageExtOrder = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg"}

// Book is the loaded, validated publishing input for one book directory.
type Book struct {
	Dir      string
	Config   config.Config
	Plan     layoutplan.Plan
	Chapters []Chapter
}

// Chapter is one folder under chapters/ with its markdown and local images.
type Chapter struct {
	Name     string
	Dir      string
	Markdown string
	Title    string
	Images   []string
}

// Load reads configuration, layout.json, chapters, and images, then validates
// that layout image paths exist on disk.
func Load(bookDir string) (Book, error) {
	if strings.TrimSpace(bookDir) == "" {
		return Book{}, fmt.Errorf("book directory is required")
	}
	bookDir = filepath.Clean(bookDir)

	cfg, err := config.LoadForBook(bookDir)
	if err != nil {
		return Book{}, fmt.Errorf("load configuration: %w", err)
	}

	plan, err := layoutplan.LoadFromBookDir(bookDir)
	if err != nil {
		return Book{}, fmt.Errorf("load layout.json: %w", err)
	}

	chapters, err := loadChapters(bookDir)
	if err != nil {
		return Book{}, err
	}

	if err := validateLayoutImages(bookDir, plan); err != nil {
		return Book{}, err
	}

	return Book{
		Dir:      bookDir,
		Config:   cfg,
		Plan:     plan,
		Chapters: chapters,
	}, nil
}

func loadChapters(bookDir string) ([]Chapter, error) {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chapters directory not found: %s", chaptersDir)
		}
		return nil, err
	}

	var dirs []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no chapter directories found in %s", chaptersDir)
	}

	chapters := make([]Chapter, 0, len(dirs))
	for _, entry := range dirs {
		chapter, err := loadChapter(bookDir, entry.Name())
		if err != nil {
			return nil, err
		}
		chapters = append(chapters, chapter)
	}
	return chapters, nil
}

func loadChapter(bookDir, name string) (Chapter, error) {
	chapterDir := filepath.Join(bookDir, "chapters", name)
	markdownPath, err := firstMarkdownFile(chapterDir)
	if err != nil {
		return Chapter{}, fmt.Errorf("chapter %s: %w", name, err)
	}

	parsed, err := markdown.ParseFile(markdownPath)
	if err != nil {
		rel := filepath.Join("chapters", name, filepath.Base(markdownPath))
		return Chapter{}, fmt.Errorf("%s: %w", rel, err)
	}

	images, err := imageFiles(chapterDir)
	if err != nil {
		return Chapter{}, fmt.Errorf("chapter %s: %w", name, err)
	}
	relImages := make([]string, 0, len(images))
	for _, image := range images {
		rel, err := filepath.Rel(bookDir, image)
		if err != nil {
			rel = image
		}
		relImages = append(relImages, rel)
	}

	relMarkdown, err := filepath.Rel(bookDir, markdownPath)
	if err != nil {
		relMarkdown = markdownPath
	}

	return Chapter{
		Name:     name,
		Dir:      filepath.Join("chapters", name),
		Markdown: relMarkdown,
		Title:    parsed.Title,
		Images:   relImages,
	}, nil
}

func firstMarkdownFile(chapterDir string) (string, error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("no markdown file")
	}
	return filepath.Join(chapterDir, names[0]), nil
}

func imageFiles(chapterDir string) ([]string, error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		return nil, err
	}

	byExt := make(map[string][]string, len(imageExtOrder))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !isImageExt(ext) {
			continue
		}
		byExt[ext] = append(byExt[ext], filepath.Join(chapterDir, entry.Name()))
	}

	var results []string
	seen := make(map[string]bool)
	for _, ext := range imageExtOrder {
		paths := byExt[ext]
		sort.Strings(paths)
		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true
			results = append(results, path)
		}
	}
	return results, nil
}

func isImageExt(ext string) bool {
	for _, candidate := range imageExtOrder {
		if ext == candidate {
			return true
		}
	}
	return false
}

func validateLayoutImages(bookDir string, plan layoutplan.Plan) error {
	for i, image := range plan.Images {
		src := image.Source()
		path := src
		if !filepath.IsAbs(path) {
			path = filepath.Join(bookDir, src)
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("layout.images[%d]: file not found: %s", i, src)
			}
			return fmt.Errorf("layout.images[%d]: %s: %w", i, src, err)
		}
	}
	return nil
}
