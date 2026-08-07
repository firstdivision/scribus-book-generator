package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	BleedTop          float64
	BleedBottom       float64
	BleedInside       float64
	BleedOutside      float64
	ImageBorderRGB    [3]int
	ImageBorderPt     float64
	ImageSpaceTop     float64
	ImageSpaceBottom  float64
	ImageSpaceInside  float64
	ImageSpaceOutside float64
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
	}
}

type bookConfigFile struct {
	Template string `yaml:"template"`
}

type templateConfigFile struct {
	Document struct {
		Units     string `yaml:"units"`
		Layout    string `yaml:"layout"`
		FirstPage string `yaml:"first_page"`
	} `yaml:"document"`
	Page struct {
		WidthMM            float64 `yaml:"width_mm"`
		HeightMM           float64 `yaml:"height_mm"`
		Size               string  `yaml:"size"`
		Orientation        string  `yaml:"orientation"`
		Layout             string  `yaml:"layout"`
		FirstPage          string  `yaml:"first_page"`
		BackgroundColorRGB *[3]int `yaml:"background_color_rgb"`
	} `yaml:"page"`
	Bleed struct {
		Top     float64 `yaml:"top"`
		Bottom  float64 `yaml:"bottom"`
		Inside  float64 `yaml:"inside"`
		Outside float64 `yaml:"outside"`
	} `yaml:"bleed"`
	SafetyMargin struct {
		Top     float64 `yaml:"top"`
		Bottom  float64 `yaml:"bottom"`
		Inside  float64 `yaml:"inside"`
		Outside float64 `yaml:"outside"`
	} `yaml:"safety_margin"`
	Images struct {
		Border struct {
			ColorRGB [3]int  `yaml:"color_rgb"`
			WidthPt  float64 `yaml:"width_pt"`
		} `yaml:"border"`
		SpacingMM struct {
			Top     float64 `yaml:"top"`
			Bottom  float64 `yaml:"bottom"`
			Inside  float64 `yaml:"inside"`
			Outside float64 `yaml:"outside"`
		} `yaml:"spacing_mm"`
	} `yaml:"images"`
}

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

	templateName := strings.TrimSpace(bookFile.Template)
	if templateName == "" {
		return cfg, nil
	}

	templatePath, err := resolveTemplatePath(bookDir, templateName)
	if err != nil {
		return cfg, err
	}

	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return cfg, err
	}

	var templateFile templateConfigFile
	if err := yaml.Unmarshal(templateData, &templateFile); err != nil {
		return cfg, err
	}

	cfg.DocumentUnits = defaultString(templateFile.Document.Units, cfg.DocumentUnits)
	cfg.PageLayout = defaultString(firstNonEmpty(templateFile.Document.Layout, templateFile.Page.Layout), cfg.PageLayout)
	cfg.FirstPage = defaultString(firstNonEmpty(templateFile.Document.FirstPage, templateFile.Page.FirstPage), cfg.FirstPage)
	cfg.PageSize = defaultString(templateFile.Page.Size, cfg.PageSize)
	cfg.PageOrientation = defaultString(templateFile.Page.Orientation, cfg.PageOrientation)
	if templateFile.Page.BackgroundColorRGB != nil {
		cfg.PageBackgroundRGB = templateFile.Page.BackgroundColorRGB
	}

	if templateFile.Page.WidthMM > 0 && templateFile.Page.HeightMM > 0 {
		cfg.PageWidth = templateFile.Page.WidthMM
		cfg.PageHeight = templateFile.Page.HeightMM
	} else if widthMM, heightMM, ok := pageDimensionsMM(cfg.PageSize, cfg.PageOrientation); ok {
		cfg.PageWidth = widthMM
		cfg.PageHeight = heightMM
	}
	if templateFile.SafetyMargin.Top > 0 {
		cfg.MarginTop = templateFile.SafetyMargin.Top
	}
	if templateFile.SafetyMargin.Bottom > 0 {
		cfg.MarginBottom = templateFile.SafetyMargin.Bottom
	}
	if templateFile.SafetyMargin.Inside > 0 {
		cfg.MarginLeft = templateFile.SafetyMargin.Inside
	}
	if templateFile.SafetyMargin.Outside > 0 {
		cfg.MarginRight = templateFile.SafetyMargin.Outside
	}

	if templateFile.Bleed.Top > 0 {
		cfg.BleedTop = templateFile.Bleed.Top
	}
	if templateFile.Bleed.Bottom > 0 {
		cfg.BleedBottom = templateFile.Bleed.Bottom
	}
	if templateFile.Bleed.Inside > 0 {
		cfg.BleedInside = templateFile.Bleed.Inside
	}
	if templateFile.Bleed.Outside > 0 {
		cfg.BleedOutside = templateFile.Bleed.Outside
	}
	if templateFile.Images.Border.ColorRGB != [3]int{} {
		cfg.ImageBorderRGB = templateFile.Images.Border.ColorRGB
	}
	if templateFile.Images.Border.WidthPt > 0 {
		cfg.ImageBorderPt = templateFile.Images.Border.WidthPt
	}
	if templateFile.Images.SpacingMM.Top > 0 {
		cfg.ImageSpaceTop = templateFile.Images.SpacingMM.Top
	}
	if templateFile.Images.SpacingMM.Bottom > 0 {
		cfg.ImageSpaceBottom = templateFile.Images.SpacingMM.Bottom
	}
	if templateFile.Images.SpacingMM.Inside > 0 {
		cfg.ImageSpaceInside = templateFile.Images.SpacingMM.Inside
	}
	if templateFile.Images.SpacingMM.Outside > 0 {
		cfg.ImageSpaceOutside = templateFile.Images.SpacingMM.Outside
	}

	return cfg, nil
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

func resolveTemplatePath(bookDir, templateName string) (string, error) {
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
