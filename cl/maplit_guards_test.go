package cl

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestMapLitStoredValueChains(t *testing.T) {
	for _, kind := range []string{"interface", "change type", "change interface"} {
		for _, shared := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: " single", true: " shared"}[shared], func(t *testing.T) {
				c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
				var value ssa.Value
				switch kind {
				case "interface":
					value = &ssa.MakeInterface{X: c}
				case "change type":
					value = &ssa.ChangeType{X: c}
				case "change interface":
					value = &ssa.ChangeInterface{X: c}
				}
				store := &ssa.Store{Val: value}
				refs := []ssa.Instruction{store}
				if shared {
					refs = append(refs, &ssa.Return{Results: []ssa.Value{value}})
				}
				*value.Referrers() = refs
				extra, ok := skipStoredValue(store)
				if ok == shared {
					t.Fatalf("accepted=%v for shared=%v", ok, shared)
				}
				if ok && (len(extra) != 1 || extra[0] != value.(ssa.Instruction)) {
					t.Fatalf("unexpected chain: %v", extra)
				}
			})
		}
	}
}

func TestMapLitCompositeRejectsAliasedStorage(t *testing.T) {
	for _, kind := range []string{"not alloc", "other load", "escaped alloc", "no fields", "shared field", "read field", "shared box"} {
		t.Run(kind, func(t *testing.T) {
			alloc := &ssa.Alloc{}
			load := &ssa.UnOp{Op: token.MUL, X: alloc}
			field := &ssa.FieldAddr{X: alloc, Field: 0}
			c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
			store := &ssa.Store{Addr: field, Val: c}
			*alloc.Referrers() = []ssa.Instruction{field, load}
			*field.Referrers() = []ssa.Instruction{store}
			switch kind {
			case "not alloc":
				load.X = c
			case "other load":
				*alloc.Referrers() = append(*alloc.Referrers(), &ssa.UnOp{Op: token.MUL, X: alloc})
			case "escaped alloc":
				*alloc.Referrers() = append(*alloc.Referrers(), &ssa.Return{Results: []ssa.Value{alloc}})
			case "no fields":
				*alloc.Referrers() = []ssa.Instruction{load}
			case "shared field":
				*field.Referrers() = append(*field.Referrers(), &ssa.UnOp{Op: token.MUL, X: field})
			case "read field":
				*field.Referrers() = []ssa.Instruction{&ssa.UnOp{Op: token.MUL, X: field}}
			case "shared box":
				box := &ssa.MakeInterface{X: c}
				store.Val = box
				*box.Referrers() = []ssa.Instruction{store, &ssa.Return{Results: []ssa.Value{box}}}
			}
			if _, _, _, _, ok := mapValueComplit(load); ok {
				t.Fatal("accepted aliased or incomplete composite storage")
			}
		})
	}
}

func TestMapLitDynamicKeysRemainUnrolled(t *testing.T) {
	if got := collectLargeMapLits(nil); got != nil {
		t.Fatal("nil function produced plans")
	}
	fn := buildTestSSA(t, `package foo
func F(k string) map[string]int {
 m:=make(map[string]int)
 for i:=0;i<30;i++ {m[k]=i}
 return m
}`)
	if got := collectLargeMapLits(fn); got != nil {
		t.Fatal("dynamic-key map was collected")
	}
}

func TestMapLitRejectsDynamicKeyAndEarlyRead(t *testing.T) {
	for _, mode := range []string{"dynamic", "early read"} {
		var src strings.Builder
		src.WriteString("package foo\nfunc F(k string) map[string]int { m:=make(map[string]int)\n")
		for i := 0; i < 30; i++ {
			key := fmt.Sprintf("%q", fmt.Sprintf("k%d", i))
			if mode == "dynamic" {
				key = "k"
			}
			fmt.Fprintf(&src, "m[%s]=%d\n", key, i)
			if mode == "early read" && i == 0 {
				src.WriteString("println(len(m))\n")
			}
		}
		src.WriteString("return m }")
		if got := collectLargeMapLits(buildTestSSA(t, src.String())); got != nil {
			t.Fatalf("accepted %s map", mode)
		}
	}
}

