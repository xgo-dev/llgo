package lto

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		value string
		want  Mode
	}{
		{"thin", Thin},
		{"full", Full},
	}
	for _, tt := range tests {
		got, err := Parse(tt.value)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("Parse(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	for _, value := range []string{"", "true", "false", "off", "thinlto", "fulllto", "fat"} {
		if _, err := Parse(value); err == nil {
			t.Fatalf("Parse(%q) expected error", value)
		}
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{Off, "off"},
		{Full, "full"},
		{Thin, "thin"},
		{Mode(99), "Mode(99)"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Fatalf("%v.String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestModeEnabled(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{Off, false},
		{Full, true},
		{Thin, true},
		{Mode(99), false},
	}
	for _, tt := range tests {
		if got := tt.mode.Enabled(); got != tt.want {
			t.Fatalf("%v.Enabled() = %t, want %t", tt.mode, got, tt.want)
		}
	}
}

func TestClangFlag(t *testing.T) {
	if got := Off.ClangFlag(); got != "" {
		t.Fatalf("Off.ClangFlag() = %q, want empty", got)
	}
	if got := Full.ClangFlag(); got != "-flto=full" {
		t.Fatalf("Full.ClangFlag() = %q, want -flto=full", got)
	}
	if got := Thin.ClangFlag(); got != "-flto=thin" {
		t.Fatalf("Thin.ClangFlag() = %q, want -flto=thin", got)
	}
}

func TestPassPluginLinkerFlags(t *testing.T) {
	if (PassPlugin{}).Enabled() {
		t.Fatal("empty plugin should be disabled")
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		if got, err := (PassPlugin{}).LinkerFlags(goos); err != nil || got != nil {
			t.Fatalf("disabled plugin flags on %s = %v, err = %v; want nil, nil", goos, got, err)
		}
	}
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			plugin := PassPlugin{Path: "/tmp/plugin dir,with comma/LLGOLTOPlugin.so"}
			want := []string{"-Xlinker", "--load-pass-plugin=" + plugin.Path}
			got, err := plugin.LinkerFlags(goos)
			if err != nil {
				t.Fatalf("LinkerFlags() error: %v", err)
			}
			if !sameStrings(got, want) {
				t.Fatalf("LinkerFlags() = %v, want %v", got, want)
			}
		})
	}
	if got, err := (PassPlugin{Path: "C:/plugins/LLGOLTOPlugin.dll"}).LinkerFlags("windows"); err == nil || got != nil {
		t.Fatalf("windows plugin flags = %v, err = %v; want nil and an unsupported error", got, err)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
