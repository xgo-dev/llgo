//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"reflect"
	"runtime"
	"simd/archsimd"
	"testing"
	"unsafe"
)

type nativeVectorBox struct {
	Tag   byte
	Value archsimd.Float32x4
	Tail  byte
}
type definedVector archsimd.Float32x4

//llgo:cdecl
type nativeCallbackBox struct {
	Tag   uint64
	Value archsimd.Float32x4
	Fn    func(uint64) uint64
	Tail  uint64
}

func TestSIMDNativeCallbackFieldReflection(t *testing.T) {
	holder := struct {
		Box      nativeCallbackBox
		Sentinel uint64
	}{Sentinel: 77}
	if unsafe.Sizeof(holder.Box) != 48 || unsafe.Offsetof(holder.Box.Tail) != 40 || reflect.TypeFor[nativeCallbackBox]().Field(3).Offset != 40 {
		t.Fatal("native callback field changed source/reflection offsets")
	}
	reflect.ValueOf(&holder.Box).Elem().FieldByName("Tail").SetUint(99)
	if holder.Box.Tail != 99 || holder.Sentinel != 77 {
		t.Fatalf("reflection wrote outside Tail: tail=%d sentinel=%d", holder.Box.Tail, holder.Sentinel)
	}
}

var nativeVectorGlobal nativeVectorBox
var nativeVectorHeap []*nativeVectorBox
var nativeVectorZero [0]archsimd.Float32x4
var nativeVectorZeroHeap = new([0]archsimd.Float32x4)

//go:noinline
func nativeVectorStack() uintptr {
	var x nativeVectorBox
	return uintptr(unsafe.Pointer(&x.Value)) & 15
}

func TestSIMDNativeLayout(t *testing.T) {
	if uintptr(unsafe.Pointer(&nativeVectorZero))&15 != 0 || uintptr(unsafe.Pointer(nativeVectorZeroHeap))&15 != 0 {
		t.Fatal("zero-sized vector storage alignment")
	}
	var v archsimd.Float32x4
	var b nativeVectorBox
	if unsafe.Alignof(v) != 16 || unsafe.Sizeof(v) != 16 || unsafe.Alignof(b) != 16 || unsafe.Sizeof(b) != 48 || unsafe.Offsetof(b.Value) != 16 || unsafe.Offsetof(b.Tail) != 32 {
		t.Fatalf("source layout: vector=%d/%d box=%d/%d offsets=%d/%d", unsafe.Sizeof(v), unsafe.Alignof(v), unsafe.Sizeof(b), unsafe.Alignof(b), unsafe.Offsetof(b.Value), unsafe.Offsetof(b.Tail))
	}
	if nativeVectorStack() != 0 || uintptr(unsafe.Pointer(&nativeVectorGlobal.Value))&15 != 0 {
		t.Fatal("stack/global alignment")
	}
	var zeroTail struct {
		Tag  byte
		Tail [0]archsimd.Float32x4
	}
	if unsafe.Sizeof(zeroTail) != 32 || unsafe.Alignof(zeroTail) != 16 || unsafe.Offsetof(zeroTail.Tail) != 16 || reflect.TypeOf(zeroTail).Size() != 32 {
		t.Fatal("zero-sized SIMD tail layout")
	}
	rt := reflect.TypeOf(b)
	if rt.Align() != 16 || rt.FieldAlign() != 16 || rt.Size() != 48 || rt.Field(1).Offset != 16 || reflect.TypeOf(v).Align() != 16 {
		t.Fatal("reflection layout", rt)
	}
	// A new defined type and legal pointer conversion must preserve alignment.
	var d definedVector
	if unsafe.Alignof(d) != 16 || reflect.TypeOf(d).Align() != 16 {
		t.Fatal("defined type lost alignment")
	}
	ptr := (*archsimd.Float32x4)(new(definedVector))
	*ptr = archsimd.BroadcastFloat32x4(7)
	if uintptr(unsafe.Pointer(ptr))&15 != 0 {
		t.Fatal("converted heap pointer alignment")
	}
	nativeVectorHeap = nil
	for i := 0; i < 80; i++ {
		p := new(nativeVectorBox)
		p.Value = archsimd.BroadcastFloat32x4(float32(i))
		p.Tag, p.Tail = byte(i), byte(i+1)
		nativeVectorHeap = append(nativeVectorHeap, p)
	}
	runtime.GC()
	for i, p := range nativeVectorHeap {
		if uintptr(unsafe.Pointer(p))&15 != 0 || uintptr(unsafe.Pointer(&p.Value))&15 != 0 {
			t.Fatal("heap alignment", i)
		}
		var got [4]float32
		p.Value.StoreArray(&got)
		if got != [4]float32{float32(i), float32(i), float32(i), float32(i)} || p.Tag != byte(i) || p.Tail != byte(i+1) {
			t.Fatal("heap/GC value", i, got)
		}
	}
}

