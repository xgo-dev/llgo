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

// Package simd describes Go SIMD types independently of LLVM lowering.
package simd

import (
	"fmt"
	"go/types"
)

// Kind identifies a canonical archsimd type's role. It does not select an
// instruction or a calling convention.
type Kind uint8

const (
	// Derived has the SIMD representation but no canonical vector/mask identity.
	// go/types does not retain the right-hand side of a defined type declaration:
	// `type M archsimd.Mask32x4` and `type V archsimd.Int32x4` have identical
	// underlying structs. Their representation remains known in both cases.
	Derived Kind = iota
	Vector
	Mask
)

// GoLayout describes addressable Go storage, in bytes. LLVM's default vector
// alignment and the representation used during computation must not replace it.
type GoLayout struct {
	Size  int64
	Align int64
}

// Descriptor describes one fixed-width, pointer-free, non-comparable SIMD type.
// Element is the Go storage lane kind, including signedness. Mask storage uses
// full-width signed integer lanes; a predicate value is a separate lowering
// decision and does not change this layout.
type Descriptor struct {
	WidthBits int
	Lanes     int
	Element   types.BasicKind
	Kind      Kind
	GoLayout  GoLayout
}

// Classifier recognizes vectors carrying markers from one verified archsimd
// package instance. Its zero value does not recognize any SIMD types.
type Classifier struct {
	pkg     *types.Package
	markers map[*types.Named]int
}

// NewClassifier checks the fixed-width type markers for goarch, which must be
// amd64, arm64, or wasm. The loader must first verify that archsimd was loaded
// from the selected toolchain's standard-library simd/archsimd source, with
// matching GOARCH and the SIMD experiment enabled. Package paths alone cannot
// prove source provenance. The same package instance must be shared throughout
// its importing type graph; a second package with the same path is not trusted.
//
// This constructor validates type representation only. It does not enable the
// experiment or claim that the target CPU or LLGo supports any SIMD operation.
func NewClassifier(goarch string, archsimd *types.Package) (*Classifier, error) {
	var widths []int
	switch goarch {
	case "amd64":
		widths = []int{128, 256, 512}
	case "arm64", "wasm":
		widths = []int{128}
	default:
		return nil, fmt.Errorf("simd: unsupported architecture %q", goarch)
	}
	if archsimd == nil || archsimd.Path() != "simd/archsimd" {
		return nil, fmt.Errorf("simd: expected verified simd/archsimd package")
	}
	c := &Classifier{pkg: archsimd, markers: make(map[*types.Named]int)}
	for _, width := range widths {
		name := fmt.Sprintf("v%d", width)
		obj, ok := archsimd.Scope().Lookup(name).(*types.TypeName)
		if !ok || obj.IsAlias() || obj.Pkg() != archsimd {
			return nil, fmt.Errorf("simd: missing canonical %s marker", name)
		}
		marker, ok := obj.Type().(*types.Named)
		if !ok || marker.Obj() != obj || !validMarker(marker, archsimd, width) {
			return nil, fmt.Errorf("simd: invalid %s marker", name)
		}
		c.markers[marker] = width
	}
	if goarch != "amd64" {
		for _, name := range []string{"v256", "v512"} {
			if archsimd.Scope().Lookup(name) != nil {
				return nil, fmt.Errorf("simd: %s marker is unavailable on %s", name, goarch)
			}
		}
	}
	return c, nil
}

func validMarker(marker *types.Named, pkg *types.Package, width int) bool {
	// These are the generated declarations in Go 1.27
	// src/simd/archsimd/types_{amd64,arm64,wasm}.go. A zero-length function
	// array reserves no bytes but makes the enclosing vector non-comparable.
	if marker.TypeParams().Len() != 0 {
		return false
	}
	st, ok := marker.Underlying().(*types.Struct)
	if !ok || st.NumFields() != 1 {
		return false
	}
	field := st.Field(0)
	if field.Name() != fmt.Sprintf("_%d", width) || field.Pkg() != pkg || field.Embedded() {
		return false
	}
	array, ok := types.Unalias(field.Type()).(*types.Array)
	if !ok || array.Len() != 0 {
		return false
	}
	fn, ok := types.Unalias(array.Elem()).(*types.Signature)
	return ok && fn.Recv() == nil && fn.TypeParams().Len() == 0 &&
		fn.Params().Len() == 0 && fn.Results().Len() == 0 && !fn.Variadic()
}

// Classify recognizes a canonical vector shape carrying one of c's markers.
// Aliases retain canonical identity. Defined types and unnamed structs that
// retain the true marker are recognized with Kind Derived. A similarly named
// type, a copied marker shape, and a containing aggregate are not SIMD types.
func (c *Classifier) Classify(typ types.Type) (Descriptor, bool) {
	if c == nil || typ == nil {
		return Descriptor{}, false
	}
	typ = types.Unalias(typ)
	st, ok := typ.Underlying().(*types.Struct)
	if !ok || st.NumFields() != 2 {
		return Descriptor{}, false
	}
	marker, ok := types.Unalias(st.Field(0).Type()).(*types.Named)
	if !ok {
		return Descriptor{}, false
	}
	width, ok := c.markers[marker]
	if !ok || st.Field(0).Embedded() || st.Field(1).Embedded() {
		return Descriptor{}, false
	}
	lanes, ok := types.Unalias(st.Field(1).Type()).(*types.Array)
	if !ok {
		return Descriptor{}, false
	}
	elem, ok := types.Unalias(lanes.Elem()).(*types.Basic)
	if !ok {
		return Descriptor{}, false
	}
	laneBits := scalarBits(elem.Kind())
	if laneBits == 0 || lanes.Len() != int64(width/laneBits) {
		return Descriptor{}, false
	}
	desc := Descriptor{
		WidthBits: width,
		Lanes:     int(lanes.Len()),
		Element:   elem.Kind(),
		Kind:      Derived,
		// Go's cmd/compile/internal/types/size.go simdify forces alignment
		// to 8, independently of lane kind or hardware register width.
		GoLayout: GoLayout{Size: int64(width / 8), Align: 8},
	}
	if named, ok := typ.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() == c.pkg && c.pkg.Scope().Lookup(obj.Name()) == obj {
			name := fmt.Sprintf("%sx%d", scalarName(elem.Kind()), desc.Lanes)
			if obj.Name() == name {
				desc.Kind = Vector
			} else if elem.Info()&types.IsInteger != 0 && elem.Info()&types.IsUnsigned == 0 &&
				obj.Name() == fmt.Sprintf("Mask%dx%d", laneBits, desc.Lanes) {
				desc.Kind = Mask
			}
		}
	}
	return desc, true
}

func scalarBits(kind types.BasicKind) int {
	switch kind {
	case types.Int8, types.Uint8:
		return 8
	case types.Int16, types.Uint16:
		return 16
	case types.Int32, types.Uint32, types.Float32:
		return 32
	case types.Int64, types.Uint64, types.Float64:
		return 64
	}
	return 0
}

func scalarName(kind types.BasicKind) string {
	switch kind {
	case types.Int8:
		return "Int8"
	case types.Int16:
		return "Int16"
	case types.Int32:
		return "Int32"
	case types.Int64:
		return "Int64"
	case types.Uint8:
		return "Uint8"
	case types.Uint16:
		return "Uint16"
	case types.Uint32:
		return "Uint32"
	case types.Uint64:
		return "Uint64"
	case types.Float32:
		return "Float32"
	case types.Float64:
		return "Float64"
	}
	return ""
}
