package build

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestLLGoFileCompilerLanguage(t *testing.T) {
	ctx := &context{buildConf: &Config{}}
	for _, tc := range []struct {
		name string
		file string
		args []string
		want []string
	}{
		{name: "C default", file: "p.c", want: []string{"-x", "c"}},
		{name: "Objective-C default", file: "p.m", want: []string{"-x", "objective-c"}},
		{name: "C++ extension", file: "p.cpp"},
		{name: "uppercase C is C++", file: "p.C"},
		{name: "uppercase M is Objective-C++", file: "p.M"},
		{name: "explicit Objective-C", file: "p.c", args: []string{"-x", "objective-c"}, want: []string{"-x", "c", "-x", "objective-c"}},
		{name: "joined language", file: "p.c", args: []string{"-xc++"}, want: []string{"-x", "c", "-xc++"}},
		{name: "reset language", file: "p.c", args: []string{"-x", "none"}, want: []string{"-x", "c", "-x", "none"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.args)
			got := llgoFileCompilerArgs(ctx, tc.args, tc.file)
			want := append(slices.Clone(tc.want), debugInfoCompilerArgs(ctx.buildConf, &ctx.crossCompile)...)
			if !slices.Equal(got, want) {
				t.Fatalf("compiler arguments = %q, want %q", got, want)
			}
			if !reflect.DeepEqual(tc.args, before) {
				t.Fatal("compiler arguments modified the caller's slice")
			}
		})
	}
}

func TestBuildCgoObjectiveC(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Objective-C integration fixture uses Foundation")
	}
	if testing.Short() {
		t.Skip("builds and runs a native executable")
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "objc")
	if _, err := Do([]string{"./testdata/cgoobjc"}, conf); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "objc: 3 4" {
		t.Fatalf("Objective-C fixture: %v\n%s", err, output)
	}
}
