//go:build !llgo
// +build !llgo

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

package cl

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/gogen/packages"
	llssa "github.com/goplus/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestOffsetOfBuiltinArgPromotedFieldChain(t *testing.T) {
	src := `package foo

import "unsafe"

type level1 struct{ B byte }
type level2 struct {
	C byte
	level1
}
type outer struct {
	A byte
	level2
}

type genericOuter[T any] struct {
	A T
	level1
}

type genericNested[T any] struct {
	A T
	level2
}

func genericPromoted[T any](v *genericOuter[T]) (uintptr, uintptr) {
	return unsafe.Offsetof(v.B), unsafe.Offsetof(v.level1)
}

func genericMulti[T any](v *genericNested[T]) (uintptr, uintptr) {
	return unsafe.Offsetof(v.B), unsafe.Offsetof(v.level2)
}

func genericExplicit[T any](v *genericNested[T]) uintptr {
	return unsafe.Offsetof(v.level2)
}
`
	ssaPkg, fset := buildOffsetofPackage(t, src)
	ctx := &context{prog: newLLSSAProg(t), fset: fset}

	t.Run("single-level promotion", func(t *testing.T) {
		args := offsetofArgs(t, ssaPkg.Func("genericPromoted"), 2)
		promoted := offsetOfArg(t, ctx, args[0])
		embedded := offsetOfArg(t, ctx, args[1])
		if promoted != embedded {
			t.Fatalf("promoted offset = %d, embedded offset = %d", promoted, embedded)
		}
	})
	t.Run("multi-level promotion", func(t *testing.T) {
		args := offsetofArgs(t, ssaPkg.Func("genericMulti"), 2)
		promoted := offsetOfArg(t, ctx, args[0])
		embedded := offsetOfArg(t, ctx, args[1])
		if promoted != embedded+1 {
			t.Fatalf("promoted offset = %d, want embedded offset + 1 (%d)", promoted, embedded+1)
		}
	})
	t.Run("explicit selector is detected", func(t *testing.T) {
		args := offsetofArgs(t, ssaPkg.Func("genericExplicit"), 1)
		load := args[0].(*gossa.UnOp)
		field := load.X.(*gossa.FieldAddr)
		if !ctx.isExplicitFieldAddr(field) {
			t.Fatal("explicit FieldAddr was not detected")
		}
	})
	if _, ok := ctx.offsetOfBuiltinArg(gossa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])); ok {
		t.Fatal("offsetOfBuiltinArg returned true for a non-load argument")
	}
}

