/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package wasmresume

import "unsafe"

const defaultFrameBlockSize = uintptr(2 << 10)

type frameBlock struct {
	prev         *frameBlock
	begin, end   uintptr
	stackPointer uintptr
}

type frameStorage struct {
	current *frameBlock
}

func (s *frameStorage) allocate(
	size, align uintptr, allocate Allocator,
) unsafe.Pointer {
	if size == 0 || align == 0 || align&(align-1) != 0 || allocate == nil {
		return nil
	}
	if frame, ok := allocateFromBlock(s.current, size, align); ok {
		return frame
	}

	payload, ok := addUintptr(size, unsafe.Sizeof(uintptr(0)))
	if !ok {
		return nil
	}
	payload, ok = addUintptr(payload, align-1)
	if !ok {
		return nil
	}
	if payload < defaultFrameBlockSize {
		payload = defaultFrameBlockSize
	}
	total, ok := addUintptr(unsafe.Sizeof(frameBlock{}), payload)
	if !ok {
		return nil
	}
	raw := allocate(total)
	if raw == nil {
		return nil
	}
	block := (*frameBlock)(raw)
	block.prev = s.current
	block.begin, ok = addUintptr(uintptr(raw), unsafe.Sizeof(frameBlock{}))
	if !ok {
		panic("wasmresume: frame block address overflow")
	}
	block.end, ok = addUintptr(uintptr(raw), total)
	if !ok {
		panic("wasmresume: frame block address overflow")
	}
	block.stackPointer = block.begin
	s.current = block
	frame, ok := allocateFromBlock(block, size, align)
	if !ok {
		panic("wasmresume: new frame block is too small")
	}
	return frame
}

func allocateFromBlock(block *frameBlock, size, align uintptr) (unsafe.Pointer, bool) {
	if block == nil {
		return nil, false
	}
	header, ok := addUintptr(block.stackPointer, unsafe.Sizeof(uintptr(0)))
	if !ok {
		return nil, false
	}
	frame, ok := alignUintptr(header, align)
	if !ok {
		return nil, false
	}
	next, ok := addUintptr(frame, size)
	if !ok || next > block.end {
		return nil, false
	}
	*(*uintptr)(unsafe.Pointer(frame - unsafe.Sizeof(uintptr(0)))) = block.stackPointer
	block.stackPointer = next
	return unsafe.Pointer(frame), true
}

func (s *frameStorage) releaseFrame(
	frame unsafe.Pointer, size uintptr, release Releaser,
) {
	if s.current == nil || frame == nil || size == 0 {
		panic("wasmresume: invalid frame release")
	}
	address := uintptr(frame)
	end, ok := addUintptr(address, size)
	if !ok {
		panic("wasmresume: invalid frame release")
	}

	block := s.current
	for block != nil && (address < block.begin || end > block.stackPointer) {
		block = block.prev
	}
	if block == nil {
		panic("wasmresume: frame is not owned by this context")
	}
	previous := *(*uintptr)(unsafe.Pointer(address - unsafe.Sizeof(uintptr(0))))
	if previous < block.begin || previous >= address {
		panic("wasmresume: invalid frame allocation header")
	}
	if block != s.current && release == nil {
		panic("wasmresume: missing frame block reclaimer")
	}
	for s.current != block {
		current := s.current
		s.current = current.prev
		release(unsafe.Pointer(current))
	}
	block.stackPointer = previous
	if previous == block.begin && block.prev != nil {
		if release == nil {
			panic("wasmresume: missing frame block reclaimer")
		}
		s.current = block.prev
		release(unsafe.Pointer(block))
	}
}

func (s *frameStorage) close(release Releaser) {
	if s.current != nil && release == nil {
		panic("wasmresume: missing frame block reclaimer")
	}
	for block := s.current; block != nil; {
		previous := block.prev
		release(unsafe.Pointer(block))
		block = previous
	}
	s.current = nil
}

func addUintptr(left, right uintptr) (uintptr, bool) {
	sum := left + right
	return sum, sum >= left
}

func alignUintptr(value, align uintptr) (uintptr, bool) {
	next, ok := addUintptr(value, align-1)
	if !ok {
		return 0, false
	}
	return next &^ (align - 1), true
}
