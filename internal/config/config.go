package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scribus-book-generator/internal/layout/chapterheadings"
	"scribus-book-generator/internal/layout/pagenumbering"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PageWidth         float64
	PageHeight        float64
	MarginTop         float64
	MarginLeft        float64
	MarginRight       float64
	MarginBottom      float64
	PageLayout        string
	FirstPage         string
	DocumentUnits     string
	PageSize          string
	PageOrientation   string
	PageBackgroundRGB *[3]int
	PageNumbers       pagenumbering.Settings
	BleedTop          float64
	BleedBottom       float64
	BleedInside       float64
	BleedOutside      float64
	ChapterHeadings   chapterheadings.Settings
	Images            ImageDefaults
}

func Default() Config {
	return Config{
		PageWidth:       210,
		PageHeight:      297,
		MarginTop:       20,
		MarginLeft:      20,
		MarginRight:     20,
		MarginBottom:    20,
		PageLayout:      "single_page",
		FirstPage:       "right",
		DocumentUnits:   "mm",
		PageSize:        "A4",
		PageOrientation: "portrait",
		PageNumbers:     pagenumbering.DefaultSettings(),
		ChapterHeadings: chapterheadings.DefaultSettings(),
		Images:          DefaultImageDefaults(),
	}
}

type bookConfigFile struct {
	Template  string        `yaml:"template"`
	Overrides *TemplateFile `yaml:"overrides"`
}

// TemplateFile is the YAML shape shared by template files and the book.yaml
// `overrides` block. Every leaf is a pointer so that an explicit zero can be
// distinguished from an unset value.
type TemplateFile struct {
	Document        DocumentTemplate       `yaml:"document,omitempty"`
	Page            PageTemplate           `yaml:"page,omitempty"`
	Bleed           SidesTemplate          `yaml:"bleed,omitempty"`
	SafetyMargin    SidesTemplate          `yaml:"safety_margin,omitempty"`
	ChapterHeadings ChapterHeadingTemplate `yaml:"chapter_headings,omitempty"`
	Images          ImageTemplate          `yaml:"images,omitempty"`
	PageNumbers     PageNumberTemplate     `yaml:"page_numbers,omitempty"`
}

type DocumentTemplate struct {
	Units     *string `yaml:"units,omitempty"`
	Layout    *string `yaml:"layout,omitempty"`
	FirstPage *string `yaml:"first_page,omitempty"`
}

type PageTemplate struct {
	WidthMM            *float64 `yaml:"width_mm,omitempty"`
	HeightMM           *float64 `yaml:"height_mm,omitempty"`
	Size               *string  `yaml:"size,omitempty"`
	Orientation        *string  `yaml:"orientation,omitempty"`
	Layout             *string  `yaml:"layout,omitempty"`
	FirstPage          *string  `yaml:"first_page,omitempty"`
	BackgroundColorRGB *[3]int  `yaml:"background_color_rgb,omitempty"`
}

type SidesTemplate struct {
	Top     *float64 `yaml:"top,omitempty"`
	Bottom  *float64 `yaml:"bottom,omitempty"`
	Inside  *float64 `yaml:"inside,omitempty"`
	Outside *float64 `yaml:"outside,omitempty"`
}

type FontTemplate struct {
	Family *string  `yaml:"family,omitempty"`
	Style  *string  `yaml:"style,omitempty"`
	SizePt *float64 `yaml:"size_pt,omitempty"`
}

type ChapterHeadingBorderTemplate struct {
	ColorRGB []int    `yaml:"color_rgb,omitempty"`
	WidthPt  *float64 `yaml:"width_pt,omitempty"`
}

type ChapterHeadingTemplate struct {
	BackgroundColorRGB []int                                    `yaml:"background_color_rgb,omitempty"`
	Borders            map[string]*ChapterHeadingBorderTemplate `yaml:"borders,omitempty"`
	Font               FontTemplate                             `yaml:"font,omitempty"`
	ColorRGB           []int                                    `yaml:"color_rgb,omitempty"`
	Alignment          *string                                  `yaml:"alignment,omitempty"`
	SpacingMM          struct {
		Top    *float64 `yaml:"top,omitempty"`
		Bottom *float64 `yaml:"bottom,omitempty"`
	} `yaml:"spacing_mm,omitempty"`
}

