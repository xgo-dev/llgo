/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestConfigureWasmReflectBridges(t *testing.T) {
	tests := []struct {
		name     string
		target   *llssa.Target
		src      string
		expected bool
	}{
		{
			"reachable WASI reflection",
			&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"},
			`package main; import "reflect"; func main() { reflect.ValueOf(func() {}).Call(nil) }`,
			true,
		},
		{
			"dead WASI reflection",
			&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"},
			`package main; import "reflect"; func dead(v reflect.Value) { v.Call(nil) }; func main() {}`,
			false,
		},
		{
			"fmt reflection metadata does not require bridges",
			&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"},
			`package main; import "fmt"; func main() { _ = fmt.Sprintf("%v", 1) }`,
			false,
		},
		{
			"GoJS reflection uses libffi",
			&llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "gojs"},
			`package main; import "reflect"; func main() { reflect.ValueOf(func() {}).Call(nil) }`,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			prog := llssa.NewProgram(test.target)
			defer prog.Dispose()
			ctx := &context{
				prog:    prog,
				progSSA: pkg.Prog,
				initial: []*packages.Package{nil, {}, {Types: types.NewPackage("example.com/missing", "missing")}, {Types: pkg.Pkg}},
			}
			configureWasmReflectBridges(ctx)
			if got := test.target.WasmReflectBridges; got != test.expected {
				t.Fatalf("WasmReflectBridges = %v, want %v", got, test.expected)
			}
		})
	}

	configureWasmReflectBridges(nil)
	if roots := wasmReflectRoots(nil); roots != nil {
		t.Fatalf("wasmReflectRoots(nil) = %v", roots)
	}
}

func TestConfigureWasmFuncInfoEntries(t *testing.T) {
	tests := []struct {
		name      string
		target    *llssa.Target
		buildMode BuildMode
		src       string
		want      bool
	}{
		{
			name:      "plain executable",
			target:    &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "gojs"},
			buildMode: BuildModeExe,
			src:       `package main; func main() {}`,
		},
		{
			name:      "reachable FuncForPC",
			target:    &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "gojs"},
			buildMode: BuildModeExe,
			src:       `package main; import "runtime"; func main() { _ = runtime.FuncForPC(0) }`,
			want:      true,
		},
		{
			name:      "dead FuncForPC",
			target:    &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "gojs"},
			buildMode: BuildModeExe,
			src:       `package main; import "runtime"; func dead() { _ = runtime.FuncForPC(0) }; func main() {}`,
		},
		{
			name:      "wasm library",
			target:    &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "gojs"},
			buildMode: BuildModeCArchive,
			src:       `package main; func main() {}`,
			want:      true,
		},
		{
			name:      "native",
			target:    &llssa.Target{GOOS: "linux", GOARCH: "amd64"},
			buildMode: BuildModeExe,
			src:       `package main; import "runtime"; func main() { _ = runtime.FuncForPC(0) }`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			prog := llssa.NewProgram(test.target)
			defer prog.Dispose()
			ctx := &context{
				prog:      prog,
				progSSA:   pkg.Prog,
				initial:   []*packages.Package{{Types: pkg.Pkg}},
				buildConf: &Config{BuildMode: test.buildMode},
			}
			configureWasmFuncInfoEntries(ctx)
			if got := test.target.WasmFuncInfoEntries; got != test.want {
				t.Fatalf("WasmFuncInfoEntries = %v, want %v", got, test.want)
			}
		})
	}

	configureWasmFuncInfoEntries(nil)
	if programUsesRuntimeFuncForPC(nil, nil) || isRuntimeFuncForPC(nil) {
		t.Fatal("nil program unexpectedly requires Wasm function entries")
	}
	withFuncForPC := buildWasmReflectTestProgram(t, `package p; import "runtime"; func f() { _ = runtime.FuncForPC(0) }`)
	if !programUsesRuntimeFuncForPC(withFuncForPC.Prog, nil) {
		t.Fatal("whole-program scan did not find runtime.FuncForPC")
	}
	withoutFuncForPC := buildWasmReflectTestProgram(t, `package p; func f() {}`)
	if programUsesRuntimeFuncForPC(withoutFuncForPC.Prog, nil) {
		t.Fatal("whole-program scan found an absent runtime.FuncForPC")
	}
}