func TestOffsetOfHelpersAndSourceLineCache(t *testing.T) {
	src := `package foo

import "unsafe"

type embedded struct{ B byte }
type outer[T any] struct {
	A T
	embedded
}

func promoted[T any](v *outer[T]) uintptr {
	return unsafe.Offsetof(v.B)
}

func explicit[T any](v *outer[T]) uintptr {
	return unsafe.Offsetof(v.embedded)
}
`
	ssaPkg, fset := buildOffsetofPackage(t, src)
	ctx := &context{prog: newLLSSAProg(t), fset: fset}
	arg := offsetofArgs(t, ssaPkg.Func("promoted"), 1)[0]
	load := arg.(*gossa.UnOp)
	field := load.X.(*gossa.FieldAddr)

	name, ok := fieldAddrName(field)
	if !ok || name != "B" {
		t.Fatalf("fieldAddrName = %q, %v; want B, true", name, ok)
	}
	if _, _, ok := fieldAddrStruct(field); !ok {
		t.Fatal("fieldAddrStruct returned false for valid FieldAddr")
	}
	parent := field.X.(*gossa.FieldAddr)
	if ctx.isExplicitFieldAddr(parent) {
		t.Fatal("promoted FieldAddr parent should not be considered explicit")
	}
	explicitArg := offsetofArgs(t, ssaPkg.Func("explicit"), 1)[0]
	explicitLoad := explicitArg.(*gossa.UnOp)
	explicitField := explicitLoad.X.(*gossa.FieldAddr)
	if !ctx.isExplicitFieldAddr(explicitField) {
		t.Fatal("explicit FieldAddr should be considered explicit")
	}
	if _, ok := fieldAddrName(&gossa.FieldAddr{}); ok {
		t.Fatal("fieldAddrName returned true for invalid FieldAddr")
	}
	if ctx.isExplicitFieldAddr(&gossa.FieldAddr{}) != true {
		t.Fatal("invalid FieldAddr should be treated as explicit")
	}
	if _, ok := ctx.offsetOfFieldAddr(&gossa.FieldAddr{}); ok {
		t.Fatal("offsetOfFieldAddr returned true for invalid FieldAddr")
	}
	if _, ok := ctx.offsetOfFieldChain(&gossa.FieldAddr{}); ok {
		t.Fatal("offsetOfFieldChain returned true for invalid FieldAddr")
	}
	if _, ok := ctx.offsetOfBuiltinArg(&gossa.UnOp{
		Op: token.NOT,
		X:  gossa.NewConst(constant.MakeBool(true), types.Typ[types.Bool]),
	}); ok {
		t.Fatal("offsetOfBuiltinArg returned true for a non-deref UnOp")
	}
	if _, ok := ctx.offsetOfBuiltinArg(&gossa.UnOp{
		Op: token.MUL,
		X:  gossa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
	}); ok {
		t.Fatal("offsetOfBuiltinArg returned true for deref of a non-FieldAddr")
	}
	if _, ok := ctx.offsetOfBuiltinArg(&gossa.UnOp{
		Op: token.MUL,
		X:  &gossa.FieldAddr{},
	}); ok {
		t.Fatal("offsetOfBuiltinArg returned true for invalid FieldAddr")
	}

	stubStruct := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "F", types.Typ[types.Int]),
	}, nil)
	validSynthetic := &gossa.FieldAddr{
		X:     fakeSSAValue{typ: types.NewPointer(stubStruct)},
		Field: 0,
	}
	if ctx.isExplicitFieldAddr(validSynthetic) {
		t.Fatal("FieldAddr without source position should not be considered explicit")
	}
	if _, _, ok := fieldAddrStruct(&gossa.FieldAddr{
		X:     fakeSSAValue{typ: types.Typ[types.Int]},
		Field: 0,
	}); ok {
		t.Fatal("fieldAddrStruct returned true for non-pointer receiver")
	}
	if _, _, ok := fieldAddrStruct(&gossa.FieldAddr{
		X:     fakeSSAValue{typ: types.NewPointer(types.Typ[types.Int])},
		Field: 0,
	}); ok {
		t.Fatal("fieldAddrStruct returned true for pointer to non-struct receiver")
	}
	if _, _, ok := fieldAddrStruct(&gossa.FieldAddr{
		X:     fakeSSAValue{typ: types.NewPointer(stubStruct)},
		Field: stubStruct.NumFields(),
	}); ok {
		t.Fatal("fieldAddrStruct returned true for out-of-range field")
	}

	file := filepath.Join(t.TempDir(), "lines.go")
	if err := os.WriteFile(file, []byte("first\nsecond\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	line, ok := ctx.sourceLine(file, 2)
	if !ok || line != "second" {
		t.Fatalf("sourceLine = %q, %v; want second, true", line, ok)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	line, ok = ctx.sourceLine(file, 2)
	if !ok || line != "second" {
		t.Fatalf("cached sourceLine = %q, %v; want second, true", line, ok)
	}
	if _, ok := ctx.sourceLine(file, 0); ok {
		t.Fatal("sourceLine returned true for line 0")
	}
	if _, ok := (&context{}).sourceLine(filepath.Join(t.TempDir(), "missing.go"), 1); ok {
		t.Fatal("sourceLine returned true for a missing file")
	}

	explicitFile := fset.Position(explicitField.Pos()).Filename
	if explicitFile == "" {
		t.Fatal("explicit FieldAddr has no filename")
	}
	shortLineCtx := &context{
		prog: newLLSSAProg(t),
		fset: fset,
		srcLines: map[string][]string{
			explicitFile: make([]string, fset.Position(explicitField.Pos()).Line),
		},
	}
	if shortLineCtx.isExplicitFieldAddr(explicitField) {
		t.Fatal("FieldAddr with column beyond source line should not be considered explicit")
	}
	if err := os.Remove(explicitFile); err != nil {
		t.Fatal(err)
	}
	if (&context{prog: newLLSSAProg(t), fset: fset}).isExplicitFieldAddr(explicitField) {
		t.Fatal("FieldAddr with missing source file should not be considered explicit")
	}
}

func TestCompileOffsetOfPromotedFieldChain(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `package foo

import "unsafe"

type level1 struct{ B byte }
type level2 struct {
	C byte
	level1
}
type outer[T any] struct {
	A T
	level2
}

func promoted[T any](v *outer[T]) uintptr {
	return unsafe.Offsetof(v.B)
}

func Offset(v *outer[int]) uintptr {
	return promoted(v)
}
`)
	if fn := m.NamedFunction("foo.Offset"); fn.IsNil() {
		t.Fatal("missing compiled Offset function")
	}
}

func TestInstrHelperEdges(t *testing.T) {
	ctx := &context{prog: newLLSSAProg(t)}

	mustPanic(t, "funcPCABI0 invalid arguments", func() {
		ctx.funcPCABI0Value(nil, fakeSSAValue{typ: types.Typ[types.Int]})
	})
	mustPanic(t, "syscall missing arguments", func() {
		ctx.syscallIntrinsic(nil, nil, nil)
	})

	variadicSig := ctx.syscallFnSig(2)
	if got := variadicSig.Params().Len(); got != 3 {
		t.Fatalf("syscallFnSig params = %d, want 3", got)
	}
	if !variadicSig.Variadic() {
		t.Fatal("syscallFnSig should be variadic")
	}

	sig := ctx.syscallFnSigFixed(2)
	if got := sig.Params().Len(); got != 2 {
		t.Fatalf("syscallFnSigFixed params = %d, want 2", got)
	}
	if got := sig.Results().Len(); got != 1 {
		t.Fatalf("syscallFnSigFixed results = %d, want 1", got)
	}

	ssaPkg, _ := buildOffsetofPackage(t, `package foo
func fakeUnknown() {}
`)
	fn := ssaPkg.Func("fakeUnknown")
	ctx = &context{
		prog:   newLLSSAProg(t),
		goTyps: ssaPkg.Pkg,
		loaded: make(map[*types.Package]*pkgInfo),
	}
	ctx.prog.SetLinkname("foo.fakeUnknown", "llgo.fakeUnknown")
	old, hadOld := llgoInstrs["fakeUnknown"]
	llgoInstrs["fakeUnknown"] = llgoInstrBase + 0x7f
	defer func() {
		if hadOld {
			llgoInstrs["fakeUnknown"] = old
		} else {
			delete(llgoInstrs, "fakeUnknown")
		}
	}()
	mustPanic(t, "unknown llgo instruction ftype", func() {
		ctx.call(nil, llssa.Call, &gossa.CallCommon{Value: fn})
	})
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s did not panic", name)
		}
	}()
	fn()
}

