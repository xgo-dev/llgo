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
		{"Go with unrelated bit", Type{Kind_: uint8(Func) | KindStdcall}, false},
		{"native struct", Type{Kind_: uint8(Struct) | KindStdcall, TFlag: TFlagNative}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.typ.IsStdcall(); got != tc.want {
				t.Fatalf("IsStdcall() = %v, want %v", got, tc.want)
			}
		})
	}
}
