//go:build !llgo
// +build !llgo

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

package ssa_test

import (
	"flag"
	"go/importer"
	"go/types"
	"io"
	"log"
	"runtime"
	"testing"

	"github.com/goplus/llgo/cl/cltest"
	"github.com/goplus/llgo/ssa"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
	m.Run()
}

func TestFromTestlibgo(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testlibgo")
}

func TestFromTestgo(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testgo")
}

func TestFromTestpy(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testpy")
}

func TestFromTestlibc(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testlibc")
}

func TestFromTestrt(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testrt")
}

func TestFromTestdata(t *testing.T) {
	cltest.FromDir(t, "", "../cl/_testdata")
}

func TestMakeInterface(t *testing.T) {
	prog := ssa.NewProgram(nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(ssa.PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("main", types.NewSignatureType(nil, nil, nil, nil, nil, false), ssa.InC)
	b := fn.MakeBody(1)
	b.MakeInterface(prog.Any(), prog.IntVal(100, prog.Int64()))
	b.MakeInterface(prog.Any(), prog.FloatVal(100, prog.Float64()))
	b.DeferData()
	b.Return()
}

/*
func TestCallback(t *testing.T) {
	ctx := llvm.NewContext()
	mod := ctx.NewModule("foo/bar")

	tc := llvm.FunctionType(ctx.VoidType(), nil, false)
	callback := llvm.PointerType(tc, 0)
	params := []llvm.Type{callback}

	tfn := llvm.FunctionType(ctx.VoidType(), params, false)
	f := llvm.AddFunction(mod, "fn", tfn)
	b := ctx.NewBuilder()
	blk := llvm.AddBasicBlock(f, "")
	b.SetInsertPointAtEnd(blk)

	arg := f.Param(0)
	// arg = b.CreateLoad(tc, arg, "")
	b.CreateCall(tc, arg, nil, "")
	b.CreateRetVoid()

	expected := `; ModuleID = 'foo/bar'
`
	if v := mod.String(); v != expected {
		t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", v, expected)
	}
}

/*
func TestMap(t *testing.T) {
	var m typeutil.Map
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	m.Set(sig, 1)
	csig := (*ssa.CFuncPtr)(sig)
	m.Set(csig, 2)
	if v := m.At(sig); v.(int) != 1 {
		t.Fatal("At(sig):", v)
	}
	if v := m.At(csig); v.(int) != 2 {
		t.Fatal("At(csig):", v)
	}
}
*/
