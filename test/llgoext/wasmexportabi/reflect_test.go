//go:build llgo && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package wasmexportabi

import (
	"reflect"
	"testing"
	"unsafe"
)

const wideReflectWord = uintptr(1<<40 | 515)

func wordText(n uintptr) string {
	switch n {
	case 515:
		return "small"
	case wideReflectWord:
		return "wide"
	}
	return "wrong argument"
}

type wordReceiver struct{}

func (*wordReceiver) Word(n uintptr) string { return wordText(n) }

//llgo:cdecl
type nativeWordText func(uintptr) string

//llgo:cdecl
type nativeWordRecord func(uintptr) reflectTextRecord

//go:linkname nativeTextEntry C.llgo_test_reflect_text
func nativeTextEntry() nativeWordText

//go:linkname nativeRecordEntry C.llgo_test_reflect_tagged
func nativeRecordEntry() nativeWordRecord

func TestWasmReflectCallWord(t *testing.T) {
	delimiter := t.Name()
	captured := func(n uintptr) string { return delimiter + wordText(n) }
	returned := reflect.ValueOf(func() func(uintptr) string { return captured }).Call(nil)[0]
	if returned.CanAddr() || returned.CanSet() {
		t.Fatal("returned Go closure is addressable or settable")
	}
	var receiver *wordReceiver
	made := reflect.MakeFunc(reflect.TypeOf(wordText), func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(wordText(uintptr(in[0].Uint())))}
	})
	for _, n := range []uintptr{515, wideReflectWord} {
		args := []reflect.Value{reflect.ValueOf(n)}
		for _, tc := range []struct {
			name string
			fn   reflect.Value
			want string
		}{
			{"Go function", reflect.ValueOf(wordText), wordText(n)},
			{"capturing closure", reflect.ValueOf(captured), delimiter + wordText(n)},
			{"returned Go closure", returned, delimiter + wordText(n)},
			{"method with nil receiver", reflect.ValueOf(receiver).MethodByName("Word"), wordText(n)},
			{"MakeFunc", made, wordText(n)},
		} {
			if got := tc.fn.Call(args)[0].String(); got != tc.want {
				t.Errorf("%s(%#x) returned %q, want %q", tc.name, n, got, tc.want)
			}
		}
	}
	if got := made.Interface().(func(uintptr) string)(wideReflectWord); got != "wide" {
		t.Fatalf("typed MakeFunc call returned %q", got)
	}
}

func TestWasmReflectNativeResults(t *testing.T) {
	args := []reflect.Value{reflect.ValueOf(uintptr(515))}
	text := reflect.ValueOf(nativeTextEntry()).Call(args)[0].Interface().(string)
	if text != "word" || !stringPointerSlotOK(&text) {
		t.Errorf("unexpected native string or pointer padding: %q", text)
	}
	// On Memory32 this record uses the string header's padding for a live field.
	r := reflect.ValueOf(nativeRecordEntry()).Call(args)[0].Interface().(reflectTextRecord)
	if r.Slot.Tag != 0x12345678 || r.Len != 4 || r.Slot.Data == nil || *r.Slot.Data != 'w' {
		t.Errorf("unexpected native record: %+v", r)
	}
}

func TestWasmReflectNativeWordRange(t *testing.T) {
	if unsafe.Offsetof(reflectSlot{}.Tag) == 4 {
		defer func() {
			err, ok := recover().(error)
			if !ok || err.Error() != "runtime error: WebAssembly ABI integer conversion out of range" {
				t.Errorf("passing %#x to C uintptr_t panicked with %v", wideReflectWord, err)
			}
		}()
	}
	args := []reflect.Value{reflect.ValueOf(wideReflectWord)}
	if got := reflect.ValueOf(nativeTextEntry()).Call(args)[0].String(); got != "wide" {
		t.Errorf("C received the wrong uintptr_t, returned %q", got)
	}
}

