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

package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"math"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestCVectorValueAndArrayOperations(t *testing.T) {
	for _, target := range []*Target{{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "amd64"}, {GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			rawPkg := types.NewPackage("example.com/c", "c")
			arr := types.NewArray(types.Typ[types.Float32], 4)
			raw := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "_Ctype_F4", nil), arr, nil)
			prog.SetCVectorType("example.com/c._Ctype_F4", 4)
			typ := prog.rawType(raw)
			if typ.ll.String() != "<4 x float>" || prog.storageType(typ) != typ.ll || sizes.Alignof(raw) != 16 || sizes.Sizeof(raw) != 16 {
				t.Fatal("lost native vector representation")
			}
			plain := prog.rawType(arr)
			if plain.ll.TypeKind() != llvm.ArrayTypeKind || sizes.Alignof(arr) != 4 {
				t.Fatal("ordinary array changed")
			}
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("p", "p")
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", raw)), types.NewTuple(types.NewParam(token.NoPos, nil, "", raw)), false)
			fn := pkg.NewFunc("p.roundtrip", sig, CDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			value := fn.Param(0)
			array := b.ChangeType(plain, value)
			restored := b.ChangeType(typ, array)
			lane := b.Index(restored, prog.IntVal(2, prog.Int()), func() (Expr, bool) { t.Fatal("vector index took an address"); return Nil, false })
			rebuilt := b.Aggregate(typ, lane, lane, lane, lane)
			b.BinOp(token.EQL, restored, rebuilt)
			ptr := b.AllocaT(typ)
			b.Store(ptr, restored)
			b.Return(b.Load(ptr))
			b.EndBuild()
			values := []Expr{prog.FloatVal(1, prog.Float32()), prog.FloatVal(2, prog.Float32()), prog.FloatVal(3, prog.Float32()), prog.FloatVal(4, prog.Float32())}
			constant := prog.ConstArray(typ, values)
			global := pkg.NewVarEx("p.vector", prog.Pointer(typ))
			global.Init(constant)
			if !strings.Contains(fn.impl.String(), "extractelement <4 x float>") {
				t.Fatal(fn.impl.String())
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err, pkg.String())
			}
		})
	}
}

func TestNativeVectorSourceLayouts(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()
	sizes := prog.TypeSizes(types.SizesFor("gc", "arm64"))
	simd := types.NewPackage("simd/archsimd", "archsimd")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, simd, "v128", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, simd, "align", types.NewArray(types.Typ[types.Uint64], 0), false),
	}, nil), nil)
	vector := types.NewNamed(types.NewTypeName(token.NoPos, simd, "Float32x4", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, simd, "tag", marker, false), types.NewField(token.NoPos, simd, "lanes", types.NewArray(types.Typ[types.Float32], 4), false),
	}, nil), nil)
	rawPkg := types.NewPackage("example.com/p", "p")
	fields := func(pos token.Pos, elem types.Type) []*types.Var {
		return []*types.Var{
			types.NewField(pos, rawPkg, "tag", types.Typ[types.Byte], false), types.NewField(pos+1, rawPkg, "v", elem, false), types.NewField(pos+2, rawPkg, "tail", types.Typ[types.Byte], false),
		}
	}
	nativeFields := fields(10, vector)
	native := types.NewNamed(types.NewTypeName(1, rawPkg, "Native", nil), types.NewStruct(nativeFields, nil), nil)
	prog.SetTypeCallConv("example.com/p.Native", CDecl)
	prog.SetNativeStructFields([]token.Pos{10, 11, 12})
	if sizes.Sizeof(native) != 48 || sizes.Alignof(native) != 16 || sizes.Offsetsof(nativeFields)[1] != 16 {
		t.Fatal("native registration did not enable vector source layout")
	}
	ordinaryFields := fields(20, vector)
	ordinary := types.NewStruct(ordinaryFields, nil)
	cvector := types.NewNamed(types.NewTypeName(2, rawPkg, "CVector", nil), types.NewArray(types.Typ[types.Float32], 4), nil)
	prog.SetCVectorType("example.com/p.CVector", 4)
	importedFields := fields(30, cvector)
	imported := types.NewStruct(importedFields, nil)
	for _, tc := range []struct {
		name                    string
		typ                     types.Type
		fields                  []*types.Var
		size, alignment, offset int64
	}{
		{"Go", ordinary, ordinaryFields, 48, 16, 16},
		{"cdecl", native, nativeFields, 48, 16, 16},
		{"C vector", imported, importedFields, 48, 16, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if sizes.Sizeof(tc.typ) != tc.size || sizes.Alignof(tc.typ) != tc.alignment || sizes.Offsetsof(tc.fields)[1] != tc.offset {
				t.Fatalf("size/align/offset = %d/%d/%d", sizes.Sizeof(tc.typ), sizes.Alignof(tc.typ), sizes.Offsetsof(tc.fields)[1])
			}
			ty := prog.Type(tc.typ, GoDecl)
			if prog.SizeOf(ty) != uint64(tc.size) || prog.AlignOf(ty) != uint64(tc.alignment) || prog.OffsetOf(ty, 1) != uint64(tc.offset) {
				t.Fatal("source layout differs from LLVM", ty.ll.String())
			}
		})
	}
	if sizes.Sizeof(types.NewArray(native, 3)) != 144 || sizes.Alignof(types.NewArray(native, 3)) != 16 {
		t.Fatal("native array layout")
	}
	outerFields := fields(40, native)
	outer := types.NewStruct(outerFields, nil)
	if sizes.Sizeof(outer) != 80 || sizes.Offsetsof(outerFields)[1] != 16 {
		t.Fatal("nested native layout")
	}
	if sizes.Alignof(vector) != 16 || sizes.Sizeof(vector) != 16 {
		t.Fatal("archsimd storage is not natively aligned")
	}
}

