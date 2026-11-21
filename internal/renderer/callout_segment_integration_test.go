package renderer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScribusCalloutSegmentIntegration(t *testing.T) {
	runScribusProbe(t, "callout_segment_integration.py", "CALLOUT_SEGMENT_INTEGRATION_PASS")
}

func TestScribusCalloutFlowIntegration(t *testing.T) {
	runScribusProbe(t, "callout_flow_integration.py", "CALLOUT_FLOW_INTEGRATION_PASS")
}

func runScribusProbe(t *testing.T, name, marker string) {
	t.Helper()
	if os.Getenv("BOOKGEN_SCRIBUS_INTEGRATION") != "1" {
		t.Skip("set BOOKGEN_SCRIBUS_INTEGRATION=1 to run Scribus integration tests")
	}
	script := committedScriptPath(t)
	probe := filepath.Join(filepath.Dir(filepath.Dir(script)), "internal", "renderer", "testdata", name)
	resultPath := filepath.Join(t.TempDir(), "result.txt")
	cmd := exec.Command("xvfb-run", "-a", "scribus", "-g", "-py", probe)
	cmd.Env = append(os.Environ(), "BOOKGEN_SCRIPT="+script, "BOOKGEN_PROBE_RESULT="+resultPath)
	output, err := cmd.CombinedOutput()
	result, resultErr := os.ReadFile(resultPath)
	if err != nil || resultErr != nil || !strings.Contains(string(result), marker) {
		t.Fatalf("Scribus callout segment probe failed: %v, result: %v\n%s\n%s", err, resultErr, output, result)
	}
}
