package main

import (
	"testing"
)

func TestRunRequiresBookDirectory(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Fatalf("got exit code %d, want 2", code)
	}

	if code := run([]string{"-v"}); code != 2 {
		t.Fatalf("got exit code %d, want 2", code)
	}
}

func TestOpenRequiresExistingDocument(t *testing.T) {
	if code := run([]string{"open"}); code != 2 {
		t.Fatalf("got exit code %d, want 2", code)
	}
	if code := run([]string{"open", "/nonexistent/book.sla"}); code != 1 {
		t.Fatalf("got exit code %d, want 1", code)
	}
}
