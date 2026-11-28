// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

// Package frontcache describes the frontend information needed from an
// already compiled dependency, independently of its instruction bodies.
package frontcache

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"strconv"
	"strings"
)

// Types binds exported type identities to this invocation's checked types.
// Package IDs, rather than import paths, distinguish test and alternate
// packages. Ambiguous identities fail closed instead of selecting a type.
type Types struct {
	packages map[*types.Package]string
	keys     map[types.Type]string
	values   map[string]types.Type
	added    map[types.Type]bool
	fset     *token.FileSet
}

func NewTypes(packages map[*types.Package]string, sets ...*token.FileSet) *Types {
	t := &Types{
		packages: packages,
		keys:     make(map[types.Type]string),
		values:   make(map[string]types.Type),
		added:    make(map[types.Type]bool),
	}
	if len(sets) != 0 {
		t.fset = sets[0]
	}
	return t
}

func (t *Types) qualifier(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	if id, ok := t.packages[pkg]; ok {
		return id
	}
	return pkg.Path()
}

// Key deliberately ignores parameter names and signature receivers, just as
// Go function-type identity does. Method receivers are exported separately.
// Named bodies are validated by the owning package's input fingerprint.
func (t *Types) Key(typ types.Type) string {
	if typ == nil {
		return ""
	}
	typ = types.Unalias(typ)
	if key, ok := t.keys[typ]; ok {
		return key
	}
	var b strings.Builder
	writeQuote := func(s string) {
		b.WriteString(strconv.Quote(s))
	}
	list := func(tuple *types.Tuple) {
		for i := 0; i < tuple.Len(); i++ {
			writeQuote(t.Key(tuple.At(i).Type()))
			b.WriteByte(';')
		}
	}
	switch v := typ.(type) {
	case *types.Basic:
		b.WriteString("basic:")
		b.WriteString(strconv.Itoa(int(v.Kind())))
	case *types.Named:
		b.WriteString("named:")
		writeQuote(t.qualifier(v.Obj().Pkg()))
		b.WriteByte(':')
		writeQuote(v.Obj().Name())
		b.WriteByte('[')
		for i := 0; i < v.TypeArgs().Len(); i++ {
			writeQuote(t.Key(v.TypeArgs().At(i)))
			b.WriteByte(';')
		}
		b.WriteByte(']')
		if obj := v.Obj(); obj.Pkg() != nil && obj.Parent() != nil && obj.Parent() != obj.Pkg().Scope() {
			b.WriteString("local:")
			writeQuote(Position(t.fset, obj.Pos()))
		}
	case *types.Pointer:
		if v.Kind() != types.GoPtr {
			b.WriteString("pointer:")
			b.WriteString(strconv.Itoa(int(v.Kind())))
			b.WriteByte(':')
			writeQuote(t.Key(v.Elem()))
		} else {
			b.WriteString("pointer:")
			writeQuote(t.Key(v.Elem()))
		}
	case *types.Slice:
		b.WriteString("slice:")
		writeQuote(t.Key(v.Elem()))
	case *types.Array:
		b.WriteString("array:")
		b.WriteString(strconv.FormatInt(v.Len(), 10))
		b.WriteByte(':')
		writeQuote(t.Key(v.Elem()))
	case *types.Map:
		b.WriteString("map:")
		writeQuote(t.Key(v.Key()))
		b.WriteByte(':')
		writeQuote(t.Key(v.Elem()))
	case *types.Chan:
		b.WriteString("chan:")
		b.WriteString(strconv.Itoa(int(v.Dir())))
		b.WriteByte(':')
		writeQuote(t.Key(v.Elem()))
	case *types.Signature:
		if v.CallConv() != types.Go {
			b.WriteString("func:")
			b.WriteString(strconv.Itoa(int(v.CallConv())))
			b.WriteByte(':')
			b.WriteString(strconv.FormatBool(v.Variadic()))
			b.WriteByte('[')
		} else {
			b.WriteString("func:")
			b.WriteString(strconv.FormatBool(v.Variadic()))
			b.WriteByte('[')
		}
		for i := 0; i < v.TypeParams().Len(); i++ {
			writeQuote(t.Key(v.TypeParams().At(i)))
			b.WriteByte(';')
		}
		b.WriteString("](")
		list(v.Params())
		b.WriteString(")(")
		list(v.Results())
		b.WriteByte(')')
	case *types.Tuple:
		b.WriteString("tuple:")
		list(v)
	case *types.Struct:
		if v.NativeLayout() {
			b.WriteString("struct:native:")
		} else {
			b.WriteString("struct:")
		}
		for i := 0; i < v.NumFields(); i++ {
			f := v.Field(i)
			owner := ""
			if !f.Exported() {
				owner = t.qualifier(f.Pkg())
			}
			writeQuote(owner)
			b.WriteByte(':')
			writeQuote(f.Name())
			b.WriteByte(':')
			b.WriteString(strconv.FormatBool(f.Embedded()))
			b.WriteByte(':')
			writeQuote(v.Tag(i))
			b.WriteByte(':')
			writeQuote(t.Key(f.Type()))
			b.WriteByte(';')
		}
	case *types.SIMDVec:
		b.WriteString("simd:")
		b.WriteString(strconv.Itoa(int(v.Elem().Kind())))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(v.Len()))
		b.WriteByte(':')
		b.WriteString(strconv.FormatBool(v.IsMask()))
	case *types.Interface:
		v.Complete()
		b.WriteString("interface:")
		for i := 0; i < v.NumMethods(); i++ {
			m := v.Method(i)
			owner := ""
			if !m.Exported() {
				owner = t.qualifier(m.Pkg())
			}
			writeQuote(owner)
			b.WriteByte(':')
			writeQuote(m.Name())
			b.WriteByte(':')
			writeQuote(t.Key(m.Type()))
			b.WriteByte(';')
		}
		// Non-basic interfaces cannot be runtime values. Retaining their
		// terms and comparability also prevents unrelated constraints (such as
		// predeclared comparable vs empty interface{}) from sharing a key.
		if !v.IsMethodSet() {
			if v.IsComparable() {
				b.WriteString("comparable;")
			}
			for i := 0; i < v.NumEmbeddeds(); i++ {
				b.WriteString("embed:")
				writeQuote(t.Key(v.EmbeddedType(i)))
				b.WriteByte(';')
			}
		}
	case *types.Union:
		b.WriteString("union:")
		for i := 0; i < v.Len(); i++ {
			b.WriteString(strconv.FormatBool(v.Term(i).Tilde()))
			b.WriteByte(':')
			writeQuote(t.Key(v.Term(i).Type()))
			b.WriteByte(';')
		}
	case *types.TypeParam:
		// Parameters are scoped to their enclosing generic declaration and are
		// never looked up globally in Types.Lookup. Omit token.Pos so
		// source-parsed and .llgotypes-loaded generic declarations produce
		// identical function and signature identities.
		b.WriteString("parameter:")
		writeQuote(t.qualifier(v.Obj().Pkg()))
		b.WriteByte(':')
		writeQuote(v.Obj().Name())
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(v.Index()))
	default:
		return ""
	}
	key := b.String()
	t.keys[typ] = key
	return key
}

