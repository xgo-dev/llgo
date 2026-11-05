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
	marker := llvm.ArrayType(llvm.VectorType(ctx.Int32Type(), 4), 0)
	for _, tc := range []struct {
		name               string
		typ, first, second llvm.Type
	}{
		{"embedded-int", outer, ctx.Int64Type(), ctx.Int32Type()},
		{"embedded-float", structure(ctx.Int8Type(), structure(ctx.FloatType(), ctx.FloatType())), ctx.Int64Type(), ctx.FloatType()},
		{"nested-padding", structure(ctx.Int8Type(), outer), ctx.Int64Type(), ctx.Int64Type()},
		{"nested-array", llvm.ArrayType(inner, 2), ctx.Int64Type(), ctx.Int64Type()},
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
			}
		})
	}
}
