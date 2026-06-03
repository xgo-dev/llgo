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

func boolCStr(v bool) *c.Char {
	if v {
		return c.Str("true")
	}
	return c.Str("false")
}

func PrintBool(v bool) {
	c.Fprintf(c.Stderr, boolCStr(v))
}

func PrintByte(v byte) {
	c.Fputc(c.Int(v), c.Stderr)
}

func PrintUint(v uint64) {
	c.Fprintf(c.Stderr, printFormatPrefixUInt, v)
}

func PrintInt(v int64) {
	c.Fprintf(c.Stderr, printFormatPrefixInt, v)
}

func PrintFloat(v float64) {
	c.Fputs(c.AllocaCStr(formatFloat(v)), c.Stderr)
}

func PrintComplex(v complex128) {
	c.Fputs(c.AllocaCStr(formatComplex(v)), c.Stderr)
}

func PrintHex(v uint64) {
	c.Fprintf(c.Stderr, printFormatPrefixHex, v)
}

func PrintPointer(p unsafe.Pointer) {
	// Match Go's builtin print/println pointer formatting (0x... even for nil).
	c.Fprintf(c.Stderr, c.Str("0x"))
	c.Fprintf(c.Stderr, printFormatPrefixHex, uintptr(p))
}

func PrintString(s String) {
	c.Fwrite(s.data, 1, uintptr(s.len), c.Stderr)
}

func PrintSlice(s Slice) {
	print("[", s.len, "/", s.cap, "]", s.data)
}

func PrintEface(e Eface) {
	print("(", e._type, ",", e.data, ")")
}

func PrintIface(i Iface) {
	print("(", i.tab, ",", i.data, ")")
}
