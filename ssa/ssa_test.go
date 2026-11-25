package ssa

import (
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestBuildVerifyAndSummarizeIR(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := NewPackage("example.com/demo", "demo", spec)
	pkg.Exports["example.com/demo.Entry"] = "demo_entry"

	i64 := types.Typ[types.Int64]
	goPtrI64 := types.NewPointerWithKind(i64, types.GoPtr)
	cPtrI64 := types.NewPointerWithKind(i64, types.CPtr)
	cbSig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "x", i64)),
		types.NewTuple(types.NewParam(0, nil, "", i64)),
		false,
	)
	entrySig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "n", i64)),
		types.NewTuple(types.NewParam(0, nil, "", i64)),
		false,
	)

	fn := pkg.NewFunction("example.com/demo.Entry", entrySig)
	entry := fn.NewBasicBlock("entry")
	b := NewBuilder(entry)

	// NoScan stack/heap allocation for pointer-free int64
	slot := b.Alloc(i64, true, true)
	b.Store(slot, fn.Params[0])
	// Pointer-bearing heap cell with explicit write barrier & safepoint
	cell := b.Alloc(goPtrI64, true, false)
	b.WriteBarrier(cell, slot, BarrierStore)
	b.Safepoint(cell)
	_ = b.PtrCast(slot, cPtrI64)

	// First-class closure construction and call
	clo := b.MakeClosure(cbSig, "example.com/demo.helper", nil)
	res := b.CallClosure(clo, i64, fn.Params[0])
	_ = b.CallDirect("C.puts", types.CDecl, nil)
	readMeth := types.NewFunc(0, nil, "Read", types.NewSignature(nil, nil, types.NewTuple(types.NewParam(0, nil, "", i64)), false))
	_ = b.CallInterface(b.Const(types.NewInterfaceType([]*types.Func{readMeth}, nil), "iface"), "Read", 0, i64)
	b.KeepAlive(cell)
	b.Return(res)

	if err := VerifyPackage(pkg); err != nil {
		t.Fatalf("VerifyPackage failed: %v\n%s", err, pkg.String())
	}
	text := pkg.String()
	for _, want := range []string{"alloc.heap.noscan", "gc.writebarrier.store", "gc.safepoint", "makeclosure @example.com/demo.helper", "call.closure", "ptrcast"} {
		if !strings.Contains(text, want) {
			t.Fatalf("IR dump missing %q:\n%s", want, text)
		}
	}

	sum := Summarize(pkg)
	raw, err := MarshalSummary(sum)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := UnmarshalSummary(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sum, roundTrip) {
		t.Fatalf("summary round-trip mismatch: got %+v, want %+v", roundTrip, sum)
	}
	if !sum.HasSafepoints || !sum.HasWriteBarrier ||
		!reflect.DeepEqual(sum.DirectCalls, []string{"C.puts"}) ||
		!reflect.DeepEqual(sum.ClosureRefs, []string{"example.com/demo.helper"}) ||
		!reflect.DeepEqual(sum.InterfaceCalls, []string{"Read"}) {
		t.Fatalf("unexpected summary: %+v", sum)
	}
}

