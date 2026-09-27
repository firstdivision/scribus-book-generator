package renderer

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Scribus treats every Type 1 font file that begins with "%!" as a hex-encoded
// PFA and hex-decodes its eexec section when embedding it in a PDF. Some
// distributions ship Type 1 files that start with "%!" but store the eexec
// section in binary (for example Debian's fonts-urw-base35 *.t1 files), so the
// embedded font program ends up corrupt and viewers or printers such as Lulu
// report that the font cannot be extracted. When fontconfig offers another file
// for the same PostScript font (an .otf or .pfb copy), those broken files are
// hidden from Scribus so it loads the usable copy instead.

const maxType1HeaderBytes = 1 << 20

type fontFile struct {
	Path           string
	PostScriptName string
}

// listFontconfigFonts returns the font files fontconfig reports, or nil when fc-list is unavailable.
func listFontconfigFonts() []fontFile {
	output, err := exec.Command("fc-list", "--format", "%{file}\t%{postscriptname}\n").Output()
	if err != nil {
		return nil
	}
	return parseFontconfigList(output)
}

func parseFontconfigList(output []byte) []fontFile {
	var fonts []fontFile
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		path, psName, ok := strings.Cut(scanner.Text(), "\t")
		path = strings.TrimSpace(path)
		psName = strings.TrimSpace(psName)
		if !ok || path == "" || psName == "" {
			continue
		}
		fonts = append(fonts, fontFile{Path: path, PostScriptName: psName})
	}
	return fonts
}

// type1FileHasBinaryEexec reports whether path is a "%!"-prefixed Type 1 font
// whose eexec section is binary rather than hex, which Scribus cannot embed.
func type1FileHasBinaryEexec(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxType1HeaderBytes))
	if err != nil {
		return false
	}
	return type1DataHasBinaryEexec(data)
}

func type1DataHasBinaryEexec(data []byte) bool {
	if !bytes.HasPrefix(data, []byte("%!")) {
		return false
	}
	marker := bytes.Index(data, []byte("eexec"))
	if marker < 0 {
		return false
	}
	payload := bytes.TrimLeft(data[marker+len("eexec"):], " \t\r\n")
	if len(payload) == 0 {
		return false
	}
	// Adobe Type 1 spec: the eexec section is hex only if its first four bytes are all hex digits.
	sample := payload[:min(4, len(payload))]
	for _, b := range sample {
		if !isHexDigit(b) {
			return true
		}
	}
	return false
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// unembeddableFontFiles returns the sorted broken Type 1 files whose PostScript
// font is also available from at least one other, embeddable file.
func unembeddableFontFiles(fonts []fontFile, isBroken func(string) bool) []string {
	broken := map[string]bool{}
	usable := map[string]bool{}
	for _, font := range fonts {
		if _, seen := broken[font.Path]; seen {
			continue
		}
		broken[font.Path] = isBroken(font.Path)
		if !broken[font.Path] {
			usable[font.PostScriptName] = true
		}
	}

	var rejected []string
	seen := map[string]bool{}
	for _, font := range fonts {
		if seen[font.Path] || !broken[font.Path] || !usable[font.PostScriptName] {
			continue
		}
		seen[font.Path] = true
		rejected = append(rejected, font.Path)
	}
	sort.Strings(rejected)
	return rejected
}

// fontconfigOverride renders a fontconfig file that loads baseConfig and rejects the given font files.
func fontconfigOverride(baseConfig string, rejected []string) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n")
	b.WriteString("<!DOCTYPE fontconfig SYSTEM \"urn:fontconfig:fonts.dtd\">\n")
	b.WriteString("<fontconfig>\n")
	b.WriteString("  <include ignore_missing=\"yes\">")
	xml.EscapeText(&b, []byte(baseConfig))
	b.WriteString("</include>\n")
	b.WriteString("  <selectfont>\n    <rejectfont>\n")
	for _, path := range rejected {
		b.WriteString("      <pattern><patelt name=\"file\"><string>")
		xml.EscapeText(&b, []byte(path))
		b.WriteString("</string></patelt></pattern>\n")
	}
	b.WriteString("    </rejectfont>\n  </selectfont>\n</fontconfig>\n")
	return b.String()
}

// baseFontconfigFile returns the fontconfig file the override should include.
// A relative "fonts.conf" resolves to fontconfig's default configuration.
func baseFontconfigFile() string {
	if current := strings.TrimSpace(os.Getenv("FONTCONFIG_FILE")); current != "" {
		if abs, err := filepath.Abs(current); err == nil {
			return abs
		}
		return current
	}
	return "fonts.conf"
}

// scribusFontEnvironment returns the environment for the Scribus process and a
// cleanup function. When broken Type 1 files have usable alternatives, it points
// FONTCONFIG_FILE at a temporary override that hides them from Scribus.
func scribusFontEnvironment(stderr io.Writer) ([]string, func(), error) {
	noop := func() {}
	rejected := unembeddableFontFiles(listFontconfigFonts(), type1FileHasBinaryEexec)
	if len(rejected) == 0 {
		return nil, noop, nil
	}

	dir, err := os.MkdirTemp("", "bookgen-fontconfig-")
	if err != nil {
		return nil, noop, fmt.Errorf("create fontconfig override directory: %w", err)
	}
	cleanup := func() { os.RemoveAll(dir) }
	confPath := filepath.Join(dir, "fonts.conf")
	if err := os.WriteFile(confPath, []byte(fontconfigOverride(baseFontconfigFile(), rejected)), 0o600); err != nil {
		cleanup()
		return nil, noop, fmt.Errorf("write fontconfig override: %w", err)
	}

	fmt.Fprintf(stderr, "hiding %d Type 1 font file(s) with binary eexec data from Scribus so embeddable copies are used instead\n", len(rejected))
	return append(os.Environ(), "FONTCONFIG_FILE="+confPath), cleanup, nil
}