func TestMapLitConstantCallWrappers(t *testing.T) {
	pkg := buildTestSSA(t, `package foo
func box(x int) any { return x }
func F() any { return box(1) }
`)
	var call *ssa.Call
	for _, i := range pkg.Blocks[0].Instrs {
		if c, ok := i.(*ssa.Call); ok {
			call = c
		}
	}
	if call == nil {
		t.Fatal("missing constructor call")
	}
	for _, v := range []ssa.Value{&ssa.MakeInterface{X: call}, &ssa.ChangeType{X: call}, &ssa.ChangeInterface{X: call}} {
		if got, ok := constantMakeCall(v); !ok || got != call {
			t.Fatal("lost constant constructor through wrapper")
		}
	}
	call.Call.Args[0] = ssa.NewConst(nil, types.NewPointer(types.Typ[types.Int]))
	if _, ok := constantMakeCall(call); ok {
		t.Fatal("accepted a nil constant argument that lowering cannot emit")
	}
}

func TestMapLitCompositeClassificationGuards(t *testing.T) {
	c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	for _, kind := range []string{"shared load", "nonstruct", "missing field"} {
		alloc := &ssa.Alloc{}
		load := &ssa.UnOp{Op: token.MUL, X: alloc}
		field := &ssa.FieldAddr{X: alloc, Field: 0}
		store := &ssa.Store{Addr: field, Val: c}
		update := &ssa.MapUpdate{Value: load}
		*alloc.Referrers() = []ssa.Instruction{field, load}
		*field.Referrers() = []ssa.Instruction{store}
		*load.Referrers() = []ssa.Instruction{update}
		typ := types.NewStruct([]*types.Var{types.NewVar(0, nil, "a", types.Typ[types.Int]), types.NewVar(0, nil, "b", types.Typ[types.Int])}, nil)
		plan := &mapLitPlan{structTy: typ}
		if kind == "shared load" {
			*load.Referrers() = append(*load.Referrers(), &ssa.Return{Results: []ssa.Value{load}})
		}
		if kind == "nonstruct" {
			plan.structTy = types.Typ[types.Int]
		}
		if classifyStaticComplit([]*ssa.MapUpdate{update}, plan) {
			t.Fatalf("accepted %s", kind)
		}
	}
}

func TestMapLitSafeToDelayReferences(t *testing.T) {
	fn := buildTestSSA(t, `package foo
func F() map[int]int {m:=make(map[int]int);m[1]=1;return m}`)
	var update *ssa.MapUpdate
	for _, i := range fn.Blocks[0].Instrs {
		if u, ok := i.(*ssa.MapUpdate); ok {
			update = u
		}
	}
	if update == nil {
		t.Fatal("missing update")
	}
	m := update.Map.(*ssa.MakeMap)
	for _, r := range []ssa.Instruction{&ssa.MapUpdate{Map: m}, &ssa.Store{Addr: m}} {
		*m.Referrers() = []ssa.Instruction{update, r}
		if mapLitSafeToDelay(m, []*ssa.MapUpdate{update}) {
			t.Fatal("accepted unplanned map use")
		}
	}
}

func TestMapLitCompileFallbacks(t *testing.T) {
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg := prog.NewPackage("fallback", "fallback")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := pkg.NewFunc("test", sig, llssa.GoDecl)
	b := fn.MakeBody(1)
	p := &context{prog: prog, pkg: pkg, inCFunc: true}
	c := ssa.NewConst(constant.MakeInt64(7), types.Typ[types.Int])
	got := p.compileMapLitConst(b, c)
	if got.Type == nil {
		t.Fatal("C calling-convention constant is missing")
	}
	if _, ok := p.compileMapLitComplit(b, c, types.Typ[types.Int]); ok {
		t.Fatal("accepted non-composite value")
	}
	p.mapLitLoops = map[*ssa.MapUpdate]*mapLitPlan{}
	if p.compileMapLitUpdate(b, &ssa.MapUpdate{}) {
		t.Fatal("accepted unplanned update")
	}
	b.Return()
}
