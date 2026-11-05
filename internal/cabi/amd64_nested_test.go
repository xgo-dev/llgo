package cabi

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAMD64NestedAggregateOffsets(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tr := NewTransformer(prog, "x86_64-unknown-linux-gnu", "", false)
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	structure := func(fields ...llvm.Type) llvm.Type { return ctx.StructType(fields, false) }
	name := func(typ llvm.Type) string {
		if typ.IsNil() {
			return "none"
		}
		return typ.String()
	}
	inner := structure(ctx.Int8Type(), ctx.Int32Type())
	outer := structure(ctx.Int8Type(), inner)
	// A zero-sized vector array adds 16-byte alignment without scalar fields,
	// leaving the second eightbyte entirely padding in the tail cases below.
	marker := llvm.ArrayType(llvm.VectorType(ctx.Int32Type(), 4), 0)
	alignedInt := structure(llvm.ArrayType(ctx.Int64Type(), 0), ctx.Int32Type())
	alignedFloat := structure(llvm.ArrayType(ctx.Int64Type(), 0), ctx.FloatType())
	for _, tc := range []struct {
		name               string
		typ, first, second llvm.Type
	}{
		{"embedded-int", outer, ctx.Int64Type(), ctx.Int32Type()},
		{"embedded-float", structure(ctx.Int8Type(), structure(ctx.FloatType(), ctx.FloatType())), ctx.Int64Type(), ctx.FloatType()},
		{"nested-padding", structure(ctx.Int8Type(), outer), ctx.Int64Type(), ctx.Int64Type()},
		{"nested-array", llvm.ArrayType(inner, 2), ctx.Int64Type(), ctx.Int64Type()},
		{"separated-integers", structure(ctx.Int32Type(), alignedInt), ctx.Int64Type(), ctx.Int32Type()},
		{"separated-floats", structure(ctx.FloatType(), alignedFloat), ctx.DoubleType(), ctx.FloatType()},
		{"separated-mixed", structure(ctx.FloatType(), alignedInt), ctx.DoubleType(), ctx.Int32Type()},
		{"aligned-array-stride", llvm.ArrayType(alignedInt, 2), ctx.Int64Type(), ctx.Int32Type()},
		{"empty-tail-register", structure(marker, llvm.ArrayType(ctx.Int8Type(), 3)), ctx.Int64Type(), llvm.Type{}},
		{"empty-float-tail", structure(marker, ctx.FloatType(), ctx.FloatType()), llvm.VectorType(ctx.FloatType(), 2), llvm.Type{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ftyp := llvm.FunctionType(tc.typ, []llvm.Type{tc.typ}, false)
			for _, index := range []int{0, 1} {
				info := tr.GetTypeInfo(ctx, ftyp, tc.typ, index)
				kind := AttrWidthType2
				if tc.second.IsNil() {
					kind = AttrWidthType
				}
				if info.Kind != kind || info.Type1 != tc.first || info.Type2 != tc.second {
					t.Errorf("index %d: kind %v, components %s, %s; want %s, %s", index, info.Kind, name(info.Type1), name(info.Type2), name(tc.first), name(tc.second))
				}
				if !tc.second.IsNil() {
					offset := int(tr.td.ElementOffset(structure(info.Type1, info.Type2), 1))
					if offset != 8 {
						t.Errorf("index %d: second component offset %d, want 8", index, offset)
					}
				}
			}
			for _, optimize := range []bool{false, true} {
				mod := ctx.NewModule(tc.name)
				defer mod.Dispose()
				mod.SetDataLayout(prog.DataLayout())
				mod.SetTarget("x86_64-unknown-linux-gnu")
				b := ctx.NewBuilder()
				defer b.Dispose()
				echo := llvm.AddFunction(mod, "echo", ftyp)
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(echo, "entry"))
				b.CreateRet(echo.Param(0))
				caller := llvm.AddFunction(mod, "caller", ftyp)
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
				b.CreateRet(llvm.CreateCall(b, ftyp, echo, []llvm.Value{caller.Param(0)}))
				NewTransformer(prog, "x86_64-unknown-linux-gnu", "", optimize).TransformModule(tc.name, mod)
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatalf("optimize=%v: %v\n%s", optimize, err, mod.String())
				}
				for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
					for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
						for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
							if !inst.IsAAllocaInst().IsNil() && tr.Sizeof(inst.AllocatedType()) < tr.Sizeof(tc.typ) {
								t.Fatalf("optimize=%v: undersized ABI temporary %s for %s", optimize, inst.String(), tc.typ.String())
							}
						}
					}
				}
			}
		})
	}
}
