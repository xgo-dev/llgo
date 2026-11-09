//go:build llgo && !baremetal

package gotest

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"structs"
)

type explicitlyAligned[A any] struct {
	_       A
	Value   uint64
	Pointer *int
}

type aligned64 = explicitlyAligned[structs.Align64]

var explicitGlobal aligned64
var explicitZeroGlobal [0]aligned64

//go:noinline
func explicitStackAlignment() uintptr {
	var value aligned64
	return uintptr(unsafe.Pointer(&value)) & 63
}

//go:noinline
func explicitAlignedIdentity(value aligned64) aligned64 { return value }

type explicitLarge struct {
	_      structs.Align64
	Values [8193]uint64
}

//go:noinline
func explicitLargeResult(value uint64) (byte, explicitLarge) {
	var out explicitLarge
	out.Values[0], out.Values[8192] = value, value+1
	return 7, out
}

func checkExplicitAlignment[A any](t *testing.T, alignment uintptr) {
	t.Helper()
	typ := reflect.TypeFor[explicitlyAligned[A]]()
	var value explicitlyAligned[A]
	if unsafe.Alignof(value) != alignment || typ.Align() != int(alignment) || typ.FieldAlign() != int(alignment) || typ.Size() != unsafe.Sizeof(value) || typ.Field(2).Offset != unsafe.Offsetof(value.Pointer) {
		t.Fatalf("source/reflection layout mismatch for %s", typ)
	}
	check := func(p unsafe.Pointer) {
		t.Helper()
		if uintptr(p)&(alignment-1) != 0 {
			t.Fatalf("%s address %p does not satisfy alignment %d", typ, p, alignment)
		}
	}
	p := new(explicitlyAligned[A])
	check(unsafe.Pointer(p))
	n := new(int)
	*n = 37
	p.Value, p.Pointer = 29, n
	items := make([]explicitlyAligned[A], 1, 1)
	items[0] = *p
	for i := 0; i < 70; i++ {
		items = append(items, *p)
	}
	var array [3]explicitlyAligned[A]
	for i := range array {
		check(unsafe.Pointer(&array[i]))
	}
	boxed := any(*p)
	// A multi-field struct is boxed indirectly; verify the box itself, not a
	// freshly aligned local copy obtained by a type assertion.
	box := reflect.ValueOf(&boxed).Elem().InterfaceData()
	if box[1]&(alignment-1) != 0 {
		t.Fatal("interface box alignment")
	}
	ch := make(chan explicitlyAligned[A], 2)
	ch <- *p
	reflected := reflect.New(typ)
	check(reflected.UnsafePointer())
	reflected.Elem().Set(reflect.ValueOf(*p))
	rs := reflect.MakeSlice(reflect.SliceOf(typ), 0, 1)
	for i := 0; i < 20; i++ {
		rs = reflect.Append(rs, reflected.Elem())
	}
	check(rs.Index(0).Addr().UnsafePointer())
	rc := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 1)
	rc.Send(reflected.Elem())
	runtime.GC()
	for i := range items {
		check(unsafe.Pointer(&items[i]))
		if items[i].Value != 29 || *items[i].Pointer != 37 {
			t.Fatal("slice/GC contents")
		}
		if i > 0 && uintptr(unsafe.Pointer(&items[i]))-uintptr(unsafe.Pointer(&items[i-1])) != unsafe.Sizeof(value) {
			t.Fatal("slice stride")
		}
	}
	if got := <-ch; got.Value != 29 || *got.Pointer != 37 {
		t.Fatal("channel contents")
	}
	got, ok := rc.Recv()
	if !ok || got.FieldByName("Value").Uint() != 29 || got.FieldByName("Pointer").Elem().Int() != 37 {
		t.Fatal("reflected channel contents")
	}
	if got := boxed.(explicitlyAligned[A]); got.Value != 29 || *got.Pointer != 37 {
		t.Fatal("interface contents")
	}
	reflected.Elem().FieldByName("Pointer").Elem().SetInt(41)
	if *p.Pointer != 41 {
		t.Fatal("reflected field access")
	}
	runtime.KeepAlive(items)
}

