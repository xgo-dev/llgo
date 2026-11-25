package cabi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestArm64HFAParameterLayout(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, target := range []struct {
		goos, triple string
		stackAlign   uint64
	}{
		{"linux", "aarch64-unknown-linux-gnu", 8},
		{"windows", "aarch64-pc-windows-gnu", 0},
		{"windows", "aarch64-pc-windows-msvc", 0},
		{"darwin", "arm64-apple-darwin", 0},
		{"darwin", "aarch64-unknown-darwin", 0},
		{"darwin", "aarch64-APPLE-DARWIN", 0},
		{"darwin", "aarch64-unknown-macosx", 0},
		{"linux", "", 8},
		{"darwin", "arm64", 0},
		// Explicit LLVM targets override the program's default target.
		{"linux", "aarch64-apple-darwin", 0},
		{"linux", "aarch64-unknown-macosx", 0},
		{"linux", "aarch64-unknown-macosx14.0.0", 0},
		{"darwin", "aarch64-unknown-linux-gnu", 8},
	} {
		t.Run(target.goos+"/"+target.triple, func(t *testing.T) {
			prog := llvmgen.NewProgram(&llvmgen.Target{GOOS: target.goos, GOARCH: "arm64"})
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			tr := NewTransformer(prog, target.triple, "", true)
			for _, base := range []llvm.Type{ctx.FloatType(), ctx.DoubleType()} {
				for n := 1; n <= 4; n++ {
					// Nested arrays must use the same flattened HFA parameter.
					typ := ctx.StructType([]llvm.Type{llvm.ArrayType(base, n)}, false)
					info := checkTypeInfo(t, tr, typ, 1, AttrWidthType, llvm.ArrayType(base, n).String())
					if uint64(info.StackAlign) != target.stackAlign {
						t.Fatalf("%s: stack alignment = %d, want %d", typ, info.StackAlign, target.stackAlign)
					}
					checkTypeInfo(t, tr, typ, 0, AttrNone, typ.String())
				}
			}
			// A zero-sized alignment carrier retains a 16-byte-aligned HFA
			// without contributing another homogeneous element.
			aligned := ctx.StructType([]llvm.Type{llvm.ArrayType(ctx.IntType(128), 0), llvm.ArrayType(ctx.DoubleType(), 2)}, false)
			info := checkTypeInfo(t, tr, aligned, 1, AttrWidthType, "[2 x double]")
			wantAlign := target.stackAlign
			if wantAlign != 0 {
				wantAlign = 16
			}
			if uint64(info.StackAlign) != wantAlign {
				t.Fatalf("aligned HFA: stack alignment = %d, want %d", info.StackAlign, wantAlign)
			}
			// Verify both definitions and indirect calls carry the attribute;
			// changing only the declaration cannot fix stack argument offsets.
			const ir = `
%HFA = type { float, [2 x float] }
%Padded = type { [0 x i128], float }
define double @callee(double %a, %HFA %h) { ret double %a }
define float @padded(%Padded %h) {
  %f = extractvalue %Padded %h, 1
  ret float %f
}
define double @caller(ptr %p, double %a, %HFA %h) {
  %r = call double %p(double %a, %HFA %h)
  ret double %r
}`
			path := filepath.Join(t.TempDir(), "hfa.ll")
			if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
				t.Fatal(err)
			}
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mod, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			defer mod.Dispose()
			tr.TransformModule("test", mod)
			callee := mod.NamedFunction("callee")
			if got := callee.GlobalValueType().ParamTypes()[1].String(); got != "[3 x float]" {
				t.Fatalf("parameter = %s\n%s", got, mod.String())
			}
			check := func(attr llvm.Attribute) {
				t.Helper()
				if target.stackAlign == 0 {
					if !attr.IsNil() {
						t.Fatal("unexpected stack alignment", mod.String())
					}
				} else if attr.IsNil() || attr.GetEnumValue() != target.stackAlign {
					t.Fatal("missing stack alignment", mod.String())
				}
			}
			check(callee.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("alignstack")))
			// A one-float array is only four bytes; its aligned source object
			// still needs sixteen bytes of storage when rebuilding the body.
			padded := mod.NamedFunction("padded")
			allocas := 0
			for inst := padded.EntryBasicBlock().FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if !inst.IsAAllocaInst().IsNil() {
					if tr.Sizeof(inst.AllocatedType()) < 16 || inst.Alignment() < 16 {
						t.Fatal("undersized HFA reconstruction storage", padded.String())
					}
					allocas++
				}
			}
			if allocas != 1 {
				t.Fatalf("got %d reconstruction allocations\n%s", allocas, padded.String())
			}
			calls := 0
			for bb := mod.NamedFunction("caller").FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if call := inst.IsACallInst(); !call.IsNil() {
						check(call.GetCallSiteEnumAttribute(2, llvm.AttributeKindID("alignstack")))
						calls++
					}
				}
			}
			if calls != 1 {
				t.Fatalf("got %d calls\n%s", calls, mod.String())
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err, mod.String())
			}
		})
	}
}
