package packages

import "testing"

func TestParseRuntimeGoMinor(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{"go1.21.13", 21},
		{"go1.26.0", 26},
		{"go1.26rc1", 26},
		{"devel go1.27", 0},
		{"go2.0", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := parseRuntimeGoMinor(tt.version); got != tt.want {
			t.Fatalf("parseRuntimeGoMinor(%q) = %d, want %d", tt.version, got, tt.want)
		}
	}
}
