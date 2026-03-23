package scaling

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  Hello World  ", "hello world"},
		{"UPPER", "upper"},
		{"already lower", "already lower"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := Normalize(tt.input); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		if got := Clamp(tt.v, tt.min, tt.max); got != tt.want {
			t.Errorf("Clamp(%v, %v, %v) = %v, want %v", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		input string
		width int
		want  string
	}{
		{"hello world foo bar", 11, "hello world\nfoo bar"},
		{"short", 20, "short"},
		{"", 10, ""},
	}
	for _, tt := range tests {
		if got := Wrap(tt.input, tt.width); got != tt.want {
			t.Errorf("Wrap(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
	}
}
