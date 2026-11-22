//go:build llgo

package gotest

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type reflectBulkCallback func(int32) int32

//llgo:cdecl
type reflectBulkCallbackAlias = func(int32) int32

//llgo:cdecl
type reflectBulkCallbackIdentity = func(reflectBulkCallbackAlias) reflectBulkCallbackAlias

//llgo:cdecl
type reflectBulkNativeArray [1]reflectBulkCallback

//llgo:cdecl
type reflectBulkRecord struct {
	Before    uint32
	Callbacks [3]reflectBulkCallback
	After     [3]uint32
}

//llgo:cdecl
type reflectBulkSingleRecord struct {
	Callback  reflectBulkCallback
	Callbacks [1]reflectBulkCallback
	Tail      [2]uint32
}

func reflectBulkAdd7(x int32) int32  { return x + 7 }
func reflectBulkAdd11(x int32) int32 { return x + 11 }
func reflectBulkAdd23(x int32) int32 { return x + 23 }

func reflectBulkGoIdentity(f func(int32) int32) func(int32) int32 { return f }

func TestReflectNativeCallbackConvertFrame(t *testing.T) {
	src := reflect.ValueOf(reflectBulkGoIdentity)
	// Composite Elem retains the raw descriptor; TypeFor exposes its Go signature.
	dst := reflect.TypeOf([1]reflectBulkCallbackIdentity{}).Elem()
	if src.Type() != reflect.TypeFor[reflectBulkCallbackIdentity]() || dst == src.Type() {
		t.Fatal("expected equal public signatures with distinct physical frames")
	}
	defer func() {
		const want = "reflect.Value.Convert: C callback with a different ABI must be a direct function reference"
		if got := recover(); got != want {
			t.Fatalf("conversion panicked with %v, want %q", got, want)
		}
	}()
	src.Convert(dst)
}

func newReflectBulkRecord(callbacks [3]reflectBulkCallback, marker uint32) *reflectBulkRecord {
	return &reflectBulkRecord{
		Before:    marker,
		Callbacks: callbacks,
		// Keep enough sentinel storage to contain an incorrectly widened copy.
		After: [3]uint32{marker + 1, marker + 2, marker + 3},
	}
}

func checkReflectBulkRecord(t *testing.T, record *reflectBulkRecord, want [3]int32, marker uint32) {
	t.Helper()
	if record.Before != marker || record.After != [3]uint32{marker + 1, marker + 2, marker + 3} {
		t.Fatalf("callback operation changed neighboring fields: %#x, %#x", record.Before, record.After)
	}
	checkReflectBulkCallbacks(t, record.Callbacks, want)
}

func checkReflectBulkCallbacks(t *testing.T, callbacks [3]reflectBulkCallback, want [3]int32) {
	t.Helper()
	for i, fn := range callbacks {
		checkReflectBulkCallback(t, i, fn, want[i])
	}
}

func checkReflectBulkCallback(t *testing.T, i int, fn reflectBulkCallback, want int32) {
	t.Helper()
	if (fn == nil) != (want == 0) {
		t.Fatalf("callback %d: nil=%v, want nil=%v", i, fn == nil, want == 0)
	}
	if fn != nil {
		if got := fn(5); got != want {
			t.Fatalf("callback %d(5) = %d, want %d", i, got, want)
		}
	}
}

func TestReflectNativeCallbackCopy(t *testing.T) {
	types := []struct {
		name          string
		native, slice bool
	}{
		{"native array", true, false}, {"native slice", true, true},
		{"Go array", false, false}, {"Go slice", false, true},
	}
	makeValue := func(native, slice bool, callbacks [3]reflectBulkCallback, marker uint32) (reflect.Value, func(*testing.T, [3]int32)) {
		value := reflect.ValueOf(&callbacks).Elem()
		check := func(t *testing.T, want [3]int32) { checkReflectBulkCallbacks(t, callbacks, want) }
		if native {
			record := newReflectBulkRecord(callbacks, marker)
			value = reflect.ValueOf(record).Elem().Field(1)
			check = func(t *testing.T, want [3]int32) { checkReflectBulkRecord(t, record, want, marker) }
		}
		if slice {
			value = value.Slice(0, value.Len())
		}
		return value, check
	}
	for _, dstType := range types {
		for _, srcType := range types {
			t.Run(dstType.name+" from "+srcType.name, func(t *testing.T) {
				dst, checkDst := makeValue(dstType.native, dstType.slice, [3]reflectBulkCallback{reflectBulkAdd23, reflectBulkAdd23, reflectBulkAdd23}, 0x11223344)
				src, checkSrc := makeValue(srcType.native, srcType.slice, [3]reflectBulkCallback{reflectBulkAdd7, nil, reflectBulkAdd11}, 0x55667788)
				if n := reflect.Copy(dst, src); n != 3 {
					t.Fatalf("Copy returned %d, want 3", n)
				}
				checkDst(t, [3]int32{12, 0, 16})
				checkSrc(t, [3]int32{12, 0, 16})
			})
		}
	}
}