func TestWasmReflectResultSlots(t *testing.T) {
	type pair struct{ X, Y int64 }
	args := []reflect.Value{reflect.ValueOf(uintptr(1))}
	record := reflect.ValueOf(func(uintptr) pair { return pair{3, 5} }).Call(args)
	tuple := reflect.ValueOf(func(uintptr) (int64, int64) { return 7, 11 }).Call(args)
	if got := record[0].Interface().(pair); got != (pair{3, 5}) {
		t.Errorf("record result = %+v", got)
	}
	if x, y := tuple[0].Int(), tuple[1].Int(); x != 7 || y != 11 {
		t.Errorf("tuple results = (%d, %d)", x, y)
	}
}

func TestWasmReflectNativeCallbackResult(t *testing.T) {
	native := nativeTextEntry()
	direct := reflect.ValueOf(native)
	callback := reflect.ValueOf(nativeTextEntry).Call(nil)[0]
	if callback.CanAddr() || callback.CanSet() {
		t.Fatal("returned native callback is addressable or settable")
	}
	if callback.Type() != direct.Type() || callback.UnsafePointer() != direct.UnsafePointer() {
		t.Fatal("reflect.Call changed the native callback's type or code pointer")
	}
	if got := callback.Interface().(nativeWordText)(515); got != "word" {
		t.Fatalf("typed native callback returned %q", got)
	}
	goType := reflect.TypeFor[func(uintptr) string]()
	if got := callback.Convert(goType).Interface().(func(uintptr) string)(515); got != "word" {
		t.Fatalf("converted Go callback returned %q", got)
	}
	args := []reflect.Value{reflect.ValueOf(uintptr(515))}
	if got := callback.Call(args)[0].String(); got != "word" {
		t.Fatalf("reflected native callback returned %q", got)
	}

	// Both directions of the MakeFunc frame carry a single native code pointer.
	identity := reflect.MakeFunc(reflect.TypeFor[func(nativeWordText) nativeWordText](),
		func(in []reflect.Value) []reflect.Value {
			if got := in[0].Call(args)[0].String(); got != "word" {
				t.Fatalf("MakeFunc argument returned %q", got)
			}
			return in
		})
	fn := identity.Interface().(func(nativeWordText) nativeWordText)
	if got := fn(native)(515); got != "word" {
		t.Fatalf("MakeFunc result returned %q", got)
	}
}

func TestWasmReflectCallbackStorage(t *testing.T) {
	captured := t.Name()
	key := reflect.ValueOf(1)
	args := []reflect.Value{reflect.ValueOf(uintptr(515))}
	for _, tc := range []struct {
		fn   reflect.Value
		want string
	}{
		{reflect.ValueOf(nativeTextEntry).Call(nil)[0], "word"},
		{reflect.ValueOf(func(uintptr) string { return captured }), captured},
	} {
		typ := tc.fn.Type()
		m := reflect.MakeMap(reflect.MapOf(key.Type(), typ))
		m.SetMapIndex(key, tc.fn)
		record := reflect.StructOf([]reflect.StructField{{Name: "Callback", Type: typ}})
		field := reflect.New(record).Elem().Field(0)
		field.Set(tc.fn)
		for _, v := range []reflect.Value{tc.fn.Convert(typ), m.MapIndex(key), field} {
			if v.Type() != typ || v.UnsafePointer() != tc.fn.UnsafePointer() {
				t.Fatalf("reflection changed the type or code pointer of %v", typ)
			}
			if got := v.Call(args)[0].String(); got != tc.want {
				t.Fatalf("%v returned %q, want %q", typ, got, tc.want)
			}
		}
	}
}

//llgo:cdecl
type nativeInt64Callback func(int64) int64

func TestWasmReflectConvertMadeNativeFunc(t *testing.T) {
	typ := reflect.TypeFor[nativeInt64Callback]()
	var calls int
	made := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		calls++
		return in
	})
	args := []reflect.Value{reflect.ValueOf(int64(17))}
	for _, fn := range []reflect.Value{made, made.Convert(typ)} {
		if got := fn.Call(args)[0].Int(); got != 17 {
			t.Fatalf("callback returned %d, want 17", got)
		}
	}
	if calls != 2 {
		t.Fatalf("callback invoked %d times, want 2", calls)
	}
}

func nextCallbackValue(n int64) int64 { return n + 1 }

