package ssa

import (
	"bytes"
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"log"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

func TestChangeTypeNamedInterfaceIdentity(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", LLVMTarget: "wasm32-unknown-emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			testChangeTypeNamedInterfaceIdentity(t, target)
		})
	}
}

func testChangeTypeNamedInterfaceIdentity(t *testing.T, target *Target) {
	prog := NewProgram(target)
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkgTypes := types.NewPackage("example.com/interfaceidentity", "interfaceidentity")
	method := types.NewFunc(token.NoPos, pkgTypes, "M", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	underlying := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	src := types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "Source", nil), underlying, nil)
	dst := types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "Destination", nil), underlying, nil)
	pkg := prog.NewPackage(pkgTypes.Name(), pkgTypes.Path())
	fn := pkg.NewFunc("Convert", types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", src)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", dst)), false), GoDecl)
	b := fn.MakeBody(1)
	// Materialize descriptors before measuring cache allocation alone.
	b.abiType(dst)
	pkg.NewVarEx("removable", prog.Pointer(prog.Int()))
	replacement := pkg.NewVarEx("replacement", prog.Pointer(prog.Int()))
	var converted Expr
	func() {
		var trace bytes.Buffer
		previousOutput, previousDebug := log.Writer(), debugInstr
		defer func() {
			log.SetOutput(previousOutput)
			debugInstr = previousDebug
		}()
		log.SetOutput(&trace)
		SetDebug(DbgFlagInstruction)
		converted = b.ChangeType(prog.Type(dst, GoDecl), fn.Param(0))
		if !strings.Contains(trace.String(), "ChangeType "+dst.String()) {
			t.Fatalf("named interface conversion omitted its debug trace: %s", trace.String())
		}
	}()
	cacheCount := func() int {
		count := 0
		for name := range pkg.vars {
			if strings.HasPrefix(name, "_llgo_ifaceconv$") {
				count++
			}
		}
		return count
	}
	if got := cacheCount(); got != 1 {
		t.Fatalf("first conversion allocated %d caches, want 1", got)
	}
	if !pkg.ReplaceVarWith("removable", replacement.Expr) {
		t.Fatal("failed to remove temporary global")
	}
	b.ChangeType(prog.Type(dst, GoDecl), fn.Param(0))
	if got := cacheCount(); got != 2 {
		t.Fatalf("conversions separated by global removal allocated %d caches, want 2", got)
	}

	if !types.Identical(converted.RawType(), dst) {
		t.Fatalf("converted type = %v, want %v", converted.RawType(), dst)
	}
	if converted.impl == fn.Param(0).impl {
		t.Fatal("conversion retained the source interface table")
	}
	if same := b.ChangeType(prog.Type(dst, GoDecl), converted); same.impl != converted.impl {
		t.Fatal("identical interface conversion rebuilt the value")
	}
	before := pkg.String()
	if same := b.ChangeInterface(prog.Type(dst, GoDecl), converted); same.impl != converted.impl || !types.Identical(same.RawType(), dst) {
		t.Fatal("same-type ChangeInterface changed the value or type")
	}
	if got := cacheCount(); got != 2 {
		t.Fatalf("same-type ChangeInterface allocated a cache: got %d, want 2", got)
	}
	if pkg.String() != before {
		t.Fatal("same-type ChangeInterface emitted additional IR")
	}
	b.Return(converted)
	b.EndBuild()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for index, target := range []types.Type{dst, types.NewInterfaceType(nil, nil).Complete(), types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "Empty", nil), types.NewInterfaceType(nil, nil).Complete(), nil)} {
		for _, commaOK := range []bool{false, true} {
			name := fmt.Sprintf("Assert%d", index)
			results := types.NewTuple(types.NewVar(token.NoPos, nil, "", target))
			if commaOK {
				name += "OK"
				results = types.NewTuple(types.NewVar(token.NoPos, nil, "", target), types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]))
			}
			fn := pkg.NewFunc(name, types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "value", src)), results, false), GoDecl)
			b := fn.MakeBody(1)
			asserted := b.TypeAssert(fn.Param(0), prog.Type(target, GoDecl), commaOK)
			if commaOK {
				b.Return(b.Extract(asserted, 0), b.Extract(asserted, 1))
			} else {
				b.Return(asserted)
			}
			b.EndBuild()
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	emptyType := types.NewInterfaceType(nil, nil).Complete()
	for i, pair := range [][2]types.Type{{src, emptyType}, {emptyType, dst}} {
		fn := pkg.NewFunc(fmt.Sprintf("Unknown%d", i), types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "value", pair[0])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", pair[1])), false), GoDecl)
		b := fn.MakeBody(1)
		converted := b.ChangeInterface(prog.Type(pair[1], GoDecl), fn.Param(0))
		if !types.Identical(converted.RawType(), pair[1]) {
			t.Fatalf("unknown conversion type = %v, want %v", converted.RawType(), pair[1])
		}
		b.Return(converted)
		b.EndBuild()
		ir := fn.impl.String()
		if i == 0 && (!strings.Contains(ir, "IfaceType") || strings.Contains(ir, "NewItab")) {
			t.Fatalf("non-empty to empty conversion must retain the dynamic type without an itab:\n%s", ir)
		}
		if i == 1 && !strings.Contains(ir, "NewItab") {
			t.Fatalf("unknown empty to non-empty conversion must build the destination itab:\n%s", ir)
		}
	}
	concrete := types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "Concrete", nil), types.Typ[types.Int], nil)
	recv := types.NewVar(token.NoPos, pkgTypes, "", types.NewPointer(concrete))
	concrete.AddMethod(types.NewFunc(token.NoPos, pkgTypes, "M", types.NewSignatureType(recv, nil, nil, nil, nil, false)))
	knownFn := pkg.NewFunc("Known", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "", dst)), false), GoDecl)
	knownBuilder := knownFn.MakeBody(1)
	known := knownBuilder.MakeInterface(prog.Type(src, GoDecl), prog.Zero(prog.Type(types.NewPointer(concrete), GoDecl)))
	known = knownBuilder.ChangeType(prog.Type(dst, GoDecl), known)
	empty := types.NewInterfaceType(nil, nil).Complete()
	known = knownBuilder.ChangeInterface(prog.Type(empty, GoDecl), known)
	known = knownBuilder.ChangeInterface(prog.Type(dst, GoDecl), known)
	selected := Expr{knownBuilder.impl.CreateSelect(prog.Val(true).impl, known.impl, known.impl, ""), known.Type}
	selected = knownBuilder.ChangeInterface(prog.Type(src, GoDecl), selected)
	packed := knownBuilder.Aggregate(known.Type, knownBuilder.Extract(known, 0), knownBuilder.Extract(known, 1))
	packed = knownBuilder.ChangeInterface(prog.Type(src, GoDecl), packed)
	if !types.Identical(knownBuilder.knownInterfaceConcreteType(packed.impl), types.NewPointer(concrete)) {
		t.Fatal("insert/extract propagation lost the known concrete type")
	}
	if got := cacheCount(); got != 2 {
		t.Fatalf("aggregate propagation allocated a cache: got %d, want 2", got)
	}
	if !types.Identical(knownBuilder.knownInterfaceConcreteType(selected.impl), types.NewPointer(concrete)) {
		t.Fatal("select propagation lost the known concrete type")
	}
	if got := cacheCount(); got != 2 {
		t.Fatalf("select propagation allocated a cache: got %d, want 2", got)
	}
	if !types.Identical(knownBuilder.Func.interfaceConcreteTypes[known.impl], types.NewPointer(concrete)) {
		t.Fatal("conversion chain lost the known concrete type")
	}
	if got := cacheCount(); got != 2 {
		t.Fatalf("known concrete conversions allocated a cache: got %d, want 2", got)
	}
	knownBuilder.Return(known)
	knownBuilder.EndBuild()
	if strings.Contains(knownFn.impl.String(), "ConvertIface") || strings.Contains(knownFn.impl.String(), "IfaceType") {
		t.Fatalf("known concrete conversion used dynamic lookup:\n%s", knownFn.impl.String())
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	name, _ := prog.abi.TypeName(dst)
	if !strings.Contains(ir, "ConvertIface\"(ptr @\""+name+"\",") || !strings.Contains(ir, "_llgo_ifaceconv$") {
		t.Fatalf("conversion does not rebuild the destination interface table:\n%s", ir)
	}
}