func TestReflectNativeCallbackCopyOverlap(t *testing.T) {
	for _, tc := range []struct {
		name         string
		dstLo, dstHi int
		srcLo, srcHi int
		want         [3]int32
	}{
		{"forward", 1, 3, 0, 3, [3]int32{12, 12, 16}},
		{"backward", 0, 3, 1, 3, [3]int32{16, 28, 28}},
		{"empty", 1, 1, 0, 3, [3]int32{12, 16, 28}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, reflectBulkAdd11, reflectBulkAdd23}, 0x11223344)
			array := reflect.ValueOf(record).Elem().Field(1)
			dst, src := array.Slice(tc.dstLo, tc.dstHi), array.Slice(tc.srcLo, tc.srcHi)
			if got, want := reflect.Copy(dst, src), min(dst.Len(), src.Len()); got != want {
				t.Fatalf("Copy returned %d, want %d", got, want)
			}
			checkReflectBulkRecord(t, record, tc.want, 0x11223344)
		})
	}
}

func TestReflectNativeCallbackClear(t *testing.T) {
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, reflectBulkAdd11, reflectBulkAdd23}, 0x11223344)
	array := reflect.ValueOf(record).Elem().Field(1)
	array.Slice(1, 1).Clear()
	checkReflectBulkRecord(t, record, [3]int32{12, 16, 28}, 0x11223344)
	array.Slice(1, 2).Clear()
	checkReflectBulkRecord(t, record, [3]int32{12, 0, 28}, 0x11223344)
	array.Slice(0, 3).Clear()
	checkReflectBulkRecord(t, record, [3]int32{}, 0x11223344)
}

func TestReflectNativeCallbackArrayTransfer(t *testing.T) {
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, nil, reflectBulkAdd11}, 0x11223344)
	array := reflect.ValueOf(record).Elem().Field(1)
	t.Run("Interface", func(t *testing.T) {
		boxed, ok := array.Interface().([3]reflectBulkCallback)
		if !ok {
			t.Fatal("boxed array lost its type identity")
		}
		checkReflectBulkCallbacks(t, boxed, [3]int32{12, 0, 16})
	})
	t.Run("TypeAssert", func(t *testing.T) {
		boxed, ok := reflect.TypeAssert[[3]reflectBulkCallback](array)
		if !ok {
			t.Fatal("asserted array lost its type identity")
		}
		checkReflectBulkCallbacks(t, boxed, [3]int32{12, 0, 16})
	})
	t.Run("map", func(t *testing.T) {
		values := make(map[int][3]reflectBulkCallback)
		reflect.ValueOf(values).SetMapIndex(reflect.ValueOf(1), array)
		checkReflectBulkCallbacks(t, values[1], [3]int32{12, 0, 16})
	})
	t.Run("channel", func(t *testing.T) {
		values := make(chan [3]reflectBulkCallback, 1)
		reflect.ValueOf(values).Send(array)
		checkReflectBulkCallbacks(t, <-values, [3]int32{12, 0, 16})
	})
	checkReflectBulkRecord(t, record, [3]int32{12, 0, 16}, 0x11223344)
}

func TestReflectNativeCallbackSliceInterface(t *testing.T) {
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, nil, reflectBulkAdd11}, 0x11223344)
	array := reflect.ValueOf(record).Elem().Field(1)
	for _, tc := range []struct {
		name   string
		value  reflect.Value
		length int
		want   []int32
	}{
		{"slice", array.Slice(0, 2), 2, []int32{12, 0, 16}},
		{"slice3", array.Slice3(1, 2, 3), 1, []int32{0, 16}},
		{"empty", array.Slice(1, 1), 0, []int32{0, 16}},
		{"zero capacity", array.Slice3(3, 3, 3), 0, []int32{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := func(t *testing.T, boxed []reflectBulkCallback, ok bool) {
				t.Helper()
				if !ok || boxed == nil || len(boxed) != tc.length || cap(boxed) != len(tc.want) {
					t.Fatalf("boxed slice: assertion=%v nil=%v len=%d cap=%d", ok, boxed == nil, len(boxed), cap(boxed))
				}
				for i, fn := range boxed[:cap(boxed)] {
					checkReflectBulkCallback(t, i, fn, tc.want[i])
				}
				checkReflectBulkRecord(t, record, [3]int32{12, 0, 16}, 0x11223344)
			}
			t.Run("Interface", func(t *testing.T) {
				boxed, ok := tc.value.Interface().([]reflectBulkCallback)
				check(t, boxed, ok)
			})
			t.Run("TypeAssert", func(t *testing.T) {
				boxed, ok := reflect.TypeAssert[[]reflectBulkCallback](tc.value)
				check(t, boxed, ok)
			})
			t.Run("Set", func(t *testing.T) {
				var boxed []reflectBulkCallback
				reflect.ValueOf(&boxed).Elem().Set(tc.value)
				check(t, boxed, true)
			})
			t.Run("Append", func(t *testing.T) {
				boxed, ok := reflect.Append(tc.value).Interface().([]reflectBulkCallback)
				check(t, boxed, ok)
			})
		})
	}
}

