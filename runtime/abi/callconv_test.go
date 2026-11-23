package abi

import "testing"

func TestNativeFunctionCallConvention(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  Type
		want bool
	}{
		{"Go", Type{Kind_: uint8(Func)}, false},
		{"cdecl", Type{Kind_: uint8(Func) | KindDirectIface, TFlag: TFlagNative}, false},
		{"stdcall", Type{Kind_: uint8(Func) | KindDirectIface | KindStdcall, TFlag: TFlagNative}, true},
		{"Go closure descriptor", Type{Kind_: uint8(Func) | KindClosureFunc}, false},
		{"native struct", Type{Kind_: uint8(Struct) | KindStdcall, TFlag: TFlagNative}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.typ.IsStdcall(); got != tc.want {
				t.Fatalf("IsStdcall() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNamedClosureDescriptors(t *testing.T) {
	storage := &namedClosureTypeUncommon{
		NamedClosureType: NamedClosureType{
			StructType: StructType{Type: Type{
				Kind_: uint8(Struct),
				TFlag: TFlagNamed | TFlagClosure | TFlagUncommon,
			}},
		},
		u: UncommonType{Mcount: 1},
	}
	public := &ClosureFuncType{
		FuncType: FuncType{Type: Type{
			Kind_: uint8(Func) | KindClosureFunc,
			TFlag: TFlagNamed | TFlagUncommon,
		}},
		Closure: &storage.StructType,
	}
	storage.PublicFunc = &public.FuncType
	if storage.PublicFuncType() != &public.FuncType || public.PhysicalClosure() != &storage.StructType {
		t.Fatal("function descriptors do not round trip")
	}
	if public.IsClosure() || public.IsStdcall() || public.Uncommon() != storage.Uncommon() ||
		public.Uncommon().Mcount != 1 {
		t.Fatal("public function has incorrect flags or method metadata")
	}

	native := &FuncType{Type: Type{Kind_: uint8(Func) | KindStdcall, TFlag: TFlagNative}}
	if !native.IsStdcall() || native.PhysicalClosure() != nil {
		t.Fatal("stdcall function was treated as a Go closure")
	}
}
