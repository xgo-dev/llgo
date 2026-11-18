//go:build !llgo

package llgoext

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// Packed source directives are not implemented yet. Drive the production ABI
// lowering from host Go and compare/link it with real Clang-generated C code.
func TestPackedCABIInterop(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("Clang is required for the C ABI oracle")
	}
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, target := range []struct{ goos, triple string }{
		{"linux", "x86_64-unknown-linux-gnu"},
		{"darwin", "x86_64-apple-darwin"},
	} {
		t.Run(target.goos, func(t *testing.T) {
			prog := ssa.NewProgram(&ssa.Target{GOOS: target.goos, GOARCH: "amd64"})
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule("packed-interop")
			defer mod.Dispose()
			mod.SetTarget(target.triple)
			mod.SetDataLayout(prog.TargetData().String())
			packed := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int64Type()}, true)
			zero := func(n int) llvm.Type {
				return ctx.StructType([]llvm.Type{llvm.ArrayType(ctx.Int8Type(), n), llvm.ArrayType(ctx.Int64Type(), 0)}, true)
			}
			stride := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int8Type()}, true)
			marker := llvm.ArrayType(llvm.VectorType(ctx.Int32Type(), 4), 0)
			cases := []struct {
				name string
				typ  llvm.Type
			}{
				{"packed", packed},
				{"nested", ctx.StructType([]llvm.Type{packed}, false)},
				{"zero1", zero(1)}, {"zero2", zero(2)}, {"zero8", zero(8)},
				{"array", ctx.StructType([]llvm.Type{llvm.ArrayType(stride, 2)}, false)},
				{"aligned", ctx.StructType([]llvm.Type{marker, packed}, false)},
			}
			b := ctx.NewBuilder()
			defer b.Dispose()
			for _, tc := range cases {
				ft := llvm.FunctionType(tc.typ, []llvm.Type{tc.typ}, false)
				callee := llvm.AddFunction(mod, "c_"+tc.name, ft)
				direct := llvm.AddFunction(mod, "llgo_direct_"+tc.name, ft)
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(direct, "entry"))
				b.CreateRet(b.CreateCall(ft, callee, []llvm.Value{direct.Param(0)}, ""))
				indirect := llvm.AddFunction(mod, "llgo_indirect_"+tc.name,
					llvm.FunctionType(tc.typ, []llvm.Type{llvm.PointerType(ctx.Int8Type(), 0), tc.typ}, false))
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(indirect, "entry"))
				b.CreateRet(b.CreateCall(ft, indirect.Param(0), []llvm.Value{indirect.Param(1)}, ""))
			}
			cabi.NewTransformer(prog, target.triple, "", false).TransformModule("packed-interop", mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
			}
			dir := t.TempDir()
			oraclePath := filepath.Join(dir, "oracle.ll")
			runPackedABICommand(t, clang, "-target", target.triple, "-S", "-emit-llvm", "-O0",
				"testdata/packed_abi.c", "-o", oraclePath)
			buf, err := llvm.NewMemoryBufferFromFile(oraclePath)
			if err != nil {
				t.Fatal(err)
			}
			oracle, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			defer oracle.Dispose()
			for _, tc := range cases {
				for _, names := range [][2]string{
					{"llgo_direct_" + tc.name, "c_" + tc.name},
					{"llgo_indirect_" + tc.name, "native_indirect_" + tc.name},
				} {
					t.Run(names[0], func(t *testing.T) {
						got, want := mod.NamedFunction(names[0]), oracle.NamedFunction(names[1])
						if got.GlobalValueType() != want.GlobalValueType() {
							t.Fatalf("LLGo: %s\nClang: %s", got.GlobalValueType(), want.GlobalValueType())
						}
						for index := 1; index <= want.ParamsCount(); index++ {
							for _, name := range []string{"sret", "byval"} {
								id := llvm.AttributeKindID(name)
								ga, wa := got.GetEnumAttributeAtIndex(index, id), want.GetEnumAttributeAtIndex(index, id)
								if ga.IsNil() != wa.IsNil() {
									t.Fatalf("parameter %d: %s differs\n%s\n%s", index, name, got, want)
								}
								if wa.IsNil() {
									continue
								}
								td := prog.TargetData()
								if td.TypeAllocSize(ga.GetTypeValue()) != td.TypeAllocSize(wa.GetTypeValue()) {
									t.Fatalf("parameter %d: %s storage sizes differ", index, name)
								}
								align := got.GetEnumAttributeAtIndex(index, llvm.AttributeKindID("align"))
								actual := uint64(td.ABITypeAlignment(ga.GetTypeValue()))
								if !align.IsNil() {
									actual = align.GetEnumValue()
								}
								align = want.GetEnumAttributeAtIndex(index, llvm.AttributeKindID("align"))
								expected := uint64(td.ABITypeAlignment(wa.GetTypeValue()))
								if !align.IsNil() {
									expected = align.GetEnumValue()
								}
								if actual != expected {
									t.Fatalf("parameter %d: %s alignment %d, Clang %d", index, name, actual, expected)
								}
							}
						}
					})
				}
			}
			t.Run("native", func(t *testing.T) {
				if runtime.GOARCH != "amd64" || runtime.GOOS != target.goos {
					t.Skip("execution requires a matching native SysV AMD64 host; signatures checked above")
				}
				loweredPath := filepath.Join(dir, "lowered.ll")
				if err := os.WriteFile(loweredPath, []byte(mod.String()), 0600); err != nil {
					t.Fatal(err)
				}
				executable := filepath.Join(dir, "interop")
				runPackedABICommand(t, clang, "-O2", "testdata/packed_abi.c", loweredPath, "-o", executable)
				runPackedABICommand(t, executable)
			})
		})
	}
}

func runPackedABICommand(t *testing.T, program string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, program, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", program, args, err, out)
	}
}
