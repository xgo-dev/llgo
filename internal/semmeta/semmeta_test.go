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

package semmeta

import (
	"testing"

	"github.com/goplus/llvm"
)

func TestRead(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	mod := ctx.NewModule("semmeta.test")
	defer mod.Dispose()

	e := NewEmitter(mod)
	e.AddUseIface("github.com/goplus/llgo/demo.main", "*_llgo_github.com/goplus/llgo/demo.Other")
	e.AddUseIface("github.com/goplus/llgo/demo.main", "*_llgo_github.com/goplus/llgo/demo.File")
	e.AddUseIfaceMethod("github.com/goplus/llgo/demo.consume", IfaceMethodUse{
		Target: "_llgo_github.com/goplus/llgo/demo.Reader",
		Sig: MethodSig{
			Name:  "Read",
			MType: "_llgo_func$readsig",
		},
	})
	e.AddInterfaceInfo("_llgo_github.com/goplus/llgo/demo.Reader", []MethodSig{
		{Name: "Close", MType: "_llgo_func$closesig"},
		{Name: "Read", MType: "_llgo_func$readsig"},
	})
	e.AddMethodInfo("*_llgo_github.com/goplus/llgo/demo.File", []MethodSlot{
		{
			Index: 1,
			Sig: MethodSig{
				Name:  "Close",
				MType: "_llgo_func$closesig",
			},
			IFn: "github.com/goplus/llgo/demo.(*File).Close",
			TFn: "github.com/goplus/llgo/demo.File.Close",
		},
		{
			Index: 0,
			Sig: MethodSig{
				Name:  "Read",
				MType: "_llgo_func$readsig",
			},
			IFn: "github.com/goplus/llgo/demo.(*File).Read",
			TFn: "github.com/goplus/llgo/demo.File.Read",
		},
	})
	e.AddUseNamedMethod("github.com/goplus/llgo/demo.lookup", "String")
	e.AddUseNamedMethod("github.com/goplus/llgo/demo.lookup", "ServeHTTP")
	e.AddReflectMethod("github.com/goplus/llgo/demo.reflectAll")

	info := Read(mod)
	if got := info.UseIface[Symbol("github.com/goplus/llgo/demo.main")]; len(got) != 2 ||
		got[0] != "*_llgo_github.com/goplus/llgo/demo.Other" ||
		got[1] != "*_llgo_github.com/goplus/llgo/demo.File" {
		t.Fatalf("UseIface mismatch: %#v", got)
	}
	if got := info.UseIfaceMethod[Symbol("github.com/goplus/llgo/demo.consume")]; len(got) != 1 ||
		got[0].Target != "_llgo_github.com/goplus/llgo/demo.Reader" ||
		got[0].Sig.Name != "Read" ||
		got[0].Sig.MType != "_llgo_func$readsig" {
		t.Fatalf("UseIfaceMethod mismatch: %#v", got)
	}
	if got := info.InterfaceInfo[Symbol("_llgo_github.com/goplus/llgo/demo.Reader")]; len(got) != 2 ||
		got[0].Name != "Close" || got[0].MType != "_llgo_func$closesig" ||
		got[1].Name != "Read" || got[1].MType != "_llgo_func$readsig" {
		t.Fatalf("InterfaceInfo mismatch: %#v", got)
	}
	if got := info.MethodInfo[Symbol("*_llgo_github.com/goplus/llgo/demo.File")]; len(got) != 2 ||
		got[0].Index != 1 || got[0].Sig.Name != "Close" ||
		got[1].Index != 0 || got[1].Sig.Name != "Read" {
		t.Fatalf("MethodInfo mismatch: %#v", got)
	}
	if got := info.UseNamedMethod[Symbol("github.com/goplus/llgo/demo.lookup")]; len(got) != 2 ||
		got[0] != "String" || got[1] != "ServeHTTP" {
		t.Fatalf("UseNamedMethod mismatch: %#v", got)
	}
	if _, ok := info.ReflectMethod[Symbol("github.com/goplus/llgo/demo.reflectAll")]; !ok {
		t.Fatal("ReflectMethod entry missing")
	}
}
