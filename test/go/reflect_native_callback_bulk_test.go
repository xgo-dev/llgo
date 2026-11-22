//go:build llgo

package gotest

import (
	"reflect"
	"testing"
	"unsafe"
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
	if src.CanConvert(dst) {
		t.Fatal("CanConvert accepted a callback with an incompatible argument and result frame")
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

func TestReflectNativeCallbackAliasCopyAppend(t *testing.T) {
	for _, tc := range []struct {
		name string
		move func(dst, src reflect.Value) int
	}{
		{"Copy", reflect.Copy},
		{"AppendSlice", func(dst, src reflect.Value) int {
			return reflect.AppendSlice(dst.Slice(0, 0), src.Slice(0, src.Len())).Len()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			native := [3]reflectBulkCallbackAlias{reflectBulkAdd7, nil, reflectBulkAdd11}
			ordinary := make([]func(int32) int32, len(native))
			a, s := reflect.ValueOf(&native).Elem(), reflect.ValueOf(ordinary)
			if n := tc.move(s, a); n != 3 {
				t.Fatalf("copied %d native callbacks to Go, want 3", n)
			}
			if ordinary[0] == nil || ordinary[0](5) != 12 || ordinary[1] != nil || ordinary[2] == nil || ordinary[2](5) != 16 {
				t.Fatal("native callbacks lost their entries in Go function slots")
			}
			ordinary[0], ordinary[1], ordinary[2] = nil, reflectBulkAdd23, ordinary[0]
			if n := tc.move(a, s); n != 3 {
				t.Fatalf("copied %d Go callbacks to native slots, want 3", n)
			}
			for i, want := range []int32{0, 28, 12} {
				checkReflectBulkCallback(t, i, native[i], want)
			}

			offset := int32(len(t.Name()))
			ordinary[0], ordinary[1] = reflectBulkAdd7, func(v int32) int32 { return v + offset }
			defer func() {
				if recover() == nil {
					t.Fatal("accepted a captured closure in a native callback slot")
				}
				if native[0] != nil {
					t.Fatal("wrote the first callback before rejecting the second")
				}
			}()
			tc.move(a, s)
		})
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

func TestReflectNativeCallbackAppend(t *testing.T) {
	const marker = 0x11223344
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, reflectBulkAdd11, reflectBulkAdd23}, marker)
	array := reflect.ValueOf(record).Elem().Field(1)
	empty := array.Slice(0, 0)
	if s := reflect.Append(empty); s.Pointer() != empty.Pointer() || s.Len() != 0 || s.Cap() != 3 {
		t.Fatal("empty append changed the slice")
	}

	s := reflect.Append(empty, reflect.ValueOf(reflectBulkCallback(reflectBulkAdd23)))
	checkReflectBulkRecord(t, record, [3]int32{28, 16, 28}, marker)
	s = reflect.AppendSlice(s, array.Slice(0, 2))
	if s.Pointer() != empty.Pointer() || s.Len() != 3 || empty.Len() != 0 {
		t.Fatal("in-capacity append changed the backing array or the original length")
	}
	checkReflectBulkRecord(t, record, [3]int32{28, 28, 16}, marker)

	grown := reflect.AppendSlice(s, s.Slice(1, 3))
	callbacks := grown.Interface().([]reflectBulkCallback)
	if grown.Pointer() == s.Pointer() || len(callbacks) != 5 {
		t.Fatal("append past capacity did not grow the slice")
	}
	for i, want := range []int32{28, 28, 16, 28, 16} {
		checkReflectBulkCallback(t, i, callbacks[i], want)
	}
	grown.Clear()
	checkReflectBulkRecord(t, record, [3]int32{28, 28, 16}, marker)
}

//llgo:cdecl
type reflectNativeCallbackPair [2]reflectBulkCallback

func TestReflectNativeCallbackPairGrow(t *testing.T) {
	rows := [4]reflectNativeCallbackPair{
		{reflectBulkAdd7}, {reflectBulkAdd11}, {reflectBulkAdd23}, {reflectBulkAdd7},
	}
	s := rows[:1:2]
	reflect.ValueOf(&s).Elem().Grow(2)
	if len(s) != 1 || cap(s) < 3 {
		t.Fatalf("Grow: len=%d cap=%d", len(s), cap(s))
	}
	s = s[:3]
	checkReflectBulkCallback(t, 0, s[0][0], 12)
	checkReflectBulkCallback(t, 1, s[1][0], 16)
	checkReflectBulkCallback(t, 2, s[2][0], 0)

	v := reflect.ValueOf(s)
	reflect.Copy(v.Slice(2, 3), v.Slice(1, 2))
	v.Slice(1, 2).Clear()
	checkReflectBulkCallback(t, 0, s[0][0], 12)
	checkReflectBulkCallback(t, 1, s[1][0], 0)
	checkReflectBulkCallback(t, 2, s[2][0], 16)
	checkReflectBulkCallback(t, 1, rows[1][0], 16)

	full := reflect.ValueOf(&rows).Elem().Slice3(0, 2, 2)
	appended := reflect.AppendSlice(full, v.Slice(2, 3)).Interface().([]reflectNativeCallbackPair)
	for i, want := range []int32{12, 16, 16} {
		checkReflectBulkCallback(t, i, appended[i][0], want)
	}
}

func TestReflectNativeCallbackSliceToArray(t *testing.T) {
	record := newReflectBulkRecord([3]reflectBulkCallback{reflectBulkAdd7, reflectBulkAdd11}, 17)
	field := reflect.ValueOf(record).Elem().Field(1)
	type callbackSlice []reflectBulkCallback
	packed := reflect.Append(field.Slice(0, 2)).Convert(reflect.TypeFor[callbackSlice]())
	ordinary := []reflectBulkCallback{reflectBulkAdd7, reflectBulkAdd11}
	different := unsafe.Sizeof(reflectNativeCallbackPair{}) != unsafe.Sizeof([2]reflectBulkCallback{})
	for _, tc := range []struct {
		name   string
		src    reflect.Value
		native bool
	}{
		{"record", packed, true},
		{"Go slice", reflect.ValueOf(ordinary), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, dst := range []reflect.Type{reflect.TypeFor[*[2]reflectBulkCallback](), reflect.TypeFor[*reflectNativeCallbackPair]()} {
				native := dst.Elem() == reflect.TypeFor[reflectNativeCallbackPair]()
				snapshot := tc.src.Convert(dst.Elem())
				checkReflectBulkCallback(t, 1, snapshot.Index(1).Interface().(reflectBulkCallback), 16)
				switch a := snapshot.Interface().(type) {
				case [2]reflectBulkCallback:
					checkReflectBulkCallback(t, 1, a[1], 16)
				case reflectNativeCallbackPair:
					checkReflectBulkCallback(t, 1, a[1], 16)
				default:
					t.Fatalf("unexpected array snapshot %T", a)
				}
				if different && native != tc.native {
					if tc.src.CanConvert(dst) {
						t.Fatalf("CanConvert accepted incompatible storage for %v", dst)
					}
					expectSliceToArrayPanic(t, "reflect: cannot convert slice to array pointer with a different callback layout", func() { tc.src.Convert(dst) })
					continue
				}
				if !tc.src.CanConvert(dst) {
					t.Fatalf("CanConvert rejected compatible storage for %v", dst)
				}
				ptr := tc.src.Convert(dst)
				if ptr.Pointer() != tc.src.Pointer() {
					t.Fatal("array pointer did not share the slice")
				}
				switch a := ptr.Interface().(type) {
				case *[2]reflectBulkCallback:
					checkReflectBulkCallback(t, 1, a[1], 16)
					a[1] = reflectBulkAdd23
				case *reflectNativeCallbackPair:
					checkReflectBulkCallback(t, 1, a[1], 16)
					a[1] = reflectBulkAdd23
				default:
					t.Fatalf("unexpected array pointer %T", a)
				}
				checkReflectBulkCallback(t, 1, tc.src.Index(1).Interface().(reflectBulkCallback), 28)
				tc.src.Index(1).Set(reflect.ValueOf(reflectBulkCallback(reflectBulkAdd11)))
			}
		})
	}
	zero := reflect.TypeFor[*[0]reflectBulkCallback]()
	for _, s := range []reflect.Value{packed.Slice(0, 0), reflect.ValueOf([]reflectBulkCallback(nil))} {
		if !s.CanConvert(zero) || s.Convert(zero).Pointer() != s.Pointer() {
			t.Fatal("zero-length array pointer lost the slice pointer")
		}
	}
	rows := []reflectNativeCallbackPair{{}, {reflectBulkAdd7, reflectBulkAdd11}}
	last := reflect.ValueOf(rows).Slice(1, 2)
	target := reflect.TypeFor[[1]reflectNativeCallbackPair]()
	ptr := last.Convert(reflect.PointerTo(target))
	if ptr.Interface().(*[1]reflectNativeCallbackPair) != (*[1]reflectNativeCallbackPair)(rows[1:]) {
		t.Fatal("native array element lost its stride")
	}
	checkReflectBulkCallback(t, 1, ptr.Elem().Index(0).Index(1).Interface().(reflectBulkCallback), 16)
	checkReflectBulkCallback(t, 1, last.Convert(target).Interface().([1]reflectNativeCallbackPair)[0][1], 16)
	checkReflectBulkRecord(t, record, [3]int32{12, 16, 0}, 17)
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

func TestReflectNativeCallbackSlotAddr(t *testing.T) {
	var ordinary reflectBulkCallback = reflectBulkAdd7
	goAddr := reflect.ValueOf(&ordinary).Elem().Addr()
	if !goAddr.CanInterface() || goAddr.Interface().(*reflectBulkCallback) != &ordinary {
		t.Fatal("ordinary callback pointer did not share its Go slot")
	}
	record := reflectBulkSingleRecord{Callback: reflectBulkAdd7, Callbacks: [1]reflectBulkCallback{reflectBulkAdd7}}
	v := reflect.ValueOf(&record).Elem()
	typ := reflect.TypeFor[*reflectBulkCallback]()
	for _, slot := range []reflect.Value{v.Field(0), v.Field(1).Index(0)} {
		addr := slot.Addr()
		if addr.Type() != typ || !addr.CanConvert(typ) {
			t.Fatal("callback slot lost its pointer type")
		}
		same := addr.Convert(typ)
		checkReflectBulkCallback(t, 0, same.Elem().Interface().(reflectBulkCallback), 12)
		if unsafe.Sizeof(reflectBulkNativeArray{}) == unsafe.Sizeof([1]reflectBulkCallback{}) {
			p, ok := reflect.TypeAssert[*reflectBulkCallback](same)
			if !same.CanInterface() || !ok || reflect.ValueOf(p).Pointer() != addr.Pointer() {
				t.Fatal("compatible callback pointer did not share the slot")
			}
			continue
		}
		box := reflect.TypeFor[any]()
		if addr.CanInterface() || same.CanInterface() || addr.CanConvert(box) {
			t.Fatal("packed callback address is exportable as a Go pointer")
		}
		const want = "reflect: cannot use a Go pointer to a packed callback"
		expectSliceToArrayPanic(t, want, func() { same.Interface() })
		expectSliceToArrayPanic(t, want, func() { reflect.TypeAssert[*reflectBulkCallback](addr) })
		expectSliceToArrayPanic(t, want, func() { reflect.TypeAssert[int](addr) })
		expectSliceToArrayPanic(t, want, func() { reflect.New(typ).Elem().Set(addr) })
		expectSliceToArrayPanic(t, want, func() { addr.Convert(box) })
		expectSliceToArrayPanic(t, want, func() { reflect.ValueOf(func(*reflectBulkCallback) {}).Call([]reflect.Value{addr}) })
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

//llgo:cdecl
type reflectBulkNamedIdentity func(reflectBulkCallbackAlias) reflectBulkCallbackAlias

func reflectBulkNativeIdentity(f reflectBulkCallbackAlias) reflectBulkCallbackAlias { return f }

func TestReflectNativeCallbackRoundTrip(t *testing.T) {
	holder := struct{ F reflectBulkCallbackIdentity }{reflectBulkNativeIdentity}
	entry := reflect.ValueOf(&holder).Elem().Field(0)
	want := entry.UnsafePointer()
	public := entry.Convert(entry.Type())
	holder.F = nil
	// Copy the public closure into ordinary Go storage before converting back.
	slot := reflect.New(public.Type()).Elem()
	slot.Set(public)
	raw := reflect.TypeOf([1]reflectBulkCallbackIdentity{}).Elem()
	for _, v := range []reflect.Value{public, slot, reflect.ValueOf(slot.Interface())} {
		for _, target := range []reflect.Type{raw, reflect.TypeFor[reflectBulkNamedIdentity]()} {
			if !v.CanConvert(target) || v.Convert(target).UnsafePointer() != want {
				t.Fatalf("conversion to %v lost the original native callback entry", target)
			}
			r := reflect.New(target).Elem()
			r.Set(v)
			if r.UnsafePointer() != want {
				t.Fatalf("assignment to %v lost the original native callback entry", target)
			}
		}
		entry.Set(v)
		if entry.UnsafePointer() != want || holder.F(reflectBulkAdd7)(5) != 12 {
			t.Fatal("assignment lost the original native callback")
		}
	}
}

func TestReflectCanConvertNativeCallback(t *testing.T) {
	dst := reflect.TypeFor[reflectBulkCallback]()
	n := int32(42)
	if reflect.ValueOf(func(x int32) int32 { return x + n }).CanConvert(dst) {
		t.Fatal("CanConvert accepted a captured closure as a native callback")
	}
	for _, f := range []func(int32) int32{nil, reflectBulkAdd7} {
		v := reflect.ValueOf(f)
		if !v.CanConvert(dst) || v.Convert(dst).IsNil() != (f == nil) {
			t.Fatal("direct native callback conversion failed")
		}
	}
}

type reflectCallbackReceiver int32

func (r reflectCallbackReceiver) Apply(v int32) int32 { return int32(r) + v }

func TestReflectCanConvertCallbackMethod(t *testing.T) {
	var receiver interface{ Apply(int32) int32 } = reflectCallbackReceiver(7)
	m := reflect.ValueOf(&receiver).Elem().Method(0)
	dst := reflect.TypeFor[reflectBulkCallback]()
	if m.CanConvert(dst) {
		out := m.Convert(dst).Call([]reflect.Value{reflect.ValueOf(int32(5))})
		if out[0].Int() != 12 {
			t.Fatal("converted method lost its receiver")
		}
	}
	receiver = nil
	if m.CanConvert(dst) {
		t.Fatal("CanConvert accepted a method on a nil interface")
	}
}
