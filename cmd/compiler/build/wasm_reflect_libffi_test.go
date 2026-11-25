package build

import (
	"github.com/xgo-dev/llgo/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/wasmworkers"
)

func TestConfigureWasmReflectLibffi(t *testing.T) {
	base, err := crosscompile.Use("js", "wasm", "", false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		conf    Config
		workers int
		tags    []string
		wantErr string
	}{
		{"default", Config{Goos: "js", Goarch: "wasm"}, 2, nil, ""},
		{"single worker", Config{Goos: "js", Goarch: "wasm", Tags: wasmReflectLibffiTag}, 1, nil, ""},
		{"native", Config{Goos: "linux", Goarch: "amd64", Tags: wasmReflectLibffiTag}, 1, nil, "requires GOOS=js"},
		{"WASI", Config{Goos: "wasip1", Goarch: "wasm", Tags: wasmReflectLibffiTag}, 1, nil, "requires GOOS=js"},
		{"worker pool", Config{Goos: "js", Goarch: "wasm", Tags: wasmReflectLibffiTag}, 2, nil, "does not support WebAssembly workers"},
		{"explicit worker tag", Config{Goos: "js", Goarch: "wasm", Tags: wasmReflectLibffiTag + ",llgo.wasm.workers"}, 1, nil, "does not support WebAssembly workers"},
		{"target worker tag", Config{Goos: "js", Goarch: "wasm", Tags: wasmReflectLibffiTag}, 1, []string{"llgo.wasm.workers"}, "does not support WebAssembly workers"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			export := base
			export.BuildTags, export.LDFLAGS = test.tags, slices.Clone(base.LDFLAGS)
			err := configureWasmReflectLibffi(&test.conf, &export, wasmworkers.Config{Count: test.workers})
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(strings.Join(export.LDFLAGS, " "), "llgo_reflect_bind"); got != (test.conf.Tags == wasmReflectLibffiTag) {
				t.Fatalf("unexpected reflection exclusions: %v", export.LDFLAGS)
			}
		})
	}
}

func TestBuildRejectsWasmReflectLibffiWorkers(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoWasmWorkers, "2")
	conf := NewDefaultConf(ModeBuild)
	conf.Target, conf.Tags = "emscripten", wasmReflectLibffiTag
	if _, err := Do(nil, conf); err == nil || !strings.Contains(err.Error(), "does not support WebAssembly workers") {
		t.Fatalf("build error = %v, want incompatible reflection backend", err)
	}
}

func TestWasmReflectLibffiProgramFeatures(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main; import "reflect"; func main() { reflect.ValueOf(func() {}).Call(nil) }`)
	for _, target := range []*llvmgen.Target{
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "gojs"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"},
	} {
		t.Run(target.WasmProfile+"/"+target.WasmProvider, func(t *testing.T) {
			prog := llvmgen.NewProgram(target)
			defer prog.Dispose()
			ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: &Config{BuildMode: BuildModeExe, Tags: wasmReflectLibffiTag}, initial: []*packages.Package{
				{Types: types.NewPackage("example.com/plain", "plain")}, {Types: pkg.Pkg},
			}}
			configureWasmReflectBridges(ctx)
			if target.WasmReflectBridges {
				t.Fatal("libffi program enabled typed reflection entries")
			}
			groups := groupInitialBuilds(ctx, nil)
			if len(groups) != 1 || groups[0].features.reflectBridges || len(groups[0].pkgs) != 2 {
				t.Fatalf("libffi reflection split compatible programs: %+v", groups)
			}
		})
	}
}
