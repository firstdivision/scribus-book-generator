package renderer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"scribus-book-generator/internal/markdown"
)

func TestScribusCalloutParserMatchesGo(t *testing.T) {
	tests := []struct {
		name, input, failure string
	}{
		{"ordered", "```callout\n first\n\n# not a heading\n```\n# Title\nBody\n```callout\nlast\n```\n", ""},
		{"no callouts", "# Title\nBody\n\nMore\n", ""},
		{"empty", "```callout\n\n```\n", "empty callout on line 1"},
		{"nested", "```callout\n```callout\n", "nested callout on line 2"},
		{"unclosed", "\n```callout\nText\n", "unclosed callout on line 2"},
		{"duplicate heading", "# First\n# Second\n", "additional H1 heading on line 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "chapter.md")
			if err := os.WriteFile(path, []byte(tt.input), 0o644); err != nil {
				t.Fatal(err)
			}
			goChapter, goErr := markdown.ParseFile(path)
			cmd := exec.Command("python3", "-c", `import importlib.util, json, pathlib, sys
spec = importlib.util.spec_from_file_location("renderer", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
print(json.dumps(module._parse_chapter_blocks(pathlib.Path(sys.argv[2]))))
`, committedScriptPath(t), path)
			output, pyErr := cmd.CombinedOutput()
			if tt.failure != "" {
				if goErr == nil || !strings.Contains(goErr.Error(), tt.failure) ||
					pyErr == nil || !strings.Contains(string(output), tt.failure) {
					t.Fatalf("want %q; Go=%v Python=%v %s", tt.failure, goErr, pyErr, output)
				}
				return
			}
			if goErr != nil || pyErr != nil {
				t.Fatalf("Go=%v Python=%v %s", goErr, pyErr, output)
			}
			var parsed []json.RawMessage
			if err := json.Unmarshal(output, &parsed); err != nil {
				t.Fatal(err)
			}
			var title string
			var blocks [][]string
			if err := json.Unmarshal(parsed[0], &title); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(parsed[1], &blocks); err != nil {
				t.Fatal(err)
			}
			var want [][]string
			for _, block := range goChapter.Blocks {
				want = append(want, []string{block.Kind, block.Text})
			}
			if title != goChapter.Title || !reflect.DeepEqual(blocks, want) {
				t.Fatalf("Go=%q %#v Python=%q %#v", goChapter.Title, want, title, blocks)
			}
		})
	}
}

func TestScribusLegacyRendererRejectsCallouts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chapter.md")
	if err := os.WriteFile(path, []byte("# Title\n```callout\ntext\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-c", `import importlib.util, pathlib, sys
spec = importlib.util.spec_from_file_location("renderer", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
module._parse_chapter_markdown(pathlib.Path(sys.argv[2]))
`, committedScriptPath(t), path)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "callout rendering is not yet supported") {
		t.Fatalf("expected explicit unsupported error; got %v: %s", err, output)
	}
}