func TestSIMDNativeContainers(t *testing.T) {
	want := [4]float32{1, 2, 3, 4}
	v := archsimd.LoadFloat32x4Array(&want)
	check := func(x archsimd.Float32x4) {
		t.Helper()
		var got [4]float32
		x.StoreArray(&got)
		if got != want {
			t.Fatalf("lanes: %v", got)
		}
	}
	xs := make([]archsimd.Float32x4, 1, 1)
	xs[0] = v
	for i := 0; i < 100; i++ {
		xs = append(xs, v)
	}
	runtime.GC()
	for i := range xs {
		if uintptr(unsafe.Pointer(&xs[i]))&15 != 0 {
			t.Fatal("slice alignment", i)
		}
		check(xs[i])
	}
	check(any(v).(archsimd.Float32x4))
	ch := make(chan archsimd.Float32x4, 2)
	ch <- v
	runtime.GC()
	check(<-ch)
	typ := reflect.TypeOf(v)
	p := reflect.New(typ)
	if p.Pointer()&15 != 0 {
		t.Fatal("reflect.New alignment")
	}
	p.Elem().Set(reflect.ValueOf(v))
	check(p.Elem().Interface().(archsimd.Float32x4))
	rs := reflect.MakeSlice(reflect.SliceOf(typ), 1, 1)
	rs.Index(0).Set(reflect.ValueOf(v))
	for i := 0; i < 40; i++ {
		rs = reflect.Append(rs, reflect.ValueOf(v))
	}
	if rs.Pointer()&15 != 0 {
		t.Fatal("reflect slice alignment")
	}
	check(rs.Index(rs.Len() - 1).Interface().(archsimd.Float32x4))
	rc := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 1)
	rc.Send(reflect.ValueOf(v))
	r, _ := rc.Recv()
	check(r.Interface().(archsimd.Float32x4))
	dyn := reflect.StructOf([]reflect.StructField{{Name: "Tag", Type: reflect.TypeFor[byte]()}, {Name: "Value", Type: typ}})
	if dyn.Align() != 16 || dyn.Field(1).Offset != 16 || dyn.Size() != 32 || reflect.New(dyn).Pointer()&15 != 0 {
		t.Fatal("reflect.StructOf layout")
	}
	for _, element := range []reflect.Type{typ, reflect.TypeFor[nativeVectorBox](), reflect.ArrayOf(0, typ)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("reflect.MapOf accepted SIMD storage", element)
				}
			}()
			reflect.MapOf(reflect.TypeFor[int](), element)
		}()
	}
	m := map[int]*archsimd.Float32x4{1: &xs[0]}
	check(*m[1])
}

//go:noinline
func nativeVectorClosure(v archsimd.Float32x4) func() archsimd.Float32x4 {
	return func() archsimd.Float32x4 { return v }
}

//go:noinline
func nativeVectorDeferred(v archsimd.Float32x4, dst *[4]float32) {
	defer func(x archsimd.Float32x4) { x.StoreArray(dst) }(v)
	runtime.GC()
}

func TestSIMDNativeCapturedStorage(t *testing.T) {
	v := archsimd.LoadFloat32x4Array(&[4]float32{1, 2, 3, 4})
	fn := nativeVectorClosure(v)
	runtime.GC()
	var got [4]float32
	fn().StoreArray(&got)
	if got != [4]float32{1, 2, 3, 4} {
		t.Fatal("closure storage", got)
	}
	got = [4]float32{}
	nativeVectorDeferred(v, &got)
	if got != [4]float32{1, 2, 3, 4} {
		t.Fatal("defer storage", got)
	}
}
