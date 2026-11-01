//go:build !llgo

package gotest

import (
	"go/types"
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

func compilerSizesForTest() map[string]types.Sizes {
	return map[string]types.Sizes{"gc": types.SizesFor("gc", runtime.GOARCH)}
}

func TestGCTypeSizes(t *testing.T) {
	if runtime.Compiler != "gc" {
		t.Fatalf("Compiler = %q", runtime.Compiler)
	}
	if types.SizesFor("llgo", runtime.GOARCH) != nil {
		t.Fatal("gc unexpectedly provides LLGo sizes")
	}
	word := unsafe.Sizeof(uintptr(0))
	if unsafe.Sizeof(sizeFunc(nil)) != word || unsafe.Sizeof(sizeNativeFunc(nil)) != word {
		t.Fatal("gc function values must occupy one word and ignore native annotations")
	}
	// All LLGo annotations are comments to gc, so these fields keep natural
	// alignment and ordinary Go padding on every native pointer width.
	type Plain struct {
		B byte
		X uint64
		Y uint16
	}
	if unsafe.Sizeof(sizePacked{}) != unsafe.Sizeof(Plain{}) || unsafe.Alignof(sizePacked{}) != unsafe.Alignof(Plain{}) || unsafe.Offsetof(sizePacked{}.X) != unsafe.Offsetof(Plain{}.X) {
		t.Fatal("gc applied packed layout")
	}
	if unsafe.Sizeof(sizePacked2{}) != unsafe.Sizeof(Plain{}) || unsafe.Alignof(sizePacked2{}) != unsafe.Alignof(Plain{}) {
		t.Fatal("gc applied pack 2 layout")
	}
	if unsafe.Sizeof(sizePadding{}) != 1 || unsafe.Sizeof(sizeNoPadding{}) != 8+unsafe.Alignof(uint64(0)) || unsafe.Sizeof(sizePaddingIf[sizeEmpty, sizeEmpty]{}) != 0 {
		t.Fatal("gc applied padding directives")
	}
	x := llgoGenericPacked[uint64]{}
	if unsafe.Sizeof(x) != 8+unsafe.Alignof(uint64(0)) || unsafe.Offsetof(x.V) != unsafe.Alignof(uint64(0)) {
		t.Fatal("gc applied generic packing")
	}
	if !reflect.TypeOf((*sizePacked)(nil)).ConvertibleTo(reflect.TypeOf((*Plain)(nil))) {
		t.Fatal("gc rejected an equal-layout pointer conversion")
	}
	type OtherEmpty struct{}
	mixed, base := sizePaddingMixed[sizeEmpty, OtherEmpty]{}, sizePaddingBase[sizeEmpty]{}
	if unsafe.Sizeof(mixed) != 0 || unsafe.Offsetof(mixed.Last) != 0 || unsafe.Sizeof(base) != 0 || unsafe.Offsetof(base.Value) != 0 {
		t.Fatal("gc applied conditional padding")
	}
}
