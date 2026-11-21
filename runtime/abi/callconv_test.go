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

func TestCallbackDescriptorIdentity(t *testing.T) {
	plain := &Type{Kind_: uint8(Func)}
	native := &QualifiedFuncType{Public: plain}
	native.Kind_, native.TFlag = uint8(Func), TFlagNative|TFlagDirectGoCallback
	frame := &QualifiedFuncType{Public: plain}
	frame.Kind_ = uint8(Func) | KindFuncPublic
	named := &Type{Kind_: uint8(Func) | KindStdcall, TFlag: TFlagNative | TFlagNamed}
	closure := &Type{Kind_: uint8(Struct), TFlag: TFlagClosure}

	for _, tc := range []struct {
		name        string
		typ, public *Type
		alias       *QualifiedFuncType
		closure     bool
	}{
		{"Go function", plain, plain, nil, false},
		{"native alias", &native.Type, plain, native, false},
		{"Go function with native frame", &frame.Type, plain, nil, false},
		{"named stdcall", named, named, nil, false},
		{"Go closure", closure, closure, nil, true},
	} {
		if got := tc.typ.PublicFunc(); got != tc.public {
			t.Errorf("%s: PublicFunc() = %p, want %p", tc.name, got, tc.public)
		}
		if got := tc.typ.NativeFuncAlias(); got != tc.alias {
			t.Errorf("%s: NativeFuncAlias() = %p, want %p", tc.name, got, tc.alias)
		}
		if got := tc.typ.IsClosure(); got != tc.closure {
			t.Errorf("%s: IsClosure() = %t, want %t", tc.name, got, tc.closure)
		}
	}
}
