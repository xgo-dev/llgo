// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

// Package frontcache describes the frontend information needed from an
// already compiled dependency, independently of its instruction bodies.
package frontcache

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

// Types binds exported type identities to this invocation's checked types.
// Package IDs, rather than import paths, distinguish test and alternate
// packages. Ambiguous identities fail closed instead of selecting a type.
type Types struct {
	packages map[*types.Package]string
	keys     map[types.Type]string
	values   map[string]types.Type
	fset     *token.FileSet
}

func NewTypes(packages map[*types.Package]string, sets ...*token.FileSet) *Types {
	t := &Types{packages: packages, keys: make(map[types.Type]string), values: make(map[string]types.Type)}
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
	list := func(tuple *types.Tuple) {
		for i := 0; i < tuple.Len(); i++ {
			fmt.Fprintf(&b, "%q;", t.Key(tuple.At(i).Type()))
		}
	}
	switch v := typ.(type) {
	case *types.Basic:
		fmt.Fprintf(&b, "basic:%d", v.Kind())
	case *types.Named:
		fmt.Fprintf(&b, "named:%q:%q[", t.qualifier(v.Obj().Pkg()), v.Obj().Name())
		for i := 0; i < v.TypeArgs().Len(); i++ {
			fmt.Fprintf(&b, "%q;", t.Key(v.TypeArgs().At(i)))
		}
		b.WriteByte(']')
		if obj := v.Obj(); obj.Pkg() != nil && obj.Parent() != nil && obj.Parent() != obj.Pkg().Scope() {
			fmt.Fprintf(&b, "local:%q", Position(t.fset, obj.Pos()))
		}
	case *types.Pointer:
		fmt.Fprintf(&b, "pointer:%q", t.Key(v.Elem()))
	case *types.Slice:
		fmt.Fprintf(&b, "slice:%q", t.Key(v.Elem()))
	case *types.Array:
		fmt.Fprintf(&b, "array:%d:%q", v.Len(), t.Key(v.Elem()))
	case *types.Map:
		fmt.Fprintf(&b, "map:%q:%q", t.Key(v.Key()), t.Key(v.Elem()))
	case *types.Chan:
		fmt.Fprintf(&b, "chan:%d:%q", v.Dir(), t.Key(v.Elem()))
	case *types.Signature:
		fmt.Fprintf(&b, "func:%v[", v.Variadic())
		for i := 0; i < v.TypeParams().Len(); i++ {
			fmt.Fprintf(&b, "%q;", t.Key(v.TypeParams().At(i)))
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
		b.WriteString("struct:")
		for i := 0; i < v.NumFields(); i++ {
			f := v.Field(i)
			owner := ""
			if !f.Exported() {
				owner = t.qualifier(f.Pkg())
			}
			fmt.Fprintf(&b, "%q:%q:%v:%q:%q;", owner, f.Name(), f.Embedded(), v.Tag(i), t.Key(f.Type()))
		}
	case *types.Interface:
		v.Complete()
		b.WriteString("interface:")
		for i := 0; i < v.NumMethods(); i++ {
			m := v.Method(i)
			owner := ""
			if !m.Exported() {
				owner = t.qualifier(m.Pkg())
			}
			fmt.Fprintf(&b, "%q:%q:%q;", owner, m.Name(), t.Key(m.Type()))
		}
		// Non-basic interfaces cannot be runtime values. Retaining their
		// terms also prevents unrelated constraints from sharing a key.
		if !v.IsMethodSet() {
			for i := 0; i < v.NumEmbeddeds(); i++ {
				fmt.Fprintf(&b, "embed:%q;", t.Key(v.EmbeddedType(i)))
			}
		}
	case *types.Union:
		b.WriteString("union:")
		for i := 0; i < v.Len(); i++ {
			fmt.Fprintf(&b, "%v:%q;", v.Term(i).Tilde(), t.Key(v.Term(i).Type()))
		}
	case *types.TypeParam:
		// Parameters are not persisted as concrete runtime types. This
		// identity is sufficient to bind their original declaration.
		fmt.Fprintf(&b, "parameter:%q:%q:%d:%q", t.qualifier(v.Obj().Pkg()), v.Obj().Name(), v.Index(), Position(t.fset, v.Obj().Pos()))
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
	t.add(typ, make(map[types.Type]bool))
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
	key := t.Key(typ)
	if previous, ok := t.values[key]; !ok {
		t.values[key] = typ
	} else if previous != nil && !types.Identical(previous, typ) {
		t.values[key] = nil
	}
	add := func(v types.Type) { t.add(v, seen) }
	switch v := typ.(type) {
	case *types.Named:
		add(v.Underlying())
		pointer := types.NewPointer(v)
		if _, ok := t.values[t.Key(pointer)]; !ok {
			t.values[t.Key(pointer)] = pointer
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
	seen := make(map[types.Type]bool)
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
	return fmt.Sprintf("%s:%d", f.Name(), f.Offset(pos))
}
