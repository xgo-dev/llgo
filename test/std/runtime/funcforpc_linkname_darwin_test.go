//go:build darwin

package runtime_test

import (
	"reflect"
	"runtime"
	"testing"
	_ "unsafe"
)

//go:linkname oneUnderscore _llgo_runtime_linkname_probe
//go:noinline
func oneUnderscore() int { return 1 }

//go:linkname twoUnderscores __llgo_runtime_linkname_probe
//go:noinline
func twoUnderscores() int { return 2 }

func TestFuncForPCDistinctUnderscoreLinknames(t *testing.T) {
	first := runtime.FuncForPC(reflect.ValueOf(oneUnderscore).Pointer())
	second := runtime.FuncForPC(reflect.ValueOf(twoUnderscores).Pointer())
	if first == nil || second == nil {
		t.Fatal("FuncForPC lost a linked function")
	}
	if first.Name() == second.Name() {
		t.Fatalf("distinct linknames share function metadata: %q", first.Name())
	}
	if oneUnderscore() != 1 || twoUnderscores() != 2 {
		t.Fatal("linked function implementations were merged")
	}
}
