//go:build go1.26
// +build go1.26

package gotest

import (
	"iter"
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

type kubeReflectTypeInterface struct {
	itab unsafe.Pointer
	data unsafe.Pointer
}

func kubeReflectTypeData(t reflect.Type) unsafe.Pointer {
	return (*kubeReflectTypeInterface)(unsafe.Pointer(&t)).data
}

//go:linkname kubeReflectUnsafeNew reflect.unsafe_New
func kubeReflectUnsafeNew(unsafe.Pointer) unsafe.Pointer

//go:linkname kubeReflectTypedmemmove reflect.typedmemmove
func kubeReflectTypedmemmove(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer)

//go:linkname kubeReflectUnsafeNewArray reflect.unsafe_NewArray
func kubeReflectUnsafeNewArray(unsafe.Pointer, int) unsafe.Pointer

//go:linkname kubeReflectMakeMap reflect.makemap
func kubeReflectMakeMap(unsafe.Pointer, int) unsafe.Pointer

//go:linkname kubeReflectMapAccess reflect.mapaccess
func kubeReflectMapAccess(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) unsafe.Pointer

//go:linkname kubeReflectMapIterInit reflect.mapiterinit
func kubeReflectMapIterInit(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer)

//go:linkname kubeReflectMapIterNext reflect.mapiternext
func kubeReflectMapIterNext(unsafe.Pointer)

//go:linkname kubeReflectIfaceE2I reflect.ifaceE2I
func kubeReflectIfaceE2I(unsafe.Pointer, any, unsafe.Pointer)

var kubeReflectPrivateSymbolRefs = [...]any{
	kubeReflectMakeMap,
	kubeReflectMapAccess,
	kubeReflectMapIterInit,
	kubeReflectMapIterNext,
	kubeReflectIfaceE2I,
}

func TestKubeReflectPrivateLinknameSymbols(t *testing.T) {
	intType := kubeReflectTypeData(reflect.TypeOf(int(0)))
	if intType == nil {
		t.Fatal("reflect int type data is nil")
	}

	p := kubeReflectUnsafeNew(intType)
	if p == nil {
		t.Fatal("reflect.unsafe_New returned nil")
	}
	*(*int)(p) = 42
	if got := *(*int)(p); got != 42 {
		t.Fatalf("unsafe_New value = %d, want 42", got)
	}

	src := 123
	dst := 0
	kubeReflectTypedmemmove(intType, unsafe.Pointer(&dst), unsafe.Pointer(&src))
	if dst != src {
		t.Fatalf("typedmemmove dst = %d, want %d", dst, src)
	}

	arr := kubeReflectUnsafeNewArray(intType, 2)
	if arr == nil {
		t.Fatal("reflect.unsafe_NewArray returned nil")
	}

	if len(kubeReflectPrivateSymbolRefs) == 0 {
		t.Fatal("private reflect symbol refs missing")
	}
}

type kubeAddressableValue struct {
	reflect.Value
	forcedAddr bool
}

var _ interface {
	Fields() iter.Seq2[reflect.StructField, reflect.Value]
	Methods() iter.Seq2[reflect.Method, reflect.Value]
} = kubeAddressableValue{}

//go:linkname kubeReflectValueAbiType reflect.Value.abiType
func kubeReflectValueAbiType(reflect.Value) unsafe.Pointer

//go:linkname kubeReflectValueAbiTypeSlow reflect.Value.abiTypeSlow
func kubeReflectValueAbiTypeSlow(reflect.Value) unsafe.Pointer

func TestKubeReflectValueGo126PromotedSymbols(t *testing.T) {
	v := reflect.ValueOf(struct{ A int }{A: 1})
	if kubeReflectValueAbiType(v) == nil {
		t.Fatal("reflect.Value.abiType returned nil")
	}
	if kubeReflectValueAbiTypeSlow(v) == nil {
		t.Fatal("reflect.Value.abiTypeSlow returned nil")
	}

	av := kubeAddressableValue{Value: v}
	fields := 0
	for field, value := range av.Fields() {
		fields++
		if field.Name != "A" || value.Int() != 1 {
			t.Fatalf("field = (%s, %v), want (A, 1)", field.Name, value.Interface())
		}
	}
	if fields != 1 {
		t.Fatalf("Fields yielded %d values, want 1", fields)
	}

	for range av.Methods() {
		t.Fatal("unexpected method on anonymous struct value")
	}
}

func TestKubeRuntimeFuncFileLineSymbol(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller returned no frame")
	}
	if fn := runtime.FuncForPC(pc); fn != nil {
		_, _ = fn.FileLine(pc)
	}
}
