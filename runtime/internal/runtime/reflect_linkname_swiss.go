//go:build swissmap

package runtime

import (
	_ "unsafe"

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/runtime/maps"
)

//go:linkname reflect_mapiterinit reflect.mapiterinit
func reflect_mapiterinit(t *abi.MapType, h *hmap, it *maps.Iter) { mapIterStart(t, h, it) }

//go:linkname reflect_mapiternext reflect.mapiternext
func reflect_mapiternext(it *maps.Iter) { mapIterNext(it) }
