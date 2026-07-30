//go:build !llgo
// +build !llgo

package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/xgo-dev/llvm"

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
)

func init() {
	llssa.Initialize(llssa.InitAll)
}

func TestGenMainModuleExecutable(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg,
		&genConfig{rtInit: true, pyInit: true})
	if mod.ExportFile != "foo.a-main" {
		t.Fatalf("unexpected export file: %s", mod.ExportFile)
	}
	ir := mod.LPkg.String()
	checks := []string{
		"define i32 @main(",
		"call void @Py_Initialize()",
		"call void @Py_Finalize()",
		"call void @\"example.com/foo.init\"()",
		"define weak void @_start()",
	}
	for _, want := range checks {
		if !strings.Contains(ir, want) {
			t.Fatalf("main module IR missing %q:\n%s", want, ir)
		}
	}
	assertInOrder(t, ir,
		"call void @Py_Initialize()",
		"call void @\"example.com/foo.init\"()",
		"call void @\"example.com/foo.main\"()",
		"call void @Py_Finalize()",
	)
}

func TestGenMainModuleWASIAsyncifyEntry(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "wasip1",
			Goarch:    "wasm",
		},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{})
	ir := mod.LPkg.String()
	checks := []string{
		`define hidden ptr @__llgo_wasm_main(ptr %0)`,
		`call void @"github.com/goplus/llgo/runtime/internal/runtime.init"()`,
		`call void @"example.com/foo.init"()`,
		`call void @"example.com/foo.main"()`,
		`call void @"github.com/goplus/llgo/runtime/internal/runtime.RunWasmMain"()`,
	}
	for _, want := range checks {
		if !strings.Contains(ir, want) {
			t.Fatalf("WASI main module IR missing %q:\n%s", want, ir)
		}
	}
	entryStart := strings.Index(ir, "define hidden i32 @__main_argc_argv(")
	if entryStart < 0 {
		t.Fatalf("WASI main module missing host entry:\n%s", ir)
	}
	entry := ir[entryStart:]
	entry = entry[:strings.Index(entry, "}\n")+2]
	if strings.Contains(entry, `call void @"example.com/foo.init"()`) ||
		strings.Contains(entry, `call void @"example.com/foo.main"()`) {
		t.Fatalf("WASI system-stack entry calls package main directly:\n%s", entry)
	}
}

func TestGenMainModuleWasmResumeEntry(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	prog.EnableWasmResumeABI(true)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "wasip1",
			Goarch:    "wasm",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{})
	if err := lowerWasmResumeModule(ctx, mod.LPkg.Module()); err != nil {
		t.Fatal(err)
	}
	ir := mod.LPkg.String()
	for _, want := range []string{
		`define ptr @__llgo_wasm_start.__llgo_wasm_main`,
		`define internal i8 @__llgo_wasm_resume.__llgo_wasm_main`,
		`@"__llgo_wasm_resume_desc.example.com/foo.init" = external global`,
		`@"__llgo_wasm_resume_desc.example.com/foo.main" = external global`,
		`define hidden ptr @__llgo_wasm_main(ptr %0)`,
		`call void @"github.com/goplus/llgo/runtime/internal/runtime.RunWasmMain"()`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("resumable main module IR missing %q:\n%s", want, ir)
		}
	}
}

func TestGenMainModuleLibrary(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeCArchive,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{})
	ir := mod.LPkg.String()
	if strings.Contains(ir, "define i32 @main") {
		t.Fatalf("library mode should not emit main function:\n%s", ir)
	}
	if !strings.Contains(ir, "@__llgo_argc = global i32 0") {
		t.Fatalf("library mode missing argc global:\n%s", ir)
	}
	if !strings.Contains(ir, "@llvm.global_ctors") {
		t.Fatalf("library mode missing constructor:\n%s", ir)
	}
}

