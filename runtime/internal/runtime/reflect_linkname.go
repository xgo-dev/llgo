/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"
	_ "unsafe"
)

//go:linkname reflect_unsafe_New reflect.unsafe_New
func reflect_unsafe_New(t *Type) unsafe.Pointer {
	return New(t)
}

//go:linkname reflect_unsafe_NewArray reflect.unsafe_NewArray
func reflect_unsafe_NewArray(t *Type, n int) unsafe.Pointer {
	return NewArray(t, n)
}

//go:linkname reflect_typedmemmove reflect.typedmemmove
func reflect_typedmemmove(t *Type, dst, src unsafe.Pointer) {
	Typedmemmove(t, dst, src)
}

//go:linkname reflect_makemap reflect.makemap
func reflect_makemap(t *maptype, cap int) *hmap {
	return MakeMap(t, cap)
}

//go:linkname reflect_mapaccess reflect.mapaccess
func reflect_mapaccess(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	elem, ok := mapaccess2(t, h, key)
	if !ok {
		return nil
	}
	return elem
}

//go:linkname reflect_mapiterinit reflect.mapiterinit
func reflect_mapiterinit(t *maptype, h *hmap, it *hiter) {
	mapiterinit(t, h, it)
}

//go:linkname reflect_mapiternext reflect.mapiternext
func reflect_mapiternext(it *hiter) {
	mapiternext(it)
}

//go:linkname reflect_ifaceE2I reflect.ifaceE2I
func reflect_ifaceE2I(t *interfacetype, src any, dst unsafe.Pointer) {
	IfaceE2I(t, *(*eface)(unsafe.Pointer(&src)), (*iface)(dst))
}
