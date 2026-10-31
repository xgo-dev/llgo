package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestLowerMainCExportModuleWindowsVectors(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	target := &llssa.Target{GOOS: "windows", GOARCH: "amd64"}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	pkg := prog.NewPackage("exportprobe", "exportprobe")
	rawPkg := types.NewPackage("simd/archsimd", "archsimd")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "v128", nil), types.NewStruct(nil, nil), nil)
	vector := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Float32x4", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, rawPkg, "tag", marker, false),
		types.NewField(token.NoPos, rawPkg, "vals", types.NewArray(types.Typ[types.Float32], 4), false),
	}, nil), nil)
	exports := []cExport{{goName: "main.Exported", cName: "Exported", sig: newSignature([]types.Type{vector}, []types.Type{vector})}}
	defineCExportWrappers(pkg, exports, nil)
	ctx := &context{
		prog: prog, buildConf: &Config{Goos: "windows", Goarch: "amd64"},
		cTransformer: cabi.NewTransformer(prog, target.Spec().Triple, "", true),
	}
	if changed, err := lowerMainCExportModule(ctx, pkg, exports); err != nil || !changed {
		t.Fatalf("C export lowering = (%t, %v)", changed, err)
	}
	mod := pkg.Module()
	for _, name := range []string{"Exported", "main.Exported"} {
		fn := mod.NamedFunction(name)
		if fn.Param(0).Type().TypeKind() != llvm.PointerTypeKind || fn.GlobalValueType().ReturnType().TypeKind() != llvm.VectorTypeKind {
			t.Fatalf("Win64 export must use an indirect vector parameter and direct vector return:\n%s", fn.String())
		}
	}
	if !strings.Contains(mod.NamedFunction("Exported").String(), "@main.Exported(ptr ") {
		t.Fatalf("export call does not match the lowered implementation:\n%s", mod.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestLowerLargeAggregatesGCPolicy(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, enabled := range []bool{false, true} {
		func() {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer prog.Dispose()
			prog.EnableGCRoots(enabled)
			mod := prog.NewPackage("large", "large").Module()
			ctx := mod.Context()
			typ := llvm.ArrayType(ctx.Int8Type(), 65537)
			fn := llvm.AddFunction(mod, "large", llvm.FunctionType(typ, nil, false))
			b := ctx.NewBuilder()
			defer b.Dispose()
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
			b.CreateRet(llvm.ConstNull(typ))
			if lowerMainCExportAggregates(prog, mod, nil) {
				t.Fatal("empty C export set requested ABI lowering")
			}
			if !lowerMainCExportAggregates(prog, mod, []cExport{{}}) {
				t.Fatal("non-empty C export set skipped ABI lowering")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if got := !mod.NamedGlobal("llvm_gc_root_chain").IsNil(); got != enabled {
				t.Fatalf("GC roots enabled=%v: root chain present=%v", enabled, got)
			}
		}()
	}
}

func TestLowerMainCExportModuleCopies(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, tc := range []struct {
		name, goos, goarch string
		target             *llssa.Target
	}{
		{
			name:   "wasm",
			goos:   "js",
			goarch: "wasm",
			target: &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
		},
		{
			name:   "native",
			goos:   "linux",
			goarch: "amd64",
			target: &llssa.Target{GOOS: "linux", GOARCH: "amd64"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := llssa.NewProgram(tc.target)
			defer prog.Dispose()
			pkg := prog.NewPackage("large", "large")
			sig := newSignature([]types.Type{types.NewArray(types.Typ[types.Uint8], 8192)}, nil)
			exports := []cExport{{goName: "large.impl", cName: "large_export", sig: sig}}
			defineCExportWrappers(pkg, exports, nil)
			ctx := &context{
				prog:         prog,
				buildConf:    &Config{Goos: tc.goos, Goarch: tc.goarch},
				cTransformer: cabi.NewTransformer(prog, tc.target.Spec().Triple, "", true),
			}
			if changed, err := lowerMainCExportModule(ctx, pkg, nil); err != nil || changed {
				t.Fatalf("empty C export module lowering = (%t, %v), want (false, nil)", changed, err)
			}
			changed, err := lowerMainCExportModule(ctx, pkg, exports)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("C export module skipped lowering")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := pkg.Module().NamedFunction("large_export").String()
			if strings.Contains(body, "load [8192 x i8]") || strings.Contains(body, "store [8192 x i8]") ||
				!strings.Contains(body, "@llvm.mem") {
				t.Fatalf("C export wrapper retained a large aggregate copy:\n%s", body)
			}
		})
	}
}
