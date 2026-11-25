package ssa

import (
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
)

func TestBuilderAndInstrFormattingAndVerify(t *testing.T) {
	pkg := NewPackage("example.com/cov", "cov", target.Spec{GOOS: "linux", GOARCH: "amd64"})
	i64 := lltypes.Typ[lltypes.Int64]
	boolT := lltypes.Typ[lltypes.Bool]
	ptrI64 := lltypes.NewPointerWithKind(i64, lltypes.GoPtr)
	cPtr := lltypes.NewPointerWithKind(i64, lltypes.CPtr)
	cFuncSig := lltypes.NewSignatureWithCallConv(
		nil,
		lltypes.NewTuple(lltypes.NewParam(0, nil, "a", i64)),
		nil,
		false,
		lltypes.CDecl,
	)
	cFuncPtr := lltypes.NewPointerWithKind(cFuncSig, lltypes.CPtr)
	doMeth := lltypes.NewFunc(0, nil, "Do", lltypes.NewSignature(
		nil,
		lltypes.NewTuple(lltypes.NewParam(0, nil, "", i64)),
		lltypes.NewTuple(lltypes.NewParam(0, nil, "", i64)),
		false,
	))
	ifaceT := lltypes.NewInterfaceType([]*lltypes.Func{doMeth}, nil)

	g := pkg.NewGlobal("cov.G", ptrI64)
	if g == nil || !g.HasPtrData {
		t.Fatalf("NewGlobal(ptrI64) = %+v", g)
	}

	sig := lltypes.NewSignature(
		lltypes.NewParam(0, nil, "r", ptrI64),
		lltypes.NewTuple(
			lltypes.NewParam(0, nil, "cond", boolT),
			lltypes.NewParam(0, nil, "iface", ifaceT),
			lltypes.NewParam(0, nil, "cfn", cFuncPtr),
		),
		lltypes.NewTuple(lltypes.NewParam(0, nil, "", i64)),
		false,
	)
	fn := pkg.NewFunction("cov.Run", sig)
	entry := fn.NewBasicBlock("")
	thenBlk := fn.NewBasicBlock("then")
	elseBlk := fn.NewBasicBlock("else")
	mergeBlk := fn.NewBasicBlock("merge")
	deadBlk := fn.NewBasicBlock("dead")

	b := NewBuilder(entry)
	c1 := b.Const(i64, "1")
	alloc := b.Alloc(i64, true, false)
	b.Store(alloc, c1)
	loaded := b.Load(alloc, i64)
	b.WriteBarrier(alloc, loaded, BarrierStore)
	b.WriteBarrier(alloc, loaded, BarrierMove)
	b.WriteBarrier(alloc, loaded, BarrierBulk)
	b.Safepoint(alloc)
	b.GCRoot(alloc, true)
	b.GCRoot(alloc, false)
	b.KeepAlive(alloc)
	_ = b.PtrCast(alloc, cPtr)
	clos := b.MakeClosure(lltypes.NewSignature(nil, nil, nil, false), "cov.cb", alloc)
	_ = b.CallDirect("cov.helper", lltypes.Go, i64, loaded)
	_ = b.CallClosure(clos, nil)
	_ = b.CallCFunc(fn.Params[3], lltypes.CDecl, nil, loaded)
	_ = b.CallInterface(fn.Params[2], "Do", 0, i64, loaded)
	b.Defer(clos, false)
	b.Defer(clos, true)
	b.RunDefers()
	b.Branch(fn.Params[1], thenBlk, elseBlk)

	b.SetBlock(thenBlk)
	v1 := b.BinOp(token.ADD, i64, loaded, c1)
	b.Jump(mergeBlk)

	b.SetBlock(elseBlk)
	v2 := b.BinOp(token.SUB, i64, loaded, c1)
	b.Jump(mergeBlk)

	b.SetBlock(mergeBlk)
	phi := b.Phi(i64, PhiEdge{Pred: thenBlk, Val: v1}, PhiEdge{Pred: elseBlk, Val: v2})
	b.Return(phi)

	b.SetBlock(deadBlk)
	b.Unreachable()

	if err := VerifyPackage(pkg); err != nil {
		t.Fatalf("VerifyPackage() unexpected error: %v", err)
	}

	for _, blk := range fn.Blocks {
		if blk.Parent() != fn {
			t.Fatalf("block %s has wrong parent", blk.Label)
		}
		if term := blk.Terminator(); term == nil {
			t.Fatalf("block %s missing terminator", blk.Label)
		} else {
			term.termMarker()
			_ = term.Successors()
		}
		for _, ins := range blk.Instrs {
			ins.instrMarker()
			if ins.Block() != blk || ins.Parent() != fn {
				t.Fatalf("instr %s has wrong block/parent", ins.String())
			}
			_ = ins.Operands()
			if ins.String() == "" {
				t.Fatalf("empty String() for %T", ins)
			}
			if val, ok := ins.(Value); ok {
				val.valMarker()
				_ = val.ID()
				_ = val.Name()
			}
		}
	}
	if (&RunDefers{}).Parent() != nil {
		t.Fatal("detached instruction Parent() != nil")
	}
	for _, p := range fn.Params {
		p.valMarker()
		_ = p.ID()
		_ = p.Name()
	}
	c1.valMarker()
	_ = c1.ID()
	_ = c1.Name()

	printed := pkg.String()
	for _, want := range []string{"cov.Run", "gc.writebarrier.store", "gc.safepoint", "gc.root.enter", "call.cdecl", "defer", "rundefers", "unreachable"} {
		if !strings.Contains(printed, want) {
			t.Fatalf("pkg.String() missing %q:\n%s", want, printed)
		}
	}

	summary := Summarize(pkg)
	if !summary.HasSafepoints || !summary.HasWriteBarrier || len(summary.DirectCalls) != 1 || len(summary.ClosureRefs) != 1 || len(summary.InterfaceCalls) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	raw, err := MarshalSummary(summary)
	if err != nil {
		t.Fatal(err)
	}
	roundtrip, err := UnmarshalSummary(raw)
	if err != nil || roundtrip.Path != summary.Path {
		t.Fatalf("UnmarshalSummary = (%+v, %v)", roundtrip, err)
	}

	if err := VerifyPackage(nil); err == nil {
		t.Fatal("VerifyPackage(nil) succeeded")
	}
	if err := VerifyFunction(nil); err == nil {
		t.Fatal("VerifyFunction(nil) succeeded")
	}
	badPkg := NewPackage("bad", "bad", target.Spec{GOOS: "linux", GOARCH: "amd64"})
	badFn := badPkg.NewFunction("bad.EmptyBlock", lltypes.NewSignature(nil, nil, nil, false))
	_ = badFn.NewBasicBlock("empty")
	if err := VerifyFunction(badFn); err == nil {
		t.Fatal("VerifyFunction accepted empty block")
	}
}