func TestCVectorRejectsIncompatibleGoStorage(t *testing.T) {
	for _, tc := range []struct {
		name  string
		raw   types.Type
		lanes int
		want  string
	}{
		{"struct", types.NewStruct(nil, nil), 4, "incompatible Go representation"},
		{"length", types.NewArray(types.Typ[types.Float32], 4), 2, "incompatible Go representation"},
		{"word", types.NewArray(types.Typ[types.Uintptr], 2), 2, "requires 128 bits"},
		{"pointer", types.NewArray(types.NewPointer(types.Typ[types.Byte]), 2), 2, "requires 128 bits"},
		{"width", types.NewArray(types.Typ[types.Uint16], 4), 4, "requires 128 bits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", "amd64"))
			pkg := types.NewPackage("p", "p")
			raw := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "V", nil), tc.raw, nil)
			prog.SetCVectorType("p.V", tc.lanes)
			defer func() {
				if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), tc.want) {
					t.Fatalf("panic = %v, want %s", got, tc.want)
				}
			}()
			prog.rawType(raw)
		})
	}
}

func TestCVectorByteConstantsAndLayoutBounds(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	sizes := prog.TypeSizes(types.SizesFor("gc", "amd64"))
	pkg := types.NewPackage("p", "p")
	raw := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Bytes", nil), types.NewArray(types.Typ[types.Byte], 16), nil)
	prog.SetCVectorType("p.Bytes", 16)
	constant := prog.ConstByteArray(prog.rawType(raw), []byte("0123456789abcdef"))
	for i, want := range []byte("0123456789abcdef") {
		got := llvm.ConstExtractElement(constant.impl, llvm.ConstInt(prog.tyInt32(), uint64(i), false)).ZExtValue()
		if got != uint64(want) {
			t.Fatalf("lane %d = %d, want %d", i, got, want)
		}
	}
	for _, tc := range []struct{ n, want int64 }{{0, 0}, {2, 32}, {math.MaxInt64 / 16, math.MaxInt64 / 16 * 16}, {math.MaxInt64/16 + 1, -1}} {
		if got := sizes.Sizeof(types.NewArray(raw, tc.n)); got != tc.want {
			t.Errorf("array length %d: size %d, want %d", tc.n, got, tc.want)
		}
	}
	for _, tc := range []struct {
		name string
		tail types.Type
		want int64
	}{
		{"zero tail", types.NewArray(types.Typ[types.Byte], 0), 32},
		{"overflow tail", types.NewArray(types.Typ[types.Byte], math.MaxInt64), -1},
		{"overflow vector", types.NewArray(raw, math.MaxInt64), -1},
	} {
		st := types.NewStruct([]*types.Var{types.NewField(0, pkg, "v", raw, false), types.NewField(0, pkg, "tail", tc.tail, false)}, nil)
		if got := sizes.Sizeof(st); got != tc.want {
			t.Errorf("%s: size %d, want %d", tc.name, got, tc.want)
		}
	}
	// Function fields in native structs occupy one code pointer.
	fields := []*types.Var{
		types.NewField(0, pkg, "v", raw, false), types.NewField(0, pkg, "f", NoArgsNoRet, false), types.NewField(0, pkg, "tail", types.Typ[types.Byte], false),
	}
	st := types.NewNamed(types.NewTypeName(0, pkg, "Callback", nil), types.NewStruct(fields, nil), nil)
	prog.SetTypeCallConv("p.Callback", CDecl, fields...)
	// Offsetsof must work before Sizeof/Alignof, even without source positions.
	if offsets := sizes.Offsetsof(fields); offsets[1] != 16 || offsets[2] != 24 {
		t.Fatalf("native callback offsets = %v, want [0 16 24]", offsets)
	}
	if got := sizes.Sizeof(st); got != 32 {
		t.Fatalf("native callback struct size = %d, want 32", got)
	}
	zeroFields := []*types.Var{
		types.NewField(0, pkg, "v", raw, false),
		types.NewField(0, pkg, "tail", types.NewArray(types.Typ[types.Byte], 0), false),
	}
	zero := types.NewNamed(types.NewTypeName(0, pkg, "Zero", nil), types.NewStruct(zeroFields, nil), nil)
	prog.SetTypeCallConv("p.Zero", CDecl, zeroFields...)
	if sizes.Offsetsof(zeroFields)[1] != 16 || sizes.Sizeof(zero) != 16 {
		t.Fatal("native zero-sized tail changed layout")
	}
	// Unregistered fields with the same names and NoPos retain Go layout.
	ordinary := []*types.Var{
		types.NewField(0, pkg, "v", raw, false),
		types.NewField(0, pkg, "f", NoArgsNoRet, false),
		types.NewField(0, pkg, "tail", types.Typ[types.Byte], false),
	}
	if got := sizes.Offsetsof(ordinary)[2]; got != 32 {
		t.Fatalf("ordinary callback tail offset = %d, want 32", got)
	}
}

func TestVectorMapDiagnostics(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "amd64"))
	pkg := types.NewPackage("p", "p")
	vector := types.NewNamed(types.NewTypeName(0, pkg, "V", nil), types.NewArray(types.Typ[types.Float32], 4), nil)
	prog.SetCVectorType("p.V", 4)
	array := types.NewArray(vector, 2)
	record := types.NewStruct([]*types.Var{types.NewField(0, pkg, "v", array, false)}, nil)
	for _, typ := range []types.Type{vector, types.NewAlias(types.NewTypeName(0, pkg, "Alias", nil), vector), array, record, types.NewArray(vector, 0)} {
		for _, key := range []bool{true, false} {
			m := types.NewMap(types.Typ[types.Int], typ)
			want := "map value"
			if key {
				m, want = types.NewMap(typ, types.Typ[types.Int]), "map key"
			}
			if err := prog.ValidateVectorMaps(m); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: %v", m, err)
			}
		}
	}
	for _, typ := range []types.Type{
		types.NewMap(types.NewPointer(vector), types.NewPointer(record)),
		types.NewMap(vector.Underlying(), vector.Underlying()),
	} {
		if err := prog.ValidateVectorMaps(typ); err != nil {
			t.Fatal(err)
		}
	}
}
