package c

import (
	"go/build"
	"slices"
	"testing"
)

func TestWasmCTypeFileSelection(t *testing.T) {
	for _, test := range []struct {
		name string
		goos string
		tags []string
		want string
	}{
		{name: "js Memory64", goos: "js", want: "ctypes_wasm64.go"},
		{name: "js wasm32 target", goos: "js", tags: []string{"tinygo.wasm"}, want: "ctypes_wasm.go"},
		{name: "WASI wasm32", goos: "wasip1", want: "ctypes_wasm.go"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := build.Default
			ctx.GOOS = test.goos
			ctx.GOARCH = "wasm"
			ctx.BuildTags = test.tags
			pkg, err := ctx.ImportDir(".", 0)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(pkg.GoFiles, test.want) {
				t.Fatalf("GoFiles = %v, want %s", pkg.GoFiles, test.want)
			}
		})
	}
}