type PageNumberTemplate struct {
	Enabled     *bool         `yaml:"enabled,omitempty"`
	StartOnPage *int          `yaml:"start_on_page,omitempty"`
	StartNumber *int          `yaml:"start_number,omitempty"`
	Format      *string       `yaml:"format,omitempty"`
	Position    *string       `yaml:"position,omitempty"`
	Font        FontTemplate  `yaml:"font,omitempty"`
	ColorRGB    []int         `yaml:"color_rgb,omitempty"`
	OffsetMM    SidesTemplate `yaml:"offset_mm,omitempty"`
	HideOn      []string      `yaml:"hide_on,omitempty"`
}

// LoadForBook resolves the effective configuration for a book directory:
// built-in defaults, then the referenced template, then book.yaml `overrides`.
func LoadForBook(bookDir string) (Config, error) {
	cfg := Default()
	bookConfigPath := filepath.Join(bookDir, "book.yaml")

	bookFileData, err := os.ReadFile(bookConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var bookFile bookConfigFile
	if err := yaml.Unmarshal(bookFileData, &bookFile); err != nil {
		return cfg, err
	}

	return Resolve(bookDir, bookFile.Template, bookFile.Overrides)
}

// Resolve merges defaults, the named template (may be empty), and overrides (may be nil).
func Resolve(bookDir, templateName string, overrides *TemplateFile) (Config, error) {
	cfg := Default()

	templateName = strings.TrimSpace(templateName)
	if templateName != "" {
		templateFile, err := LoadTemplate(bookDir, templateName)
		if err != nil {
			return cfg, err
		}
		cfg, err = ApplyTemplate(cfg, templateFile)
		if err != nil {
			return cfg, fmt.Errorf("template %s: %w", templateName, err)
		}
	}

	if overrides != nil {
		var err error
		cfg, err = ApplyTemplate(cfg, *overrides)
		if err != nil {
			return cfg, fmt.Errorf("overrides: %w", err)
		}
	}

	return cfg, nil
}

// LoadTemplate resolves templateName relative to bookDir and parses it.
func LoadTemplate(bookDir, templateName string) (TemplateFile, error) {
	templatePath, err := ResolveTemplatePath(bookDir, templateName)
	if err != nil {
		return TemplateFile{}, err
	}
	return ReadTemplateFile(templatePath)
}

// ReadTemplateFile parses a template YAML file without applying it.
func ReadTemplateFile(path string) (TemplateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TemplateFile{}, err
	}
	var templateFile TemplateFile
	if err := yaml.Unmarshal(data, &templateFile); err != nil {
		return TemplateFile{}, err
	}
	return templateFile, nil
}

