//go:build !llgo && !wasm

package cgo

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
)

// Build in-process so host Go coverage records the compiler paths exercised by
// the fixture, even though its source and runtime assertions live under test/.
// mode optionally selects a single build mode; omission uses exe.
func buildCExportFixture(t *testing.T, dir, output string, mode ...string) {
	t.Helper()
	t.Chdir(dir)
	conf := build.NewDefaultConf(build.ModeBuild)
	conf.OutFile = output
	if len(mode) != 0 {
		conf.BuildMode = build.BuildMode(mode[0])
	}
	if _, err := build.Do([]string{"."}, conf); err != nil {
		t.Fatal(err)
	}
}
