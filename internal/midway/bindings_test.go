package midway

import (
	"go/types"
	"testing"
)

func TestRewriteGenericReceiverInstances(t *testing.T) {
	for _, recv := range []string{"Box[T]", "*Box[T]"} {
		t.Run(recv, func(t *testing.T) {
			pkg, _ := testCheck(t, testSIMDImports(t), "client", `package client
 import "simd"
 type Box[T any] struct{}
 func (`+recv+`) Width() int {var v simd.Uint32s;return v.Len()}
 var Global Box[int]
 func Identity(x Box[int]) Box[int] {return x}
 func Use() int {x:=Identity(Box[int]{});return x.Width()+Global.Width()}
 `, true)
			for _, name := range []string{"Box", "Global", "Identity", "Use"} {
				for _, width := range targetWidths("amd64") {
					if pkg.Scope().Lookup(name+suffix(width)) == nil {
						t.Fatalf("missing %s variant %d", name, width)
					}
				}
			}
		})
	}
}

func TestRewriteIndependentInitializers(t *testing.T) {
	for _, tc := range []struct {
		name, decl string
		want       []string
	}{
		{"earlier RHS", "var A,B=simd.BroadcastUint32s(1),A", []string{"A", "B"}},
		{"forward dependency", "var A,B=simd.BroadcastUint32s(1),C;var C=42", []string{"A", "C", "B"}},
		{"tuple RHS", "func pair()(simd.Uint32s,int){return simd.BroadcastUint32s(1),C};var A,B=pair();var C=42", []string{"C", "A", "B"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, info := testCheck(t, testSIMDImports(t), "client", "package client;import \"simd\";"+tc.decl, true)
			var got []string
			for _, init := range info.InitOrder {
				for _, v := range init.Lhs {
					if v.Name() != "_" {
						got = append(got, v.Name())
					}
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("init order %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("init order %v, want %v", got, tc.want)
				}
			}
			if tc.name == "tuple RHS" && len(info.InitOrder[1].Lhs) != 2 {
				t.Fatal("multi-result initializer was split")
			}
		})
	}
}

func TestRewriteShadowedPredeclaredTypes(t *testing.T) {
	imports := testSIMDImports(t)
	testCheck(t, imports, "foreign", `package foreign
 import "simd"
 func Make() (simd.Uint32s,string,error) {return simd.Uint32s{},"ok",nil}
 `, true)
	pkg, _ := testCheck(t, imports, "client", `package client
 import ("simd";"foreign")
 type int = string
 type bool = string
 type string = []byte
 type error = []byte
 var Width=(simd.Uint32s{}).Len()
 var Flag=(simd.Uint32s{}).Len()>0
 var Vector, Text, Err = foreign.Make()
 `, true)
	if !types.Identical(pkg.Scope().Lookup("Width").Type(), types.Typ[types.Int]) {
		t.Fatal("lost builtin int identity")
	}
	if !types.Identical(pkg.Scope().Lookup("Flag").Type(), types.Typ[types.Bool]) {
		t.Fatal("lost builtin bool identity")
	}
	if !types.Identical(pkg.Scope().Lookup("Text").Type(), types.Typ[types.String]) {
		t.Fatal("lost builtin string identity")
	}
	if !types.Identical(pkg.Scope().Lookup("Err").Type(), types.Universe.Lookup("error").Type()) {
		t.Fatal("lost builtin error identity")
	}
}

func TestRewriteBlankTypeParameters(t *testing.T) {
	testCheck(t, testSIMDImports(t), "client", `package client
 import "simd"
 func Width[_, _ any, T ~int](x T) T {var v simd.Uint32s;return x+T(v.Len())}
 func Use() int {return Width[int,string](3)}
 `, true)
}

func TestRewriteImportShadowsPredeclaredType(t *testing.T) {
	pkg, _ := testCheck(t, testSIMDImports(t), "client", `package client
 import ("simd";int "unsafe")
 var Width=(simd.Uint32s{}).Len()
 var _=int.Sizeof(0)
 `, true)
	if !types.Identical(pkg.Scope().Lookup("Width").Type(), types.Typ[types.Int]) {
		t.Fatal("lost builtin int identity")
	}
}

func TestRewriteBlankFunctions(t *testing.T) {
	pkg, _ := testCheck(t, testSIMDImports(t), "client", `package client
import "simd"
func _() int {var v simd.Uint32s;return v.Len()}
func _() int {var v simd.Uint32s;return v.Len()}
type Number int
func (Number) _() int {var v simd.Uint32s;return v.Len()}
func (Number) _() int {var v simd.Uint32s;return v.Len()}
func Width() int {var v simd.Uint32s;return v.Len()}
`, true)
	for _, width := range targetWidths("amd64") {
		if pkg.Scope().Lookup("Number"+suffix(width)) != nil {
			t.Fatal("blank method created unnecessary receiver variants")
		}
	}
}
