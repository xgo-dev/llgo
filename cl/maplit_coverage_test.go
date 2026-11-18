package cl

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func firstMapLitUpdate(t *testing.T, fn *ssa.Function) *ssa.MapUpdate {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if update, ok := instr.(*ssa.MapUpdate); ok {
				return update
			}
		}
	}
	t.Fatal("missing map update")
	return nil
}

func TestMapLitMalformedCoverage(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { X int }
func F() map[int]item { var s item; s.X = 1; return map[int]item{0: s} }
`)
	var makeMap *ssa.MakeMap
	var update *ssa.MapUpdate
	var load *ssa.UnOp
	var alloc *ssa.Alloc
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch x := instr.(type) {
			case *ssa.MakeMap:
				makeMap = x
			case *ssa.MapUpdate:
				update = x
			case *ssa.UnOp:
				if x.Op == token.MUL {
					load = x
				}
			case *ssa.Alloc:
				alloc = x
			}
		}
	}
	if makeMap == nil || update == nil || load == nil || alloc == nil {
		t.Fatal("missing map composite SSA values")
	}
	setMapLitRefs(makeMap, &ssa.Store{})
	if mapLitSafeToDelay(makeMap, []*ssa.MapUpdate{update}) {
		t.Fatal("early store publication was accepted")
	}
	setMapLitRefs(makeMap, update)
	setMapLitRefs(alloc, load)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("load without field stores was accepted")
	}
	setMapLitRefs(alloc)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("allocation without referrers was accepted")
	}
}

func TestMapLitComplitMalformedReferrers(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { X int }
func F() map[int]item { var s item; s.X = 1; return map[int]item{0: s} }
`)
	var load *ssa.UnOp
	var alloc *ssa.Alloc
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch x := instr.(type) {
			case *ssa.UnOp:
				if x.Op == token.MUL {
					load = x
				}
			case *ssa.Alloc:
				alloc = x
			}
		}
	}
	if load == nil || alloc == nil {
		t.Fatal("missing composite values")
	}
	other := &ssa.UnOp{Op: token.MUL, X: alloc}
	setMapLitRefs(alloc, load, other)
	setMapLitRefs(load, other)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("unrelated load accepted")
	}
	setMapLitRefs(alloc, load)
	setMapLitRefs(load)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("missing load ref accepted")
	}
}

func TestMapLitConvertedValueRejectsInvalidChain(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("invalid conversion chain did not panic")
		}
	}()
	p := &context{}
	var b llssa.Builder
	p.mapLitConvertedValue(b, &ssa.Const{}, nil, llssa.Expr{})
}

func TestMapLitClassifyAndComplitGuardsFromRealSSA(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { X int }
func F() map[int]item { var s item; s.X = 1; return map[int]item{0: s} }
`)
	update := firstMapLitUpdate(t, fn)
	load, alloc, _, _, ok := mapValueComplit(update.Value)
	if !ok {
		t.Fatal("safe composite was rejected")
	}
	plan := &mapLitPlan{}
	origLoadRefs := append([]ssa.Instruction(nil), *load.Referrers()...)
	setMapLitRefs(load, origLoadRefs[0], &ssa.Return{})
	if classifyStaticComplit([]*ssa.MapUpdate{update}, plan) {
		t.Fatal("shared composite load was accepted")
	}
	setMapLitRefs(load, origLoadRefs...)

	var field *ssa.FieldAddr
	for _, ref := range *alloc.Referrers() {
		if fa, ok := ref.(*ssa.FieldAddr); ok {
			field = fa
			break
		}
	}
	if field == nil {
		t.Fatal("missing field address")
	}
	origFieldRefs := append([]ssa.Instruction(nil), *field.Referrers()...)
	setMapLitRefs(field, origFieldRefs[0], &ssa.Return{})
	if _, _, _, _, ok := mapValueComplit(update.Value); ok {
		t.Fatal("shared field address was accepted")
	}
	setMapLitRefs(field, origFieldRefs...)

	origAllocRefs := append([]ssa.Instruction(nil), *alloc.Referrers()...)
	setMapLitRefs(alloc, append(origAllocRefs, &ssa.Return{})...)
	if _, _, _, _, ok := mapValueComplit(update.Value); ok {
		t.Fatal("escaped composite storage was accepted")
	}
	setMapLitRefs(alloc, origAllocRefs...)
}

func TestMapLitValueComplitRejectsSharedBoxedField(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { V any }
func box(x int) any { return x }
func F() (any, map[int]item) {
	v := box(1)
	return v, map[int]item{0: {V: v}}
}
`)
	if _, _, _, _, ok := mapValueComplit(firstMapLitUpdate(t, fn).Value); ok {
		t.Fatal("shared boxed field was accepted")
	}
}

