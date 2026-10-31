package cl

import (
	"go/constant"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func newMapInitContext(t *testing.T) *context {
	t.Helper()
	prog := ssatest.NewProgram(t, nil)
	return &context{prog: prog, pkg: prog.NewPackage("foo", "foo")}
}

func foldableMapPackage(t *testing.T) *ssa.Package {
	t.Helper()
	return buildSSAPackage(t, `package foo
var M = map[string][]int{
	"a": {1, 2},
	"b": {3},
}
func Use() int { return M["a"][0] + M["b"][0] }
`)
}

func emptyMapSlicePackage(t *testing.T) *ssa.Package {
	t.Helper()
	return buildSSAPackage(t, `package foo
var M = map[string][]int{"e": {}}
func Use() int { return len(M["e"]) }
`)
}

func findGlobalMakeMapStore(t *testing.T, pkg *ssa.Package, name string) (*ssa.Global, *ssa.Store, *ssa.MakeMap) {
	t.Helper()
	global, ok := pkg.Members[name].(*ssa.Global)
	if !ok {
		t.Fatalf("missing global %s: %T", name, pkg.Members[name])
	}
	initFn := pkg.Func("init")
	if initFn == nil {
		t.Fatal("missing package initializer")
	}
	for _, block := range initFn.Blocks {
		for _, instr := range block.Instrs {
			store, ok := instr.(*ssa.Store)
			if !ok || staticInitRootGlobal(store.Addr) != global {
				continue
			}
			makeMap, ok := store.Val.(*ssa.MakeMap)
			if ok {
				return global, store, makeMap
			}
		}
	}
	t.Fatalf("missing MakeMap store for %s", name)
	return nil, nil, nil
}

func appendInitInstr(pkg *ssa.Package, instr ssa.Instruction) {
	initFn := pkg.Func("init")
	initFn.Blocks[0].Instrs = append(initFn.Blocks[0].Instrs, instr)
}

func TestInstructionConsumes(t *testing.T) {
	v := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	other := ssa.NewConst(constant.MakeInt64(2), types.Typ[types.Int])
	if !instructionConsumes(&ssa.Store{Val: v}, v) {
		t.Fatal("store of value should consume it")
	}
	if instructionConsumes(&ssa.Store{Val: other}, v) {
		t.Fatal("store of a different value should not consume it")
	}
	if instructionConsumes((*ssa.Store)(nil), v) {
		t.Fatal("nil store should not consume")
	}
	if !instructionConsumes(&ssa.MapUpdate{Value: v}, v) {
		t.Fatal("map update of value should consume it")
	}
	if instructionConsumes(&ssa.MapUpdate{Value: other}, v) {
		t.Fatal("map update of a different value should not consume it")
	}
	if instructionConsumes((*ssa.MapUpdate)(nil), v) {
		t.Fatal("nil map update should not consume")
	}
	if instructionConsumes(new(ssa.Call), v) {
		t.Fatal("unrelated instruction should not consume")
	}
}

func TestStaticInitBoundedArraySizeAllowed(t *testing.T) {
	if !staticInitBoundedArraySizeAllowed(types.NewArray(types.Typ[types.Int], 0)) {
		t.Fatal("empty array should be allowed")
	}
	if !staticInitBoundedArraySizeAllowed(types.NewArray(types.Typ[types.Uint8], maxStaticInitArrayElements)) {
		t.Fatal("byte array at the slice cap should be allowed")
	}
	if staticInitBoundedArraySizeAllowed(types.NewArray(types.Typ[types.Uint8], maxStaticInitArrayElements+1)) {
		t.Fatal("byte array above the slice cap should be rejected")
	}
	if staticInitBoundedArraySizeAllowed(types.NewArray(types.Typ[types.Int], maxStaticInitArrayElements+1)) {
		t.Fatal("non-byte array above the cap should be rejected")
	}
}

func TestStaticSliceInitFromValueRejectsInvalidConsumer(t *testing.T) {
	if _, ok := staticSliceInitFromValue(nil, nil); ok {
		t.Fatal("nil consumer should be rejected")
	}
	if _, ok := staticSliceInitFromValue(&ssa.Slice{X: new(ssa.Alloc)}, new(ssa.MapUpdate)); ok {
		t.Fatal("consumer without a basic block should be rejected")
	}

	pkg := buildSSAPackage(t, `package foo
var M = map[string][]int{"a": {1}}
func local() []int { return []int{9} }
func Use() int { return M["a"][0] + local()[0] }
`)
	_, _, makeMap := findGlobalMakeMapStore(t, pkg, "M")
	updates := mapUpdatesInBlockOrder(pkg.Func("init"))[makeMap]
	if len(updates) == 0 {
		t.Fatal("expected map updates")
	}
	var foreign *ssa.Slice
	for _, block := range pkg.Func("local").Blocks {
		for _, instr := range block.Instrs {
			if slice, ok := instr.(*ssa.Slice); ok {
				foreign = slice
			}
		}
	}
	if foreign == nil {
		t.Fatal("missing slice in local()")
	}
	if _, ok := staticSliceInitFromValue(foreign, updates[0]); ok {
		t.Fatal("slice alloc from another function should be rejected")
	}
}

func TestCollectAllocStoresMapUpdateConsumer(t *testing.T) {
	setRefs := func(value ssa.Value, refs ...ssa.Instruction) {
		t.Helper()
		referrers := value.Referrers()
		if referrers == nil {
			t.Fatalf("%T does not track referrers", value)
		}
		*referrers = refs
	}

	alloc := new(ssa.Alloc)
	slice := &ssa.Slice{X: alloc}
	update := &ssa.MapUpdate{Value: slice}
	leaf := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	store := &ssa.Store{Addr: alloc, Val: leaf}
	setRefs(alloc, store, slice)
	setRefs(slice, update)

	var stores []staticInitStore
	var instrs []ssa.Instruction
	if !collectAllocStores(alloc, slice, update, nil, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
		t.Fatal("rejected an alloc consumed only by MapUpdate")
	}
	if len(stores) != 1 || stores[0].store != store || stores[0].value != leaf {
		t.Fatalf("unexpected collected stores: %+v", stores)
	}
}

func TestCollectStaticMapInitsEarlyExits(t *testing.T) {
	noInit := buildSSAPackage(t, `package foo
const A = 1
`)
	ctx := newMapInitContext(t)
	ctx.collectStaticMapInits(noInit)
	if ctx.staticMapSliceValues != nil {
		t.Fatalf("const-only package produced map folds: %+v", ctx.staticMapSliceValues)
	}

	notSynthetic := foldableMapPackage(t)
	initFn := notSynthetic.Func("init")
	if initFn == nil {
		t.Fatal("expected package initializer")
	}
	initFn.Synthetic = ""
	ctx = newMapInitContext(t)
	ctx.collectStaticMapInits(notSynthetic)
	if ctx.staticMapSliceValues != nil {
		t.Fatalf("non-synthetic init produced map folds: %+v", ctx.staticMapSliceValues)
	}

	ctx = newMapInitContext(t)
	ctx.collectStaticMapInits(buildSSAPackage(t, `package foo
var G = 1
func Use() int { return G }
`))
	if ctx.staticMapSliceValues != nil {
		t.Fatalf("package without map[K][]T produced map folds: %+v", ctx.staticMapSliceValues)
	}
}

func TestStaticMapSliceGlobalsFilters(t *testing.T) {
	pkg := buildSSAPackage(t, `package foo
var Keep = map[string][]int{"a": {1}}
var Skip = map[string][]int{"a": {1}}
var __cgo_cb = map[string][]int{"a": {1}}
var Rewritten = map[string][]int{"a": {1}}
var Linked = map[string][]int{"a": {1}}
var NotMap = 1
var NotSlice = map[string]int{"a": 1}
func Use() int { return Keep["a"][0] + Skip["a"][0] + NotMap + NotSlice["a"] + Rewritten["a"][0] + Linked["a"][0] + __cgo_cb["a"][0] }
`)
	prog := ssatest.NewProgram(t, nil)
	prog.SetLinkname("foo.Linked", "C.Linked")
	prog.SetLocalityInfo(llssa.FullName(pkg.Pkg, "Keep"), llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	ctx := &context{
		prog:     prog,
		pkg:      prog.NewPackage("foo", "foo"),
		skips:    map[string]none{"Skip": {}},
		rewrites: map[string]string{"Rewritten": "x"},
	}
	eligible := ctx.staticMapSliceGlobals(pkg)
	if len(eligible) != 0 {
		names := make([]string, 0, len(eligible))
		for g := range eligible {
			names = append(names, g.Name())
		}
		t.Fatalf("eligible globals = %v, want none after locality/skip/rewrite filters", names)
	}

	withoutLocality := &context{
		prog:     prog,
		pkg:      ctx.pkg,
		skips:    ctx.skips,
		rewrites: ctx.rewrites,
	}
	prog.SetLocalityInfo(llssa.FullName(pkg.Pkg, "Keep"), llssa.LocalityInfo{})
	keepOnly := withoutLocality.staticMapSliceGlobals(pkg)
	if _, ok := keepOnly[pkg.Members["Keep"].(*ssa.Global)]; !ok || len(keepOnly) != 1 {
		names := make([]string, 0, len(keepOnly))
		for g := range keepOnly {
			names = append(names, g.Name())
		}
		t.Fatalf("eligible globals without locality = %v, want [Keep]", names)
	}
}

func TestFindSoleMakeMapStoresRejectsAmbiguousStores(t *testing.T) {
	pkg := foldableMapPackage(t)
	ctx := newMapInitContext(t)
	eligible := ctx.staticMapSliceGlobals(pkg)
	if len(eligible) != 1 {
		t.Fatalf("eligible = %d, want 1", len(eligible))
	}
	global, store, makeMap := findGlobalMakeMapStore(t, pkg, "M")
	initFn := pkg.Func("init")

	t.Run("nonempty path", func(t *testing.T) {
		orig := store.Addr
		store.Addr = &ssa.IndexAddr{
			X:     global,
			Index: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
		}
		defer func() { store.Addr = orig }()
		if _, ok := findSoleMakeMapStores(initFn, eligible)[global]; ok {
			t.Fatal("store through a non-root path should be rejected")
		}
	})

	t.Run("non MakeMap value", func(t *testing.T) {
		orig := store.Val
		store.Val = ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
		defer func() { store.Val = orig }()
		if _, ok := findSoleMakeMapStores(initFn, eligible)[global]; ok {
			t.Fatal("non-MakeMap store should be rejected")
		}
	})

	t.Run("duplicate and already ambiguous", func(t *testing.T) {
		dup := &ssa.Store{Addr: store.Addr, Val: makeMap}
		third := &ssa.Store{Addr: store.Addr, Val: makeMap}
		orig := append([]ssa.Instruction(nil), initFn.Blocks[0].Instrs...)
		appendInitInstr(pkg, dup)
		appendInitInstr(pkg, third)
		defer func() { initFn.Blocks[0].Instrs = orig }()
		if _, ok := findSoleMakeMapStores(initFn, eligible)[global]; ok {
			t.Fatal("duplicate MakeMap stores should be rejected")
		}
	})
}

func TestMapUpdatesInBlockOrderIgnoresNonMakeMap(t *testing.T) {
	pkg := foldableMapPackage(t)
	_, _, makeMap := findGlobalMakeMapStore(t, pkg, "M")
	initFn := pkg.Func("init")
	appendInitInstr(pkg, &ssa.MapUpdate{
		Map:   new(ssa.Global),
		Key:   ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
		Value: ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
	})
	updates := mapUpdatesInBlockOrder(initFn)
	if len(updates) != 1 {
		t.Fatalf("updates for %d maps, want 1", len(updates))
	}
	if got := len(updates[makeMap]); got != 2 {
		t.Fatalf("MakeMap updates = %d, want 2", got)
	}
}

func TestTryStaticMapInitRejectsEscapingMakeMap(t *testing.T) {
	mustNotFold := func(t *testing.T, mutate func(*ssa.Package, *ssa.Global, *ssa.Store, *ssa.MakeMap, []*ssa.MapUpdate)) {
		t.Helper()
		pkg := foldableMapPackage(t)
		global, store, makeMap := findGlobalMakeMapStore(t, pkg, "M")
		ordered := mapUpdatesInBlockOrder(pkg.Func("init"))[makeMap]
		mutate(pkg, global, store, makeMap, ordered)
		ctx := newMapInitContext(t)
		ctx.tryStaticMapInit(global, ctx.staticMapSliceGlobals(pkg)[global], store, makeMap, ordered)
		if len(ctx.staticMapSliceValues) != 0 {
			t.Fatalf("escaping MakeMap was folded: %+v", ctx.staticMapSliceValues)
		}
	}

	t.Run("extra store referrer", func(t *testing.T) {
		mustNotFold(t, func(_ *ssa.Package, _ *ssa.Global, _ *ssa.Store, makeMap *ssa.MakeMap, _ []*ssa.MapUpdate) {
			refs := makeMap.Referrers()
			*refs = append(*refs, &ssa.Store{Val: makeMap})
		})
	})
	t.Run("map update of a different map", func(t *testing.T) {
		mustNotFold(t, func(_ *ssa.Package, _ *ssa.Global, _ *ssa.Store, makeMap *ssa.MakeMap, _ []*ssa.MapUpdate) {
			refs := makeMap.Referrers()
			*refs = append(*refs, &ssa.MapUpdate{Map: new(ssa.MakeMap)})
		})
	})
	t.Run("unsupported referrer", func(t *testing.T) {
		mustNotFold(t, func(_ *ssa.Package, _ *ssa.Global, _ *ssa.Store, makeMap *ssa.MakeMap, _ []*ssa.MapUpdate) {
			refs := makeMap.Referrers()
			*refs = append(*refs, new(ssa.Call))
		})
	})
	t.Run("referrer update missing from blocks", func(t *testing.T) {
		mustNotFold(t, func(_ *ssa.Package, _ *ssa.Global, _ *ssa.Store, makeMap *ssa.MakeMap, _ []*ssa.MapUpdate) {
			refs := makeMap.Referrers()
			*refs = append(*refs, &ssa.MapUpdate{Map: makeMap})
		})
	})
	t.Run("block update missing from referrers", func(t *testing.T) {
		mustNotFold(t, func(_ *ssa.Package, _ *ssa.Global, _ *ssa.Store, makeMap *ssa.MakeMap, ordered []*ssa.MapUpdate) {
			extra := &ssa.MapUpdate{
				Map:   makeMap,
				Key:   ordered[0].Key,
				Value: ordered[0].Value,
			}
			refs := makeMap.Referrers()
			orig := append([]ssa.Instruction(nil), *refs...)
			next := orig[:0]
			removed := false
			for _, ref := range orig {
				if !removed && ref == ordered[1] {
					removed = true
					continue
				}
				next = append(next, ref)
			}
			*refs = append(next, extra)
		})
	})
	t.Run("type mismatch", func(t *testing.T) {
		pkg := foldableMapPackage(t)
		global, store, makeMap := findGlobalMakeMapStore(t, pkg, "M")
		ordered := mapUpdatesInBlockOrder(pkg.Func("init"))[makeMap]
		ctx := newMapInitContext(t)
		wrong := types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.String]))
		ctx.tryStaticMapInit(global, wrong, store, makeMap, ordered)
		if len(ctx.staticMapSliceValues) != 0 {
			t.Fatalf("mismatched map element type was folded: %+v", ctx.staticMapSliceValues)
		}
	})
}

