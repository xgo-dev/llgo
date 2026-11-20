package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

func TestWasmHostLayoutHeaderLoads(t *testing.T) {
	markers, err := importer.Default().Import("structs")
	if err != nil {
		t.Fatal(err)
	}
	field := func(name string, typ types.Type) *types.Var {
		return types.NewField(token.NoPos, nil, name, typ, false)
	}
	text := types.Typ[types.String]
	bytes := types.NewSlice(types.Typ[types.Byte])
	record := types.NewStruct([]*types.Var{
		field("String", text), field("Bytes", bytes),
		field("Word", types.Typ[types.Uintptr]), field("Pointer", types.NewPointer(types.Typ[types.Byte])),
	}, nil)
	legacyPkg := types.NewPackage("example.com/hostfields", "hostfields")
	legacy := types.NewNamed(types.NewTypeName(token.NoPos, legacyPkg, "Legacy", nil), record, nil)
	host := types.NewStruct([]*types.Var{
		field("_", markers.Scope().Lookup("HostLayout").Type()),
		field("Items", types.NewArray(record, 2)),
		field("Legacy", legacy),
	}, nil)
	testWasmStringProfiles(t, "host", func(t *testing.T, prog Program, pkg Package) {
		prog.SetTypeCallConv(legacyPkg.Path()+".Legacy", CDecl)
		var observers []llvm.Value
		for _, tc := range []struct {
			name string
			cc   CallConv
		}{{"native", CDecl}, {"local", GoDecl}} {
			global := pkg.NewVar(tc.name, types.NewPointer(host), tc.cc)
			result := types.NewParam(token.NoPos, nil, "", types.Typ[types.Uint32])
			sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(result), false)
			fn := pkg.NewFunc(tc.name+".read", sig, GoDecl)
			b := fn.MakeBody(1)
			items := b.FieldAddr(global.Expr, 1)
			item := b.IndexAddr(items, prog.IntVal(1, prog.Int()))
			textAddr, bytesAddr := b.FieldAddr(item, 0), b.FieldAddr(item, 1)
			for _, addr := range []Expr{textAddr, bytesAddr} {
				b.impl.CreateStore(llvm.ConstInt(prog.tyInt(), 0xa5a5a5a500001000, false), addr.impl)
			}
			for i := 2; i < 4; i++ {
				typ := prog.Elem(b.FieldAddr(item, i).Type)
				if prog.isNativeStorage(typ) || prog.SizeOf(typ) != 8 {
					t.Fatalf("%s field %d lost its Go storage width", tc.name, i)
				}
				native := prog.Elem(b.FieldAddr(b.FieldAddr(global.Expr, 2), i).Type)
				if !prog.isNativeStorage(native) || prog.SizeOf(native) != 4 {
					t.Fatalf("%s cdecl field %d lost its native storage width", tc.name, i)
				}
			}

			// Exercise both a direct field load and a copy of the containing record.
			whole := b.Load(item)
			if whole.Type != prog.rawType(record) {
				t.Fatalf("%s by-value copy retained its storage view", tc.name)
			}
			status := llvm.ConstNull(prog.tyInt32())
			for _, header := range []Expr{b.Load(textAddr), b.Load(bytesAddr), b.Field(whole, 0), b.Field(whole, 1)} {
				slot := b.impl.CreateExtractValue(header.impl, 0, "")
				high := b.impl.CreateExtractValue(slot, 1, "")
				low := llvm.CreatePtrToInt(b.impl, b.impl.CreateExtractValue(slot, 0, ""), prog.tyInt32())
				status = b.impl.CreateOr(status, high, "")
				status = b.impl.CreateOr(status, b.impl.CreateXor(low, llvm.ConstInt(prog.tyInt32(), 0x1000, false), ""), "")
			}
			b.Return(Expr{status, prog.Uint32()})
			b.EndBuild()
			b.Dispose()
			observers = append(observers, fn.impl)
		}
		checkNativeStringZeroResults(t, pkg, observers...)
	})
}

// A typed aggregate load must also work through a plain Go pointer. Count
// instructions before optimization so neither the header traversal nor the
// scratch slot's GC metadata can expand with the number of array elements.
func TestWasmHeaderArrayLoad(t *testing.T) {
	prog := newJ32Program(t)
	prog.EnableGCRoots(true)
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	text := types.Typ[types.String]
	cell := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Label", text, false),
		types.NewField(token.NoPos, nil, "Code", types.Typ[types.Uint32], false),
	}, nil)
	array := types.NewArray(types.NewArray(cell, 32), 32)
	src := types.NewParam(token.NoPos, nil, "src", types.NewPointer(array))
	result := types.NewParam(token.NoPos, nil, "", text)
	fn := pkg.NewFunc("read", types.NewSignatureType(nil, nil, nil, types.NewTuple(src), types.NewTuple(result), false), GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	value := b.Load(fn.Param(0))
	last := b.impl.CreateExtractValue(b.impl.CreateExtractValue(value.impl, 31, ""), 31, "")
	b.Return(Expr{b.impl.CreateExtractValue(last, 0, ""), prog.String()})
	b.EndBuild()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	instructions := 0
	for f := pkg.mod.FirstFunction(); !f.IsNil(); f = llvm.NextFunction(f) {
		for block := f.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				instructions++
			}
		}
	}
	if instructions > 128 {
		t.Fatalf("loading 32x32 string headers generated %d instructions", instructions)
	}
}

func TestWasmHeaderSnapshotSourceAndStorage(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	stringSize := prog.SizeOf(prog.String())
	threshold := int64(llabi.MinAggregateCopySize / stringSize)
	lengths := []int64{8, 9, threshold - 1, 9, threshold}
	params := make([]*types.Var, len(lengths))
	for i, length := range lengths {
		array := types.NewArray(types.Typ[types.String], length)
		params[i] = types.NewParam(token.NoPos, nil, "", types.NewPointer(array))
	}
	fn := pkg.NewFunc("snapshots", types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), nil, false), GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	var scratch llvm.Value
	for i, n := range lengths {
		src := fn.Param(i)
		value := b.Load(src).SetVolatile(true)
		source := value.memoryInstruction()
		if source.IsALoadInst().IsNil() || source.Operand(0) != src.impl || !source.IsVolatile() {
			t.Fatalf("[%d]string lost its volatile source load:\n%s", n, fn.impl.String())
		}
		if n == 8 {
			continue // Inlined headers have no scratch storage.
		}
		slot := value.impl.Operand(0)
		if value.impl.IsVolatile() {
			t.Fatalf("[%d]string marked its private reload volatile", n)
		}
		if n == threshold {
			if slot.IsACallInst().IsNil() {
				t.Fatalf("[%d]string needs a scanned heap snapshot", n)
			}
		} else if scratch.IsNil() {
			scratch = slot
		} else if scratch != slot {
			t.Fatal("successive header snapshots allocated distinct stack slots")
		}
	}
	b.Return()
	b.EndBuild()
	if scratch.IsAAllocaInst().IsNil() || scratch.Operand(0).ZExtValue() != uint64(threshold-1)*stringSize || scratch.Alignment() < 8 {
		t.Fatalf("shared stack slot has insufficient storage: %s", scratch.String())
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