func TestGenMainModuleLibraryInitializesRuntime(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, mode := range []BuildMode{BuildModeCArchive, BuildModeCShared} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				buildConf: &Config{
					BuildMode: mode,
					Goos:      "linux",
					Goarch:    "amd64",
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{rtInit: true})
			ir := mod.LPkg.String()
			checks := []string{
				"define internal void @__llgo_runtime_ctor()",
				"call void @\"github.com/goplus/llgo/runtime/internal/runtime.init\"()",
				"call void @\"example.com/foo.init\"()",
			}
			if mode == BuildModeCShared {
				checks = append(checks, `@__llgo_runtime_ctor_init = hidden constant ptr @__llgo_runtime_ctor, section ".init_array"`)
			} else {
				checks = append(checks, "@llvm.global_ctors = appending global")
			}
			for _, want := range checks {
				if !strings.Contains(ir, want) {
					t.Fatalf("library module IR missing %q:\n%s", want, ir)
				}
			}
			if strings.Contains(ir, "define i32 @main") {
				t.Fatalf("library mode should not emit main function:\n%s", ir)
			}
		})
	}
}

func TestGenMainModuleTestLibraryDefersMainInit(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, mode := range []BuildMode{BuildModeCArchive, BuildModeCShared} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				mode: ModeTest,
				buildConf: &Config{
					Mode:      ModeTest,
					BuildMode: mode,
					Goos:      "linux",
					Goarch:    "amd64",
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{rtInit: true})
			ir := mod.LPkg.String()
			if !strings.Contains(ir, "call void @\"github.com/goplus/llgo/runtime/internal/runtime.init\"()") {
				t.Fatalf("test library constructor missing runtime init:\n%s", ir)
			}
			if strings.Contains(ir, "call void @\"example.com/foo.init\"()") {
				t.Fatalf("test library constructor initialized test main before the C runner supplied argc/argv:\n%s", ir)
			}
		})
	}
}

func TestGenMainModuleInstallsLocalContextWhenNeeded(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	prog := llssa.NewProgram(nil)
	runtimePkg := types.NewPackage(llssa.PkgRuntime, "runtime")
	contextName := types.NewTypeName(token.NoPos, runtimePkg, "LocalContext", nil)
	contextType := types.NewNamed(contextName, types.NewStruct(nil, nil), nil)
	runtimePkg.Scope().Insert(contextName)
	contextPointer := types.NewPointer(contextType)
	enterParams := types.NewTuple(types.NewParam(token.NoPos, runtimePkg, "ctx", contextPointer))
	enterResults := types.NewTuple(types.NewParam(token.NoPos, runtimePkg, "previous", types.Typ[types.Uintptr]))
	runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, "EnterLocalContext", types.NewSignatureType(nil, nil, nil, enterParams, enterResults, false)))
	leaveParams := types.NewTuple(
		types.NewParam(token.NoPos, runtimePkg, "ctx", contextPointer),
		types.NewParam(token.NoPos, runtimePkg, "previous", types.Typ[types.Uintptr]),
	)
	runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, "LeaveLocalContext", types.NewSignatureType(nil, nil, nil, leaveParams, nil, false)))
	prog.SetRuntime(runtimePkg)
	prog.SetLocalityInfo("example.com/state.Value", llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
	prog.SetLocalStorage("example.com/state.Value", llssa.LocalStoragePackage)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{}).LPkg.String()
	assertInOrder(t, ir,
		"EnterLocalContext",
		`call void @"example.com/foo.init"()`,
		`call void @"example.com/foo.main"()`,
		"LeaveLocalContext",
	)
}

func assertInOrder(t *testing.T, s string, wants ...string) {
	t.Helper()
	offset := 0
	for _, want := range wants {
		i := strings.Index(s[offset:], want)
		if i < 0 {
			t.Fatalf("main module IR missing ordered entry %q after byte %d:\n%s", want, offset, s)
		}
		offset += i + len(want)
	}
}
