//go:build llgo && go1.25

package gotest

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestReflectNativeCallbackArrayTypeAssert(t *testing.T) {
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, nil, reflectBulkAdd11}, 0x11223344)
	boxed, ok := reflect.TypeAssert[[3]reflectBulkCallback](reflect.ValueOf(record).Elem().Field(1))
	if !ok {
		t.Fatal("asserted array lost its type identity")
	}
	checkReflectBulkCallbacks(t, boxed, [3]int32{12, 0, 16})
	checkReflectBulkRecord(t, record, [3]int32{12, 0, 16}, 0x11223344)
}

func TestReflectNativeCallbackSliceTypeAssert(t *testing.T) {
	testReflectCallbackSlice(t, reflect.TypeAssert[[]reflectBulkCallback])
}

func TestReflectNativeCallbackSingleArrayTypeAssert(t *testing.T) {
	for _, tc := range reflectCallbackSingleArrays() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value.Type() == reflect.TypeFor[reflectBulkNativeArray]() {
				boxed, ok := reflect.TypeAssert[reflectBulkNativeArray](tc.value)
				if !ok {
					t.Fatal("asserted native array lost its type identity")
				}
				checkReflectBulkCallback(t, 0, boxed[0], tc.want)
			} else {
				boxed, ok := reflect.TypeAssert[[1]reflectBulkCallback](tc.value)
				if !ok {
					t.Fatal("asserted Go array lost its type identity")
				}
				checkReflectBulkCallback(t, 0, boxed[0], tc.want)
			}
		})
	}
}

func TestReflectNativeCallbackSlotAddrTypeAssert(t *testing.T) {
	for _, slot := range reflectBulkCallbackSlots() {
		addr := slot.Addr()
		if unsafe.Sizeof(reflectBulkNativeArray{}) == unsafe.Sizeof([1]reflectBulkCallback{}) {
			same := addr.Convert(reflect.TypeFor[*reflectBulkCallback]())
			p, ok := reflect.TypeAssert[*reflectBulkCallback](same)
			if !ok || reflect.ValueOf(p).Pointer() != addr.Pointer() {
				t.Fatal("asserted callback pointer did not share the slot")
			}
			continue
		}
		const want = "reflect: cannot use a Go pointer to a packed callback"
		expectPanicEqual(t, want, func() { reflect.TypeAssert[*reflectBulkCallback](addr) })
		expectPanicEqual(t, want, func() { reflect.TypeAssert[int](addr) })
	}
}

func TestReflectNativeCallbackFieldTypeAssert(t *testing.T) {
	for _, callback := range []reflectBulkCallback{nil, reflectBulkAdd7} {
		record := reflectBulkSingleRecord{
			Callback: callback, Callbacks: [1]reflectBulkCallback{reflectBulkAdd11},
			Tail: [2]uint32{0x11223344, 0x55667788},
		}
		fn, ok := reflect.TypeAssert[reflectBulkCallback](reflect.ValueOf(&record).Elem().Field(0))
		if !ok || (fn == nil) != (callback == nil) {
			t.Fatalf("callback assertion=%v nil=%v, want nil=%v", ok, fn == nil, callback == nil)
		}
		if fn != nil && fn(5) != 12 {
			t.Fatal("asserted callback lost its target")
		}
		if record.Callbacks[0](5) != 16 || record.Tail != [2]uint32{0x11223344, 0x55667788} {
			t.Fatal("TypeAssert changed neighboring fields")
		}
	}
}

//llgo:cdecl
type reflectAliasField struct {
	Callback reflectBulkCallbackAlias
	Marker   uint32
}

func TestReflectNativeAliasTypeAssert(t *testing.T) {
	record := reflectAliasField{Callback: reflectBulkAdd7, Marker: 17}
	field := reflect.ValueOf(&record).Elem().Field(0)
	var boxed any = field.Interface()
	for _, v := range []reflect.Value{field, reflect.ValueOf(&boxed).Elem()} {
		fn, ok := reflect.TypeAssert[func(int32) int32](v)
		if !ok || fn == nil || fn(5) != 12 {
			t.Fatal("asserting a native alias as its Go signature lost the callback")
		}
	}
	record.Callback = nil
	if fn, ok := reflect.TypeAssert[func(int32) int32](field); !ok || fn != nil {
		t.Fatal("asserting a nil native alias produced a callback")
	}
}

func TestReflectHigherOrderAliasTypeAssert(t *testing.T) {
	field := reflect.ValueOf(struct{ F reflectBulkCallbackIdentity }{reflectBulkNativeIdentity}).Field(0)
	identity, ok := reflect.TypeAssert[func(func(int32) int32) func(int32) int32](field)
	if !ok || identity == nil || identity(reflectBulkAdd7)(5) != 12 {
		t.Fatal("asserting a higher-order native alias lost its callable Go frame")
	}
}

func TestReflectNativeCallbackReadOnlyTypeAssert(t *testing.T) {
	wrapped := struct{ record reflectBulkSingleRecord }{reflectBulkSingleRecord{
		Callback: reflectBulkAdd7, Callbacks: [1]reflectBulkCallback{reflectBulkAdd11},
		Tail: [2]uint32{0x11223344, 0x55667788},
	}}
	record := reflect.ValueOf(&wrapped).Elem().Field(0)
	callback, array := record.Field(0), record.Field(1)
	slice := array.Slice(0, 1)
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"callback", func() { reflect.TypeAssert[reflectBulkCallback](callback) }},
		{"array", func() { reflect.TypeAssert[[1]reflectBulkCallback](array) }},
		{"slice", func() { reflect.TypeAssert[[]reflectBulkCallback](slice) }},
		{"mismatched", func() { reflect.TypeAssert[int](callback) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectPanicEqual(t, "reflect.TypeAssert: cannot return value obtained from unexported field or method", tc.call)
		})
	}
}
