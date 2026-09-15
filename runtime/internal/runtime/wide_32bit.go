//go:build 386 || arm || mips || mipsle || wasm

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

package runtime

// NewChan64 follows runtime.makechan64. It preserves the range check for a
// wide make operand on 32-bit targets before calling the native-int fast path.
func NewChan64(eltSize int, cap64 int64) *Chan {
	cap := int(cap64)
	if int64(cap) != cap64 {
		panicMakeChanSize()
	}
	return NewChan(eltSize, cap)
}

// PanicExtendIndex and PanicExtendIndexU preserve a 64-bit index on 32-bit
// targets. Their high/low-word interface follows runtime.panicExtendIndex in
// the standard Go runtime.
func PanicExtendIndex(hi int, lo uint, y int) {
	panic(boundsError{x: int64(hi)<<32 + int64(lo), signed: true, y: y, code: boundsIndex})
}

func PanicExtendIndexU(hi uint, lo uint, y int) {
	panic(boundsError{x: int64(hi)<<32 + int64(lo), signed: false, y: y, code: boundsIndex})
}

// MakeSlice64 follows runtime.makeslice64. The compiler uses it on 32-bit
// targets when either source operand is wider than int, so an out-of-range
// value is rejected before it can be truncated into an apparently valid size.
func MakeSlice64(len64, cap64 int64, etSize int) Slice {
	len := int(len64)
	if int64(len) != len64 {
		panicmakeslicelen()
	}
	cap := int(cap64)
	if int64(cap) != cap64 {
		panicmakeslicecap()
	}
	return MakeSlice(len, cap, etSize)
}
