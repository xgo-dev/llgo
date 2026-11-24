package ir

import (
	"go/token"
	gotypes "go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
)

func TestBuildVerifyAndSummarizeIR(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := NewPackage("example.com/demo", "demo", spec)
	pkg.Exports["example.com/demo.Entry"] = "demo_entry"

	i64 := &lltypes.Basic{Kind: gotypes.Int64}
	goPtrI64 := &lltypes.Pointer{Elem: i64, Kind: lltypes.GoPtr}
	nativePtrI64 := &lltypes.Pointer{Elem: i64, Kind: lltypes.NativePtr}
	cbSig := &lltypes.Func{
		CallConv: lltypes.GoDecl,
		Params:   []*lltypes.Var{{Name: "x", Type: i64}},
		Results:  []*lltypes.Var{{Type: i64}},
	}
	entrySig := &lltypes.Func{
		CallConv: lltypes.GoDecl,
		Params:   []*lltypes.Var{{Name: "n", Type: i64}},
		Results:  []*lltypes.Var{{Type: i64}},
	}

	fn := pkg.NewFunc("example.com/demo.Entry", entrySig)
	entry := fn.NewBlock("entry")
	b := NewBuilder(entry)

	// NoScan stack/heap allocation for pointer-free int64
	slot := b.Alloc(i64, true, true)
	b.Store(slot, fn.Params[0])
	// Pointer-bearing heap cell with explicit write barrier & safepoint
	cell := b.Alloc(goPtrI64, true, false)
	b.WriteBarrier(cell, slot, BarrierStore)
	b.Safepoint(cell)
	_ = b.PtrCast(slot, nativePtrI64)

	// First-class closure construction and call
	clo := b.MakeClosure(cbSig, "example.com/demo.helper", nil)
	res := b.CallClosure(clo, i64, fn.Params[0])
	_ = b.CallDirect("C.puts", lltypes.CDecl, nil)
	_ = b.CallInterface(b.Const(&lltypes.Interface{Empty: false}, "iface"), "Read", 0, i64)
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
	if !sum.HasSafepoints || !sum.HasWriteBarrier || !reflect.DeepEqual(sum.InterfaceCalls, []string{"Read"}) {
		t.Fatalf("unexpected summary: %+v", sum)
	}
}

func TestVerifyRejectsInvalidGCAndCFGRules(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	i64 := &lltypes.Basic{Kind: gotypes.Int64}
	goPtr := &lltypes.Pointer{Elem: i64, Kind: lltypes.GoPtr}
	nativePtr := &lltypes.Pointer{Elem: i64, Kind: lltypes.NativePtr}
	voidSig := &lltypes.Func{CallConv: lltypes.GoDecl}

	t.Run("noscan on pointer type", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunc("p.f", voidSig)
		b := NewBuilder(fn.NewBlock("entry"))
		b.Alloc(goPtr, true, true)
		b.Return()
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "alloc.noscan") {
			t.Fatalf("got %v, want alloc.noscan error", err)
		}
	})

	t.Run("write barrier on native ptr", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunc("p.f", voidSig)
		b := NewBuilder(fn.NewBlock("entry"))
		p := b.Const(nativePtr, "null")
		b.WriteBarrier(p, b.Const(i64, "0"), BarrierStore)
		b.Return()
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "GoPtr") {
			t.Fatalf("got %v, want GoPtr write barrier error", err)
		}
	})

	t.Run("missing terminator", func(t *testing.T) {
		pkg := NewPackage("p", "p", spec)
		fn := pkg.NewFunc("p.f", voidSig)
		b := NewBuilder(fn.NewBlock("entry"))
		b.BinOp(token.ADD, i64, b.Const(i64, "1"), b.Const(i64, "2"))
		if err := VerifyPackage(pkg); err == nil || !strings.Contains(err.Error(), "missing terminator") {
			t.Fatalf("got %v, want missing terminator error", err)
		}
	})
}