// Add records all constituent types, including the receiver-free method-value
// signatures and pointer types that SSA may synthesize from declarations.
func (t *Types) Add(typ types.Type) {
	if t.added == nil {
		t.added = make(map[types.Type]bool)
	}
	t.add(typ, t.added)
}

func (t *Types) add(typ types.Type, seen map[types.Type]bool) {
	if typ == nil {
		return
	}
	typ = types.Unalias(typ)
	if seen[typ] {
		return
	}
	seen[typ] = true
	if !hasTypeParam(typ) {
		key := t.Key(typ)
		if previous, ok := t.values[key]; !ok {
			t.values[key] = typ
		} else if previous != nil && !types.Identical(previous, typ) {
			t.values[key] = nil
		}
	}
	add := func(v types.Type) { t.add(v, seen) }
	switch v := typ.(type) {
	case *types.Named:
		add(v.Underlying())
		pointer := types.NewPointer(v)
		ptrKey := t.Key(pointer)
		if _, ok := t.values[ptrKey]; !ok {
			t.values[ptrKey] = pointer
		}
		for i := 0; i < v.NumMethods(); i++ {
			add(v.Method(i).Type())
		}
		for i := 0; i < v.TypeArgs().Len(); i++ {
			add(v.TypeArgs().At(i))
		}
	case *types.Signature:
		if v.Recv() != nil {
			add(v.Recv().Type())
		}
		add(v.Params())
		add(v.Results())
	case *types.Tuple:
		for i := 0; i < v.Len(); i++ {
			add(v.At(i).Type())
		}
	case *types.Struct:
		for i := 0; i < v.NumFields(); i++ {
			add(v.Field(i).Type())
		}
	case *types.Interface:
		for i := 0; i < v.NumMethods(); i++ {
			add(v.Method(i).Type())
		}
	case *types.Pointer:
		add(v.Elem())
	case *types.Slice:
		add(v.Elem())
	case *types.Array:
		add(v.Elem())
	case *types.Chan:
		add(v.Elem())
	case *types.Map:
		add(v.Key())
		add(v.Elem())
	}
}

