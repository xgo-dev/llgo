package deadcode

import (
	"reflect"
	"testing"

	"github.com/goplus/llgo/internal/metadata"
)

func TestAnalyze(t *testing.T) {
	tests := []deadcodeCase{
		// type I interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func (T) N() {}
		// func main() { use(T{}) }
		// func use(i I) { i.M() }
		{
			name:  "keeps interface method implementation",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				use := b.Symbol("pkg.use")
				typ := b.Symbol("_llgo_pkg.T")
				iface := b.Symbol("_llgo_iface$I")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.AddIfaceEntry(iface, []metadata.MethodSig{mSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
				})
				b.AddEdge(main, use)
				b.AddEdge(main, typ)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddUseIfaceMethod(use, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    mSig,
				}})
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				useJ := b.Symbol("pkg.useJ")
				useI := b.Symbol("pkg.useI")
				typ := b.Symbol("_llgo_pkg.T")
				iface := b.Symbol("_llgo_iface$I")
				compatibleIface := b.Symbol("_llgo_iface$J")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.AddIfaceEntry(iface, []metadata.MethodSig{mSig, nSig})
				b.AddIfaceEntry(compatibleIface, []metadata.MethodSig{mSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddEdge(main, useJ)
				b.AddEdge(main, useI)
				b.AddEdge(main, typ)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddUseIfaceMethod(useI, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    mSig,
				}})
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				use := b.Symbol("pkg.use")
				typ := b.Symbol("_llgo_pkg.T")
				iface := b.Symbol("_llgo_iface$I")
				ifaceMSig := methodSigWithType(b, "M", "_llgo_func$int")
				typeMSig := methodSigWithType(b, "M", "_llgo_func$string")

				b.AddIfaceEntry(iface, []metadata.MethodSig{ifaceMSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, typeMSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddEdge(main, use)
				b.AddEdge(main, typ)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddUseIfaceMethod(use, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    ifaceMSig,
				}})
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				useI := b.Symbol("pkg.useI")
				makeIface := b.Symbol("pkg.makeIface")
				typ := b.Symbol("_llgo_pkg.T")
				iface := b.Symbol("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.AddIfaceEntry(iface, []metadata.MethodSig{mSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddEdge(main, useI)
				b.AddEdge(main, makeIface)
				b.AddEdge(makeIface, typ)
				b.AddUseIfaceMethod(useI, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    mSig,
				}})
				b.AddUseIface(makeIface, []metadata.Symbol{typ})
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				use := b.Symbol("pkg.use")
				typ := b.Symbol("_llgo_pkg.T")
				child := b.Symbol("_llgo_pkg.Child")
				iface := b.Symbol("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.AddIfaceEntry(iface, []metadata.MethodSig{mSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddMethodInfo(child, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*Child).M", "pkg.Child.M"),
				})
				b.AddTypeChild(typ, child)
				b.AddEdge(main, use)
				b.AddEdge(main, typ)
				b.AddEdge(typ, child)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddUseIfaceMethod(use, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    mSig,
				}})
			}),
			want: map[string][]int{
				"_llgo_pkg.Child": {0},
				"_llgo_pkg.T":     {0},
			},
		},
		// type T struct{}
		// func (T) M() {}
		// func (T) N() {}
		// func main() { use(T{}) }
		// func use(v any) { reflect.ValueOf(v).MethodByName("M") }
		{
			name:  "constant MethodByName keeps same-name method",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				use := b.Symbol("pkg.use")
				typ := b.Symbol("_llgo_pkg.T")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
				})
				b.AddEdge(main, use)
				b.AddEdge(main, typ)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddUseNamedMethod(use, []metadata.Name{mSig.Name})
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				use := b.Symbol("pkg.use")
				typ := b.Symbol("_llgo_pkg.T")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")
				unexportedSig := methodSig(b, "m")

				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
					methodSlot(b, nSig, "pkg.(*T).N", "pkg.T.N"),
					methodSlot(b, unexportedSig, "pkg.(*T).m", "pkg.T.m"),
				})
				b.AddEdge(main, use)
				b.AddEdge(main, typ)
				b.AddUseIface(main, []metadata.Symbol{typ})
				b.AddReflectMethod(use)
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
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				useT := b.Symbol("pkg.useT")
				callU := b.Symbol("pkg.callU")
				useU := b.Symbol("pkg.useU")
				typT := b.Symbol("_llgo_pkg.T")
				typU := b.Symbol("_llgo_pkg.U")
				ifaceI := b.Symbol("_llgo_iface$I")
				ifaceJ := b.Symbol("_llgo_iface$J")
				mSig := methodSig(b, "M")
				nSig := methodSig(b, "N")

				b.AddIfaceEntry(ifaceI, []metadata.MethodSig{mSig})
				b.AddIfaceEntry(ifaceJ, []metadata.MethodSig{nSig})
				b.AddMethodInfo(typT, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddMethodInfo(typU, []metadata.MethodSlot{
					methodSlot(b, nSig, "pkg.(*U).N", "pkg.U.N"),
				})
				b.AddEdge(main, useT)
				b.AddEdge(main, typT)
				b.AddUseIface(main, []metadata.Symbol{typT})
				b.AddUseIfaceMethod(useT, []metadata.IfaceMethodDemand{{
					Target: ifaceI,
					Sig:    mSig,
				}})
				b.AddEdge(b.Symbol("pkg.T.M"), callU)
				b.AddEdge(callU, useU)
				b.AddEdge(callU, typU)
				b.AddUseIface(callU, []metadata.Symbol{typU})
				b.AddUseIfaceMethod(useU, []metadata.IfaceMethodDemand{{
					Target: ifaceJ,
					Sig:    nSig,
				}})
			}),
			want: map[string][]int{
				"_llgo_pkg.T": {0},
				"_llgo_pkg.U": {0},
			},
		},
		// type I interface{ M() }
		// type T struct{}
		// func (T) M() {}
		// func main() { _ = T{} }
		// func unreachable(i I) { i.M(); reflect.ValueOf(i).MethodByName("M") }
		{
			name:  "ignores unreachable semantic facts",
			roots: []string{"pkg.main"},
			summary: buildPackage(func(b *metadata.Builder) {
				main := b.Symbol("pkg.main")
				unreachable := b.Symbol("pkg.unreachable")
				typ := b.Symbol("_llgo_pkg.T")
				iface := b.Symbol("_llgo_iface$I")
				mSig := methodSig(b, "M")

				b.AddIfaceEntry(iface, []metadata.MethodSig{mSig})
				b.AddMethodInfo(typ, []metadata.MethodSlot{
					methodSlot(b, mSig, "pkg.(*T).M", "pkg.T.M"),
				})
				b.AddEdge(main, typ)
				b.AddUseIface(unreachable, []metadata.Symbol{typ})
				b.AddUseIfaceMethod(unreachable, []metadata.IfaceMethodDemand{{
					Target: iface,
					Sig:    mSig,
				}})
				b.AddUseNamedMethod(unreachable, []metadata.Name{mSig.Name})
				b.AddReflectMethod(unreachable)
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

type deadcodeCase struct {
	name    string
	summary *metadata.PackageMeta
	roots   []string
	want    map[string][]int
}

func buildPackage(fn func(*metadata.Builder)) *metadata.PackageMeta {
	b := metadata.NewBuilder()
	fn(b)
	return b.Build()
}

func methodSig(b *metadata.Builder, name string) metadata.MethodSig {
	return methodSigWithType(b, name, "_llgo_func$X")
}

func methodSigWithType(b *metadata.Builder, name, mtype string) metadata.MethodSig {
	return metadata.MethodSig{
		Name:  b.Name(name),
		MType: b.Symbol(mtype),
	}
}

func methodSlot(b *metadata.Builder, sig metadata.MethodSig, ifn, tfn string) metadata.MethodSlot {
	return metadata.MethodSlot{
		Sig: sig,
		IFn: b.Symbol(ifn),
		TFn: b.Symbol(tfn),
	}
}

func newSummary(t *testing.T, pkgs ...*metadata.PackageMeta) *metadata.GlobalSummary {
	t.Helper()
	summary, err := metadata.NewGlobalSummary(pkgs)
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}
	return summary
}
