package buildtags

import (
	"go/build"
	"reflect"
	"runtime"
	"testing"
)

func TestMatch(t *testing.T) {
	ctx := build.Default
	ctx.GOOS = "linux"
	ctx.GOARCH = "amd64"
	ctx.BuildTags = []string{"webkit2_41"}
	for _, test := range []struct {
		expr string
		want bool
	}{
		{expr: "linux", want: true},
		{expr: "windows", want: false},
		{expr: "linux,amd64", want: true},
		{expr: "linux arm64", want: true},
		{expr: "linux && amd64", want: true},
		{expr: "linux || windows", want: true},
		{expr: "linux && (amd64 || arm64)", want: true},
		{expr: "linux\nwindows", want: false},
		{expr: "!webkit2_41", want: false},
		{expr: "webkit2_41", want: true},
	} {
		t.Run(test.expr, func(t *testing.T) {
			if got := Match(&ctx, test.expr); got != test.want {
				t.Fatalf("Match(%q) = %v, want %v", test.expr, got, test.want)
			}
		})
	}
}

func TestCheckTags(t *testing.T) {
	tests := []struct {
		name       string
		buildFlags []string
		testTags   map[string]bool
		want       map[string]bool
	}{
		{
			name:       "mywindows tags",
			buildFlags: []string{"-tags", "mywindows"},
			testTags: map[string]bool{
				"mywindows":         false,
				"!mywindows":        false,
				"mywindows,myamd64": false,
			},
			want: map[string]bool{
				"mywindows":         true,
				"!mywindows":        false,
				"mywindows,myamd64": runtime.GOARCH == "myamd64",
			},
		},
		{
			name:       "non-mywindows tags",
			buildFlags: []string{"-tags", "mylinux"},
			testTags: map[string]bool{
				"mywindows":          false,
				"!mywindows":         false,
				"mylinux,myamd64":    false,
				"!mywindows,myamd64": false,
			},
			want: map[string]bool{
				"mywindows":          false,
				"!mywindows":         true,
				"mylinux,myamd64":    runtime.GOARCH == "myamd64",
				"!mywindows,myamd64": runtime.GOARCH == "myamd64",
			},
		},
		{
			name:       "multiple tags",
			buildFlags: []string{"-tags", "mywindows,myamd64"},
			testTags: map[string]bool{
				"mywindows":         false,
				"myamd64":           false,
				"mywindows,myamd64": false,
				"mylinux,myamd64":   false,
			},
			want: map[string]bool{
				"mywindows":         true,
				"myamd64":           true,
				"mywindows,myamd64": true,
				"mylinux,myamd64":   false,
			},
		},
		{
			name:       "tags with equals format",
			buildFlags: []string{"-tags=mywindows,myamd64"},
			testTags: map[string]bool{
				"mywindows":         false,
				"myamd64":           false,
				"mywindows,myamd64": false,
			},
			want: map[string]bool{
				"mywindows":         true,
				"myamd64":           true,
				"mywindows,myamd64": true,
			},
		},
		{
			name:       "complex tag combinations",
			buildFlags: []string{"-tags", "mylinux,myamd64"},
			testTags: map[string]bool{
				"mywindows":          false,
				"!mywindows":         false,
				"mylinux":            false,
				"mylinux,myamd64":    false,
				"!mywindows,myamd64": false,
			},
			want: map[string]bool{
				"mywindows":          false,
				"!mywindows":         true,
				"mylinux":            true,
				"mylinux,myamd64":    true,
				"!mywindows,myamd64": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTags := make(map[string]bool)
			for tag := range tt.testTags {
				testTags[tag] = false
			}
			CheckTags(tt.buildFlags, testTags)
			if !reflect.DeepEqual(testTags, tt.want) {
				t.Errorf("CheckTags() = %v, want %v", testTags, tt.want)
			}
		})
	}
}

func TestParseBuildTags(t *testing.T) {
	tests := []struct {
		name       string
		buildFlags []string
		want       []string
	}{
		{
			name:       "space separated tags",
			buildFlags: []string{"-tags", "mywindows myamd64"},
			want:       []string{"mywindows", "myamd64"},
		},
		{
			name:       "equals format",
			buildFlags: []string{"-tags=mywindows,myamd64"},
			want:       []string{"mywindows", "myamd64"},
		},
		{
			name:       "multiple -tags flags",
			buildFlags: []string{"-tags", "mywindows", "-tags", "myamd64"},
			want:       []string{"mywindows", "myamd64"},
		},
		{
			name:       "duplicate tags",
			buildFlags: []string{"-tags", "mywindows myamd64", "-tags=mywindows"},
			want:       []string{"mywindows", "myamd64"},
		},
		{
			name:       "empty tags",
			buildFlags: []string{},
			want:       []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseBuildTags(tt.buildFlags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBuildTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