type fakeSSAValue struct {
	typ types.Type
}

func (v fakeSSAValue) Name() string                    { return "fake" }
func (v fakeSSAValue) String() string                  { return "fake" }
func (v fakeSSAValue) Type() types.Type                { return v.typ }
func (v fakeSSAValue) Parent() *gossa.Function         { return nil }
func (v fakeSSAValue) Referrers() *[]gossa.Instruction { return nil }
func (v fakeSSAValue) Pos() token.Pos                  { return token.NoPos }

func buildOffsetofPackage(t *testing.T, src string) (*gossa.Package, *token.FileSet) {
	t.Helper()
	dir := t.TempDir()
	filename := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(filename, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage(f.Name.Name, f.Name.Name)
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssaPkg, fset
}

func offsetOfArg(t *testing.T, ctx *context, arg gossa.Value) int {
	t.Helper()
	if _, ok := ctx.offsetOfBuiltinArg(arg); !ok {
		t.Fatal("offsetOfBuiltinArg returned false")
	}
	load := arg.(*gossa.UnOp)
	field := load.X.(*gossa.FieldAddr)
	got, ok := ctx.offsetOfFieldChain(field)
	if !ok {
		t.Fatal("offsetOfFieldChain returned false")
	}
	return got
}

func offsetofArgs(t *testing.T, fn *gossa.Function, want int) []gossa.Value {
	t.Helper()
	if fn == nil {
		t.Fatal("missing function")
	}
	var args []gossa.Value
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*gossa.Call)
			if !ok {
				continue
			}
			builtin, ok := call.Call.Value.(*gossa.Builtin)
			if ok && builtin.Name() == "Offsetof" {
				args = append(args, call.Call.Args[0])
			}
		}
	}
	if len(args) != want {
		t.Fatalf("found %d Offsetof calls in %s, want %d", len(args), fn.Name(), want)
	}
	return args
}
