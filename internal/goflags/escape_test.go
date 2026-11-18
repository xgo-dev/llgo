package goflags

import (
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
)

func TestEscapeDiagnosticScopesAndCounts(t *testing.T) {
	for _, tc := range []struct {
		flags                  []string
		selected, dependencies int
		loader                 []string
	}{
		{[]string{"-gcflags=-m"}, 1, 0, []string{"-gcflags="}},
		{[]string{"-gcflags=all=-m=2"}, 2, 2, []string{"-gcflags=all="}},
		{[]string{"-gcflags=all=-m", "-gcflags=-m=2"}, 2, 1, []string{"-gcflags=all=", "-gcflags="}},
		{[]string{"-gcflags=-m", "-gcflags=all=-m=2"}, 2, 2, []string{"-gcflags=", "-gcflags=all="}},
		{[]string{"-gcflags=all=-m=2", "-gcflags="}, 0, 2, []string{"-gcflags=all=", "-gcflags="}},
		{[]string{"-gcflags=all=-m=2", "-gcflags=all="}, 0, 0, []string{"-gcflags=all=", "-gcflags=all="}},
		{[]string{"-gcflags=--m --m=true"}, 2, 0, []string{"-gcflags="}},
		{[]string{"-gcflags=-m=2 -m=false -m"}, 1, 0, []string{"-gcflags="}},
		{[]string{"-gcflags=all='-lang=go1.27' '-m=2' -N"}, 2, 2, []string{"-gcflags=all=-lang=go1.27 -N"}},
	} {
		conf := new(build.Config)
		if err := ApplyBuildFlags(conf, tc.flags); err != nil {
			t.Fatal(err)
		}
		if conf.EscapeDiagnostics != tc.selected || conf.DependencyEscapeDiagnostics != tc.dependencies || !reflect.DeepEqual(conf.LoaderBuildFlags, tc.loader) {
			t.Fatalf("%v: levels=%d/%d loader=%q", tc.flags, conf.EscapeDiagnostics, conf.DependencyEscapeDiagnostics, conf.LoaderBuildFlags)
		}
		// Reapplying flags must retain scope and override information, while
		// the host metadata loader never sees native diagnostic controls.
		if err := ApplyBuildFlags(conf, nil); err != nil {
			t.Fatal(err)
		}
		if conf.EscapeDiagnostics != tc.selected || conf.DependencyEscapeDiagnostics != tc.dependencies || !reflect.DeepEqual(conf.LoaderBuildFlags, tc.loader) {
			t.Fatal("reapplication changed diagnostics")
		}
	}
}
