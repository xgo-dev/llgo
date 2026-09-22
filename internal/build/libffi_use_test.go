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
	"runtime"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestProgramUsesLibffi(t *testing.T) {
	if programUsesLibffi(nil) {
		t.Fatal("nil context may not require libffi")
	}
	if isLibffiCall(nil, libffiOptions{}) {
		t.Fatal("nil call may not require libffi")
	}
	if programHasSetFinalizerPtr(nil) {
		t.Fatal("nil program may not provide SetFinalizerPtr")
	}
	tests := []struct {
		name string
		src  string
		opts libffiOptions
		want bool
	}{
		{"no reflection", `package p; func f() int { return 1 }`, libffiOptions{}, false},
		{"metadata only", `package p; import "reflect"; func f() reflect.Type { return reflect.TypeOf(1) }`, libffiOptions{}, false},
		{"sequence", `package p; import "reflect"; func f(v reflect.Value) { _ = v.Seq() }`, libffiOptions{}, false},
		{"sequence two", `package p; import "reflect"; func f(v reflect.Value) { _ = v.Seq2() }`, libffiOptions{}, false},
		{"value call", `package p; import "reflect"; func f(v reflect.Value) { v.Call(nil) }`, libffiOptions{}, true},
		{"call slice", `package p; import "reflect"; func f(v reflect.Value) { v.CallSlice(nil) }`, libffiOptions{}, true},
		{"make func", `package p; import "reflect"; func f(t reflect.Type, fn func([]reflect.Value) []reflect.Value) { reflect.MakeFunc(t, fn) }`, libffiOptions{}, true},
		{"named finalizer", `package p; import "runtime"; type T int; func fin(*T) {}; func f(p *T) { runtime.SetFinalizer(p, fin) }`, libffiOptions{setFinalizerPtr: true}, false},
		{"nil finalizer", `package p; import "runtime"; type T int; func f(p *T) { runtime.SetFinalizer(p, nil) }`, libffiOptions{setFinalizerPtr: true}, false},
		{"closure finalizer", `package p; import "runtime"; type T int; func f(p *T) { n := 1; runtime.SetFinalizer(p, func(*T) { _ = n }) }`, libffiOptions{setFinalizerPtr: true}, true},
		{"finalizer without ptr", `package p; import "runtime"; type T int; func f(p *T) { n := 1; runtime.SetFinalizer(p, func(*T) { _ = n }) }`, libffiOptions{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			if got := analyzeProgramUse(pkg.Prog, nil).usesLibffi(test.opts); got != test.want {
				t.Fatalf("usesLibffi() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProgramUsesLibffiReachability(t *testing.T) {
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
		{"higher-order make func", `package main; import "reflect"; func invoke(makeFunc func(reflect.Type, func([]reflect.Value) []reflect.Value) reflect.Value) { makeFunc(reflect.TypeOf(func() {}), func([]reflect.Value) []reflect.Value { return nil }) }; func main() { invoke(reflect.MakeFunc) }`, true},
		{"interface call", `package main; import "reflect"; type caller interface { Call([]reflect.Value) []reflect.Value }; func main() { var call caller = reflect.Value{}; call.Call(nil) }`, true},
		{"unrelated reflect bound method", `package main; import "reflect"; func main() { typ := reflect.TypeOf(0); name := typ.String; _ = name() }`, false},
		{"dead sequence", `package main; import "reflect"; func dead(v reflect.Value) { _ = v.Seq() }; func main() {}`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, test.src)
			roots := []*ssa.Function{pkg.Func("init"), pkg.Func("main")}
			if got := analyzeProgramUse(pkg.Prog, roots).usesLibffi(libffiOptions{}); got != test.want {
				t.Fatalf("reachable usesLibffi() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLibffiProgramUseSharesExecutableAnalysis(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main
import "reflect"
func main() { reflect.ValueOf(func() {}).Call(nil) }
`)
	target := &llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	ctx := &context{
		prog:      prog,
		progSSA:   pkg.Prog,
		initial:   []*packages.Package{{Types: pkg.Pkg}},
		buildConf: &Config{BuildMode: BuildModeExe},
		mode:      ModeBuild,
	}
	configureWasmReflectBridges(ctx)
	analysis := ctx.programUse
	if analysis == nil {
		t.Fatal("WASI configuration did not analyze the program")
	}
	if libffiProgramUse(ctx) != analysis {
		t.Fatal("CheckFFI executable scan used a different program analysis")
	}
	if !programUsesLibffi(ctx) {
		t.Fatal("reachable reflect.Call did not require libffi")
	}
}

func TestLibffiUseExecutableRoots(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main; func main() {}`)
	ctx := &context{
		progSSA:   pkg.Prog,
		initial:   []*packages.Package{{Types: pkg.Pkg}},
		buildConf: &Config{BuildMode: BuildModeExe},
		mode:      ModeBuild,
	}
	if !libffiUseExecutableRoots(ctx) {
		t.Fatal("main executable should use init/main roots")
	}
	ctx.mode = ModeGen
	if libffiUseExecutableRoots(ctx) {
		t.Fatal("package generation should scan every function")
	}
	ctx.mode = ModeBuild
	ctx.buildConf.BuildMode = BuildModeCArchive
	if libffiUseExecutableRoots(ctx) {
		t.Fatal("c-archive should scan every function")
	}
	if libffiUseExecutableRoots(nil) {
		t.Fatal("nil context should not use executable roots")
	}
	if libffiUseExecutableRoots(&context{}) {
		t.Fatal("empty context should not use executable roots")
	}
}

func TestLibffiNameHelpers(t *testing.T) {
	if isLibffiReflectName("Seq") || isLibffiReflectName("Seq2") || !isLibffiReflectName("MakeFunc") {
		t.Fatal("libffi reflect names include Call/CallSlice/MakeFunc only")
	}
	if skipLibffiCallSite("fmt") || !skipLibffiCallSite("reflect") || !skipLibffiCallSite("runtime") {
		t.Fatal("libffi call sites skip reflect and runtime")
	}
	if setFinalizerCallNeedsFFI(nil) != true {
		t.Fatal("nil SetFinalizer call must keep libffi")
	}
}

func TestLibffiOptionsAndGuards(t *testing.T) {
	if libffiOptionsFor(nil) != (libffiOptions{}) {
		t.Fatal("nil context should use empty libffi options")
	}
	if libffiProgramUse(nil) == nil {
		t.Fatal("nil context should still produce a program-use view")
	}
	if (*programUse)(nil).usesLibffi(libffiOptions{}) {
		t.Fatal("nil program-use should not require libffi")
	}
	if programMayCallLibffiIndirectly(nil, libffiOptions{}) {
		t.Fatal("nil program-use should not report indirect libffi")
	}
	if isLibffiIndirectFunction(nil, libffiOptions{windows: true}) {
		t.Fatal("nil function is not an indirect libffi entry")
	}
	if programUseFor(nil) != nil {
		t.Fatal("nil context has no cached program-use")
	}
	analyzeProgramUse(nil, nil).eachFunction(nil)
	(*programUse)(nil).eachFunction(func(*ssa.Function) {
		t.Fatal("nil program-use visited a function")
	})

	emptyGOOS := llssa.NewProgram(&llssa.Target{GOARCH: "amd64"})
	defer emptyGOOS.Dispose()
	if got := libffiOptionsFor(&context{prog: emptyGOOS}); got.windows != (runtime.GOOS == "windows") {
		t.Fatalf("empty GOOS windows = %v, want host %s", got.windows, runtime.GOOS)
	}
	win := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "amd64"})
	defer win.Dispose()
	if !libffiOptionsFor(&context{prog: win}).windows {
		t.Fatal("windows target should set windows libffi options")
	}

	pkg := buildWasmReflectTestProgram(t, `package main; func F() {}`)
	ctx := &context{
		progSSA: pkg.Prog,
		initial: []*packages.Package{
			nil,
			{},
			{Types: types.NewPackage("example.com/lib", "lib")},
			{Types: types.NewPackage("example.com/missing", "main")},
			{Types: pkg.Pkg},
		},
		buildConf: &Config{BuildMode: BuildModeExe},
		mode:      ModeBuild,
	}
	if libffiUseExecutableRoots(ctx) {
		t.Fatal("main package without main should not use executable roots")
	}
}

func TestLibffiWindowsNewCallback(t *testing.T) {
	pkg := buildSSAPackage(t, "syscall", `package syscall
func NewCallback(fn any) uintptr { return 0 }
func NewCallbackCDecl(fn any) uintptr { return 0 }
func Open() {}
func useCallback() { NewCallback(nil) }
func useCDecl() { NewCallbackCDecl(nil) }
func useOpen() { Open() }
`)
	opts := libffiOptions{windows: true}
	if !isSyscallNewCallback(pkg.Func("NewCallback")) || !isSyscallNewCallback(pkg.Func("NewCallbackCDecl")) {
		t.Fatal("syscall callback constructors should require libffi on windows")
	}
	if isSyscallNewCallback(nil) || isSyscallNewCallback(pkg.Func("Open")) {
		t.Fatal("non-callback syscall functions should not require libffi")
	}
	if !functionCallsLibffi(pkg.Func("useCallback"), opts) || !functionCallsLibffi(pkg.Func("useCDecl"), opts) {
		t.Fatal("direct NewCallback calls should require libffi on windows")
	}
	if functionCallsLibffi(pkg.Func("useOpen"), opts) || functionCallsLibffi(pkg.Func("useCallback"), libffiOptions{}) {
		t.Fatal("Open and non-windows NewCallback should not require libffi")
	}
	if !isLibffiIndirectName("syscall", "NewCallback", opts) || !isLibffiIndirectName("syscall", "NewCallbackCDecl", opts) {
		t.Fatal("escaped syscall callback names should require libffi on windows")
	}
	if isLibffiIndirectName("syscall", "NewCallback", libffiOptions{}) || isLibffiIndirectName("syscall", "Open", opts) {
		t.Fatal("non-windows or non-callback syscall names should not require libffi")
	}
}

func TestUnwrapSSAValue(t *testing.T) {
	if unwrapSSAValue(&ssa.MakeInterface{}) != nil {
		t.Fatal("MakeInterface with a nil operand should unwrap to nil")
	}
	pkg := buildWasmReflectTestProgram(t, `package p
type I interface{ M() }
type J interface{ M() }
type T int
type P *int
func (T) M() {}
func f(i I, t T, p P, n int) any {
	var j J = i
	q := (*int)(p)
	_ = int(t)
	_ = float64(n)
	_ = q
	return j
}
`)
	fn := pkg.Func("f")
	if fn == nil {
		t.Fatal("missing f")
	}
	var sawChangeIface, sawChangeType, sawConvert bool
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			v, ok := instr.(ssa.Value)
			if !ok {
				continue
			}
			unwrapped := unwrapSSAValue(v)
			switch instr.(type) {
			case *ssa.ChangeInterface:
				sawChangeIface = true
				if unwrapped == v {
					t.Fatal("ChangeInterface was not unwrapped")
				}
			case *ssa.ChangeType:
				sawChangeType = true
				if unwrapped == v {
					t.Fatal("ChangeType was not unwrapped")
				}
			case *ssa.Convert:
				sawConvert = true
				if unwrapped == v {
					t.Fatal("Convert was not unwrapped")
				}
			}
		}
	}
	if !sawChangeIface || !sawChangeType || !sawConvert {
		t.Fatalf("unwrap coverage changeIface=%v changeType=%v convert=%v", sawChangeIface, sawChangeType, sawConvert)
	}
}

func buildSSAPackage(t *testing.T, path, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked := types.NewPackage(path, file.Name.Name)
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
