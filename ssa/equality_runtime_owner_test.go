package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestRuntimeEqualityRegularMemoryProducerRetainsTypeContext(t *testing.T) {
	runtimeTypes, err := importer.For("source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []Target{
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			prog.SetRuntime(func() *types.Package { return runtimeTypes })
			pkg := prog.NewPackage("equalityowner", "equalityowner")
			builder := pkg.NewFunc("witness", NoArgsNoRet, GoDecl).MakeBody(1)
			field := func(name string, typ types.Type) *types.Var {
				return types.NewField(token.NoPos, nil, name, typ, false)
			}
			for _, test := range []struct {
				name    string
				typ     types.Type
				regular bool
				entry   string
			}{
				{"pointer-free", types.NewArray(types.Typ[types.Uint64], 3), true, ".memequal_varlen"},
				{"pointer-bearing", types.NewStruct([]*types.Var{
					field("P", types.NewPointer(types.Typ[types.Uint64])), field("Tail", types.Typ[types.Uint64]),
				}, nil), true, ".memequal_varlen"},
				{"float", types.NewArray(types.Typ[types.Float64], 3), false, ".arrayequal"},
				{"padding", types.NewStruct([]*types.Var{
					field("Small", types.Typ[types.Byte]), field("Wide", types.Typ[types.Uint64]),
				}, nil), false, ".structequal"},
				{"blank-field", types.NewStruct([]*types.Var{
					field("A", types.Typ[types.Uint64]), field("_", types.Typ[types.Uint64]), field("B", types.Typ[types.Uint64]),
				}, nil), false, ".structequal"},
			} {
				t.Run(test.name, func(t *testing.T) {
					if prog.IsRegularMemory(test.typ) != test.regular {
						t.Fatal("independent regular-memory fixture changed", test.name)
					}
					global := pkg.NewVarEx(test.name, prog.Pointer(prog.Uintptr()))
					fields := builder.abiCommonFields(test.typ, test.name, false, global.impl)
					if test.regular && fields[0].ZExtValue() <= fields[1].ZExtValue() {
						t.Fatal("fixture must distinguish real Size_ from PtrBytes")
					}
					equal := fields[7]
					if equal.OperandsCount() != 2 {
						t.Fatal("type equality lost its concrete two-word function value", equal)
					}
					entry := pkg.FuncOf(equal.Operand(0).Name())
					if entry == nil || !entry.NeedsEnv() || !strings.HasSuffix(entry.Name(), test.entry) {
						t.Fatalf("%s equality entry = %s; want a real env-bearing %s", test.name, equal, test.entry)
					}
					if equal.Operand(1) != builder.abiType(test.typ).impl {
						t.Fatal("equality producer substituted an invented size/GC closure for its actual Type*", equal)
					}
				})
			}
			builder.Return()
			builder.EndBuild()
		})
	}
}
