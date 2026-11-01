// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// LLGoTypeAttrs controls native representation and maximum field alignment.
// A zero Pack preserves the target's ordinary alignment rules.
type LLGoTypeAttrs struct {
	Native   bool
	Pack     int64
	CallConv string
}

type llgoDeclKey struct {
	pkg  *Package
	name string
}
type llgoFieldKey struct {
	pkg *Package
	pos token.Pos
}
type llgoTypeDecl struct {
	attrs   LLGoTypeAttrs
	ref     ast.Expr
	imports map[string][]string // import path -> local names; empty uses the package name
}

func (d llgoTypeDecl) importsAs(pkg *Package, name string) bool {
	for _, local := range d.imports[pkg.Path()] {
		if local == "" {
			local = pkg.Name()
		}
		if local == name {
			return true
		}
	}
	return false
}

// LLGoLayout retains attributes that go/types.Type does not encode. Keys
// include package identity: independent checkers may reuse token positions.
// Register syntax before type checking, and retain the same layout when
// checking importing packages. Processed field tags and generated metadata
// constants preserve these attributes through standard Go export data.
type LLGoLayout struct {
	metadataMu    sync.Mutex
	mu            sync.RWMutex
	types         map[llgoDeclKey]llgoTypeDecl
	structs       map[llgoFieldKey]llgoDeclKey
	attrs         map[llgoFieldKey]LLGoTypeAttrs
	padding       map[llgoFieldKey]string
	fieldAttrs    map[*Var]LLGoTypeAttrs
	fieldPadding  map[*Var]string
	packed        atomic.Bool
	nativeFuncs   map[string]*Named
	nativePkg     *Package
	recordedTypes map[Type]bool
}

const llgoTypeMetadataPrefix = "LLGoTypeLayout_"

// HasPacked reports whether code may access storage below its natural alignment.
func (m *LLGoLayout) HasPacked() bool {
	return m.packed.Load()
}

func llgoObjectName(t Type) *TypeName {
	if _, ok := t.(*TypeParam); ok {
		return nil
	}
	if n, ok := t.(interface{ Obj() *TypeName }); ok {
		return n.Obj()
	}
	return nil
}

func (m *LLGoLayout) TypeAttrs(t Type) LLGoTypeAttrs {
	attrs := LLGoTypeAttrs{}
	if st, ok := t.Underlying().(*Struct); ok && st.NumFields() > 0 {
		attrs, _, _ = llgoTagLayout(st.Tag(0))
	}
	if obj := llgoObjectName(t); obj != nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		source := m.typeAttrs(llgoDeclKey{obj.Pkg(), obj.Name()}, make(map[llgoDeclKey]bool))
		if source.Native {
			attrs.Native, attrs.CallConv = true, source.CallConv
		}
		if source.Pack != 0 {
			attrs.Pack = source.Pack
		}
	}
	return attrs
}

func (m *LLGoLayout) typeAttrs(key llgoDeclKey, seen map[llgoDeclKey]bool) LLGoTypeAttrs {
	if key.pkg != nil && key.pkg.Path() == "llgo.internal/native" {
		cc := "C"
		if strings.HasPrefix(key.name, "Stdcall_") {
			cc = "stdcall"
		}
		return LLGoTypeAttrs{Native: true, CallConv: cc}
	}
	if seen[key] {
		return LLGoTypeAttrs{}
	}
	seen[key] = true
	d := m.types[key]
	if key.pkg != nil {
		if c, ok := key.pkg.Scope().Lookup(llgoTypeMetadataPrefix + key.name).(*Const); ok && c.Val().Kind() == constant.String {
			if attrs, _, found := llgoTagLayout(constant.StringVal(c.Val())); found {
				d.attrs = attrs
			}
		}
	}
	inherited := LLGoTypeAttrs{}
	ref := d.ref
	for {
		switch r := ref.(type) {
		case *ast.ParenExpr:
			ref = r.X
		case *ast.IndexExpr:
			ref = r.X
		case *ast.IndexListExpr:
			ref = r.X
		default:
			goto resolved
		}
	}
resolved:
	switch ref := ref.(type) {
	case *ast.Ident:
		inherited = m.typeAttrs(llgoDeclKey{key.pkg, ref.Name}, seen)
		if inherited == (LLGoTypeAttrs{}) && key.pkg != nil {
			for _, pkg := range key.pkg.Imports() {
				if d.importsAs(pkg, ".") {
					inherited = m.typeAttrs(llgoDeclKey{pkg, ref.Name}, seen)
					if inherited != (LLGoTypeAttrs{}) {
						break
					}
				}
			}
		}
	case *ast.SelectorExpr:
		if id, ok := ref.X.(*ast.Ident); ok && key.pkg != nil {
			for _, pkg := range key.pkg.Imports() {
				if d.importsAs(pkg, id.Name) {
					inherited = m.typeAttrs(llgoDeclKey{pkg, ref.Sel.Name}, seen)
					break
				}
			}
		}
	}
	inherited.Native = inherited.Native || d.attrs.Native
	if d.attrs.CallConv != "" {
		inherited.CallConv = d.attrs.CallConv
	}
	if d.attrs.Pack != 0 {
		inherited.Pack = d.attrs.Pack
	}
	return inherited
}