func (t *Types) AddPackage(pkg *types.Package, info *types.Info) {
	if t.added == nil {
		t.added = make(map[types.Type]bool)
	}
	seen := t.added
	if pkg != nil {
		for _, name := range pkg.Scope().Names() {
			t.add(pkg.Scope().Lookup(name).Type(), seen)
		}
	}
	if info == nil {
		return
	}
	for _, tv := range info.Types {
		t.add(tv.Type, seen)
	}
	for _, obj := range info.Defs {
		if obj != nil {
			t.add(obj.Type(), seen)
		}
	}
	for _, instance := range info.Instances {
		t.add(instance.Type, seen)
	}
}

func (t *Types) Lookup(key string) (types.Type, bool) {
	typ := t.values[key]
	return typ, typ != nil && key != ""
}

// Bind retains the exported identity of a private type imported without its
// original lexical scope. Package fingerprints validate its defining source.
func (t *Types) Bind(key string, typ types.Type) bool {
	if old, exists := t.values[key]; exists && (old == nil || !types.Identical(old, typ)) {
		return false
	}
	t.keys[typ], t.values[key] = key, typ
	return true
}

func (t *Types) PackageID(pkg *types.Package) string { return t.qualifier(pkg) }

// Position is stable across independent token.FileSets.
func Position(fset *token.FileSet, pos token.Pos) string {
	if fset == nil || !pos.IsValid() {
		return ""
	}
	f := fset.File(pos)
	if f == nil {
		return ""
	}
	return f.Name() + ":" + strconv.Itoa(f.Offset(pos))
}

func hasTypeParam(typ types.Type) bool {
	visited := make(map[types.Type]bool)
	var visit func(types.Type) bool
	visit = func(tt types.Type) bool {
		if tt == nil {
			return false
		}
		tt = types.Unalias(tt)
		if visited[tt] {
			return false
		}
		visited[tt] = true
		switch t := tt.(type) {
		case *types.TypeParam:
			return true
		case *types.Named:
			if tp := t.TypeParams(); tp != nil && tp.Len() > 0 {
				if ta := t.TypeArgs(); ta == nil || ta.Len() == 0 {
					return true
				}
			}
			if ta := t.TypeArgs(); ta != nil {
				for i := 0; i < ta.Len(); i++ {
					if visit(ta.At(i)) {
						return true
					}
				}
			}
			return visit(t.Underlying())
		case *types.Pointer:
			return visit(t.Elem())
		case *types.Slice:
			return visit(t.Elem())
		case *types.Array:
			return visit(t.Elem())
		case *types.Chan:
			return visit(t.Elem())
		case *types.Map:
			return visit(t.Key()) || visit(t.Elem())
		case *types.Signature:
			if tp := t.TypeParams(); tp != nil && tp.Len() > 0 {
				return true
			}
			if rtp := t.RecvTypeParams(); rtp != nil && rtp.Len() > 0 {
				return true
			}
			if t.Recv() != nil && visit(t.Recv().Type()) {
				return true
			}
			if params := t.Params(); params != nil {
				for i := 0; i < params.Len(); i++ {
					if visit(params.At(i).Type()) {
						return true
					}
				}
			}
			if results := t.Results(); results != nil {
				for i := 0; i < results.Len(); i++ {
					if visit(results.At(i).Type()) {
						return true
					}
				}
			}
			return false
		case *types.Tuple:
			for i := 0; i < t.Len(); i++ {
				if visit(t.At(i).Type()) {
					return true
				}
			}
			return false
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				if visit(t.Field(i).Type()) {
					return true
				}
			}
			return false
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				if visit(t.Method(i).Type()) {
					return true
				}
			}
			for i := 0; i < t.NumEmbeddeds(); i++ {
				if visit(t.EmbeddedType(i)) {
					return true
				}
			}
			return false
		case *types.Union:
			for i := 0; i < t.Len(); i++ {
				if visit(t.Term(i).Type()) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return visit(typ)
}