func TestDirectInterfaceDataKinds(t *testing.T) {
	ptr := types.NewPointer(types.Typ[types.Int])
	cases := []types.Type{
		ptr,
		types.Typ[types.UnsafePointer],
		types.NewMap(types.Typ[types.Int], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	}
	for _, typ := range append([]types.Type(nil), cases...) {
		cases = append(cases, types.NewArray(typ, 1),
			types.NewStruct([]*types.Var{types.NewVar(token.NoPos, nil, "Value", typ)}, nil))
	}
	for _, typ := range append([]types.Type(nil), cases...) {
		cases = append(cases, types.NewNamed(types.NewTypeName(token.NoPos, nil, "Named", nil), typ, nil))
	}
	for _, typ := range cases {
		for _, is32Bits := range []bool{false, true} {
			if !directIfaceType(typ) {
				t.Fatalf("%v should use direct interface storage", typ)
			}
			kind, _, _ := abi.DataKindOf(typ, 0, is32Bits)
			if kind != abi.Pointer {
				t.Fatalf("direct interface type %v has data kind %v, want Pointer", typ, kind)
			}
		}
	}
	for _, typ := range []types.Type{types.Typ[types.Int], types.Typ[types.Float64],
		types.NewArray(ptr, 2), types.NewSlice(ptr), types.NewStruct(nil, nil)} {
		if directIfaceType(typ) {
			t.Fatalf("%v should use boxed interface storage", typ)
		}
	}
}