func (m *LLGoLayout) StructAttrs(field *Var) LLGoTypeAttrs {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if attrs, ok := m.fieldAttrs[field]; ok {
		return attrs
	}
	key := llgoFieldKey{field.Pkg(), field.Pos()}
	attrs := m.attrs[key]
	if decl, ok := m.structs[key]; ok {
		inherited := m.typeAttrs(decl, make(map[llgoDeclKey]bool))
		attrs.Native = attrs.Native || inherited.Native
		if inherited.Pack != 0 {
			attrs.Pack = inherited.Pack
		}
	}
	return attrs
}

func (m *LLGoLayout) Padding(field *Var) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if padding, ok := m.fieldPadding[field]; ok {
		return padding
	}
	return m.padding[llgoFieldKey{field.Pkg(), field.Pos()}]
}

// RecordType restores exported metadata without needing source comments or
// source positions. It also supports synthetic structs built by clients.
func (m *LLGoLayout) RecordType(t Type) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fieldAttrs == nil {
		m.fieldAttrs = make(map[*Var]LLGoTypeAttrs)
		m.fieldPadding = make(map[*Var]string)
	}
	if m.recordedTypes == nil {
		m.recordedTypes = make(map[Type]bool)
	}
	seen := m.recordedTypes
	var visit func(Type)
	visit = func(t Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch t := t.Underlying().(type) {
		case *Struct:
			for i := 0; i < t.NumFields(); i++ {
				f := t.Field(i)
				if attrs, pad, ok := llgoTagLayout(t.Tag(i)); ok {
					m.fieldAttrs[f], m.fieldPadding[f] = attrs, pad
					if attrs.Pack != 0 {
						m.packed.Store(true)
					}
				}
				visit(f.Type())
			}
		case *Array:
			visit(t.Elem())
		case *Slice:
			visit(t.Elem())
		case *Pointer:
			visit(t.Elem())
		case *Chan:
			visit(t.Elem())
		case *Map:
			visit(t.Key())
			visit(t.Elem())
		case *Signature:
			for _, tuple := range []*Tuple{t.Params(), t.Results()} {
				for i := 0; i < tuple.Len(); i++ {
					visit(tuple.At(i).Type())
				}
			}
		}
	}
	visit(t)
}

func (m *LLGoLayout) RecordPackage(pkg *Package) {
	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		m.RecordType(obj.Type())
		if _, ok := obj.(*TypeName); ok {
			attrs := m.TypeAttrs(obj.Type())
			if attrs != (LLGoTypeAttrs{}) && pkg.Scope().Lookup(llgoTypeMetadataPrefix+name) == nil {
				pkg.Scope().Insert(NewConst(token.NoPos, pkg, llgoTypeMetadataPrefix+name, Typ[String], constant.MakeString(llgoLayoutTag("", attrs, ""))))
			}
		}
	}
}

type llgoDirective struct {
	name, args string
	pos        token.Pos
}

func llgoDirectives(group *ast.CommentGroup) (out []llgoDirective) {
	if group == nil {
		return
	}
	for _, c := range group.List {
		if !strings.HasPrefix(c.Text, "//") {
			continue
		}
		parts := strings.Fields(strings.TrimPrefix(c.Text, "//"))
		if len(parts) != 0 {
			out = append(out, llgoDirective{parts[0], strings.Join(parts[1:], " "), c.Pos()})
		}
	}
	return
}

