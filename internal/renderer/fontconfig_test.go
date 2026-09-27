package renderer

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	binaryEexecType1 = []byte("%!PS-AdobeFont-1.0: Test-Demi 1.00\ncurrentfile eexec\r\xe9\x8d\t\xd7`\xa3\xc2,\x00\x01")
	hexEexecType1    = []byte("%!PS-AdobeFont-1.0: Test-Demi 1.00\ncurrentfile eexec\nD9D66F633B846A989B9974B0179FC6CC\n")
)

func TestType1DataHasBinaryEexec(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"binary eexec PFA", binaryEexecType1, true},
		{"hex eexec PFA", hexEexecType1, false},
		{"PFB segment header", []byte("\x80\x01\x10\x00\x00\x00%!PS-AdobeFont-1.0 eexec\x80\x02"), false},
		{"OpenType", []byte("OTTO\x00\x0b\x00\x80"), false},
		{"no eexec", []byte("%!PS-AdobeFont-1.0: Test"), false},
	}
	for _, tc := range cases {
		if got := type1DataHasBinaryEexec(tc.data); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseFontconfigList(t *testing.T) {
	got := parseFontconfigList([]byte("/a/Demi.t1\tURWBookman-Demi\n\n/b/NoName.ttf\t\n/c/Demi.otf\tURWBookman-Demi\n"))
	want := []fontFile{
		{Path: "/a/Demi.t1", PostScriptName: "URWBookman-Demi"},
		{Path: "/c/Demi.otf", PostScriptName: "URWBookman-Demi"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestUnembeddableFontFilesOnlyRejectsBrokenFilesWithAlternatives(t *testing.T) {
	fonts := []fontFile{
		{Path: "/fonts/type1/URWBookman-Demi.t1", PostScriptName: "URWBookman-Demi"},
		{Path: "/fonts/opentype/URWBookman-Demi.otf", PostScriptName: "URWBookman-Demi"},
		{Path: "/fonts/type1/URWBookman-Demi.t1", PostScriptName: "URWBookman-Demi"},
		{Path: "/fonts/type1/Lonely.t1", PostScriptName: "Lonely-Regular"},
		{Path: "/fonts/type1/Other.t1", PostScriptName: "Other-Bold"},
		{Path: "/fonts/X11/Other.pfb", PostScriptName: "Other-Bold"},
	}
	isBroken := func(path string) bool { return strings.HasSuffix(path, ".t1") }
	got := unembeddableFontFiles(fonts, isBroken)
	want := []string{"/fonts/type1/Other.t1", "/fonts/type1/URWBookman-Demi.t1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFontconfigOverrideIncludesBaseAndRejectsFiles(t *testing.T) {
	conf := fontconfigOverride("fonts.conf", []string{"/fonts/a&b.t1"})
	for _, fragment := range []string{
		`<include ignore_missing="yes">fonts.conf</include>`,
		`<rejectfont>`,
		`<patelt name="file"><string>/fonts/a&amp;b.t1</string></patelt>`,
	} {
		if !strings.Contains(conf, fragment) {
			t.Fatalf("override missing %q:\n%s", fragment, conf)
		}
	}
}

func TestScribusScriptRejectsBinaryEexecType1Fonts(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "Broken.t1")
	usable := filepath.Join(dir, "Usable.t1")
	if err := os.WriteFile(broken, binaryEexecType1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usable, hexEexecType1, 0o644); err != nil {
		t.Fatal(err)
	}
	script := `import importlib.util, sys
spec = importlib.util.spec_from_file_location("scribus_generate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
class PDF:
    def __init__(self):
        self.fontEmbedding = None
        self.fonts = []
class Scribus:
    def __init__(self, heading_file):
        self.heading_file = heading_file
    def getXFontNames(self):
        return [("Heading Demi", "Heading", "Heading-Demi", 0, 1, self.heading_file)]
for path in sys.argv[2:]:
    try:
        module._configure_pdf_export_compat(Scribus(path), PDF(), required_fonts=["Heading Demi"])
        print("ok")
    except RuntimeError as exc:
        print("error:", exc)
`
	output, err := exec.Command("python3", "-c", script, committedScriptPath(t), usable, broken).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to execute pdf export helper: %v\n%s", err, output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 || lines[0] != "ok" {
		t.Fatalf("expected hex-eexec font to be accepted, got %q", output)
	}
	if !strings.HasPrefix(lines[1], "error:") || !strings.Contains(lines[1], "'Heading Demi'") || !strings.Contains(lines[1], broken) {
		t.Fatalf("expected binary-eexec font to be rejected with its name and file, got %q", lines[1])
	}
}