func TestWasmReflectConvertGoCallback(t *testing.T) {
	type goCallback func(int64) int64
	typ := reflect.TypeFor[nativeInt64Callback]()
	for _, tc := range []struct {
		name    string
		fn      any
		wantNil bool
	}{
		{"direct", nextCallbackValue, false},
		{"named", goCallback(nextCallbackValue), false},
		{"nil", (func(int64) int64)(nil), true},
		{"named nil", goCallback(nil), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := reflect.ValueOf(tc.fn).Convert(typ)
			fn, ok := value.Interface().(nativeInt64Callback)
			if !ok || value.Type() != typ || (fn == nil) != tc.wantNil {
				t.Fatal("conversion lost the destination's native function representation")
			}
			if fn != nil && fn(41) != 42 {
				t.Fatal("converted native callback returned the wrong result")
			}
		})
	}
	fn := nextCallbackValue
	converted := reflect.ValueOf(&fn).Elem().Convert(typ)
	fn = nil
	if got := converted.Interface().(nativeInt64Callback)(41); got != 42 {
		t.Fatalf("conversion retained the mutable source slot: got %d", got)
	}
}

func TestWasmReflectConvertGoCallbackConstraints(t *testing.T) {
	check := func(name string, fn any, typ reflect.Type, want string) {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != want {
					t.Fatalf("conversion panicked with %v, want %q", got, want)
				}
			}()
			reflect.ValueOf(fn).Convert(typ)
		})
	}
	offset := int64(len(t.Name()))
	check("capture", func(n int64) int64 { return n + offset }, reflect.TypeFor[nativeInt64Callback](),
		"reflect.Value.Convert: C callback must not capture variables")
	const incompatibleABI = "reflect.Value.Convert: C callback with a different ABI must be a direct function reference"
	check("variadic", func(...byte) {}, reflect.TypeFor[sliceSink](), incompatibleABI)
	if unsafe.Offsetof(reflectSlot{}.Tag) == 4 {
		check("ABI", wordText, reflect.TypeFor[nativeWordText](), incompatibleABI)
	}
}

func TestWasmReflectSetGoCallback(t *testing.T) {
	var fn nativeInt64Callback
	slot := reflect.ValueOf(&fn).Elem()
	slot.Set(reflect.ValueOf(nextCallbackValue))
	if got := fn(41); got != 42 {
		t.Fatalf("native callback returned %d, want 42", got)
	}
	var zero func(int64) int64
	slot.Set(reflect.ValueOf(zero))
	if fn != nil {
		t.Fatal("setting a nil function produced a non-nil native callback")
	}
	defer func() {
		if got := recover(); got != "reflect.Set: C callback must not capture variables" {
			t.Fatalf("setting a capturing closure panicked with %v", got)
		}
	}()
	offset := int64(len(t.Name()))
	slot.Set(reflect.ValueOf(func(n int64) int64 { return n + offset }))
}

func TestWasmReflectSetGoCallbackIncompatibleABI(t *testing.T) {
	if unsafe.Offsetof(reflectSlot{}.Tag) != 4 {
		t.Skip("requires Memory32")
	}
	defer func() {
		if got := recover(); got != "reflect.Set: C callback with a different ABI must be a direct function reference" {
			t.Fatalf("setting an incompatible Go uintptr entry panicked with %v", got)
		}
	}()
	var fn nativeWordText
	reflect.ValueOf(&fn).Elem().Set(reflect.ValueOf(wordText))
}

//llgo:cdecl
type aliasWordText = func(uintptr) string

func aliasTextEntry() aliasWordText { return aliasWordText(nativeTextEntry()) }