func TestExplicitAlignmentStorage(t *testing.T) {
	checkExplicitAlignment[structs.Align8](t, 8)
	checkExplicitAlignment[structs.Align16](t, 16)
	checkExplicitAlignment[structs.Align32](t, 32)
	checkExplicitAlignment[structs.Align64](t, 64)
	if explicitStackAlignment() != 0 || uintptr(unsafe.Pointer(&explicitGlobal))&63 != 0 || uintptr(unsafe.Pointer(&explicitZeroGlobal))&63 != 0 || uintptr(unsafe.Pointer(new([0]aligned64)))&63 != 0 {
		t.Fatal("stack/global/zero-sized storage alignment")
	}
	if got := explicitAlignedIdentity(aligned64{Value: 91}); got.Value != 91 {
		t.Fatal("Go by-value call")
	}
	if (aligned64{Value: 1}) == (aligned64{Value: 2}) {
		t.Fatal("aligned value comparison")
	}
	// Eight-byte alignment remains supported in map values and comparable keys.
	key := explicitlyAligned[structs.Align8]{Value: 71}
	m := map[explicitlyAligned[structs.Align8]]explicitlyAligned[structs.Align8]{key: key}
	if m[key] != key {
		t.Fatal("eight-byte aligned map storage")
	}
}

// Run with -goroutine-stack-size=1MB on fixed-stack Wasm targets.
func TestExplicitAlignmentLargeCopies(t *testing.T) {
	// Exercise late aggregate-copy and large-result lowering, plus a deferred
	// by-value capture. These compiler-created copies must preserve the value.
	flag, large := explicitLargeResult(83)
	if flag != 7 || large.Values[0] != 83 || large.Values[8192] != 84 || uintptr(unsafe.Pointer(&large))&63 != 0 {
		t.Fatal("large aligned return")
	}
	copy := large
	large.Values[0] = 99
	defer func(v explicitLarge) {
		if v.Values[0] != 83 || v.Values[8192] != 84 || uintptr(unsafe.Pointer(&v))&63 != 0 {
			t.Fatal("aligned defer capture")
		}
	}(copy)
	copy.Values[8192] = 101
}

func TestExplicitAlignmentReflection(t *testing.T) {
	typ := reflect.StructOf([]reflect.StructField{
		{Name: "Alignment", Type: reflect.TypeFor[structs.Align64]()},
		{Name: "Data", Type: reflect.TypeFor[[24]byte]()},
	})
	if typ.Align() != 64 || typ.Size() != 64 || typ.Field(1).Offset != 0 {
		t.Fatal("dynamic struct layout")
	}
	p := reflect.New(typ)
	if p.Pointer()&63 != 0 {
		t.Fatal("dynamic struct allocation")
	}
	p.Elem().Field(1).Index(23).SetUint(17)
	if p.Elem().Field(1).Index(23).Uint() != 17 {
		t.Fatal("dynamic struct write")
	}
	for _, tc := range []struct {
		name    string
		f       func()
		message string
	}{
		{"map-value", func() { reflect.MapOf(reflect.TypeFor[int](), reflect.TypeFor[aligned64]()) }, "alignment above 8 bytes"},
		{"map-key", func() { reflect.MapOf(reflect.TypeFor[aligned64](), reflect.TypeFor[int]()) }, "alignment above 8 bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if v := recover(); v == nil || !strings.Contains(fmt.Sprint(v), tc.message) {
					t.Fatalf("diagnostic: %v", v)
				}
			}()
			tc.f()
		})
	}
	// Pointers preserve the ordinary calling convention and map representation.
	x := aligned64{Value: 23}
	got := reflect.ValueOf(func(p *aligned64) uint64 { return p.Value }).Call([]reflect.Value{reflect.ValueOf(&x)})
	if got[0].Uint() != 23 {
		t.Fatal("reflection pointer call")
	}
	reflect.MapOf(reflect.TypeFor[int](), reflect.TypeFor[*aligned64]())
}
