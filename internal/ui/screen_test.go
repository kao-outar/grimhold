package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPadIgnoresANSI(t *testing.T) {
	s := HPBar(5, 10, 10)
	padded := pad(s, 30)
	if n := visibleLen(padded); n != 30 {
		t.Fatalf("visible width %d, want 30 (%q)", n, stripANSI(padded))
	}
	if utf8.RuneCountInString(padded) <= 30 {
		t.Fatal("expected raw length to include escape codes")
	}
}

func TestDangerLabelColors(t *testing.T) {
	cases := []struct {
		danger int
		color  string
	}{
		{0, escGreen},
		{1, escYellow},
		{2, escMagenta},
		{3, escRed},
		{4, escRed},
	}
	for _, tc := range cases {
		label := DangerLabel(tc.danger)
		if !strings.Contains(label, tc.color) {
			t.Fatalf("danger %d label %q missing %q", tc.danger, label, tc.color)
		}
		if !strings.HasSuffix(label, escReset) {
			t.Fatalf("danger %d label not reset: %q", tc.danger, label)
		}
	}
}
