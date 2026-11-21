package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
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

func TestWasmHeaderSnapshotCopyLowering(t *testing.T) {
	for _, tc := range []struct{ volatile, delayed, sparse bool }{
		{},
		{delayed: true},
		{volatile: true},
		{volatile: true, delayed: true},
		{sparse: true},
		{delayed: true, sparse: true},
		{volatile: true, sparse: true},
		{volatile: true, delayed: true, sparse: true},
	} {
		prog := newJ32Program(t)
		setTestRuntime(t, prog)
		pkg := prog.NewPackage("p", "example.com/p")
		var aggregate types.Type = types.NewArray(types.Typ[types.String], 256)
		lastHeader := 255
		if tc.sparse {
			// One header still needs memory lowering at the 4 KiB boundary.
			aggregate = types.NewStruct([]*types.Var{
				types.NewField(token.NoPos, nil, "Data", types.NewArray(types.Typ[types.Uint64], 510), false),
				types.NewField(token.NoPos, nil, "Text", types.Typ[types.String], false),
			}, nil)
			lastHeader = 1
		}
		ptr := types.NewPointer(aggregate)
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "src", ptr), types.NewParam(token.NoPos, nil, "dst", ptr))
		fn := pkg.NewFunc("copyHeaders", types.NewSignatureType(nil, nil, nil, params, nil, false), GoDecl)
		b := fn.MakeBody(1)
		src, dst := fn.Param(0), fn.Param(1)
		value := b.Load(src).SetVolatile(tc.volatile)
		if tc.delayed {
			// An intervening call can collect and overwrite the source.
			sig := llvm.FunctionType(prog.ctx.VoidType(), []llvm.Type{src.impl.Type()}, false)
			mutate := llvm.AddFunction(pkg.mod, "mutate", sig)
			b.impl.CreateCall(sig, mutate, []llvm.Value{src.impl}, "")
		}
		b.Store(dst, value)
		if tc.delayed {
			last := b.impl.CreateExtractValue(value.impl, lastHeader, "")
			b.impl.CreateStore(last, dst.impl)
		}
		b.Return()
		b.EndBuild()
		b.Dispose()
		cfg := llabi.AggregateLoweringConfig{GoWordSize: prog.GoWordSize(), GCRoots: true, Wasm: true}
		llabi.LowerLargeAggregates(prog.TargetData(), pkg.mod, cfg)
		llabi.LowerAggregateCopies(prog.TargetData(), pkg.mod, cfg)
		if err := llvm.VerifyModule(pkg.mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}

		var slot, clear, read, write, field llvm.Value
		allocs := 0
		for block := fn.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if !inst.IsAGetElementPtrInst().IsNil() && !slot.IsNil() && inst.Operand(0) == slot {
					field = inst
				}
				if inst.IsACallInst().IsNil() {
					continue
				}
				name := inst.CalledValue().Name()
				switch {
				case strings.HasSuffix(name, ".AllocU"):
					allocs++
					slot = inst
				case strings.HasPrefix(name, "__llgo_clear_header_range"):
					clear = inst
				case strings.HasPrefix(name, "llvm.memmove"), strings.HasPrefix(name, "llvm.memcpy"):
					if inst.Operand(1) == src.impl {
						if !read.IsNil() {
							t.Fatal("aggregate copy read its source twice")
						}
						read = inst
					}
					if inst.Operand(0) == dst.impl {
						write = inst
					}
				}
			}
		}
		if read.IsNil() || !strings.HasPrefix(read.CalledValue().Name(), "llvm.memmove") || write.IsNil() || clear.IsNil() {
			t.Fatalf("missing source read, destination write or padding normalization:\n%s", fn.impl)
		}
		if tc.volatile || tc.delayed {
			if allocs != 1 || read.Operand(0) != slot || (read.Operand(3).ZExtValue() != 0) != tc.volatile ||
				clear.Operand(0) != slot || write.Operand(1) != slot || write.Operand(3).ZExtValue() != 0 {
				t.Fatalf("snapshot must retain the source read and its volatility:\n%s", fn.impl)
			}
			if tc.delayed && (!strings.HasPrefix(write.CalledValue().Name(), "llvm.memcpy") || field.IsNil() ||
				field.Operand(2).ZExtValue() != uint64(lastHeader) || !strings.Contains(fn.impl.String(), "gcroot.entry")) {
				t.Fatalf("delayed assignment must retain one rooted, normalized snapshot:\n%s", fn.impl)
			}
		} else if allocs != 0 || read != write || clear.Operand(0) != dst.impl {
			t.Fatalf("immediate assignment must normalize in the destination:\n%s", fn.impl)
		}
	}
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
	lengths := []int64{1, 2, 8, 9, threshold - 1, 9, threshold}
	params := make([]*types.Var, len(lengths))
	for i, length := range lengths {
		array := types.NewArray(types.Typ[types.String], length)
		params[i] = types.NewParam(token.NoPos, nil, "", types.NewPointer(array))
	}
	fn := pkg.NewFunc("snapshots", types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), nil, false), GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	var scratch, consume, heapSlot llvm.Value
	for i, n := range lengths {
		src := fn.Param(i)
		value := b.Load(src).SetVolatile(true)
		source := value.memoryInstruction()
		if source.IsALoadInst().IsNil() || source.Operand(0) != src.impl || !source.IsVolatile() {
			t.Fatalf("[%d]string lost its volatile source load:\n%s", n, fn.impl.String())
		}
		if n == 1 {
			continue // A single string stays in SSA; larger arrays need copy lowering.
		}
		slot := value.impl.Operand(0)
		if value.impl.IsVolatile() {
			t.Fatalf("[%d]string marked its private reload volatile", n)
		}
		if n == threshold {
			if slot.IsACallInst().IsNil() {
				t.Fatalf("[%d]string needs a scanned heap snapshot", n)
			}
			// A by-value consumer still needs the normalized aggregate in SSA.
			heapSlot = slot
			sig := llvm.FunctionType(prog.ctx.VoidType(), []llvm.Type{value.impl.Type()}, false)
			consume = b.impl.CreateCall(sig, llvm.AddFunction(pkg.mod, "consume", sig), []llvm.Value{value.impl}, "")
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
	cfg := llabi.AggregateLoweringConfig{GoWordSize: prog.GoWordSize(), Wasm: true}
	llabi.LowerLargeAggregates(prog.TargetData(), pkg.mod, cfg)
	llabi.LowerAggregateCopies(prog.TargetData(), pkg.mod, cfg)
	arg := consume.Operand(0)
	if arg.IsALoadInst().IsNil() || arg.Operand(0) != heapSlot {
		t.Fatalf("by-value call lost its normalized snapshot:\n%s", fn.impl)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
