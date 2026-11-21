package renderer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenInScribusWaitsAndPassesDocument(t *testing.T) {
	dir := t.TempDir()
	scribus := filepath.Join(dir, "scribus")
	if err := os.WriteFile(scribus, []byte("#!/bin/sh\nprintf '%s' \"$1\" > \"$OPEN_RESULT\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(dir, "my document.sla")
	if err := os.WriteFile(doc, []byte("document"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := filepath.Join(dir, "result")
	t.Setenv("PATH", dir)
	t.Setenv("OPEN_RESULT", result)
	if err := OpenInScribus(doc, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != doc {
		t.Fatalf("got argument %q; want %q", output, doc)
	}
}

func TestOpenInScribusRejectsNonFile(t *testing.T) {
	err := OpenInScribus(t.TempDir(), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
