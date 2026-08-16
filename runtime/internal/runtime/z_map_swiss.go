//go:build swissmap

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/runtime/maps"
)

type Map = maps.Map
type maptype = abi.MapType
type arraytype = abi.ArrayType
type structtype = abi.StructType

type slice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

func typedmemmove(typ *_type, dst, src unsafe.Pointer) { Typedmemmove(typ, dst, src) }

func MakeSmallMap() *Map                                               { return makemap_small() }
func MakeMap(t *maptype, hint int) *Map                                { return makemap(t, hint, nil) }
func MapAssign(t *maptype, h *Map, key unsafe.Pointer) unsafe.Pointer  { return mapassign(t, h, key) }
func MapAccess1(t *maptype, h *Map, key unsafe.Pointer) unsafe.Pointer { return mapaccess1(t, h, key) }
func MapAccess2(t *maptype, h *Map, key unsafe.Pointer) (unsafe.Pointer, bool) {
	return mapaccess2(t, h, key)
}
func MapDelete(t *maptype, h *Map, key unsafe.Pointer) { mapdelete(t, h, key) }
func MapClear(t *maptype, h *Map)                      { mapclear(t, h) }

func MapAccess1Fast32(t *maptype, h *Map, key uint32) unsafe.Pointer {
	return mapaccess1_fast32(t, h, key)
}

func MapAccess2Fast32(t *maptype, h *Map, key uint32) (unsafe.Pointer, bool) {
	return mapaccess2_fast32(t, h, key)
}

func MapAssignFast32(t *maptype, h *Map, key uint32) unsafe.Pointer {
	return mapassign_fast32(t, h, key)
}

func MapAssignFast32Ptr(t *maptype, h *Map, key unsafe.Pointer) unsafe.Pointer {
	return mapassign_fast32ptr(t, h, key)
}

func MapDeleteFast32(t *maptype, h *Map, key uint32) {
	mapdelete_fast32(t, h, key)
}

func MapAccess1Fast64(t *maptype, h *Map, key uint64) unsafe.Pointer {
	return mapaccess1_fast64(t, h, key)
}

func MapAccess2Fast64(t *maptype, h *Map, key uint64) (unsafe.Pointer, bool) {
	return mapaccess2_fast64(t, h, key)
}

func MapAssignFast64(t *maptype, h *Map, key uint64) unsafe.Pointer {
	return mapassign_fast64(t, h, key)
}

func MapAssignFast64Ptr(t *maptype, h *Map, key unsafe.Pointer) unsafe.Pointer {
	return mapassign_fast64ptr(t, h, key)
}

func MapDeleteFast64(t *maptype, h *Map, key uint64) {
	mapdelete_fast64(t, h, key)
}

func MapAccess1FastStr(t *maptype, h *Map, key string) unsafe.Pointer {
	return mapaccess1_faststr(t, h, key)
}

func MapAccess2FastStr(t *maptype, h *Map, key string) (unsafe.Pointer, bool) {
	return mapaccess2_faststr(t, h, key)
}

func MapAssignFastStr(t *maptype, h *Map, key string) unsafe.Pointer {
	return mapassign_faststr(t, h, key)
}

func MapDeleteFastStr(t *maptype, h *Map, key string) {
	mapdelete_faststr(t, h, key)
}

type llgoMapIter struct {
	maps.Iter
	ready bool
}

func NewMapIter(t *maptype, h *Map) *llgoMapIter {
	it := &llgoMapIter{ready: true}
	mapIterStart(t, h, &it.Iter)
	return it
}

func MapIterNext(it *llgoMapIter) (ok bool, k unsafe.Pointer, v unsafe.Pointer) {
	if !it.ready {
		mapIterNext(&it.Iter)
		it.ready = true
	}
	k, v = it.Key(), it.Elem()
	if k == nil {
		return false, nil, nil
	}
	it.ready = false
	return true, k, v
}

func MapLen(h *Map) int {
	if h == nil {
		return 0
	}
	return int(h.Used())
}