func TestStaticSliceInitFromValueEmptySliceReferrers(t *testing.T) {
	pkg := emptyMapSlicePackage(t)
	_, _, makeMap := findGlobalMakeMapStore(t, pkg, "M")
	updates := mapUpdatesInBlockOrder(pkg.Func("init"))[makeMap]
	if len(updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(updates))
	}
	slice, ok := updates[0].Value.(*ssa.Slice)
	if !ok {
		t.Fatalf("empty map value = %T, want *ssa.Slice", updates[0].Value)
	}
	alloc, ok := slice.X.(*ssa.Alloc)
	if !ok {
		t.Fatalf("empty slice source = %T, want *ssa.Alloc", slice.X)
	}
	if _, ok := staticSliceInitFromValue(slice, updates[0]); !ok {
		t.Fatal("empty constant slice should be statically constructible")
	}

	t.Run("extra alloc referrer", func(t *testing.T) {
		refs := alloc.Referrers()
		orig := append([]ssa.Instruction(nil), *refs...)
		*refs = append(*refs, new(ssa.Call))
		defer func() { *refs = orig }()
		if _, ok := staticSliceInitFromValue(slice, updates[0]); ok {
			t.Fatal("empty slice alloc with an extra referrer should be rejected")
		}
	})
	t.Run("extra slice referrer", func(t *testing.T) {
		refs := slice.Referrers()
		orig := append([]ssa.Instruction(nil), *refs...)
		*refs = append(*refs, new(ssa.Call))
		defer func() { *refs = orig }()
		if _, ok := staticSliceInitFromValue(slice, updates[0]); ok {
			t.Fatal("empty slice with an extra referrer should be rejected")
		}
	})
}

