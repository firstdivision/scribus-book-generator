package main

import "testing"

func TestRunRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"one.pdf", "two.pdf"},
		{"-dpi", "0", "book.pdf"},
		{"-dpi", "299", "book.pdf"},
		{"-dpi", "1201", "book.pdf"},
		{"-dpi", "abc", "book.pdf"},
		{"book.sla"},
	} {
		if code := run(args); code != 2 {
			t.Errorf("run(%v) = %d, want 2", args, code)
		}
	}
}

func TestRunMissingInput(t *testing.T) {
	if code := run([]string{"missing.pdf"}); code != 1 {
		t.Fatalf("got exit code %d, want 1", code)
	}
}
