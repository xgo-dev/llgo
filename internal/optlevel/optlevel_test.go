package optlevel

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want Level
	}{
		{"0", O0},
		{"1", O1},
		{"2", O2},
		{"3", O3},
		{"s", Os},
		{"z", Oz},
		{"O2", O2},
		{"-O3", O3},
		{"  -Oz  ", Oz},
		{"o1", O1},
		{"-os", Os},
		{" -Z ", Oz},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	tests := []string{"", "-", "O", "fast", "4"}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			got, err := Parse(in)
			if err == nil {
				t.Fatalf("Parse(%q) expected error", in)
			}
			if got != Unset {
				t.Fatalf("Parse(%q) = %v, want %v on error", in, got, Unset)
			}
			if !strings.Contains(err.Error(), in) {
				t.Fatalf("Parse(%q) error %q does not mention input", in, err)
			}
		})
	}
}

func TestLevelFormatting(t *testing.T) {
	tests := []struct {
		level  Level
		valid  bool
		name   string
		flag   string
		string string
	}{
		{O0, true, "O0", "-O0", "O0"},
		{O1, true, "O1", "-O1", "O1"},
		{O2, true, "O2", "-O2", "O2"},
		{O3, true, "O3", "-O3", "O3"},
		{Os, true, "Os", "-Os", "Os"},
		{Oz, true, "Oz", "-Oz", "Oz"},
		{Unset, false, "", "", "unset"},
		{Level(255), false, "", "", "unset"},
	}
	for _, tt := range tests {
		t.Run(tt.string, func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.valid {
				t.Fatalf("%v.IsValid() = %v, want %v", tt.level, got, tt.valid)
			}
			if got := tt.level.Name(); got != tt.name {
				t.Fatalf("%v.Name() = %q, want %q", tt.level, got, tt.name)
			}
			if got := tt.level.Flag(); got != tt.flag {
				t.Fatalf("%v.Flag() = %q, want %q", tt.level, got, tt.flag)
			}
			if got := tt.level.String(); got != tt.string {
				t.Fatalf("%v.String() = %q, want %q", tt.level, got, tt.string)
			}
		})
	}
}