// ApplyTemplate layers the set (non-nil) values of templateFile over cfg.
func ApplyTemplate(cfg Config, templateFile TemplateFile) (Config, error) {
	cfg.DocumentUnits = defaultString(deref(templateFile.Document.Units), cfg.DocumentUnits)
	cfg.PageLayout = defaultString(firstNonEmpty(deref(templateFile.Document.Layout), deref(templateFile.Page.Layout)), cfg.PageLayout)
	cfg.FirstPage = defaultString(firstNonEmpty(deref(templateFile.Document.FirstPage), deref(templateFile.Page.FirstPage)), cfg.FirstPage)
	cfg.PageSize = defaultString(deref(templateFile.Page.Size), cfg.PageSize)
	cfg.PageOrientation = defaultString(deref(templateFile.Page.Orientation), cfg.PageOrientation)
	if templateFile.Page.BackgroundColorRGB != nil {
		cfg.PageBackgroundRGB = templateFile.Page.BackgroundColorRGB
	}
	pageNumbers, err := parsePageNumberSettings(templateFile.PageNumbers, cfg.PageNumbers)
	if err != nil {
		return cfg, err
	}
	cfg.PageNumbers = pageNumbers
	cfg.ChapterHeadings, err = parseChapterHeadingSettings(templateFile.ChapterHeadings, cfg.ChapterHeadings)
	if err != nil {
		return cfg, err
	}

	if (templateFile.Page.WidthMM == nil) != (templateFile.Page.HeightMM == nil) {
		return cfg, fmt.Errorf("page.width_mm and page.height_mm must be set together")
	}
	if templateFile.Page.WidthMM != nil && templateFile.Page.HeightMM != nil {
		cfg.PageWidth = *templateFile.Page.WidthMM
		cfg.PageHeight = *templateFile.Page.HeightMM
	} else if templateFile.Page.Size != nil || templateFile.Page.Orientation != nil {
		if widthMM, heightMM, ok := pageDimensionsMM(cfg.PageSize, cfg.PageOrientation); ok {
			cfg.PageWidth = widthMM
			cfg.PageHeight = heightMM
		}
	}
	if cfg.PageWidth <= 0 || cfg.PageHeight <= 0 {
		return cfg, fmt.Errorf("page.width_mm and page.height_mm must be > 0")
	}

	setFloat(&cfg.MarginTop, templateFile.SafetyMargin.Top)
	setFloat(&cfg.MarginBottom, templateFile.SafetyMargin.Bottom)
	setFloat(&cfg.MarginLeft, templateFile.SafetyMargin.Inside)
	setFloat(&cfg.MarginRight, templateFile.SafetyMargin.Outside)
	if cfg.MarginTop < 0 || cfg.MarginBottom < 0 || cfg.MarginLeft < 0 || cfg.MarginRight < 0 {
		return cfg, fmt.Errorf("safety_margin values must be >= 0")
	}

	setFloat(&cfg.BleedTop, templateFile.Bleed.Top)
	setFloat(&cfg.BleedBottom, templateFile.Bleed.Bottom)
	setFloat(&cfg.BleedInside, templateFile.Bleed.Inside)
	setFloat(&cfg.BleedOutside, templateFile.Bleed.Outside)
	if cfg.BleedTop < 0 || cfg.BleedBottom < 0 || cfg.BleedInside < 0 || cfg.BleedOutside < 0 {
		return cfg, fmt.Errorf("bleed values must be >= 0")
	}

	cfg.Images, err = parseImageDefaults(templateFile.Images, cfg.Images)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func setFloat(dst *float64, src *float64) {
	if src != nil {
		*dst = *src
	}
}

func parseChapterHeadingSettings(raw ChapterHeadingTemplate, defaults chapterheadings.Settings) (chapterheadings.Settings, error) {
	settings := defaults

	if raw.Font.Family != nil {
		settings.Font.Family = strings.TrimSpace(*raw.Font.Family)
	}
	if raw.Font.Style != nil {
		settings.Font.Style = strings.TrimSpace(*raw.Font.Style)
	}
	if raw.Font.SizePt != nil {
		settings.Font.SizePt = *raw.Font.SizePt
	}
	if raw.ColorRGB != nil {
		if len(raw.ColorRGB) != 3 {
			return settings, fmt.Errorf("chapter_headings.color_rgb must contain exactly 3 integers")
		}
		settings.ColorRGB = [3]int{raw.ColorRGB[0], raw.ColorRGB[1], raw.ColorRGB[2]}
	}
	if raw.BackgroundColorRGB != nil {
		if len(raw.BackgroundColorRGB) != 3 {
			return settings, fmt.Errorf("chapter_headings.background_color_rgb must contain exactly 3 integers")
		}
		settings.BackgroundColorRGB = &[3]int{raw.BackgroundColorRGB[0], raw.BackgroundColorRGB[1], raw.BackgroundColorRGB[2]}
	}
	if raw.Borders != nil {
		settings.Borders = make(map[string]chapterheadings.Border, len(raw.Borders))
		for side, value := range raw.Borders {
			border := chapterheadings.Border{ColorRGB: settings.ColorRGB, WidthPt: 1}
			if value == nil {
				border.WidthPt = 0
			} else {
				if value.WidthPt != nil {
					border.WidthPt = *value.WidthPt
				}
				if value.ColorRGB != nil {
					if len(value.ColorRGB) != 3 {
						return settings, fmt.Errorf("chapter_headings.borders.%s.color_rgb must contain exactly 3 integers", side)
					}
					border.ColorRGB = [3]int{value.ColorRGB[0], value.ColorRGB[1], value.ColorRGB[2]}
				}
			}
			settings.Borders[side] = border
		}
	}
	if raw.Alignment != nil {
		settings.Alignment = chapterheadings.Alignment(strings.TrimSpace(*raw.Alignment))
	}
	if raw.SpacingMM.Top != nil {
		settings.SpacingMM.Top = *raw.SpacingMM.Top
	}
	if raw.SpacingMM.Bottom != nil {
		settings.SpacingMM.Bottom = *raw.SpacingMM.Bottom
	}

	if err := settings.Validate(); err != nil {
		return settings, err
	}
	return settings, nil
}

func parsePageNumberSettings(raw PageNumberTemplate, defaults pagenumbering.Settings) (pagenumbering.Settings, error) {
	settings := defaults

	if raw.Enabled != nil {
		settings.Enabled = *raw.Enabled
	}
	if raw.StartOnPage != nil {
		settings.StartOnPage = *raw.StartOnPage
	}
	if raw.StartNumber != nil {
		settings.StartNumber = *raw.StartNumber
	}
	if trimmed := strings.TrimSpace(deref(raw.Format)); trimmed != "" {
		settings.Format = pagenumbering.NumberFormat(trimmed)
	}
	if trimmed := strings.TrimSpace(deref(raw.Position)); trimmed != "" {
		settings.Position = pagenumbering.Position(trimmed)
	}
	if trimmed := strings.TrimSpace(deref(raw.Font.Family)); trimmed != "" {
		settings.Font.Family = trimmed
	}
	if trimmed := strings.TrimSpace(deref(raw.Font.Style)); trimmed != "" {
		settings.Font.Style = trimmed
	}
	if raw.Font.SizePt != nil {
		settings.Font.SizePt = *raw.Font.SizePt
	}
	if raw.ColorRGB != nil {
		if len(raw.ColorRGB) != 3 {
			return settings, fmt.Errorf("page_numbers.color_rgb must contain exactly 3 integers")
		}
		settings.ColorRGB = [3]int{raw.ColorRGB[0], raw.ColorRGB[1], raw.ColorRGB[2]}
	}
	if raw.OffsetMM.Top != nil {
		settings.OffsetMM.Top = *raw.OffsetMM.Top
	}
	if raw.OffsetMM.Bottom != nil {
		settings.OffsetMM.Bottom = *raw.OffsetMM.Bottom
	}
	if raw.OffsetMM.Inside != nil {
		settings.OffsetMM.Inside = *raw.OffsetMM.Inside
	}
	if raw.OffsetMM.Outside != nil {
		settings.OffsetMM.Outside = *raw.OffsetMM.Outside
	}
	if raw.HideOn != nil {
		settings.HideOn = make([]pagenumbering.PageRole, 0, len(raw.HideOn))
		for _, role := range raw.HideOn {
			settings.HideOn = append(settings.HideOn, pagenumbering.PageRole(strings.TrimSpace(role)))
		}
	}

	if err := settings.Validate(); err != nil {
		return settings, err
	}

	return settings, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func pageDimensionsMM(size, orientation string) (float64, float64, bool) {
	size = strings.ToUpper(strings.TrimSpace(size))
	orientation = strings.ToLower(strings.TrimSpace(orientation))

	width, height, ok := paperSizeMM(size)
	if !ok {
		return 0, 0, false
	}

	if orientation == "landscape" && width < height {
		width, height = height, width
	}
	if orientation == "portrait" && width > height {
		width, height = height, width
	}

	return width, height, true
}

func paperSizeMM(size string) (float64, float64, bool) {
	switch size {
	case "A4":
		return 210, 297, true
	case "LETTER":
		return 215.9, 279.4, true
	default:
		return 0, 0, false
	}
}

// ResolveTemplatePath locates templateName as an absolute path, relative to
// bookDir, or under a templates/ directory in bookDir or any of its parents.
func ResolveTemplatePath(bookDir, templateName string) (string, error) {
	if filepath.IsAbs(templateName) {
		return templateName, nil
	}

	bookTemplatePath := filepath.Join(bookDir, templateName)
	if _, err := os.Stat(bookTemplatePath); err == nil {
		return bookTemplatePath, nil
	}

	searchRoot := bookDir
	if absBookDir, err := filepath.Abs(bookDir); err == nil {
		searchRoot = absBookDir
	}

	for {
		templatesDir := filepath.Join(searchRoot, "templates")
		if info, err := os.Stat(templatesDir); err == nil && info.IsDir() {
			matches, err := filepath.Glob(filepath.Join(templatesDir, "*", templateName))
			if err != nil {
				return "", err
			}
			if len(matches) > 0 {
				return matches[0], nil
			}
			if match, err := filepath.Glob(filepath.Join(templatesDir, templateName)); err == nil && len(match) > 0 {
				return match[0], nil
			}
		}

		parent := filepath.Dir(searchRoot)
		if parent == searchRoot {
			break
		}
		searchRoot = parent
	}

	return "", fmt.Errorf("template %q not found under any templates directory", templateName)
}

// TemplateRef is a template discovered under a project's templates/ directory.
type TemplateRef struct {
	// Name is the value to store in book.yaml `template:`, relative to templates/.
	Name string
	Path string
}

// ListTemplates returns every *.yaml file under <root>/templates, sorted by Name.
func ListTemplates(root string) ([]TemplateRef, error) {
	templatesDir := filepath.Join(root, "templates")
	var refs []TemplateRef
	err := filepath.WalkDir(templatesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return err
		}
		refs = append(refs, TemplateRef{Name: filepath.ToSlash(rel), Path: path})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}

// FindProjectRoot walks up from startDir to the nearest directory containing templates/.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "templates")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no templates directory found above %s", startDir)
		}
		dir = parent
	}
}
