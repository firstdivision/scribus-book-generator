package book

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"scribus-book-generator/internal/config"
	"scribus-book-generator/internal/markdown"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify lowercases s and replaces runs of non-alphanumerics with single hyphens.
func Slugify(s string) string {
	slug := nonSlugChars.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(slug, "-")
}

// Create initialises a new book directory with chapters/ and a book.yaml.
// It refuses to reuse a directory that already contains a book.yaml.
func Create(bookDir, template, title string) error {
	bookDir = filepath.Clean(bookDir)
	if _, err := os.Stat(filepath.Join(bookDir, "book.yaml")); err == nil {
		return fmt.Errorf("%s already contains a book.yaml", bookDir)
	}
	if err := os.MkdirAll(filepath.Join(bookDir, "chapters"), 0o755); err != nil {
		return err
	}
	file := File{Template: strings.TrimSpace(template)}
	file.Layout.Title = strings.TrimSpace(title)
	return WriteFile(bookDir, file)
}

// CreateChapter adds chapters/<n>-<slug>/text.md containing an H1 title, where n
// is one more than the highest existing numeric prefix.
func CreateChapter(bookDir, title string) (Chapter, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Chapter{}, fmt.Errorf("chapter title is required")
	}
	slug := Slugify(title)
	if slug == "" {
		return Chapter{}, fmt.Errorf("chapter title %q yields an empty directory name", title)
	}

	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0o755); err != nil {
		return Chapter{}, err
	}
	next, err := nextChapterNumber(chaptersDir)
	if err != nil {
		return Chapter{}, err
	}

	name := fmt.Sprintf("%d-%s", next, slug)
	chapterDir := filepath.Join(chaptersDir, name)
	if err := os.Mkdir(chapterDir, 0o755); err != nil {
		return Chapter{}, err
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "text.md"), []byte("# "+title+"\n"), 0o644); err != nil {
		return Chapter{}, err
	}
	return Chapter{
		Name:     name,
		Dir:      filepath.Join("chapters", name),
		Markdown: filepath.Join("chapters", name, "text.md"),
		Title:    title,
	}, nil
}

func nextChapterNumber(chaptersDir string) (int, error) {
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return 0, err
	}
	highest := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix, _, _ := strings.Cut(entry.Name(), "-")
		if n, err := strconv.Atoi(prefix); err == nil && n > highest {
			highest = n
		}
	}
	return highest + 1, nil
}

// ImportImages copies srcPaths into chapters/<chapterName>/ and returns the
// book-relative paths written. Existing files are never overwritten.
func ImportImages(bookDir, chapterName string, srcPaths []string) ([]string, error) {
	chapterDir := filepath.Join(bookDir, "chapters", chapterName)
	if info, err := os.Stat(chapterDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("chapter directory not found: %s", chapterDir)
	}

	var written []string
	for _, src := range srcPaths {
		base := filepath.Base(src)
		if !isImageExt(strings.ToLower(filepath.Ext(base))) {
			return written, fmt.Errorf("%s is not a supported image type", base)
		}
		dst := filepath.Join(chapterDir, base)
		if _, err := os.Stat(dst); err == nil {
			return written, fmt.Errorf("%s already exists in %s", base, chapterName)
		}
		if err := copyFile(src, dst); err != nil {
			return written, err
		}
		written = append(written, filepath.Join("chapters", chapterName, base))
	}
	return written, nil
}

// WriteChapterMarkdown atomically writes content to the book-relative markdown
// path rel, creating the chapter directory if needed.
func WriteChapterMarkdown(bookDir, rel, content string) error {
	rel = filepath.Clean(rel)
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("markdown path must be relative to the book: %q", rel)
	}
	target := filepath.Join(bookDir, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return writeAtomic(target, []byte(content))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// ListChapters inventories chapters/ without requiring any to exist or to
// contain markdown yet; a chapter with no .md file has an empty Title.
func ListChapters(bookDir string, sorting config.ImageSorting) ([]Chapter, error) {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	chapters := make([]Chapter, 0, len(names))
	for _, name := range names {
		chapterDir := filepath.Join(chaptersDir, name)
		chapter := Chapter{Name: name, Dir: filepath.Join("chapters", name)}

		if markdownPath, err := firstMarkdownFile(chapterDir); err == nil {
			chapter.Markdown, _ = filepath.Rel(bookDir, markdownPath)
			if parsed, err := markdown.ParseFile(markdownPath); err == nil {
				chapter.Title = parsed.Title
			} else {
				return nil, fmt.Errorf("%s: %w", chapter.Markdown, err)
			}
		}

		imagePaths, err := imageFiles(chapterDir, sorting)
		if err != nil {
			return nil, fmt.Errorf("chapter %s: %w", name, err)
		}
		for _, image := range imagePaths {
			rel, err := filepath.Rel(bookDir, image)
			if err != nil {
				rel = image
			}
			chapter.Images = append(chapter.Images, rel)
		}
		chapters = append(chapters, chapter)
	}
	return chapters, nil
}
