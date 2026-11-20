//go:build !llgo || !wasip1 || !wasm

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
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"unsafe"
)

const decimalPrintLLGoFiles = ""

func PrintUint(v uint64) {
	printDecimal(v, false)
}

func PrintInt(v int64) {
	u := uint64(v)
	negative := v < 0
	if negative {
		u = 0 - u
	}
	printDecimal(u, negative)
}

func printDecimal(v uint64, negative bool) {
	// uint64 needs at most 20 digits. Use stack storage explicitly because
	// printing is also used on panic paths and before runtime initialization.
	buf := c.Alloca(21)
	i := uintptr(21)
	for {
		i--
		*(*byte)(unsafe.Add(buf, i)) = '0' + byte(v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	if negative {
		i--
		*(*byte)(unsafe.Add(buf, i)) = '-'
	}
	c.Fwrite(unsafe.Add(buf, i), 1, 21-i, c.Stderr)
}