func llgoLayoutDirective(name string) bool {
	switch name {
	case "llgo:padding", "llgo:no-padding", "llgo:padding-if", "llgo:pack", "llgo:packed":
		return true
	}
	return false
}

// ParseSyntax records layout attributes before any unsafe constant is folded.
// warn may be nil. Errors refer to the original source comment positions.
func (m *LLGoLayout) ParseSyntax(fset *token.FileSet, pkg *Package, files []*ast.File, warn func(error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.types == nil {
		m.types = make(map[llgoDeclKey]llgoTypeDecl)
		m.structs = make(map[llgoFieldKey]llgoDeclKey)
		m.attrs = make(map[llgoFieldKey]LLGoTypeAttrs)
		m.padding = make(map[llgoFieldKey]string)
	}
	used := make(map[token.Pos]bool)
	errAt := func(d llgoDirective, message string) error {
		return fmt.Errorf("%s: //%s %s", fset.Position(d.pos), d.name, message)
	}
	nativeVariables := make(map[string]string)
	for _, file := range files {
		for _, group := range file.Comments {
			for _, d := range llgoDirectives(group) {
				if d.name != "go:linkname" {
					continue
				}
				words := strings.Fields(d.args)
				if len(words) == 2 {
					if strings.HasPrefix(words[1], "C.") {
						nativeVariables[words[0]] = "C"
					}
					if strings.HasPrefix(words[1], "stdcall.") {
						nativeVariables[words[0]] = "stdcall"
					}
				}
			}
		}
	}
	for _, file := range files {
		imports := make(map[string][]string)
		for _, imp := range file.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			}
			// Retain every file-local binding, including multiple names for one path.
			imports[path] = append(imports[path], name)
		}
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && (strings.HasPrefix(fd.Name.Name, llgoTypeMetadataPrefix) || strings.HasPrefix(fd.Name.Name, "__llgo_native_")) {
				return fmt.Errorf("%s: %s is reserved for LLGo layout metadata", fset.Position(fd.Pos()), fd.Name.Name)
			}
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			// These declarations are emitted in package scope by the compiler.
			// Reject source collisions before injecting any helper definitions.
			for _, spec := range gd.Specs {
				var names []*ast.Ident
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					names = []*ast.Ident{spec.Name}
				case *ast.ValueSpec:
					names = spec.Names
				}
				for _, name := range names {
					if strings.HasPrefix(name.Name, llgoTypeMetadataPrefix) || strings.HasPrefix(name.Name, "__llgo_native_") {
						return fmt.Errorf("%s: %s is reserved for LLGo layout metadata", fset.Position(name.Pos()), name.Name)
					}
				}
			}
			if gd.Tok == token.VAR {
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					for _, name := range vs.Names {
						if cc := nativeVariables[name.Name]; cc != "" && vs.Type != nil {
							if len(vs.Names) > 1 && llgoHasAnonymousCallback(vs.Type) {
								return fmt.Errorf("%s: declare each C-linked callback variable separately", fset.Position(name.Pos()))
							}
							vs.Type = m.nativeFunctionExpr(pkg, vs.Type, cc)
						}
					}
				}
			}
			if gd.Tok != token.TYPE {
				continue
			}
			cc, err := LLGoTypeDeclCallConv(fset, gd)
			if err != nil {
				return err
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				key := llgoDeclKey{pkg, ts.Name.Name}
				d := llgoTypeDecl{ref: ts.Type, imports: imports, attrs: LLGoTypeAttrs{Native: cc.BG != "", CallConv: cc.BG}}
				dirs := llgoDirectives(ts.Doc)
				if len(gd.Specs) == 1 {
					dirs = append(llgoDirectives(gd.Doc), dirs...)
				}
				for _, directive := range dirs {
					switch directive.name {
					case "llgo:packed", "llgo:pack":
						if d.attrs.Pack != 0 {
							return errAt(directive, "conflicts with another pack directive")
						}
						pack := int64(1)
						if directive.name == "llgo:pack" {
							pack, _ = strconv.ParseInt(directive.args, 10, 64)
						}
						if directive.name == "llgo:packed" && directive.args != "" || pack < 1 || pack > 16 || pack&(pack-1) != 0 {
							return errAt(directive, "requires a power-of-two alignment from 1 to 16 (packed takes no argument)")
						}
						if _, ok := ts.Type.(*ast.StructType); !ok {
							return errAt(directive, "requires a struct type declaration")
						}
						d.attrs.Pack = pack
						m.packed.Store(true)
						used[directive.pos] = true
					}
				}
				m.types[key] = d
				// An annotation that changes an alias's representation creates a
				// distinct native type. An ordinary alias keeps the target identity.
				if d.attrs.Native && ts.Assign.IsValid() {
					ts.Assign = token.NoPos
				}
				generic := ts.TypeParams != nil && len(ts.TypeParams.List) != 0
				var parseErr error
				llgoInspect(ts.Type, func(node ast.Node) bool {
					st, ok := node.(*ast.StructType)
					if !ok || parseErr != nil {
						return parseErr == nil
					}
					owner, attrs := key, d.attrs
					if node != ts.Type {
						// Packing caps the outer field's alignment; it does not
						// change the internal layout of its anonymous struct type.
						owner.name, attrs.Pack = "", 0
					}
					if parseErr = m.parseStruct(st, owner, attrs, generic, used, errAt, warn); parseErr != nil {
						return false
					}
					return true
				})
				if parseErr != nil {
					return parseErr
				}
			}
		}
		// Anonymous and function-local structs also have field directives.
		var parseErr error
		llgoInspect(file, func(node ast.Node) bool {
			if parseErr != nil {
				return false
			}
			st, ok := node.(*ast.StructType)
			if !ok || len(st.Fields.List) == 0 {
				return true
			}
			pos := llgoFieldPositions(st.Fields.List[0])[0]
			if _, exists := m.attrs[llgoFieldKey{pkg, pos}]; exists {
				return true
			}
			parseErr = m.parseStruct(st, llgoDeclKey{pkg, ""}, LLGoTypeAttrs{}, false, used, errAt, warn)
			return parseErr == nil
		})
		if parseErr != nil {
			return parseErr
		}
		for _, group := range file.Comments {
			for _, d := range llgoDirectives(group) {
				if llgoLayoutDirective(d.name) && !used[d.pos] {
					return errAt(d, "is not on a valid struct type or field declaration")
				}
			}
		}
	}
	return nil
}

