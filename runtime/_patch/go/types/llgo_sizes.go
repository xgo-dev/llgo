// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

// This file is also compiled by the LLGo compiler itself. Run go generate
// ./internal/typesizes after editing it; the generated adapter only changes
// the package clause and imports go/types. Keep layout rules in this file.

// LLGoSizes describes LLGo storage, including two-word Go function values and
// native C function pointers. Layout contains source declaration attributes;
// it must be populated before checking expressions containing unsafe builtins.
type LLGoSizes struct {
	WordSize, MaxAlign int64
	PointerSize        int64
	NativeAlign64      int64
	VectorAlign        [3]int64
	Layout             *LLGoLayout

	// These hooks supply compiler-created types that have no source declaration.
	NativeType  func(Type) bool
	NativeField func(*Var) bool
	// StorageLayout is used only after compiler type rewriting (for example
	// internal/abi runtime substitutions). Source clients should leave it nil.
	StorageLayout func(Type) (size, align int64, ok bool)
}

// LLGoSizesFor returns the LLGo layout for arch, or nil for an unknown target.
// Both "gc" and "llgo" select this model in LLGo's patched SizesFor.
func LLGoSizesFor(arch string) *LLGoSizes {
	s := &LLGoSizes{WordSize: 8, MaxAlign: 8, PointerSize: 8, NativeAlign64: 8, VectorAlign: [3]int64{16, 32, 64}}
	switch arch {
	case "amd64", "arm64", "loong64", "mips64", "mips64le", "ppc64", "ppc64le", "riscv64", "s390x", "sparc64", "wasm64":
	case "386", "arm", "mips", "mipsle", "riscv", "amd64p32":
		s.WordSize, s.PointerSize = 4, 4
		if arch == "386" {
			s.MaxAlign, s.NativeAlign64 = 4, 4
		}
	case "wasm":
		// LLGo's wasm profiles have 64-bit Go words even with Memory32.
		s.PointerSize = 4
	default:
		return nil
	}
	if arch == "arm64" || arch == "wasm" || arch == "wasm64" {
		s.VectorAlign = [3]int64{16, 16, 16}
	}
	return s
}

type llgoSizeAlign struct{ size, align int64 }

func (s *LLGoSizes) Sizeof(t Type) int64  { return s.typeLayout(t, false, false).size }
func (s *LLGoSizes) Alignof(t Type) int64 { return s.typeLayout(t, false, true).align }

func (s *LLGoSizes) Offsetsof(fields []*Var) []int64 {
	_, offsets := s.FieldsLayout(fields, false)
	return offsets
}

// FieldsLayout returns size/alignment and field offsets for a struct. Native
// selects C storage for fields; explicit declaration metadata is also honored.
func (s *LLGoSizes) FieldsLayout(fields []*Var, native bool) ([2]int64, []int64) {
	attrs := s.fieldAttrs(fields)
	native = native || attrs.Native || len(fields) > 0 && s.NativeField != nil && s.NativeField(fields[0])
	l, offsets := s.fieldsLayout(llgoFields{fields: fields}, native, attrs.Pack)
	return [2]int64{l.size, l.align}, offsets
}

func (s *LLGoSizes) fieldAttrs(fields []*Var) LLGoTypeAttrs {
	if s.Layout != nil && len(fields) != 0 {
		return s.Layout.StructAttrs(fields[0])
	}
	return LLGoTypeAttrs{}
}