func TestConfigureWasmProgramAnalysisShared(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main
import (
	"reflect"
	"runtime"
)
func main() {
	reflect.ValueOf(func() {}).Call(nil)
	_ = runtime.FuncForPC(0)
}`)
	target := &llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	ctx := &context{
		prog:      prog,
		progSSA:   pkg.Prog,
		initial:   []*packages.Package{{Types: pkg.Pkg}},
		buildConf: &Config{BuildMode: BuildModeExe},
	}

	configureWasmReflectBridges(ctx)
	analysis := ctx.wasmProgramUse
	if analysis == nil {
		t.Fatal("reflection configuration did not analyze the WebAssembly program")
	}
	configureWasmFuncInfoEntries(ctx)
	if ctx.wasmProgramUse != analysis {
		t.Fatal("reflection bridges and function metadata used different program analyses")
	}
	if !target.WasmReflectBridges || !target.WasmFuncInfoEntries {
		t.Fatalf("WebAssembly features = bridges %v, function entries %v", target.WasmReflectBridges, target.WasmFuncInfoEntries)
	}
}

func TestProgramUsesWasmReflectBridges(t *testing.T) {
	if programUsesWasmReflectBridges(nil, nil) {
		t.Fatal("nil program may not require reflection bridges")
	}
	if isWasmReflectBridgeCall(nil) {
		t.Fatal("nil call may not be a reflection bridge")
	}
	if got := ssaFunctionPackagePath(nil); got != "" {
		t.Fatalf("nil function package path = %q", got)
	}
	if got := ssaFunctionPackagePath(new(ssa.Function)); got != "" {
		t.Fatalf("synthetic function package path = %q", got)
	}
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"no reflection", `package p; func f() int { return 1 }`, false},
		{"metadata only", `package p; import "reflect"; func f() reflect.Type { return reflect.TypeOf(1) }`, false},
		{"value call", `package p; import "reflect"; func f(v reflect.Value) { v.Call(nil) }`, true},
		{"call slice", `package p; import "reflect"; func f(v reflect.Value) { v.CallSlice(nil) }`, true},
		{"make func", `package p; import "reflect"; func f(t reflect.Type, fn func([]reflect.Value) []reflect.Value) { reflect.MakeFunc(t, fn) }`, true},
		{"sequence", `package p; import "reflect"; func f(v reflect.Value) { _ = v.Seq() }`, true},
		{"sequence two", `package p; import "reflect"; func f(v reflect.Value) { _ = v.Seq2() }`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			if got := programUsesWasmReflectBridges(pkg.Prog, nil); got != test.want {
				t.Fatalf("programUsesWasmReflectBridges() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProgramUsesWasmReflectBridgesReachability(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"dead call", `package main; import "reflect"; func dead(v reflect.Value) { v.Call(nil) }; func main() {}`, false},
		{"reachable call", `package main; import "reflect"; func live(v reflect.Value) { v.Call(nil) }; func main() { live(reflect.Value{}) }`, true},
		{"function value call", `package main; import "reflect"; var call = reflect.Value.Call; func main() { call(reflect.Value{}, nil) }`, true},
		{"bound method call", `package main; import "reflect"; func main() { call := reflect.Value{}.Call; call(nil) }`, true},
		{"function value make func", `package main; import "reflect"; var makeFunc = reflect.MakeFunc; func main() { makeFunc(reflect.TypeOf(func() {}), func([]reflect.Value) []reflect.Value { return nil }) }`, true},
		{"function value sequence", `package main; import "reflect"; var sequence = reflect.Value.Seq; func main() { _ = sequence(reflect.ValueOf(1)) }`, true},
		{"interface call", `package main; import "reflect"; type caller interface { Call([]reflect.Value) []reflect.Value }; func main() { var call caller = reflect.Value{}; call.Call(nil) }`, true},
		{"unrelated bound method", `package main; type value int; func (value) M() {}; func main() { var v value; call := v.M; call() }`, false},
		{"unrelated reflect bound method", `package main; import "reflect"; func main() { typ := reflect.TypeOf(0); name := typ.String; _ = name() }`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			roots := []*ssa.Function{pkg.Func("init"), pkg.Func("main")}
			if got := programUsesWasmReflectBridges(pkg.Prog, roots); got != test.want {
				t.Fatalf("reachable programUsesWasmReflectBridges() = %v, want %v", got, test.want)
			}
		})
	}

	if programMayCallWasmReflectBridgeIndirectly(nil) {
		t.Fatal("nil reachable set may not require bridges")
	}
}

func buildWasmReflectTestProgram(t *testing.T, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked := types.NewPackage("example.com/"+file.Name.Name, file.Name.Name)
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()}, fset,
		checked, []*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}