func (m *LLGoLayout) parseStruct(st *ast.StructType, owner llgoDeclKey, attrs LLGoTypeAttrs, generic bool, used map[token.Pos]bool,
	errAt func(llgoDirective, string) error, warn func(error)) error {
	for i, field := range st.Fields.List {
		if attrs.Native {
			field.Type = m.nativeFunctionExpr(owner.pkg, field.Type, attrs.CallConv)
		}
		padding := ""
		for _, d := range llgoDirectives(field.Doc) {
			switch d.name {
			case "llgo:padding", "llgo:no-padding", "llgo:padding-if":
				if d.args != "" {
					return errAt(d, "does not accept arguments")
				}
				if padding != "" {
					return errAt(d, "conflicts with another padding directive")
				}
				if d.name == "llgo:no-padding" && (generic || i+1 != len(st.Fields.List) || len(field.Names) > 1) {
					return errAt(d, "requires the last field of a non-generic struct")
				}
				if d.name == "llgo:padding-if" && !generic {
					return errAt(d, "requires a field of a generic struct")
				}
				if d.name == "llgo:padding" && i+1 == len(st.Fields.List) && warn != nil {
					warn(errAt(d, "on the last field has no effect"))
				}
				padding = strings.TrimPrefix(d.name, "llgo:")
				used[d.pos] = true
			}
		}
		for _, pos := range llgoFieldPositions(field) {
			key := llgoFieldKey{owner.pkg, pos}
			m.attrs[key] = attrs
			if owner.name != "" {
				m.structs[key] = owner
			}
			m.padding[key] = padding
		}
		tag := ""
		if field.Tag != nil {
			tag, _ = strconv.Unquote(field.Tag.Value)
		}
		if encoded := llgoLayoutTag(tag, attrs, padding); encoded != tag {
			field.Tag = &ast.BasicLit{ValuePos: field.End(), Kind: token.STRING, Value: strconv.Quote(encoded)}
		}
	}
	return nil
}