func TestVerifyRejectsInvalidGCAndCFGRules(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	i64 := types.Typ[types.Int64]
	goPtr := types.NewPointerWithKind(i64, types.GoPtr)
	cPtr := types.NewPointerWithKind(i64, types.CPtr)
	voidSig := types.NewSignature(nil, nil, nil, false)

	t.Run("noscan on pointer type", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.f", voidSig)
		b := NewBuilder(fn.NewBasicBlock("entry"))
		b.Alloc(goPtr, true, true)
		b.Return()
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "alloc.noscan") {
			t.Fatalf("got %v, want alloc.noscan error", err)
		}
	})

	t.Run("write barrier on C ptr", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.f", voidSig)
		b := NewBuilder(fn.NewBasicBlock("entry"))
		p := b.Const(cPtr, "null")
		b.WriteBarrier(p, b.Const(i64, "0"), BarrierStore)
		b.Return()
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "GoPtr") {
			t.Fatalf("got %v, want GoPtr write barrier error", err)
		}
	})

	t.Run("untyped operands rejected without panic", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.f", voidSig)
		b := NewBuilder(fn.NewBasicBlock("entry"))
		voidCall := b.CallDirect("C.nop", types.CDecl, nil)
		b.WriteBarrier(voidCall, b.Const(i64, "0"), BarrierStore)
		b.Return()
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "untyped operand") {
			t.Fatalf("got %v, want untyped operand error", err)
		}

		pkg2 := NewPackage("p", "p", spec)
		fn2 := pkg2.NewFunction("p.f", voidSig)
		b2 := NewBuilder(fn2.NewBasicBlock("entry"))
		b2.CallClosure(b2.Const(nil, "bad"), nil)
		b2.Return()
		if err := VerifyPackage(pkg2); err == nil || !strings.Contains(err.Error(), "untyped operand") {
			t.Fatalf("got %v, want untyped operand error", err)
		}
	})

	t.Run("missing terminator", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.f", voidSig)
		b := NewBuilder(fn.NewBasicBlock("entry"))
		b.BinOp(token.ADD, i64, b.Const(i64, "1"), b.Const(i64, "2"))
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "missing terminator") {
			t.Fatalf("got %v, want missing terminator error", err)
		}
	})

	t.Run("valid diamond phi passes", func(t *testing.T) {
		boolT := types.Typ[types.Bool]
		sig := types.NewSignature(
			nil,
			types.NewTuple(
				types.NewParam(0, nil, "cond", boolT),
				types.NewParam(0, nil, "a", i64),
				types.NewParam(0, nil, "b", i64),
			),
			types.NewTuple(types.NewParam(0, nil, "", i64)),
			false,
		)
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.select", sig)
		entry := fn.NewBasicBlock("entry")
		thenBlk := fn.NewBasicBlock("then")
		elseBlk := fn.NewBasicBlock("else")
		merge := fn.NewBasicBlock("merge")

		NewBuilder(entry).Branch(fn.Params[0], thenBlk, elseBlk)
		NewBuilder(thenBlk).Jump(merge)
		NewBuilder(elseBlk).Jump(merge)
		mb := NewBuilder(merge)
		phi := mb.Phi(i64, PhiEdge{Pred: thenBlk, Val: fn.Params[1]}, PhiEdge{Pred: elseBlk, Val: fn.Params[2]})
		mb.Return(phi)
		if err := VerifyPackage(pkg); err != nil {
			t.Fatalf("valid phi rejected: %v", err)
		}
	})

	t.Run("phi referencing foreign function block and param rejected", func(t *testing.T) {
		sig := types.NewSignature(
			nil,
			types.NewTuple(types.NewParam(0, nil, "x", i64)),
			types.NewTuple(types.NewParam(0, nil, "", i64)),
			false,
		)
		pkg := NewPackage("p", "p", spec)
		other := pkg.NewFunction("p.other", sig)
		otherEntry := other.NewBasicBlock("entry")
		NewBuilder(otherEntry).Return(other.Params[0])

		fn := pkg.NewFunction("p.bad", sig)
		entry := fn.NewBasicBlock("entry")
		merge := fn.NewBasicBlock("merge")
		NewBuilder(entry).Jump(merge)
		mb := NewBuilder(merge)
		phi := mb.Phi(i64, PhiEdge{Pred: otherEntry, Val: other.Params[0]})
		mb.Return(phi)
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "does not belong to function") {
			t.Fatalf("got %v, want cross-function ownership error", err)
		}

		pkg2 := NewPackage("p", "p", spec)
		other2 := pkg2.NewFunction("p.other", sig)
		NewBuilder(other2.NewBasicBlock("entry")).Return(other2.Params[0])
		fn2 := pkg2.NewFunction("p.badparam", sig)
		e2 := fn2.NewBasicBlock("entry")
		m2 := fn2.NewBasicBlock("merge")
		NewBuilder(e2).Jump(m2)
		mb2 := NewBuilder(m2)
		mb2.Return(mb2.Phi(i64, PhiEdge{Pred: e2, Val: other2.Params[0]}))
		if err := VerifyPackage(pkg2); err == nil || !strings.Contains(err.Error(), "does not belong to function") {
			t.Fatalf("got %v, want foreign param error", err)
		}

		pkg3 := NewPackage("p", "p", spec)
		other3 := pkg3.NewFunction("p.other", sig)
		oe3 := other3.NewBasicBlock("entry")
		NewBuilder(oe3).Return(other3.Params[0])
		fn3 := pkg3.NewFunction("p.badpred", sig)
		e3 := fn3.NewBasicBlock("entry")
		m3 := fn3.NewBasicBlock("merge")
		NewBuilder(e3).Jump(m3)
		mb3 := NewBuilder(m3)
		mb3.Return(mb3.Phi(i64, PhiEdge{Pred: oe3, Val: fn3.Params[0]}))
		if err := VerifyPackage(pkg3); err == nil || !strings.Contains(err.Error(), "does not belong to function") {
			t.Fatalf("got %v, want foreign block error", err)
		}
	})

	t.Run("phi non-predecessor and placement rejected", func(t *testing.T) {
		sig := types.NewSignature(
			nil,
			types.NewTuple(types.NewParam(0, nil, "x", i64)),
			types.NewTuple(types.NewParam(0, nil, "", i64)),
			false,
		)
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunction("p.badcfg", sig)
		entry := fn.NewBasicBlock("entry")
		dead := fn.NewBasicBlock("dead")
		merge := fn.NewBasicBlock("merge")
		NewBuilder(entry).Jump(merge)
		NewBuilder(dead).Return(fn.Params[0])
		mb := NewBuilder(merge)
		mb.Return(mb.Phi(i64, PhiEdge{Pred: dead, Val: fn.Params[0]}))
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "not a CFG predecessor") {
			t.Fatalf("got %v, want non-predecessor error", err)
		}

		pkg2 := NewPackage("p", "p", spec)
		fn2 := pkg2.NewFunction("p.latephi", sig)
		e2 := fn2.NewBasicBlock("entry")
		m2 := fn2.NewBasicBlock("merge")
		NewBuilder(e2).Jump(m2)
		mb2 := NewBuilder(m2)
		_ = mb2.BinOp(token.ADD, i64, fn2.Params[0], mb2.Const(i64, "1"))
		mb2.Return(mb2.Phi(i64, PhiEdge{Pred: e2, Val: fn2.Params[0]}))
		if err := VerifyPackage(pkg2); err == nil || !strings.Contains(err.Error(), "phi instruction after non-phi") {
			t.Fatalf("got %v, want late-phi error", err)
		}
	})
}