func TestCollectStaticMapInitsFoldsConstantEntries(t *testing.T) {
	pkg := foldableMapPackage(t)
	ctx := newMapInitContext(t)
	ctx.collectStaticMapInits(pkg)
	if len(ctx.staticMapSliceValues) != 2 {
		t.Fatalf("folded updates = %d, want 2", len(ctx.staticMapSliceValues))
	}
	if len(ctx.staticInitInstrs) == 0 || len(ctx.staticInitStores) == 0 {
		t.Fatal("folded map should suppress slice construction instructions")
	}

	empty := emptyMapSlicePackage(t)
	ctx = newMapInitContext(t)
	ctx.collectStaticMapInits(empty)
	if len(ctx.staticMapSliceValues) != 1 {
		t.Fatalf("folded empty-slice updates = %d, want 1", len(ctx.staticMapSliceValues))
	}
}

func TestCollectStaticMapInitsRejectsDynamicEntries(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "dynamic key",
			src: `package foo
var k = "x"
var M = map[string][]int{k: {1}}
func Use() int { return M[k][0] }
`,
		},
		{
			name: "dynamic value",
			src: `package foo
var M = map[string][]int{"a": make([]int, 1)}
func Use() int { return M["a"][0] }
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildSSAPackage(t, test.src)
			ctx := newMapInitContext(t)
			ctx.collectStaticMapInits(pkg)
			if ctx.staticMapSliceValues != nil {
				t.Fatalf("dynamic map was folded: %+v", ctx.staticMapSliceValues)
			}
		})
	}
}

func TestCollectStaticMapInitsLinknameAndRewrite(t *testing.T) {
	pkg := buildSSAPackage(t, `package foo
var M = map[string][]int{"a": {1}}
func Use() int { return M["a"][0] }
`)
	prog := ssatest.NewProgram(t, nil)
	prog.SetLinkname(llssa.FullName(pkg.Pkg, "M"), "C.M")
	ctx := &context{prog: prog, pkg: prog.NewPackage("foo", "foo")}
	ctx.collectStaticMapInits(pkg)
	if ctx.staticMapSliceValues != nil {
		t.Fatalf("linknamed map was folded: %+v", ctx.staticMapSliceValues)
	}
}
