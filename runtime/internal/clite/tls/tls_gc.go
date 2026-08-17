//go:build llgo && !baremetal && !nogc

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package tls

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
)

const slotRegistered = 1 << iota

const maxSlotSize = 1 << 20 // 1 MiB sanity cap

type slot[T any] struct {
	value      T
	state      uintptr
	destructor func(*T)
}

func registerSlot[T any](s *slot[T]) {
	if s.state&slotRegistered != 0 {
		return
	}
	start, end := s.rootRange()
	size := uintptr(end) - uintptr(start)
	if size == 0 {
		return
	}
	if size > maxSlotSize {
		panic("tls: slot size exceeds maximum")
	}
	bdwgc.AddRoots(start, end)
	s.state |= slotRegistered
}

func deregisterSlot[T any](s *slot[T]) {
	if s == nil || s.state&slotRegistered == 0 {
		return
	}
	s.state &^= slotRegistered
	start, end := s.rootRange()
	if uintptr(end) > uintptr(start) {
		bdwgc.RemoveRoots(start, end)
	}
}

func (s *slot[T]) rootRange() (start, end c.Pointer) {
	begin := unsafe.Pointer(s)
	size := unsafe.Sizeof(*s)
	beginAddr := uintptr(begin)
	if beginAddr > ^uintptr(0)-size {
		panic("tls: pointer arithmetic overflow in rootRange")
	}
	endPtr := unsafe.Pointer(beginAddr + size)
	return c.Pointer(begin), c.Pointer(endPtr)
}
