package deadcode

import (
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
)

func TestAnalyze(t *testing.T) {
	tests := []deadcodeCase{
		// type I interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func (T) N() {}
		// func main() { use(&T{}) }
		// func use(v any) { v.(I).M() }
		{
			name:  "keeps interface method implementation",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				use := b.sym("pkg.use")
				typ := b.sym("_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.addIfaceEntry(iface, []pkgSig{mSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
				})
				b.addEdge(main, use)
				b.addEdge(main, typ)
				b.addUseIface(main, typ)
				b.addUseIfaceMethod(use, iface, mSig)
			}),
			want: map[string][]int{"_llgo_pkg.T": {0}},
		},
		// type I interface{ M(); N() }
		// type J interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func main() { useJ(T{}); useI(nil) }
		// func useJ(j J) {}
		// func useI(i I) { i.M() }
		{
			name:  "requires concrete type to implement whole interface",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				useJ := b.sym("pkg.useJ")
				useI := b.sym("pkg.useI")
				typ := b.sym("_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				compatibleIface := b.sym("_llgo_iface$J")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.addIfaceEntry(iface, []pkgSig{mSig, nSig})
				b.addIfaceEntry(compatibleIface, []pkgSig{mSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addEdge(main, useJ)
				b.addEdge(main, useI)
				b.addEdge(main, typ)
				b.addUseIface(main, typ)
				b.addUseIfaceMethod(useI, iface, mSig)
			}),
			want: map[string][]int{},
		},
		// type I interface{ M(int) }
		// type T struct{}
		// func (T) M(string) {}
		// func main() { use(T{}) }
		// func use(i I) { i.M(0) }
		{
			name:  "requires method type to match",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				use := b.sym("pkg.use")
				typ := b.sym("_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				ifaceMSig := methodSigWithType(b, "M", "_llgo_func$int")
				typMSig := methodSigWithType(b, "M", "_llgo_func$string")

				b.addIfaceEntry(iface, []pkgSig{ifaceMSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, typMSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addEdge(main, use)
				b.addEdge(main, typ)
				b.addUseIface(main, typ)
				b.addUseIfaceMethod(use, iface, ifaceMSig)
			}),
			want: map[string][]int{},
		},
		// type I interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func main() { useI(); makeIface() }
		// func useI(i I) { i.M() }
		// func makeIface() any { return T{} }
		{
			name:  "matches demand recorded before type enters interface semantics",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				useI := b.sym("pkg.useI")
				makeIface := b.sym("pkg.makeIface")
				typ := b.sym("_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.addIfaceEntry(iface, []pkgSig{mSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addEdge(main, useI)
				b.addEdge(main, makeIface)
				b.addEdge(makeIface, typ)
				b.addUseIfaceMethod(useI, iface, mSig)
				b.addUseIface(makeIface, typ)
			}),
			want: map[string][]int{"_llgo_pkg.T": {0}},
		},
		// type I interface{ M() }
		// type Child struct{}
		// func (Child) M() {}
		// type T struct{ Child }
		// func (T) M() {}
		// func main() { use(T{}) }
		// func use(i I) { i.M() }
		{
			name:  "propagates interface use through type children",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				use := b.sym("pkg.use")
				typ := b.sym("_llgo_pkg.T")
				child := b.sym("_llgo_pkg.Child")
				iface := b.sym("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.addIfaceEntry(iface, []pkgSig{mSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addMethodInfo(child, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*Child).M", "pkg.Child.M"),
				})
				b.b.AddTypeChild(typ, child)
				b.addEdge(main, use)
				b.addEdge(main, typ)
				b.addEdge(typ, child)
				b.addUseIface(main, typ)
				b.addUseIfaceMethod(use, iface, mSig)
			}),
			want: map[string][]int{
				"_llgo_pkg.Child": {0},
				"_llgo_pkg.T":     {0},
			},
		},
		// type I interface{ M() }
		// type T struct{}
		// func (*T) M() {}
		// func main() { use(T{}) }
		// func use(i I) { i.M() }
		{
			name:  "handles cyclic type children",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				typ := b.sym("_llgo_pkg.T")
				ptr := b.sym("*_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.addIfaceEntry(iface, []pkgSig{mSig})
				b.addMethodInfo(ptr, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.(*T).M"),
				})
				b.b.AddTypeChild(typ, ptr)
				b.b.AddTypeChild(ptr, typ)
				b.addEdge(main, ptr)
				b.addEdge(typ, ptr)
				b.addUseIface(main, ptr)
				b.addUseIfaceMethod(main, iface, mSig)
			}),
			want: map[string][]int{"*_llgo_pkg.T": {0}},
		},
		// type Unmarshaler interface{ UnmarshalJSON([]byte) error }
		// type RawMessage []byte
		// func (*RawMessage) UnmarshalJSON([]byte) error { return nil }
		// type Container struct{ Raw RawMessage }
		// func main() { unmarshal(&Container{}) }
		// func unmarshal(v any) { field.Addr().Interface().(Unmarshaler).UnmarshalJSON(nil) }
		{
			name:  "value type entering interface semantics keeps pointer method implementation",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				unmarshal := b.sym("pkg.unmarshal")
				containerPtr := b.sym("*_llgo_pkg.Container")
				container := b.sym("_llgo_pkg.Container")
				raw := b.sym("_llgo_pkg.RawMessage")
				rawPtr := b.sym("*_llgo_pkg.RawMessage")
				iface := b.sym("_llgo_pkg.Unmarshaler")
				unmarshalSig := methodSigWithType(b, "UnmarshalJSON", "_llgo_func$bytes_error")

				b.addIfaceEntry(iface, []pkgSig{unmarshalSig})
				b.addMethodInfo(rawPtr, []pkgSlot{
					methodSlot(b, unmarshalSig, "pkg.(*RawMessage).UnmarshalJSON", "pkg.(*RawMessage).UnmarshalJSON"),
				})
				b.b.AddTypeChild(containerPtr, container)
				b.b.AddTypeChild(container, raw)
				b.addEdge(main, unmarshal)
				b.addEdge(main, containerPtr)
				b.addEdge(containerPtr, container)
				b.addEdge(container, raw)
				b.addEdge(raw, rawPtr)
				b.b.AddTypeChild(raw, rawPtr)
				b.addUseIface(main, containerPtr)
				b.addUseIfaceMethod(unmarshal, iface, unmarshalSig)
			}),
			want: map[string][]int{"*_llgo_pkg.RawMessage": {0}},
		},
		// type T struct{}
		// func (T) M() {}
		// func (T) N() {}
		// func main() { use(T{}) }
		// func use(v any) { reflect.ValueOf(v).MethodByName("M") }
		{
			name:  "constant MethodByName keeps same-name method",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				use := b.sym("pkg.use")
				typ := b.sym("_llgo_pkg.T")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
				})
				b.addEdge(main, use)
				b.addEdge(main, typ)
				b.addUseIface(main, typ)
				b.b.AddNamedMethodUse(use, mSig.name)
			}),
			want: map[string][]int{"_llgo_pkg.T": {0}},
		},
		// type T struct{}
		// func (T) M() {}
		// func (T) N() {}
		// func (T) m() {}
		// func main() { use(T{}) }
		// func use(v any) { reflect.ValueOf(v).Method(0) }
		{
			name:  "reflection keeps exported methods only",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				use := b.sym("pkg.use")
				typ := b.sym("_llgo_pkg.T")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")
				unexportedSig := methodSig(b, "m")

				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
					methodSlot(b, unexportedSig, "pkg.(*T).m", "pkg.T.m"),
				})
				b.addEdge(main, use)
				b.addEdge(main, typ)
				b.addUseIface(main, typ)
				b.b.MarkReflect(use)
			}),
			want: map[string][]int{"_llgo_pkg.T": {0, 1}},
		},
		// type I interface{ M() }
		// type J interface{ N() }
		// type T struct{}
		// type U struct{}
		// func (T) M() { callU() }
		// func (U) N() {}
		// func main() { useT(T{}) }
		// func useT(i I) { i.M() }
		// func callU() { useU(U{}) }
		// func useU(j J) { j.N() }
		{
			name:  "live method body can introduce new interface demands",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				useT := b.sym("pkg.useT")
				callU := b.sym("pkg.callU")
				useU := b.sym("pkg.useU")
				typT := b.sym("_llgo_pkg.T")
				typU := b.sym("_llgo_pkg.U")
				ifaceI := b.sym("_llgo_iface$I")
				ifaceJ := b.sym("_llgo_iface$J")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.addIfaceEntry(ifaceI, []pkgSig{mSig})
				b.addIfaceEntry(ifaceJ, []pkgSig{nSig})
				b.addMethodInfo(typT, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addMethodInfo(typU, []pkgSlot{
					methodSlot(b, nSig, "pkg.(*U).N", "pkg.U.N"),
				})
				b.addEdge(main, useT)
				b.addEdge(main, typT)
				b.addUseIface(main, typT)
				b.addUseIfaceMethod(useT, ifaceI, mSig)
				b.addEdge(b.sym("pkg.T.M"), callU)
				b.addEdge(callU, useU)
				b.addEdge(callU, typU)
				b.addUseIface(callU, typU)
				b.addUseIfaceMethod(useU, ifaceJ, nSig)
			}),
			want: map[string][]int{
				"_llgo_pkg.T": {0},
				"_llgo_pkg.U": {0},
			},
		},
		// type Type interface{ Elem() Type }
		// type rtype struct{}
		// func (rtype) Elem() Type { return toType() }
		// func main() { init(); toType() }
		// func init() { reflectType.Elem() }
		// func toType() Type { return rtype{} }
		{
			name:  "interface demand and conversion from different reachable functions meet",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				init := b.sym("pkg.init")
				toType := b.sym("pkg.toType")
				typ := b.sym("_llgo_pkg.rtype")
				iface := b.sym("_llgo_pkg.Type")
				elemSig := methodSig(b, "Elem")

				b.addIfaceEntry(iface, []pkgSig{elemSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, elemSig, "pkg.(*rtype).Elem", "pkg.rtype.Elem"),
				})
				b.addEdge(main, init)
				b.addEdge(main, toType)
				b.addEdge(toType, typ)
				b.addUseIface(toType, typ)
				b.addUseIfaceMethod(init, iface, elemSig)
			}),
			want: map[string][]int{"_llgo_pkg.rtype": {0}},
		},
		// type Type interface{ Elem() Type; Kind() Kind }
		// type rtype struct{}
		// func (rtype) Elem() Type { return toType() }
		// func (rtype) Kind() Kind { return 0 }
		// func main() { init(); toType() }
		// func init() { reflectType.Elem() }
		// func toType() Type { return rtype{} }
		{
			name:  "duplicate interface method names do not inflate interface size",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				init := b.sym("pkg.init")
				toType := b.sym("pkg.toType")
				typ := b.sym("_llgo_pkg.rtype")
				iface := b.sym("_llgo_pkg.Type")
				elemSig := methodSig(b, "Elem")
				kindSig := methodSigWithType(b, "Kind", "_llgo_func$kind")
				altKindSig := methodSigWithType(b, "Kind", "_llgo_func$altKind")

				b.addIfaceEntry(iface, []pkgSig{elemSig, kindSig, altKindSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, elemSig, "pkg.(*rtype).Elem", "pkg.rtype.Elem"),
					methodSlot(b, kindSig, "pkg.(*rtype).Kind", "pkg.rtype.Kind"),
				})
				b.addEdge(main, init)
				b.addEdge(main, toType)
				b.addEdge(toType, typ)
				b.addUseIface(toType, typ)
				b.addUseIfaceMethod(init, iface, elemSig)
			}),
			want: map[string][]int{"_llgo_pkg.rtype": {0}},
		},
		// type I interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func main() { _ = T{} }
		// func unreachable(i I) { i.M(); reflect.ValueOf(i).MethodByName("M") }
		{
			name:  "ignores unreachable semantic facts",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *pkgBuilder) {
				main := b.sym("pkg.main")
				unreachable := b.sym("pkg.unreachable")
				typ := b.sym("_llgo_pkg.T")
				iface := b.sym("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.addIfaceEntry(iface, []pkgSig{mSig})
				b.addMethodInfo(typ, []pkgSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.addEdge(main, typ)
				b.addUseIface(unreachable, typ)
				b.addUseIfaceMethod(unreachable, iface, mSig)
				b.b.AddNamedMethodUse(unreachable, mSig.name)
				b.b.MarkReflect(unreachable)
			}),
			want: map[string][]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(newSummary(t, tt.summary), tt.roots)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Analyze() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBuildPlanWithFeedbackDropsFactsFromDeadFunction(t *testing.T) {
	summary := newSummary(t, buildPackage(func(b *pkgBuilder) {
		main := b.sym("pkg.main")
		semanticDemand := b.sym("pkg.semanticDemand")
		typ := b.sym("_llgo_pkg.T")
		iface := b.sym("_llgo_iface$I")
		mSig := methodSig(b, "M")

		b.addIfaceEntry(iface, []pkgSig{mSig})
		b.addMethodInfo(typ, []pkgSlot{
			methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
		})
		// The package Meta graph conservatively contains this edge. A later
		// ThinLTO round can remove it after cross-package constant propagation.
		b.addEdge(main, semanticDemand)
		b.addEdge(semanticDemand, typ)
		b.addUseIface(semanticDemand, typ)
		b.addUseIfaceMethod(semanticDemand, iface, mSig)
	}))

	first := BuildPlan(summary, []string{"pkg.main"})
	wantFirst := map[string][]int{"_llgo_pkg.T": {0}}
	if !reflect.DeepEqual(first.LiveSlots, wantFirst) {
		t.Fatalf("first plan LiveSlots = %#v, want %#v", first.LiveSlots, wantFirst)
	}

	second := BuildPlanWithFeedback(summary, []string{"pkg.main"}, Feedback{
		DeadFunctions: map[string]struct{}{"pkg.semanticDemand": {}},
	})
	if len(second.LiveSlots) != 0 {
		t.Fatalf("feedback plan LiveSlots = %#v, want empty", second.LiveSlots)
	}

	// A root is never suppressed by feedback, even if a stale producer lists it.
	rooted := BuildPlanWithFeedback(summary, []string{"pkg.semanticDemand"}, Feedback{
		DeadFunctions: map[string]struct{}{"pkg.semanticDemand": {}},
	})
	if !reflect.DeepEqual(rooted.LiveSlots, wantFirst) {
		t.Fatalf("rooted feedback plan LiveSlots = %#v, want %#v", rooted.LiveSlots, wantFirst)
	}
}

func TestBuildPlanWithFeedbackRefinesDynamicMethodNames(t *testing.T) {
	summary := newSummary(t, buildPackage(func(b *pkgBuilder) {
		main := b.sym("pkg.main")
		dynamic := b.sym("pkg.dynamicMethodByName")
		typ := b.sym("_llgo_pkg.T")
		keepA := methodSig(b, "KeepA")
		keepB := methodSig(b, "KeepB")
		drop := methodSig(b, "Drop")

		b.addMethodInfo(typ, []pkgSlot{
			methodSlot(b, keepA, "pkg.(*T).KeepA", "pkg.T.KeepA"),
			methodSlot(b, keepB, "pkg.(*T).KeepB", "pkg.T.KeepB"),
			methodSlot(b, drop, "pkg.(*T).Drop", "pkg.T.Drop"),
		})
		b.addEdge(main, dynamic)
		b.addEdge(dynamic, typ)
		b.addUseIface(dynamic, typ)
		b.b.MarkReflect(dynamic)
	}))

	first := BuildPlan(summary, []string{"pkg.main"})
	wantFirst := map[string][]int{"_llgo_pkg.T": {0, 1, 2}}
	if !reflect.DeepEqual(first.LiveSlots, wantFirst) {
		t.Fatalf("initial plan LiveSlots = %#v, want %#v", first.LiveSlots, wantFirst)
	}

	refined := BuildPlanWithFeedback(summary, []string{"pkg.main"}, Feedback{
		RefinedMethodNames: map[string][]string{
			"pkg.dynamicMethodByName": {"KeepA", "KeepB"},
		},
	})
	wantRefined := map[string][]int{"_llgo_pkg.T": {0, 1}}
	if !reflect.DeepEqual(refined.LiveSlots, wantRefined) {
		t.Fatalf("refined plan LiveSlots = %#v, want %#v", refined.LiveSlots, wantRefined)
	}

	// A present but empty result means LLVM proved that every dynamic call in
	// this owner disappeared. An absent result remains conservative.
	empty := BuildPlanWithFeedback(summary, []string{"pkg.main"}, Feedback{
		RefinedMethodNames: map[string][]string{"pkg.dynamicMethodByName": {}},
	})
	if len(empty.LiveSlots) != 0 {
		t.Fatalf("empty refinement LiveSlots = %#v, want empty", empty.LiveSlots)
	}
}

func TestBuildPlanWithFeedbackDoesNotRefineMixedReflectOwner(t *testing.T) {
	summary := newSummary(t, buildPackage(func(b *pkgBuilder) {
		main := b.sym("pkg.main")
		dynamic := b.sym("pkg.mixedReflection")
		typ := b.sym("_llgo_pkg.T")
		keep := methodSig(b, "Keep")
		drop := methodSig(b, "Drop")

		b.addMethodInfo(typ, []pkgSlot{
			methodSlot(b, keep, "pkg.(*T).Keep", "pkg.T.Keep"),
			methodSlot(b, drop, "pkg.(*T).Drop", "pkg.T.Drop"),
		})
		b.addEdge(main, dynamic)
		b.addEdge(dynamic, typ)
		b.addUseIface(dynamic, typ)
		// One demand may come from MethodByName while the other may come from
		// Method(index). Until metadata carries per-call IDs, owner-level name
		// feedback must not replace this mixed pair.
		b.b.MarkReflect(dynamic)
		b.b.MarkReflect(dynamic)
	}))

	got := BuildPlanWithFeedback(summary, []string{"pkg.main"}, Feedback{
		RefinedMethodNames: map[string][]string{"pkg.mixedReflection": {"Keep"}},
	})
	want := map[string][]int{"_llgo_pkg.T": {0, 1}}
	if !reflect.DeepEqual(got.LiveSlots, want) {
		t.Fatalf("mixed-reflect feedback LiveSlots = %#v, want %#v", got.LiveSlots, want)
	}
}

// ── test builder helpers ──────────────────────────────────────────────────────

type pkgSig struct {
	name  string
	mtype meta.Symbol
}

type pkgSlot struct {
	sig pkgSig
	ifn meta.Symbol
	tfn meta.Symbol
}

// pkgBuilder wraps meta.Builder and tracks iface method order so that
// addUseIfaceMethod can look up the sig index required by AddIfaceMethodUse.
type pkgBuilder struct {
	b          *meta.Builder
	ifaceOrder map[meta.Symbol][]pkgSig
}

func newPkgBuilder() *pkgBuilder {
	return &pkgBuilder{
		b:          meta.NewBuilder(),
		ifaceOrder: make(map[meta.Symbol][]pkgSig),
	}
}

func (p *pkgBuilder) sym(name string) meta.Symbol { return p.b.Sym(name) }

func (p *pkgBuilder) addEdge(src, dst meta.Symbol) {
	p.b.AddOrdinaryEdge(src, dst)
}

func (p *pkgBuilder) addUseIface(fn, typ meta.Symbol) {
	p.b.AddIfaceUse(fn, typ)
}

// addUseIfaceMethod records a demand for iface.sig from fn. The sig index is
// determined by the order in which sigs were registered via addIfaceEntry.
func (p *pkgBuilder) addUseIfaceMethod(fn, iface meta.Symbol, sig pkgSig) {
	sigs := p.ifaceOrder[iface]
	for i, s := range sigs {
		if s.name == sig.name && s.mtype == sig.mtype {
			p.b.AddIfaceMethodUse(fn, iface, uint32(i))
			return
		}
	}
	panic("addUseIfaceMethod: sig not found in iface — call addIfaceEntry first")
}

func (p *pkgBuilder) addIfaceEntry(iface meta.Symbol, sigs []pkgSig) {
	p.ifaceOrder[iface] = sigs
	for _, sig := range sigs {
		p.b.AddIfaceMethod(iface, sig.name, sig.mtype)
	}
}

func (p *pkgBuilder) addMethodInfo(typ meta.Symbol, slots []pkgSlot) {
	for _, slot := range slots {
		p.b.AddMethodSlot(typ, slot.sig.name, slot.sig.mtype, slot.ifn, slot.tfn)
	}
}

func methodSig(b *pkgBuilder, name string) pkgSig {
	return methodSigWithType(b, name, "_llgo_func$X")
}

func methodSigWithType(b *pkgBuilder, name, mtype string) pkgSig {
	typ := b.sym(mtype)
	b.addEdge(typ, typ)
	return pkgSig{name: name, mtype: typ}
}

func methodSlot(b *pkgBuilder, sig pkgSig, ifn, tfn string) pkgSlot {
	return pkgSlot{sig: sig, ifn: b.sym(ifn), tfn: b.sym(tfn)}
}

type deadcodeCase struct {
	name    string
	summary *meta.PackageMeta
	roots   []string
	want    map[string][]int
}

func buildPackage(fn func(*pkgBuilder)) *meta.PackageMeta {
	b := newPkgBuilder()
	fn(b)
	pm, err := b.b.Build()
	if err != nil {
		panic("buildPackage: " + err.Error())
	}
	return pm
}

func newSummary(t *testing.T, pkgs ...*meta.PackageMeta) *meta.GlobalSummary {
	t.Helper()
	summary, err := meta.NewGlobalSummary(pkgs)
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}
	return summary
}