func TestWasmReflectNativeFuncAlias(t *testing.T) {
	native := reflect.ValueOf(aliasTextEntry).Call(nil)[0]
	typ := reflect.TypeOf(struct{ F aliasWordText }{}).Field(0).Type
	captured := t.Name()
	goFunc := reflect.ValueOf(func(n uintptr) string { return captured + wordText(n) })
	if typ != goFunc.Type() || typ != native.Type() {
		t.Fatal("function alias lost its public type identity")
	}
	if native.UnsafePointer() != reflect.ValueOf(nativeTextEntry()).UnsafePointer() ||
		native.Interface().(aliasWordText)(515) != "word" {
		t.Fatal("function alias lost its native callback")
	}
	dst := reflect.New(typ).Elem()
	dst.Set(goFunc)
	if got := dst.Interface().(func(uintptr) string)(wideReflectWord); got != captured+"wide" {
		t.Errorf("Go closure returned %q", got)
	}
	dst.Set(native)
	args := []reflect.Value{reflect.ValueOf(uintptr(515))}
	if got := dst.Call(args)[0].String(); got != "word" {
		t.Errorf("native callback in Go storage returned %q", got)
	}
	callGo := reflect.ValueOf(func(f func(uintptr) string) string { return f(515) })
	if got := callGo.Call([]reflect.Value{native})[0].String(); got != "word" {
		t.Errorf("native argument passed to a Go function returned %q", got)
	}
	var raw aliasWordText
	slot := reflect.ValueOf(&raw).Elem()
	slot.Set(dst)
	if slot.UnsafePointer() != native.UnsafePointer() {
		t.Fatal("round trip through Go storage changed the native entry")
	}
	converted := native.Convert(typ)
	if got := converted.Call(args)[0].String(); got != "word" {
		t.Errorf("converted native callback returned %q", got)
	}
	if got := converted.Interface().(func(uintptr) string)(515); got != "word" {
		t.Errorf("boxed Go callback returned %q", got)
	}

	// Elem preserves the raw alias descriptor, while TypeOf exposes its Go signature.
	rawType := reflect.TypeOf([1]aliasWordText{}).Elem()
	zero := reflect.Zero(rawType).Convert(typ)
	if fn := zero.Interface().(func(uintptr) string); fn != nil {
		t.Fatal("converting a nil alias produced a non-nil Go function")
	}
	for _, value := range []reflect.Value{converted, zero} {
		roundTrip := value.Convert(rawType)
		fn, ok := roundTrip.Interface().(aliasWordText)
		if !ok || (fn == nil) != value.IsNil() {
			t.Fatal("conversion back to the native alias lost its representation")
		}
		if fn != nil && (roundTrip.UnsafePointer() != native.UnsafePointer() || fn(515) != "word") {
			t.Fatal("conversion back to the native alias lost its original entry")
		}
	}
}

type aliasIdentity = func(aliasWordText) aliasWordText

//llgo:cdecl
type nativeAliasIdentity = func(aliasWordText) aliasWordText

type aliasReceiver struct{}

func (aliasReceiver) Echo(f aliasWordText) aliasWordText { return f }

func echoAliasCallback(f aliasWordText) aliasWordText { return f }

func TestWasmReflectHigherOrderAlias(t *testing.T) {
	expanded := reflect.TypeFor[func(func(uintptr) string) func(uintptr) string]()
	raw := reflect.ValueOf(echoAliasCallback)
	if raw.Type() != expanded || reflect.TypeFor[aliasIdentity]() != expanded {
		t.Fatal("higher-order function alias lost its Go type identity")
	}
	callback := reflect.ValueOf(aliasTextEntry).Call(nil)[0]
	out := raw.Call([]reflect.Value{callback})[0]
	if out.UnsafePointer() != callback.UnsafePointer() || out.Interface().(aliasWordText)(515) != "word" {
		t.Fatal("higher-order function changed the raw callback")
	}
	receiver := reflect.ValueOf(aliasReceiver{})
	method := receiver.MethodByName("Echo")
	if method.Type() != expanded || method.Interface().(aliasIdentity)(aliasTextEntry())(515) != "word" {
		t.Fatal("bound method lost its public type or native callback frame")
	}
	adapted := reflect.New(expanded).Elem()
	nativeIdentity := reflect.ValueOf(struct{ F nativeAliasIdentity }{F: nativeAliasIdentity(echoAliasCallback)}).Field(0)
	for _, fn := range []reflect.Value{raw, nativeIdentity} {
		adapted.Set(fn)
		out = adapted.Call([]reflect.Value{callback})[0]
		if got := out.Interface().(func(uintptr) string)(515); got != "word" {
			t.Fatalf("adapted higher-order function returned %q", got)
		}
	}
}

