//go:build !llgo
// +build !llgo

package build

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestBuildPreservesUnexportedMethodIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unexported-method-identity")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = path
	if _, err := Do([]string{"./testdata/unexportedmethodidentity"}, conf); err != nil {
		t.Fatalf("build unexported method identity fixture: %v", err)
	}
	want := []string{"b", "b"}
	if output := runBinary(t, path); !reflect.DeepEqual(strings.Fields(output), want) {
		t.Fatalf("unexported method calls = %q, want %q", output, strings.Join(want, "\n")+"\n")
	}
}
