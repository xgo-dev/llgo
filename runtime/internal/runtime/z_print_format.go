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

	c "github.com/goplus/llgo/runtime/internal/clite"
)

func formatSpecialFloat(v float64) (string, bool) {
	switch {
	case v != v:
		return "NaN", true
	case v+v == v && v != 0 && v > 0:
		return "+Inf", true
	case v+v == v && v != 0:
		return "-Inf", true
	default:
		return "", false
	}
}

func formatFloatWithC(v float64, format *c.Char) string {
	var buf [64]byte
	c.Snprintf((*c.Char)(unsafe.Pointer(&buf[0])), uintptr(len(buf)), format, v)
	return c.GoString((*c.Char)(unsafe.Pointer(&buf[0])))
}