//llgo:cdecl
type nativeCallbackRecord struct {
	Callback nativeInt64Callback
	Tag      uint32
	Matrix   [2][2]nativeInt64Callback
	End      uint32
}

//llgo:cdecl
type nativeCallbackBox struct{ Callback nativeInt64Callback }

func TestWasmReflectNativeCallbackField(t *testing.T) {
	const tag = 0x12345678
	r := nativeCallbackRecord{Tag: tag, End: tag}
	s := reflect.ValueOf(&r).Elem()
	field := s.Field(0)
	index := s.Field(2).Slice3(1, 2, 2).Index(0).Slice(1, 2).Index(0).Addr().Elem()
	native := nativeInt64Callback(nextCallbackValue)
	// A separately boxed typed pointer carries no record provenance.
	reflect.ValueOf(&r.Callback).Elem().Set(reflect.ValueOf(nextCallbackValue))
	if r.Tag != tag || r.Callback(41) != 42 {
		t.Fatalf("setting a callback through its pointer overwrote the adjacent tag: %#x", r.Tag)
	}
	for _, value := range []reflect.Value{reflect.ValueOf(nextCallbackValue), reflect.ValueOf(&native).Elem(), reflect.ValueOf(nativeCallbackBox{native}).Field(0)} {
		field.Set(value)
		index.Set(field)
		if r.Tag != tag || r.End != tag || r.Callback(41) != 42 || r.Matrix[1][1](41) != 42 {
			t.Fatalf("setting native callbacks changed callback storage or an adjacent tag: %#x %#x", r.Tag, r.End)
		}
		field.SetZero()
		if r.Tag != tag || r.Callback != nil {
			t.Fatalf("zeroing the callback changed its adjacent tag or retained the callback: %#x", r.Tag)
		}
	}
	boxed := reflect.ValueOf(&r.Callback).Elem()
	boxed.Set(reflect.ValueOf(native))
	if r.Tag != tag || r.Callback(41) != 42 {
		t.Fatalf("setting a native pointer changed the adjacent tag: %#x", r.Tag)
	}
	boxed.SetZero()
	if r.Tag != tag || r.Callback != nil {
		t.Fatalf("zeroing a native pointer changed the adjacent tag: %#x", r.Tag)
	}
	matrix := s.Field(2)
	source := [2][2]nativeInt64Callback{{native, nil}, {nil, native}}
	matrix.Set(reflect.ValueOf(source))
	if r.End != tag || r.Matrix[0][0](41) != 42 || r.Matrix[0][1] != nil ||
		r.Matrix[1][0] != nil || r.Matrix[1][1](41) != 42 {
		t.Fatal("setting a native callback array changed its entries or adjacent field")
	}
	reflect.ValueOf(&source).Elem().Set(matrix)
	if source[0][0](41) != 42 || source[0][1] != nil || source[1][0] != nil || source[1][1](41) != 42 {
		t.Fatal("copying a native callback array lost a callback")
	}
	if unsafe.Offsetof(r.Tag) == 4 {
		var ordinary struct {
			Callback nativeInt64Callback
			Tag      uint32
		}
		upper := (*uint32)(unsafe.Add(unsafe.Pointer(&ordinary.Callback), 4))
		*upper = tag
		pointer := reflect.ValueOf(&ordinary.Callback).Elem()
		pointer.Set(reflect.ValueOf(native))
		pointer.SetZero()
		if *upper != tag || ordinary.Callback != nil {
			t.Fatal("a separately boxed Go pointer changed the padding or retained the callback")
		}
		reflect.ValueOf(&ordinary).Elem().Field(0).Set(index)
		if *upper != 0 || ordinary.Callback(41) != 42 {
			t.Fatalf("Go callback storage retained malformed upper bits: %#x", *upper)
		}
	}
	matrix.SetZero()
	if r.End != tag || r.Matrix[0][0] != nil || r.Matrix[1][1] != nil {
		t.Fatal("zeroing a native callback array changed its adjacent field or retained a callback")
	}
}
