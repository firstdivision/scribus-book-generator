package renderer

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCalloutGeometry(t *testing.T) {
	script := committedScriptPath(t)
	tests := filepath.Join(filepath.Dir(filepath.Dir(script)), "internal", "renderer", "testdata", "callout_geometry_test.py")
	output, err := exec.Command("python3", tests, script).CombinedOutput()
	if err != nil {
		t.Fatalf("callout geometry tests failed: %v\n%s", err, output)
	}
}
