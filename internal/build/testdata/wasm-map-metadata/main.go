package main

import (
	"maps"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

//go:linkname referenceHash github.com/xgo-dev/llgo/runtime/internal/runtime.typehashImpl
func referenceHash(t *abi.Type, p unsafe.Pointer, seed uintptr) uintptr

func descriptor(m any) *abi.MapType {
	return *(**abi.MapType)(unsafe.Pointer(&m))
}

func check[K comparable](keys []K) {
	m := make(map[K]int)
	mt := descriptor(m)
	for i, key := range keys {
		// The static hasher must agree with the dynamic reflection/interface
		// algorithm, including seed mixing for zero-sized keys.
		for seed := uintptr(0); seed < 8; seed++ {
			p := unsafe.Pointer(&key)
			if mt.Hasher(p, seed*17) != referenceHash(mt.Key, p, seed*17) {
				panic("static and dynamic hashes differ")
			}
		}
		m[key] = i + 1
	}
	cloned := maps.Clone(m)
	runtime.GC()
	reflected := reflect.ValueOf(m)
	for i, key := range keys {
		want := i + 1
		for j := i + 1; j < len(keys); j++ {
			if keys[j] == key {
				want = j + 1
			}
		}
		if m[key] != want || cloned[key] != want || reflected.MapIndex(reflect.ValueOf(&key).Elem()).Int() != int64(want) {
			panic("map lookup, clone or reflection after GC")
		}
		reflected.SetMapIndex(reflect.ValueOf(&key).Elem(), reflect.ValueOf(want))
	}
	for _, key := range keys {
		delete(m, key)
	}
	if len(m) != 0 {
		panic("map deletion")
	}
	runtime.KeepAlive(keys)
}

func checkDynamicMap() {
	key := reflect.StructOf([]reflect.StructField{{Name: "Text", Type: reflect.TypeOf("")}, {Name: "Index", Type: reflect.TypeOf(0)}})
	typ := reflect.MapOf(key, reflect.TypeOf([]byte{}))
	m := reflect.MakeMap(typ)
	for i := 0; i < 96; i++ {
		k := reflect.New(key).Elem()
		k.Field(0).SetString(strconv.Itoa(i))
		k.Field(1).SetInt(int64(i))
		m.SetMapIndex(k, reflect.ValueOf([]byte{byte(i)}))
	}
	runtime.GC()
	for i := 0; i < 96; i++ {
		k := reflect.New(key).Elem()
		k.Field(0).SetString(strconv.Itoa(i))
		k.Field(1).SetInt(int64(i))
		if value := m.MapIndex(k); !value.IsValid() || value.Bytes()[0] != byte(i) {
			panic("dynamic map after GC")
		}
	}
}

func checkValues() {
	closures := make(map[int]func() int)
	indirect := make(map[[17]uint64][17]uint64)
	for i := 0; i < 96; i++ {
		n := i
		closures[i] = func() int { return n + 7 }
		indirect[[17]uint64{uint64(i)}] = [17]uint64{uint64(i + 9)}
	}
	closures = maps.Clone(closures)
	indirect = maps.Clone(indirect)
	runtime.GC()
	for i := 0; i < 96; i++ {
		if closures[i]() != i+7 || indirect[[17]uint64{uint64(i)}][0] != uint64(i+9) {
			panic("bucket copying changed closure or indirect storage")
		}
	}
}

func checkNonReflexive() {
	zero := math.Copysign(0, -1)
	if (map[float64]int{0: 42})[zero] != 42 || (map[complex128]int{0: 43})[complex(zero, zero)] != 43 {
		panic("signed zero hash")
	}
	nan := math.NaN()
	m := make(map[float64]int)
	c := make(map[complex128]int)
	for i := 0; i < 32; i++ {
		m[nan] = i
		c[complex(nan, 0)] = i
	}
	if len(m) != 32 || len(c) != 32 {
		panic("NaN keys were merged")
	}
	if _, ok := m[nan]; ok {
		panic("NaN lookup succeeded")
	}
}

func checkUnhashable() {
	defer func() {
		if recover() == nil {
			panic("nil map lookup accepted an unhashable key")
		}
	}()
	var m map[any]int
	_ = m[[]int{1}]
}

func main() {
	type Text string
	type Padded struct {
		A byte
		B uint64
	}
	type Blank struct {
		A uint64
		_ uint64
	}
	var integers [96]uint64
	var texts [96]Text
	var pointers [96]*uint64
	for i := range integers {
		integers[i] = uint64(i)
		texts[i] = Text(strconv.Itoa(i))
		pointers[i] = &integers[i]
	}
	check(integers[:])
	check(texts[:])
	check(pointers[:])
	check([]bool{false, true})
	check([]uint16{0, 7, 65535})
	check([]int32{-1, 0, 7})
	check([][3]byte{{1, 2, 3}, {3, 2, 1}})
	check([][16]byte{{1}, {2}})
	check([][17]uint64{{1}, {2}})
	check([]struct{}{{}, {}})
	check([][0]uint64{{}, {}})
	check([][0]string{{}, {}})
	check([]float32{0, -1, 1.5})
	check([]float64{0, -1, 1.5})
	check([]complex64{0, 1 + 2i, -3i})
	check([]complex128{0, 1 + 2i, -3i})
	check([][2]string{{"a", "b"}, {"b", "a"}})
	check([]Padded{{1, 2}, {1, 3}})
	check([]Blank{{A: 1}, {A: 2}})
	check([]any{nil, 42, "key", Padded{1, 2}})
	checkDynamicMap()
	checkValues()
	checkNonReflexive()
	checkUnhashable()
	println("wasm map metadata passed")
}