func TestReflectNativeCallbackSingleArray(t *testing.T) {
	native := reflectBulkNativeArray{reflectBulkAdd7}
	plain := [1]reflectBulkCallback{reflectBulkAdd7}
	record := reflectBulkSingleRecord{Callbacks: plain}
	for _, tc := range []struct {
		name  string
		value reflect.Value
		want  int32
	}{
		{"native value", reflect.ValueOf(native), 12},
		{"native addressable", reflect.ValueOf(&native).Elem(), 12},
		{"Go value", reflect.ValueOf(plain), 12},
		{"Go addressable", reflect.ValueOf(&plain).Elem(), 12},
		{"field value", reflect.ValueOf(record).Field(1), 12},
		{"field addressable", reflect.ValueOf(&record).Elem().Field(1), 12},
		{"native nil", reflect.ValueOf(reflectBulkNativeArray{}), 0},
		{"Go nil", reflect.ValueOf([1]reflectBulkCallback{}), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := func(fn reflectBulkCallback, ok bool) {
				t.Helper()
				if !ok {
					t.Fatal("array lost its type identity")
				}
				checkReflectBulkCallback(t, 0, fn, tc.want)
			}
			nativeDst := reflectBulkNativeArray{reflectBulkAdd23}
			plainDst := [1]reflectBulkCallback{reflectBulkAdd23}
			recordDst := reflectBulkSingleRecord{
				Callback: reflectBulkAdd11, Callbacks: plainDst,
				Tail: [2]uint32{0x11223344, 0x55667788},
			}
			for _, dst := range []reflect.Value{
				reflect.ValueOf(&nativeDst).Elem(),
				reflect.ValueOf(&plainDst).Elem(),
				reflect.ValueOf(&recordDst).Elem().Field(1),
			} {
				if n := reflect.Copy(dst, tc.value); n != 1 {
					t.Fatalf("Copy returned %d, want 1", n)
				}
			}
			check(nativeDst[0], true)
			check(plainDst[0], true)
			check(recordDst.Callbacks[0], true)
			if recordDst.Callback == nil || recordDst.Callback(5) != 16 || recordDst.Tail != [2]uint32{0x11223344, 0x55667788} {
				t.Fatal("Copy changed neighboring fields")
			}
			if tc.value.Type() == reflect.TypeOf(native) {
				boxed, ok := tc.value.Interface().(reflectBulkNativeArray)
				check(boxed[0], ok)
				boxed, ok = reflect.TypeAssert[reflectBulkNativeArray](tc.value)
				check(boxed[0], ok)
			} else {
				boxed, ok := tc.value.Interface().([1]reflectBulkCallback)
				check(boxed[0], ok)
				boxed, ok = reflect.TypeAssert[[1]reflectBulkCallback](tc.value)
				check(boxed[0], ok)
			}
		})
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

func TestReflectNativeCallbackReadOnly(t *testing.T) {
	wrapped := struct{ record reflectBulkSingleRecord }{reflectBulkSingleRecord{
		Callback: reflectBulkAdd7, Callbacks: [1]reflectBulkCallback{reflectBulkAdd11},
		Tail: [2]uint32{0x11223344, 0x55667788},
	}}
	record := reflect.ValueOf(&wrapped).Elem().Field(0)
	callback, array := record.Field(0), record.Field(1)
	slice := array.Slice(0, 1)
	ordinary := reflect.ValueOf([]reflectBulkCallback{reflectBulkAdd23})
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"callback Interface", func() { callback.Interface() }},
		{"array Interface", func() { array.Interface() }},
		{"slice Interface", func() { slice.Interface() }},
		{"callback TypeAssert", func() { reflect.TypeAssert[reflectBulkCallback](callback) }},
		{"array TypeAssert", func() { reflect.TypeAssert[[1]reflectBulkCallback](array) }},
		{"slice TypeAssert", func() { reflect.TypeAssert[[]reflectBulkCallback](slice) }},
		{"mismatched TypeAssert", func() { reflect.TypeAssert[int](callback) }},
		{"Copy from array", func() { reflect.Copy(ordinary, array) }},
		{"Copy into array", func() { reflect.Copy(array, ordinary) }},
		{"Copy from slice", func() { reflect.Copy(ordinary, slice) }},
		{"Copy into slice", func() { reflect.Copy(slice, ordinary) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("operation on unexported field did not panic")
				}
			}()
			tc.call()
		})
	}
	// Clear permits a slice obtained from an unexported field.
	slice.Clear()
	if wrapped.record.Callbacks[0] != nil || wrapped.record.Callback(5) != 12 || wrapped.record.Tail != [2]uint32{0x11223344, 0x55667788} {
		t.Fatal("Clear changed more than the callback slice")
	}
}
