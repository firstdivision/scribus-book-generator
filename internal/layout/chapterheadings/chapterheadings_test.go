package chapterheadings

import (
	"strings"
	"testing"
)

func TestSettingsValidate(t *testing.T) {
	valid := DefaultSettings()
	if err := valid.Validate(); err != nil {
		t.Fatalf("default settings should be valid: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*Settings)
		wantErr string
	}{
		{name: "empty family", mutate: func(s *Settings) { s.Font.Family = " " }, wantErr: "font.family"},
		{name: "empty style", mutate: func(s *Settings) { s.Font.Style = "" }, wantErr: "font.style"},
		{name: "non-positive size", mutate: func(s *Settings) { s.Font.SizePt = 0 }, wantErr: "font.size_pt"},
		{name: "color below range", mutate: func(s *Settings) { s.ColorRGB[0] = -1 }, wantErr: "color_rgb"},
		{name: "color above range", mutate: func(s *Settings) { s.ColorRGB[2] = 256 }, wantErr: "color_rgb"},
		{name: "invalid alignment", mutate: func(s *Settings) { s.Alignment = "justify" }, wantErr: "alignment"},
		{name: "negative top spacing", mutate: func(s *Settings) { s.SpacingMM.Top = -1 }, wantErr: "spacing_mm.top"},
		{name: "negative bottom spacing", mutate: func(s *Settings) { s.SpacingMM.Bottom = -1 }, wantErr: "spacing_mm.bottom"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := valid
			test.mutate(&settings)
			err := settings.Validate()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestResolveAlignment(t *testing.T) {
	tests := []struct {
		alignment Alignment
		isRight   bool
		want      Alignment
	}{
		{alignment: AlignmentLeft, isRight: true, want: AlignmentLeft},
		{alignment: AlignmentCenter, isRight: false, want: AlignmentCenter},
		{alignment: AlignmentRight, isRight: true, want: AlignmentRight},
		{alignment: AlignmentInside, isRight: true, want: AlignmentLeft},
		{alignment: AlignmentOutside, isRight: true, want: AlignmentRight},
		{alignment: AlignmentInside, isRight: false, want: AlignmentRight},
		{alignment: AlignmentOutside, isRight: false, want: AlignmentLeft},
	}

	for _, test := range tests {
		got, err := ResolveAlignment(test.alignment, test.isRight)
		if err != nil {
			t.Fatalf("ResolveAlignment(%q, %t): %v", test.alignment, test.isRight, err)
		}
		if got != test.want {
			t.Fatalf("ResolveAlignment(%q, %t) = %q, want %q", test.alignment, test.isRight, got, test.want)
		}
	}
}
