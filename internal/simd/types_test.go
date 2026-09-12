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

package simd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testTypes = `package archsimd
type v128 struct { _128 [0]func() }
type Float32x4 struct { float32x4 v128; vals [4]float32 }
type Int32x4 struct { int32x4 v128; vals [4]int32 }
type Mask32x4 struct { int32x4 v128; vals [4]int32 }
type VecAlias = Float32x4
type MaskAlias = Mask32x4
`

func checkTypes(t *testing.T, path, source string, importer types.Importer) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "types.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Importer: importer}
	pkg, err := conf.Check(path, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

type packageImporter struct{ pkg *types.Package }

func (i packageImporter) Import(path string) (*types.Package, error) {
	if path == i.pkg.Path() {
		return i.pkg, nil
	}
	return nil, fmt.Errorf("unexpected import %s", path)
}

func TestCanonicalAliasesAndDerivedTypes(t *testing.T) {
	pkg := checkTypes(t, "simd/archsimd", testTypes, nil)
	classifier, err := NewClassifier("arm64", pkg)
	if err != nil {
		t.Fatal(err)
	}
	consumer := checkTypes(t, "example.com/consumer", `package consumer
import "simd/archsimd"
type Vector archsimd.Float32x4
type Predicate archsimd.Mask32x4
type Integer archsimd.Int32x4
type Alias = archsimd.Float32x4
type MaskAlias = archsimd.Mask32x4
type Float32x4 archsimd.Float32x4
type Mask32x4 archsimd.Mask32x4
`, packageImporter{pkg})
	tests := []struct {
		pkg  *types.Package
		name string
		kind Kind
		elem types.BasicKind
	}{
		{pkg, "Float32x4", Vector, types.Float32},
		{pkg, "Int32x4", Vector, types.Int32},
		{pkg, "Mask32x4", Mask, types.Int32},
		{pkg, "VecAlias", Vector, types.Float32},
		{pkg, "MaskAlias", Mask, types.Int32},
		{consumer, "Vector", Derived, types.Float32},
		{consumer, "Predicate", Derived, types.Int32},
		{consumer, "Integer", Derived, types.Int32},
		{consumer, "Alias", Vector, types.Float32},
		{consumer, "MaskAlias", Mask, types.Int32},
		{consumer, "Float32x4", Derived, types.Float32},
		{consumer, "Mask32x4", Derived, types.Int32},
	}
	for _, test := range tests {
		t.Run(test.pkg.Path()+"/"+test.name, func(t *testing.T) {
			typ := test.pkg.Scope().Lookup(test.name).Type()
			want := Descriptor{WidthBits: 128, Lanes: 4, Element: test.elem, Kind: test.kind, GoLayout: GoLayout{16, 8}}
			if got, ok := classifier.Classify(typ); !ok || got != want {
				t.Fatalf("Classify(%s) = %+v, %v; want %+v", typ, got, ok, want)
			}
			if types.Comparable(typ) {
				t.Fatal("SIMD marker must preserve non-comparability")
			}
		})
	}
	mask := consumer.Scope().Lookup("Predicate").Type()
	integer := consumer.Scope().Lookup("Integer").Type()
	if !types.Identical(mask.Underlying(), integer.Underlying()) {
		t.Fatal("regression fixture no longer demonstrates lost mask RHS identity")
	}
	if got, ok := classifier.Classify(mask.Underlying()); !ok || got.Kind != Derived {
		t.Fatalf("unnamed marked struct = %+v, %v", got, ok)
	}
}

func TestRejectsUnrelatedAndMalformedTypes(t *testing.T) {
	pkg := checkTypes(t, "simd/archsimd", testTypes+`
type copyMarker struct { _128 [0]func() }
type BadCopy struct { tag copyMarker; vals [4]float32 }
type BadExtra struct { tag v128; vals [4]float32; extra int }
type BadOrder struct { vals [4]float32; tag v128 }
type BadWidth struct { tag v128; vals [8]float32 }
type BadNoLanes struct { tag v128; vals [0]float32 }
type BadInt struct { tag v128; vals [4]int }
type BadBool struct { tag v128; vals [16]bool }
type BadPointer struct { tag v128; vals [2]*int64 }
type BadComplex struct { tag v128; vals [2]complex64 }
type BadNested struct { tag v128; vals [4][4]byte }
type BadEmbedded struct { v128; vals [4]float32 }
type BadAggregate struct { v Float32x4 }
type Lane float32
type BadNamedLane struct { tag v128; vals [4]Lane }
`, nil)
	c, err := NewClassifier("wasm", pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range pkg.Scope().Names() {
		if strings.HasPrefix(name, "Bad") {
			t.Run(name, func(t *testing.T) {
				if got, ok := c.Classify(pkg.Scope().Lookup(name).Type()); ok {
					t.Fatalf("malformed vector accepted: %+v", got)
				}
			})
		}
	}
	// The same path, names, and shapes do not establish marker identity.
	for _, path := range []string{"simd/archsimd", "example.com/archsimd"} {
		other := checkTypes(t, path, testTypes, nil)
		if got, ok := c.Classify(other.Scope().Lookup("Float32x4").Type()); ok {
			t.Fatalf("foreign package %q accepted: %+v", path, got)
		}
	}
	for _, typ := range []types.Type{nil, types.Typ[types.Float32], pkg.Scope().Lookup("v128").Type(),
		types.NewPointer(pkg.Scope().Lookup("Float32x4").Type())} {
		if got, ok := c.Classify(typ); ok {
			t.Fatalf("non-vector %v accepted: %+v", typ, got)
		}
	}
	var zero Classifier
	if _, ok := zero.Classify(pkg.Scope().Lookup("Float32x4").Type()); ok {
		t.Fatal("zero classifier accepted vector")
	}
}

func TestRejectsInvalidMarkersAndTargets(t *testing.T) {
	for _, decl := range []string{
		`type v128 struct { _128 [1]func() }`,
		`type v128 struct { _128 [0]int }`,
		`type v128 struct { _128 [0]func(int) }`,
		`type v128 struct { _128 [0]func() int }`,
		`type v128 struct { _128 [0]func(...int) }`,
		`type v128 struct { _128 [0]func(); extra int }`,
		`type v128 struct { _256 [0]func() }`,
		`type v128 [0]func()`,
		`type v128 = struct { _128 [0]func() }`,
		`type v128[T any] struct { _128 [0]func() }`,
		`type v256 struct { _256 [0]func() }`,
	} {
		t.Run(decl, func(t *testing.T) {
			pkg := checkTypes(t, "simd/archsimd", "package archsimd\n"+decl, nil)
			if _, err := NewClassifier("arm64", pkg); err == nil {
				t.Fatal("invalid marker accepted")
			}
		})
	}
	pkg := checkTypes(t, "simd/archsimd", testTypes, nil)
	for _, arch := range []string{"", "386", "arm", "wasm32", "riscv64", "amd64"} {
		if _, err := NewClassifier(arch, pkg); err == nil {
			t.Fatalf("unsupported target or missing wide markers accepted: %s", arch)
		}
	}
	for _, pkg := range []*types.Package{nil, types.NewPackage("example.com/archsimd", "archsimd")} {
		if _, err := NewClassifier("arm64", pkg); err == nil {
			t.Fatal("unrelated package accepted")
		}
	}
	wide := checkTypes(t, "simd/archsimd", testTypes+`
type v256 struct { _256 [0]func() }
type v512 struct { _512 [0]func() }
`, nil)
	for _, arch := range []string{"arm64", "wasm"} {
		if _, err := NewClassifier(arch, wide); err == nil {
			t.Fatalf("wide package accepted on %s", arch)
		}
	}
}

// TestToolchainTypes reads the actual generated declarations, including targets
// different from the host. Type-only parsing avoids target assembly/intrinsics
// and makes this a source/type-layout audit, not an SIMD execution test.
func TestToolchainTypes(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "wasm"} {
		t.Run(arch, func(t *testing.T) {
			path := filepath.Join(runtime.GOROOT(), "src", "simd", "archsimd", "types_"+arch+".go")
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			decls := file.Decls[:0]
			for _, decl := range file.Decls {
				if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
					decls = append(decls, decl)
				}
			}
			file.Decls = decls
			conf := types.Config{Sizes: types.SizesFor("gc", arch)}
			pkg, err := conf.Check("simd/archsimd", fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			c, err := NewClassifier(arch, pkg)
			if err != nil {
				t.Fatal(err)
			}
			counts := make(map[int][2]int)
			for _, name := range pkg.Scope().Names() {
				if !ast.IsExported(name) {
					continue
				}
				typ := pkg.Scope().Lookup(name).Type()
				desc, ok := c.Classify(typ)
				if !ok || (desc.Kind != Vector && desc.Kind != Mask) {
					t.Fatalf("unclassified toolchain type %s: %+v, %v", typ, desc, ok)
				}
				count := counts[desc.WidthBits]
				if desc.Kind == Mask {
					count[1]++
				} else {
					count[0]++
				}
				counts[desc.WidthBits] = count
				layout := GoLayout{Size: conf.Sizes.Sizeof(typ), Align: conf.Sizes.Alignof(typ)}
				if desc.GoLayout != layout {
					t.Errorf("%s storage = %+v; go/types = %+v", typ, desc.GoLayout, layout)
				}
				// A SIMD field after a byte starts at offset 8 even for
				// 256/512-bit vectors; LLVM's natural vector alignment must
				// not move it to offset 32/64 in a containing Go struct.
				fields := []*types.Var{
					types.NewField(token.NoPos, nil, "head", types.Typ[types.Byte], false),
					types.NewField(token.NoPos, nil, "value", typ, false),
					types.NewField(token.NoPos, nil, "tail", types.Typ[types.Byte], false),
				}
				offsets := conf.Sizes.Offsetsof(fields)
				if offsets[1] != 8 || offsets[2] != 8+desc.GoLayout.Size ||
					conf.Sizes.Sizeof(types.NewStruct(fields, nil)) != desc.GoLayout.Size+16 {
					t.Errorf("%s containing struct layout: offsets=%v", typ, offsets)
				}
				if types.Comparable(typ) {
					t.Errorf("%s unexpectedly comparable", typ)
				}
			}
			widths := []int{128}
			if arch == "amd64" {
				widths = []int{128, 256, 512}
			}
			if len(counts) != len(widths) {
				t.Fatalf("toolchain widths = %v; want %v", counts, widths)
			}
			for _, width := range widths {
				if counts[width] != [2]int{10, 4} {
					t.Errorf("%d-bit vector/mask counts = %v; want 10/4", width, counts[width])
				}
			}
		})
	}
}
