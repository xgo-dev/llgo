package buildtags

import (
	"reflect"
	"runtime"
	"testing"
)

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
			for k := range tt.testTags {
				testTags[k] = false
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
			got := parseBuildTags(tt.buildFlags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("name: %v, parseBuildTags() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