func llgoFieldPositions(field *ast.Field) []token.Pos {
	if len(field.Names) != 0 {
		positions := make([]token.Pos, len(field.Names))
		for i, name := range field.Names {
			positions[i] = name.Pos()
		}
		return positions
	}
	expr := field.Type
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.SelectorExpr:
			return []token.Pos{e.Sel.Pos()}
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		default:
			return []token.Pos{expr.Pos()}
		}
	}
}

// LLGoCallConv is the source-level calling convention shared by the compiler
// and the source patch. Parsing never constructs LLVM types.
type LLGoCallConv struct {
	BG, Name, Replacement string
	Pos                   token.Pos
}

func LLGoTypeCallConv(fset *token.FileSet, doc *ast.CommentGroup) (cc LLGoCallConv, err error) {
	seen := false
	for _, d := range llgoDirectives(doc) {
		var bg, name, replacement string
		switch d.name {
		case "llgo:cdecl":
			if d.args != "" {
				return cc, fmt.Errorf("%s: //llgo:cdecl does not accept arguments", fset.Position(d.pos))
			}
			bg, name = "C", "//llgo:cdecl"
		case "llgo:stdcall":
			if d.args != "" {
				return cc, fmt.Errorf("%s: //llgo:stdcall does not accept arguments", fset.Position(d.pos))
			}
			bg, name = "stdcall", "//llgo:stdcall"
		case "llgo:type":
			// //llgo:type has other forms beyond the two calling-convention
			// values; only C and stdcall are affected by this task.
			switch d.args {
			case "C":
				bg, name, replacement = "C", "//llgo:type C", "//llgo:cdecl"
			case "stdcall":
				bg, name, replacement = "stdcall", "//llgo:type stdcall", "//llgo:stdcall"
			default:
				continue
			}
		default:
			continue
		}
		if seen {
			return cc, fmt.Errorf("%s: conflicting calling-convention directives on one type: %s and %s", fset.Position(d.pos), cc.Name, name)
		}
		cc = LLGoCallConv{BG: bg, Name: name, Pos: d.pos, Replacement: replacement}
		seen = true
	}
	return cc, nil
}

// LLGoTypeDeclCallConv validates type-declaration placement in one place.
func LLGoTypeDeclCallConv(fset *token.FileSet, decl *ast.GenDecl) (LLGoCallConv, error) {
	doc := decl.Doc
	if len(decl.Specs) > 1 {
		for _, spec := range decl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				cc, err := LLGoTypeCallConv(fset, ts.Doc)
				if err != nil {
					return cc, err
				}
				if cc.BG != "" {
					return cc, fmt.Errorf("%s: %s applies to a single type declaration, not a grouped type block", fset.Position(cc.Pos), cc.Name)
				}
			}
		}
	}
	if len(decl.Specs) == 1 {
		if spec, ok := decl.Specs[0].(*ast.TypeSpec); ok && spec.Doc != nil {
			merged := &ast.CommentGroup{}
			if doc != nil {
				merged.List = append(merged.List, doc.List...)
			}
			merged.List = append(merged.List, spec.Doc.List...)
			doc = merged
		}
	}
	cc, err := LLGoTypeCallConv(fset, doc)
	if err != nil || cc.BG == "" {
		return cc, err
	}
	if len(decl.Specs) != 1 {
		return cc, fmt.Errorf("%s: %s applies to a single type declaration, not a grouped type block", fset.Position(cc.Pos), cc.Name)
	}
	if _, ok := decl.Specs[0].(*ast.TypeSpec); !ok {
		return cc, fmt.Errorf("%s: %s applies to a type declaration", fset.Position(cc.Pos), cc.Name)
	}
	return cc, nil
}

// llgoInspect visits nodes in preorder, with the same subtree pruning as
// ast.Inspect. Only direct children use ast.Walk: recursive Walk frames are
// large enough to exhaust a default Wasm goroutine stack on a short type.
func llgoInspect(root ast.Node, visit func(ast.Node) bool) {
	stack := []ast.Node{root}
	for len(stack) != 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !visit(node) {
			continue
		}
		start := len(stack)
		ast.Inspect(node, func(child ast.Node) bool {
			if child == node {
				return true
			}
			if child != nil {
				stack = append(stack, child)
			}
			return false
		})
		for i, j := start, len(stack)-1; i < j; i, j = i+1, j-1 {
			stack[i], stack[j] = stack[j], stack[i]
		}
	}
}