func TestMapLitValueComplitRejectsNonAllocLoad(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { X int }
func F(p *item) map[int]item { return map[int]item{0: *p} }
`)
	if _, _, _, _, ok := mapValueComplit(firstMapLitUpdate(t, fn).Value); ok {
		t.Fatal("load of a non-allocation was accepted")
	}
}

func TestMapLitClassifyMissingStructField(t *testing.T) {
	fn := buildTestSSA(t, `package foo
type item struct { X, Y int }
func F() map[int]item { var s item; s.X = 1; return map[int]item{0: s} }
`)
	update := firstMapLitUpdate(t, fn)
	if _, _, _, _, ok := mapValueComplit(update.Value); !ok {
		t.Fatal("partial composite was rejected before field classification")
	}
	if classifyStaticComplit([]*ssa.MapUpdate{update}, &mapLitPlan{}) {
		t.Fatal("composite missing a field was accepted")
	}
}

func TestMapLitConstantCallRejectsTruncatedArgs(t *testing.T) {
	fn := buildTestSSA(t, `package foo
func box(x int) any { return x }
func F() any { return box(1) }
`)
	var call *ssa.Call
	for _, instr := range fn.Blocks[0].Instrs {
		if c, ok := instr.(*ssa.Call); ok {
			call = c
		}
	}
	if call == nil {
		t.Fatal("missing boxing call")
	}
	call.Call.Args = nil
	if _, ok := constantMakeCall(call); ok {
		t.Fatal("call with truncated arguments was accepted")
	}
}

func TestMapLitCompileCoversUpdateAndComplitFallbacks(t *testing.T) {
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg := prog.NewPackage("fallback", "fallback")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "n", types.Typ[types.Int])), nil, false)
	fn := pkg.NewFunc("test", sig, llssa.GoDecl)
	b := fn.MakeBody(1)
	p := &context{prog: prog, pkg: pkg, bvals: map[ssa.Value]llssa.Expr{}}
	mapTy := p.prog.Type(types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), llssa.GoDecl)
	m := b.MakeMap(mapTy, p.prog.Val(1))
	c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])

	genericFn := buildTestSSA(t, `package foo
func F() map[int]int { m := make(map[int]int); m[1] = 1; return m }`)
	generic := firstMapLitUpdate(t, genericFn)
	p.bvals[generic.Map] = m
	if p.compileMapLitUpdate(b, generic) {
		t.Fatal("update with nil mapLitLoops was accepted")
	}

	first := &ssa.MapUpdate{Map: generic.Map, Key: c, Value: c}
	last := &ssa.MapUpdate{Map: generic.Map, Key: c, Value: c}
	plan := &mapLitPlan{kind: mapLitGeneric, updates: []*ssa.MapUpdate{first, last}}
	p.mapLitLoops = map[*ssa.MapUpdate]*mapLitPlan{first: plan, last: plan}
	if !p.compileMapLitUpdate(b, first) {
		t.Fatal("non-last planned update was not skipped")
	}
	p.staticMapSliceValues = map[*ssa.MapUpdate]llssa.Expr{last: p.compileMapLitConst(b, c)}
	if !p.compileMapLitUpdate(b, last) {
		t.Fatal("static map slice value was not compiled")
	}

	dynFn := buildTestSSA(t, `package foo
func F(n int) map[int]int { m := make(map[int]int); m[1] = n; return m }`)
	dyn := firstMapLitUpdate(t, dynFn)
	p.bvals[dyn.Map] = m
	p.mapLitLoops = map[*ssa.MapUpdate]*mapLitPlan{dyn: {kind: mapLitGeneric, updates: []*ssa.MapUpdate{dyn}}}
	if !p.compileMapLitUpdate(b, dyn) {
		t.Fatal("non-constant planned value was not compiled")
	}

	itemFn := buildTestSSA(t, `package foo
type item struct { X int }
func F() map[int]item { var s item; s.X = 1; return map[int]item{0: s} }
`)
	itemUpdate := firstMapLitUpdate(t, itemFn)
	_, alloc, _, _, ok := mapValueComplit(itemUpdate.Value)
	if !ok {
		t.Fatal("safe composite was rejected")
	}
	structTy := alloc.Type().(*types.Pointer).Elem()
	p.inCFunc = true
	if _, ok := p.compileMapLitComplit(b, itemUpdate.Value, structTy); !ok {
		t.Fatal("C calling-convention composite was rejected")
	}
	p.inCFunc = false
	if _, ok := p.compileMapLitComplit(b, itemUpdate.Value, types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "X", types.Typ[types.Int]),
		types.NewVar(0, nil, "Y", types.Typ[types.Int]),
	}, nil)); ok {
		t.Fatal("composite missing a field was compiled")
	}

	callFn := buildTestSSA(t, `package foo
type item struct { V any }
func id(x int) int { return x }
func F() map[int]item { return map[int]item{0: {V: id(1)}} }
`)
	if _, ok := p.compileMapLitComplit(b, firstMapLitUpdate(t, callFn).Value, types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "V", types.NewInterfaceType(nil, nil)),
	}, nil)); ok {
		t.Fatal("non-constant constructor field was compiled")
	}

	boxFn := buildTestSSA(t, `package foo
type item struct { V any }
func box(x int) any { return x }
func F() map[int]item { return map[int]item{0: {V: box(1)}} }
`)
	boxUpdate := firstMapLitUpdate(t, boxFn)
	_, boxAlloc, fields, _, ok := mapValueComplit(boxUpdate.Value)
	if !ok {
		t.Fatal("boxed composite was rejected")
	}
	call, ok := constantMakeCall(fields[0])
	if !ok {
		t.Fatal("missing constant boxing call")
	}
	origArg := call.Call.Args[0]
	call.Call.Args[0] = boxFn.Pkg.Func("box").Params[0]
	if _, ok := p.compileMapLitComplit(b, boxUpdate.Value, boxAlloc.Type().(*types.Pointer).Elem()); ok {
		t.Fatal("boxing call that cannot be folded was compiled")
	}
	call.Call.Args[0] = origArg

	p.bvals[itemUpdate.Map] = m
	p.mapLitLoops = map[*ssa.MapUpdate]*mapLitPlan{itemUpdate: {
		kind:     mapLitStaticComplit,
		updates:  []*ssa.MapUpdate{itemUpdate},
		structTy: types.Typ[types.Int],
	}}
	itemUpdate.Value = c
	defer func() {
		if recover() == nil {
			t.Fatal("invalid static composite plan did not panic")
		}
	}()
	p.compileMapLitUpdate(b, itemUpdate)
}

func TestMapLitConvertedValueReplaysWrappers(t *testing.T) {
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg := prog.NewPackage("wrapper", "wrapper")
	fn := pkg.NewFunc("test", types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.GoDecl)
	b := fn.MakeBody(1)
	p := &context{prog: prog, pkg: pkg}

	ssaFn := buildTestSSA(t, `package foo
type myInt int
type A interface{ M() }
type B interface{ M(); N() }
func box(x int) any { return x }
func F(x int, b B) (any, myInt, A) {
	var boxed any = 1
	return boxed, myInt(x), b
}
`)
	var mi *ssa.MakeInterface
	var ct *ssa.ChangeType
	var ci *ssa.ChangeInterface
	for _, block := range ssaFn.Blocks {
		for _, instr := range block.Instrs {
			switch x := instr.(type) {
			case *ssa.MakeInterface:
				if mi == nil {
					mi = x
				}
			case *ssa.ChangeType:
				ct = x
			case *ssa.ChangeInterface:
				ci = x
			}
		}
	}
	boxFn := ssaFn.Pkg.Func("box")
	if boxFn == nil || mi == nil || ct == nil || ci == nil {
		t.Fatalf("missing wrapper SSA: box=%v mi=%v ct=%v ci=%v", boxFn, mi, ct, ci)
	}
	fake := &ssa.Call{}
	fake.Call.Value = boxFn
	fake.Call.Args = []ssa.Value{ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])}
	folded, ok := p.foldConstantMakeValue(b, boxFn, &fake.Call)
	if !ok {
		t.Fatal("constant boxing was not folded")
	}
	for _, wrapper := range []ssa.Value{mi, ct, ci} {
		switch w := wrapper.(type) {
		case *ssa.MakeInterface:
			orig := w.X
			w.X = fake
			if p.mapLitConvertedValue(b, w, fake, folded).Type == nil {
				t.Fatal("MakeInterface wrapper was not replayed")
			}
			w.X = orig
		case *ssa.ChangeType:
			orig := w.X
			w.X = fake
			if p.mapLitConvertedValue(b, w, fake, folded).Type == nil {
				t.Fatal("ChangeType wrapper was not replayed")
			}
			w.X = orig
		case *ssa.ChangeInterface:
			orig := w.X
			w.X = fake
			if p.mapLitConvertedValue(b, w, fake, folded).Type == nil {
				t.Fatal("ChangeInterface wrapper was not replayed")
			}
			w.X = orig
		}
	}
	b.Return()
}
