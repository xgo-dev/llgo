package gotest

import "testing"

func TestGoToNativeFunctionAssertionIdentity(t *testing.T) {
	for _, value := range []any{
		assertFuncA(func(x int32) int32 { return x }),
		assertFuncA(nil),
	} {
		if _, ok := value.(assertNativeFunc); ok {
			t.Fatalf("Go function %T asserted as native function", value)
		}
	}
}

//go:noinline
func assertNativeFunction(value any) (assertNativeFunc, bool) {
	fn, ok := value.(assertNativeFunc)
	return fn, ok
}

func TestFunctionNativeCommaOK(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   any
		wantOK  bool
		wantNil bool
	}{
		{"native", assertNativeFunc(func(x int32) int32 { return x + 1 }), true, false},
		{"native nil", assertNativeFunc(nil), true, true},
		{"Go function", assertFuncA(func(x int32) int32 { return x + 1 }), false, true},
		{"Go nil", assertFuncA(nil), false, true},
		{"nil interface", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn, ok := assertNativeFunction(tc.value)
			if ok != tc.wantOK || (fn == nil) != tc.wantNil {
				t.Fatalf("assertion of %T: ok=%v nil=%v, want ok=%v nil=%v", tc.value, ok, fn == nil, tc.wantOK, tc.wantNil)
			}
			if fn != nil && fn(5) != 6 {
				t.Fatal("asserted native function did not preserve its target")
			}
		})
	}
}