func (s *LLGoSizes) typeLayout(t Type, native, alignmentOnly bool) llgoSizeAlign {
	if s.StorageLayout != nil {
		if size, align, ok := s.StorageLayout(t); ok {
			return llgoSizeAlign{size, align}
		}
	}
	if s.NativeType != nil && s.NativeType(t) {
		native = true
	}
	attrs := LLGoTypeAttrs{}
	if s.Layout != nil {
		attrs = s.Layout.TypeAttrs(t)
		native = native || attrs.Native
	}
	// Aliases are available starting in Go 1.22. Use their public interface so
	// that the source patch continues to build against older supported GOROOTs.
	if alias, ok := t.(interface{ Rhs() Type }); ok {
		return s.typeLayout(alias.Rhs(), native, alignmentOnly)
	}
	if n, ok := t.(*Named); ok {
		// A declared type owns its representation. A native enclosing struct
		// must not silently turn a named Go closure or pointer into C storage.
		native = attrs.Native || s.NativeType != nil && s.NativeType(t)
		if obj := n.Obj(); obj.Pkg() != nil {
			switch obj.Pkg().Path() {
			case "sync/atomic", "internal/runtime/atomic":
				if obj.Name() == "align64" {
					return llgoSizeAlign{0, 8}
				}
			case "simd/archsimd":
				index := -1
				switch obj.Name() {
				case "v128":
					index = 0
				case "v256":
					index = 1
				case "v512":
					index = 2
				}
				if index >= 0 {
					if st, ok := n.Underlying().(*Struct); ok && st.NumFields() == 1 {
						if a, ok := st.Field(0).Type().(*Array); ok && a.Len() == 0 {
							return llgoSizeAlign{0, s.VectorAlign[index]}
						}
					}
				}
			}
		}
	}
	word := s.WordSize
	if native {
		word = s.PointerSize
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		size := int64(0)
		switch u.Kind() {
		case Bool, Int8, Uint8:
			size = 1
		case Int16, Uint16:
			size = 2
		case Int32, Uint32, Float32:
			size = 4
		case Int64, Uint64, Float64, Complex64:
			size = 8
		case Complex128:
			size = 16
		case Invalid, Int, Uint:
			// Invalid types can occur during go/types error recovery. Keep its
			// word-sized fallback so the original diagnostic is not a panic.
			size = s.WordSize
		case Uintptr, UnsafePointer:
			size = word
		case String:
			return llgoSizeAlign{2 * s.WordSize, s.WordSize}
		default:
			panic("LLGoSizes: size of untyped basic type")
		}
		a := size
		if u.Info()&IsComplex != 0 {
			a /= 2
		}
		limit := s.MaxAlign
		if native && a == 8 {
			limit = s.NativeAlign64
		}
		if a > limit {
			a = limit
		}
		return llgoSizeAlign{size, a}
	case *Pointer, *Map, *Chan:
		return llgoSizeAlign{word, word}
	case *Signature:
		if native {
			return llgoSizeAlign{word, word}
		}
		return llgoSizeAlign{2 * word, word}
	case *Slice:
		return llgoSizeAlign{3 * s.WordSize, s.WordSize}
	case *Interface:
		return llgoSizeAlign{2 * s.WordSize, s.WordSize}
	case *Array:
		elem := s.typeLayout(u.Elem(), native, alignmentOnly)
		size := int64(0)
		if u.Len() > 0 {
			size = -1
			if elem.size >= 0 && (elem.size == 0 || u.Len() <= llgoMaxSize/elem.size) {
				size = u.Len() * elem.size
			}
		}
		return llgoSizeAlign{size, elem.align}
	case *Struct:
		fields := llgoFields{st: u}
		if fields.closure() {
			return llgoSizeAlign{2 * s.WordSize, s.WordSize}
		}
		fa := LLGoTypeAttrs{}
		if fields.len() > 0 {
			fa, _, _ = llgoTagLayout(u.Tag(0))
			if s.Layout != nil {
				if source := s.Layout.StructAttrs(fields.at(0)); source != (LLGoTypeAttrs{}) {
					fa = source
				}
			}
			native = native || fa.Native || s.NativeField != nil && s.NativeField(fields.at(0))
		}
		pack := attrs.Pack
		if pack == 0 {
			pack = fa.Pack
		}
		if alignmentOnly {
			a := int64(1)
			for i := 0; i < fields.len(); i++ {
				fa := s.typeLayout(fields.at(i).Type(), native, true).align
				if fa > a {
					a = fa
				}
			}
			if pack > 0 && a > pack {
				a = pack
			}
			return llgoSizeAlign{0, a}
		}
		l, _ := s.fieldsLayout(fields, native, pack)
		return l
	case *TypeParam, *Union:
		panic("LLGoSizes: layout of an uninstantiated type")
	}
	panic("LLGoSizes: unsupported type")
}

type llgoFields struct {
	st     *Struct
	fields []*Var
}

func (f llgoFields) len() int {
	if f.st != nil {
		return f.st.NumFields()
	}
	return len(f.fields)
}
func (f llgoFields) at(i int) *Var {
	if f.st != nil {
		return f.st.Field(i)
	}
	return f.fields[i]
}
func (f llgoFields) closure() bool {
	if f.len() != 2 || f.at(0).Name() != "$f" || f.at(1).Name() != "$data" {
		return false
	}
	_, ok := f.at(0).Type().(*Signature)
	return ok && f.at(1).Type() == Typ[UnsafePointer]
}

func (s *LLGoSizes) fieldsLayout(fields llgoFields, native bool, pack int64) (llgoSizeAlign, []int64) {
	if fields.closure() {
		return llgoSizeAlign{2 * s.WordSize, s.WordSize}, []int64{0, s.WordSize}
	}
	result := llgoSizeAlign{0, 1}
	offsets := make([]int64, fields.len())
	for i := 0; i < fields.len(); i++ {
		field := fields.at(i)
		l := s.typeLayout(field.Type(), native, false)
		a := l.align
		if pack > 0 && a > pack {
			a = pack
		}
		if a > result.align {
			result.align = a
		}
		result.size = llgoAlign(result.size, a)
		padding := ""
		if fields.st != nil {
			_, padding, _ = llgoTagLayout(fields.st.Tag(i))
		}
		if s.Layout != nil {
			if source := s.Layout.Padding(field); source != "" {
				padding = source
			}
		}
		conflict := false
		if padding == "padding-if" && l.size == 0 && result.size >= 0 {
			// Direct members and embedded bases participate, including non-adjacent
			// members. Recheck after shifting when more than two empty fields occur.
			for {
				found := false
				for j := 0; j < i; j++ {
					if offsets[j] == result.size && Identical(fields.at(j).Type(), field.Type()) {
						found = true
						break
					}
				}
				if !found {
					break
				}
				conflict = true
				result.size = llgoAlign(llgoAdd(result.size, 1), a)
				if result.size < 0 {
					break
				}
			}
		}
		offsets[i] = result.size
		result.size = llgoAdd(result.size, l.size)
		if l.size == 0 {
			interiorPadding := padding == "padding" && i+1 < fields.len()
			trailingPadding := i+1 == fields.len() && !native && result.size > 0 && padding != "no-padding" && padding != "padding-if"
			if interiorPadding || conflict || trailingPadding {
				result.size = llgoAdd(result.size, 1)
			}
		}
	}
	result.size = llgoAlign(result.size, result.align)
	return result, offsets
}

const llgoMaxSize int64 = 1<<63 - 1

func llgoAdd(x, y int64) int64 {
	if x < 0 || y < 0 || y > llgoMaxSize-x {
		return -1
	}
	return x + y
}

func llgoAlign(x, a int64) int64 {
	if x < 0 || x > llgoMaxSize-(a-1) {
		return -1
	}
	return (x + a - 1) &^ (a - 1)
}
