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

package cltest

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/semmeta"
)

func TestFormatSemMeta(t *testing.T) {
	info := semmeta.ModuleInfo{
		UseIface: map[semmeta.Symbol][]semmeta.Symbol{
			"github.com/goplus/llgo/demo.main": {
				"*_llgo_github.com/goplus/llgo/demo.Other",
				"*_llgo_github.com/goplus/llgo/demo.File",
			},
		},
		UseIfaceMethod: map[semmeta.Symbol][]semmeta.IfaceMethodUse{
			"github.com/goplus/llgo/demo.consume": {
				{
					Target: "_llgo_github.com/goplus/llgo/demo.Reader",
					Sig: semmeta.MethodSig{
						Name:  "Read",
						MType: "_llgo_func$readsig",
					},
				},
			},
		},
		InterfaceInfo: map[semmeta.Symbol][]semmeta.MethodSig{
			"_llgo_github.com/goplus/llgo/demo.Reader": {
				{Name: "Close", MType: "_llgo_func$closesig"},
				{Name: "Read", MType: "_llgo_func$readsig"},
			},
		},
		MethodInfo: map[semmeta.Symbol][]semmeta.MethodSlot{
			"*_llgo_github.com/goplus/llgo/demo.File": {
				{
					Index: 1,
					Sig: semmeta.MethodSig{
						Name:  "Close",
						MType: "_llgo_func$closesig",
					},
					IFn: "github.com/goplus/llgo/demo.(*File).Close",
					TFn: "github.com/goplus/llgo/demo.File.Close",
				},
				{
					Index: 0,
					Sig: semmeta.MethodSig{
						Name:  "Read",
						MType: "_llgo_func$readsig",
					},
					IFn: "github.com/goplus/llgo/demo.(*File).Read",
					TFn: "github.com/goplus/llgo/demo.File.Read",
				},
			},
		},
		UseNamedMethod: map[semmeta.Symbol][]string{
			"github.com/goplus/llgo/demo.lookup": {"String", "ServeHTTP"},
		},
		ReflectMethod: map[semmeta.Symbol]struct{}{
			"github.com/goplus/llgo/demo.reflectAll": {},
		},
	}

	got := FormatSemMeta(info)
	want := strings.TrimSpace(`
[UseIface]
github.com/goplus/llgo/demo.main:
    *_llgo_github.com/goplus/llgo/demo.File
    *_llgo_github.com/goplus/llgo/demo.Other

[UseIfaceMethod]
github.com/goplus/llgo/demo.consume:
    _llgo_github.com/goplus/llgo/demo.Reader Read _llgo_func$readsig

[InterfaceInfo]
_llgo_github.com/goplus/llgo/demo.Reader:
    Close _llgo_func$closesig
    Read _llgo_func$readsig

[MethodInfo]
*_llgo_github.com/goplus/llgo/demo.File:
    1 Close _llgo_func$closesig github.com/goplus/llgo/demo.(*File).Close github.com/goplus/llgo/demo.File.Close
    0 Read _llgo_func$readsig github.com/goplus/llgo/demo.(*File).Read github.com/goplus/llgo/demo.File.Read

[UseNamedMethod]
github.com/goplus/llgo/demo.lookup:
    ServeHTTP
    String

[ReflectMethod]
github.com/goplus/llgo/demo.reflectAll
`) + "\n"
	if got != want {
		t.Fatalf("FormatSemMeta mismatch\n==> got:\n%s\n==> want:\n%s", got, want)
	}
}
