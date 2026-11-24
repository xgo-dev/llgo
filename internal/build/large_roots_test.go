package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	llssa "github.com/xgo-dev/llgo/internal/llvmgen"
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
		name   string
		target *llssa.Target
		reuse  bool
	}{
		{
			name:   "wasm",
			reuse:  true,
			target: &llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
		},
		{
			name:   "native",
			reuse:  true,
			target: &llssa.Target{GOOS: "linux", GOARCH: "amd64"},
		},
		{name: "native-copy", target: &llssa.Target{GOOS: "linux", GOARCH: "amd64"}},
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
				buildConf:    &Config{Goos: tc.target.GOOS, Goarch: tc.target.GOARCH},
				cTransformer: cabi.NewTransformer(prog, tc.target.Spec().Triple, "", tc.reuse),
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
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			// Reusing the parameter home leaves its old alloca and load available
			// to debug info until LLVM cleans up unused instructions.
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses("function(dce)", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			wrapper := mod.NamedFunction("large_export")
			body := wrapper.String()
			if strings.Contains(body, "load [8192 x i8]") || strings.Contains(body, "store [8192 x i8]") {
				t.Fatalf("wrapper retained a whole-array load or store:\n%s", body)
			}
			impl := mod.NamedFunction("large.impl")
			use := impl.FirstUse()
			if use.IsNil() || !use.NextUse().IsNil() {
				t.Fatalf("expected one call to the implementation:\n%s", body)
			}
			call := use.User()
			byval := llvm.AttributeKindID("byval")
			if call.IsACallInst().IsNil() || call.CalledValue() != impl || call.InstructionParent().Parent() != wrapper ||
				wrapper.GetEnumAttributeAtIndex(1, byval).IsNil() || call.GetCallSiteEnumAttribute(1, byval).IsNil() {
				t.Fatalf("wrapper must call the implementation by value:\n%s", body)
			}
			param := wrapper.Param(0)
			if !tc.reuse {
				if call.Operand(0) == param || !strings.Contains(body, "@llvm.memcpy.") || !strings.Contains(body, "i64 8192, i1 false") {
					t.Fatalf("wrapper must copy the full array to separate storage:\n%s", body)
				}
				return
			}
			if call.Operand(0) != param || !param.FirstUse().NextUse().IsNil() ||
				strings.Contains(body, "alloca") || strings.Contains(body, "@llvm.mem") {
				t.Fatalf("wrapper must forward the incoming byval argument directly:\n%s", body)
			}
		})
	}
}
