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

package types

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	gotypes "go/types"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/directive"
)

// Object represents a named language entity (Func, Var, TypeName, Const,
// PkgName, Label, Builtin, Nil). Every Object speaks LLGo types natively.
type Object interface {
	Parent() *Scope
	Pos() token.Pos
	Pkg() *Package
	Name() string
	Type() Type
	Exported() bool
	Id() string
	String() string
	llObjectMarker()
}

// ObjectString returns the string representation of obj qualified by qf
// (mirroring go/types.ObjectString).
func ObjectString(obj Object, qf Qualifier) string {
	if obj == nil {
		return "<nil>"
	}
	switch o := obj.(type) {
	case *Func:
		var b strings.Builder
		b.WriteString("func ")
		if sig := o.Signature(); sig != nil && sig.recv != nil {
			b.WriteByte('(')
			b.WriteString(writeTypeString(sig.recv.typ, qf))
			b.WriteString(").")
		} else {
			writeQualPkg(&b, o.pkg, qf)
		}
		b.WriteString(o.name)
		if sig := o.Signature(); sig != nil {
			b.WriteString(strings.TrimPrefix(sig.String(), "func"))
		}
		return b.String()
	case *Var:
		kind := "var "
		if o.isField {
			kind = "field "
		}
		return kind + o.name + " " + writeTypeString(o.typ, qf)
	case *TypeName:
		if o.isAlias {
			return "type " + o.name + " = " + writeTypeString(o.typ, qf)
		}
		return "type " + o.name + " " + writeTypeString(o.typ, qf)
	case *Const:
		return "const " + o.name + " " + writeTypeString(o.typ, qf)
	default:
		return obj.String()
	}
}

type object struct {
	parent *Scope
	pos    token.Pos
	pkg    *Package
	name   string
	typ    Type
	goObj  gotypes.Object
}

func (o *object) Parent() *Scope { return o.parent }
func (o *object) Pos() token.Pos { return o.pos }
func (o *object) Pkg() *Package  { return o.pkg }
func (o *object) Name() string   { return o.name }
func (o *object) Type() Type     { return o.typ }
func (o *object) Exported() bool { return token.IsExported(o.name) }
func (o *object) llObjectMarker() {}

func (o *object) Id() string {
	return Id(o.pkg, o.name)
}

// Id returns the unique identifier for a name in pkg (mirroring go/types.Id).
func Id(pkg *Package, name string) string {
	if token.IsExported(name) {
		return name
	}
	path := "_"
	if pkg != nil && pkg.path != "" {
		path = pkg.path
	}
	return path + "." + name
}

// Func represents a declared function, method, or concrete closure body,
// carrying FreeVars and first-class LLGo link/directive metadata directly on
// the symbol.
type Func struct {
	object
	origin     *Func
	scope      *Scope
	freeVars   []*Var
	linkName   string
	callConv   CallConv
	isExternal bool
	isSkip     bool
	isPyList   bool
	wasmMod    string
	wasmName   string
	wasmExport string
}

// NewFunc constructs a function or method object (matching go/types.NewFunc).
func NewFunc(pos token.Pos, pkg *Package, name string, sig *Signature) *Func {
	cc := Go
	if sig != nil {
		cc = sig.callConv
	}
	return &Func{
		object:   object{pos: pos, pkg: pkg, name: name, typ: sig},
		callConv: cc,
	}
}

func (f *Func) Signature() *Signature {
	sig, _ := f.typ.(*Signature)
	return sig
}
func (f *Func) Origin() *Func {
	if f.origin != nil {
		return f.origin
	}
	return f
}
func (f *Func) Scope() *Scope           { return f.scope }
func (f *Func) FreeVars() []*Var        { return f.freeVars }
func (f *Func) SetFreeVars(vars []*Var) { f.freeVars = vars }
func (f *Func) LinkName() string        { return f.linkName }
func (f *Func) CallConv() CallConv      { return f.callConv }
func (f *Func) IsExternal() bool        { return f.isExternal }
func (f *Func) IsSkip() bool            { return f.isSkip }
func (f *Func) IsPyList() bool          { return f.isPyList }
func (f *Func) WasmExport() string      { return f.wasmExport }
func (f *Func) WasmImport() (string, string, bool) {
	if f.wasmMod == "" && f.wasmName == "" {
		return "", "", false
	}
	return f.wasmMod, f.wasmName, true
}
func (f *Func) SetLinkName(name string) { f.linkName = name }
func (f *Func) SetCallConv(cc CallConv) { f.callConv = cc }
func (f *Func) SetExternal(v bool)      { f.isExternal = v }
func (f *Func) SetSkip(v bool)          { f.isSkip = v }
func (f *Func) SetPyList(v bool)        { f.isPyList = v }

func (f *Func) FullName() string {
	var b strings.Builder
	sig := f.Signature()
	if sig != nil && sig.recv != nil && sig.recv.typ != nil {
		b.WriteByte('(')
		b.WriteString(sig.recv.typ.String())
		b.WriteString(").")
	} else if f.pkg != nil && f.pkg.path != "" {
		b.WriteString(f.pkg.path)
		b.WriteByte('.')
	}
	b.WriteString(f.name)
	return b.String()
}

func (f *Func) String() string {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(f.FullName())
	if sig := f.Signature(); sig != nil {
		b.WriteString(strings.TrimPrefix(sig.String(), "func"))
	}
	return b.String()
}

// Var represents a variable, parameter, result, or struct field.
type Var struct {
	object
	origin     *Var
	embedded   bool
	isField    bool
	linkName   string
	isExternal bool
}

// NewVar constructs a variable object (matching go/types.NewVar).
func NewVar(pos token.Pos, pkg *Package, name string, typ Type) *Var {
	return &Var{object: object{pos: pos, pkg: pkg, name: name, typ: typ}}
}

// NewParam constructs a function parameter or result variable (matching go/types.NewParam).
func NewParam(pos token.Pos, pkg *Package, name string, typ Type) *Var {
	return &Var{object: object{pos: pos, pkg: pkg, name: name, typ: typ}}
}

// NewField constructs a struct field variable (matching go/types.NewField).
func NewField(pos token.Pos, pkg *Package, name string, typ Type, embedded bool) *Var {
	return &Var{
		object:   object{pos: pos, pkg: pkg, name: name, typ: typ},
		embedded: embedded,
		isField:  true,
	}
}

func (v *Var) Origin() *Var {
	if v.origin != nil {
		return v.origin
	}
	return v
}
func (v *Var) Anonymous() bool      { return v.embedded }
func (v *Var) Embedded() bool       { return v.embedded }
func (v *Var) IsField() bool        { return v.isField }
func (v *Var) LinkName() string     { return v.linkName }
func (v *Var) IsExternal() bool     { return v.isExternal }
func (v *Var) SetLinkName(s string) { v.linkName = s }
func (v *Var) SetExternal(b bool)   { v.isExternal = b }
func (v *Var) String() string {
	kind := "var"
	if v.isField {
		kind = "field"
	}
	if v.typ != nil {
		return fmt.Sprintf("%s %s %s", kind, v.name, v.typ.String())
	}
	return fmt.Sprintf("%s %s", kind, v.name)
}

// TypeName represents a name for a defined type, type alias, or type parameter.
type TypeName struct {
	object
	isAlias      bool
	callConv     CallConv
	hasCallConv  bool
	nativeLayout bool
	hostLayout   bool
}

// NewTypeName constructs a TypeName object (matching go/types.NewTypeName).
func NewTypeName(pos token.Pos, pkg *Package, name string, typ Type) *TypeName {
	return &TypeName{object: object{pos: pos, pkg: pkg, name: name, typ: typ}}
}

func (tn *TypeName) IsAlias() bool             { return tn.isAlias }
func (tn *TypeName) CallConv() CallConv        { return tn.callConv }
func (tn *TypeName) HasExplicitCallConv() bool { return tn.hasCallConv }
func (tn *TypeName) NativeLayout() bool        { return tn.nativeLayout }
func (tn *TypeName) HostLayout() bool          { return tn.hostLayout }
func (tn *TypeName) String() string {
	if tn.typ != nil {
		return "type " + tn.name + " " + tn.typ.String()
	}
	return "type " + tn.name
}

// Const represents a declared constant.
type Const struct {
	object
	val constant.Value
}

// NewConst constructs a Const object (matching go/types.NewConst).
func NewConst(pos token.Pos, pkg *Package, name string, typ Type, val constant.Value) *Const {
	return &Const{object: object{pos: pos, pkg: pkg, name: name, typ: typ}, val: val}
}

func (c *Const) Val() constant.Value { return c.val }
func (c *Const) String() string {
	if c.val != nil {
		return fmt.Sprintf("const %s %s = %s", c.name, c.typ, c.val.ExactString())
	}
	return fmt.Sprintf("const %s %s", c.name, c.typ)
}

// PkgName represents an imported Go package binding in a file scope.
type PkgName struct {
	object
	imported *Package
}

func NewPkgName(pos token.Pos, pkg *Package, name string, imported *Package) *PkgName {
	return &PkgName{object: object{pos: pos, pkg: pkg, name: name}, imported: imported}
}

func (p *PkgName) Imported() *Package { return p.imported }
func (p *PkgName) String() string {
	if p.imported != nil {
		return fmt.Sprintf("package %s (%q)", p.name, p.imported.path)
	}
	return "package " + p.name
}

// Label represents a declared statement label.
type Label struct {
	object
}

func NewLabel(pos token.Pos, pkg *Package, name string) *Label {
	return &Label{object: object{pos: pos, pkg: pkg, name: name, typ: Typ[Invalid]}}
}

func (l *Label) String() string { return "label " + l.name }

// Builtin represents a predeclared Go built-in function.
type Builtin struct {
	object
}

func (b *Builtin) String() string { return "builtin " + b.name }

// Nil represents the predeclared value nil.
type Nil struct {
	object
}

func (n *Nil) String() string { return "nil" }

// Scope maintains a set of Objects declared in a lexical block or package.
type Scope struct {
	parent   *Scope
	children []*Scope
	elems    map[string]Object
	pos, end token.Pos
	comment  string
}

// NewScope creates a new Scope nested in parent (matching go/types.NewScope).
func NewScope(parent *Scope, pos, end token.Pos, comment string) *Scope {
	s := &Scope{
		parent:  parent,
		elems:   make(map[string]Object),
		pos:     pos,
		end:     end,
		comment: comment,
	}
	if parent != nil {
		parent.children = append(parent.children, s)
	}
	return s
}

func (s *Scope) Parent() *Scope     { return s.parent }
func (s *Scope) Len() int           { return len(s.elems) }
func (s *Scope) NumChildren() int   { return len(s.children) }
func (s *Scope) Child(i int) *Scope { return s.children[i] }
func (s *Scope) Pos() token.Pos     { return s.pos }
func (s *Scope) End() token.Pos     { return s.end }
func (s *Scope) Lookup(name string) Object {
	if s == nil {
		return nil
	}
	return s.elems[name]
}

// Names returns the scope's element names in sorted order.
func (s *Scope) Names() []string {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.elems))
	for name := range s.elems {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LookupParent searches s and enclosing scopes for name.
func (s *Scope) LookupParent(name string, pos token.Pos) (*Scope, Object) {
	for ; s != nil; s = s.parent {
		if obj := s.Lookup(name); obj != nil && (!pos.IsValid() || obj.Pos() <= pos) {
			return s, obj
		}
	}
	return nil, nil
}

// Insert attempts to insert obj into s. If an object with the same name already
// exists, Insert leaves s unchanged and returns the existing object.
func (s *Scope) Insert(obj Object) Object {
	name := obj.Name()
	if alt := s.elems[name]; alt != nil {
		return alt
	}
	s.elems[name] = obj
	setObjParent(obj, s)
	return nil
}

func setObjParent(obj Object, s *Scope) {
	switch o := obj.(type) {
	case *Func:
		o.parent = s
	case *Var:
		o.parent = s
	case *TypeName:
		o.parent = s
	case *Const:
		o.parent = s
	case *PkgName:
		o.parent = s
	case *Label:
		o.parent = s
	case *Builtin:
		o.parent = s
	case *Nil:
		o.parent = s
	}
}

// Universe is the predeclared Go universe scope speaking LLGo types natively
// (mirroring go/types.Universe).
var Universe = initUniverse()

func initUniverse() *Scope {
	u := NewScope(nil, token.NoPos, token.NoPos, "universe")
	for k := Bool; k <= String; k++ {
		b := Typ[k]
		u.Insert(NewTypeName(token.NoPos, nil, b.name, b))
	}
	// Predeclared aliases: byte = uint8, rune = int32, any = interface{}.
	u.Insert(NewAlias(NewTypeName(token.NoPos, nil, "byte", nil), Typ[Uint8]).Obj())
	u.Insert(NewAlias(NewTypeName(token.NoPos, nil, "rune", nil), Typ[Int32]).Obj())
	u.Insert(NewAlias(NewTypeName(token.NoPos, nil, "any", nil), NewInterfaceType(nil, nil)).Obj())
	// Predeclared error interface.
	errSig := NewSignature(nil, nil, NewTuple(NewParam(token.NoPos, nil, "", Typ[String])), false)
	errIface := NewInterfaceType([]*Func{NewFunc(token.NoPos, nil, "Error", errSig)}, nil)
	u.Insert(NewNamed(NewTypeName(token.NoPos, nil, "error", nil), errIface, nil).Obj())
	// Predeclared comparable interface.
	u.Insert(NewNamed(NewTypeName(token.NoPos, nil, "comparable", nil), NewInterfaceType(nil, nil), nil).Obj())
	// Predeclared constants & nil.
	u.Insert(NewConst(token.NoPos, nil, "true", Typ[UntypedBool], constant.MakeBool(true)))
	u.Insert(NewConst(token.NoPos, nil, "false", Typ[UntypedBool], constant.MakeBool(false)))
	u.Insert(NewConst(token.NoPos, nil, "iota", Typ[UntypedInt], constant.MakeInt64(0)))
	u.Insert(&Nil{object: object{name: "nil", typ: Typ[UntypedNil]}})
	// Predeclared builtins.
	for _, name := range []string{
		"append", "cap", "clear", "close", "complex", "copy", "delete",
		"imag", "len", "make", "max", "min", "new", "panic", "print",
		"println", "real", "recover",
	} {
		u.Insert(&Builtin{object: object{name: name, typ: Typ[Invalid]}})
	}
	return u
}

// Package describes an LLGo-typed Go package.
type Package struct {
	path     string
	name     string
	scope    *Scope
	complete bool
	imports  []*Package
	goPkg    *gotypes.Package
}

// NewPackage creates a new Package (matching go/types.NewPackage).
func NewPackage(path, name string) *Package {
	scope := NewScope(nil, token.NoPos, token.NoPos, fmt.Sprintf("package %q", path))
	return &Package{path: path, name: name, scope: scope}
}

func (p *Package) Path() string               { return p.path }
func (p *Package) Name() string               { return p.name }
func (p *Package) Scope() *Scope              { return p.scope }
func (p *Package) Complete() bool             { return p.complete }
func (p *Package) MarkComplete()              { p.complete = true }
func (p *Package) Imports() []*Package        { return p.imports }
func (p *Package) SetImports(list []*Package) { p.imports = list }
func (p *Package) Lookup(name string) Object {
	if p == nil || p.scope == nil {
		return nil
	}
	return p.scope.Lookup(name)
}

// PackageKind inspects p's package scope for an LLGoPackage constant and
// classifies the package using directive.ParsePackageKind.
func (p *Package) PackageKind() (directive.PackageKind, string) {
	if p == nil || p.scope == nil {
		return directive.PkgNormal, ""
	}
	if c, ok := p.scope.Lookup("LLGoPackage").(*Const); ok {
		if v := c.Val(); v != nil && v.Kind() == constant.String {
			return directive.ParsePackageKind(constant.StringVal(v))
		}
		return directive.PkgLLGo, ""
	}
	return directive.PkgNormal, ""
}

// LLGoFiles returns the string value of the LLGoFiles constant in p's package
// scope, if declared.
func (p *Package) LLGoFiles() (string, bool) {
	if p == nil || p.scope == nil {
		return "", false
	}
	if c, ok := p.scope.Lookup("LLGoFiles").(*Const); ok {
		if v := c.Val(); v != nil && v.Kind() == constant.String {
			return constant.StringVal(v), true
		}
	}
	return "", false
}

func (p *Package) String() string {
	return fmt.Sprintf("package %s (%q)", p.name, p.path)
}

// Selection describes a selector expression x.f in LLGo's type system.
type Selection struct {
	kind     SelectionKind
	recv     Type
	obj      Object
	typ      Type
	index    []int
	indirect bool
}

func (s *Selection) Kind() SelectionKind { return s.kind }
func (s *Selection) Recv() Type          { return s.recv }
func (s *Selection) Obj() Object         { return s.obj }
func (s *Selection) Type() Type          { return s.typ }
func (s *Selection) Index() []int        { return s.index }
func (s *Selection) Indirect() bool      { return s.indirect }

// MethodSet represents the ordered set of methods of a Type.
type MethodSet struct {
	list []*Selection
}

// NewMethodSet returns the method set for LLGo type T (matching go/types.NewMethodSet).
func NewMethodSet(T Type) *MethodSet {
	return methodSetOf(nil, T)
}

func (m *MethodSet) Len() int {
	if m == nil {
		return 0
	}
	return len(m.list)
}

func (m *MethodSet) At(i int) *Selection { return m.list[i] }

func (m *MethodSet) Lookup(pkg *Package, name string) *Selection {
	if m == nil {
		return nil
	}
	id := Id(pkg, name)
	for _, sel := range m.list {
		if sel.obj != nil && sel.obj.Id() == id {
			return sel
		}
	}
	return nil
}

// TypeAndValue reports the LLGo Type and constant value (if any) of an expression.
type TypeAndValue struct {
	Type        Type
	Value       constant.Value
	isType      bool
	isValue     bool
	isNil       bool
	addressable bool
	assignable  bool
	hasOk       bool
	isBuiltin   bool
}

func (tv TypeAndValue) IsType() bool      { return tv.isType }
func (tv TypeAndValue) IsValue() bool     { return tv.isValue }
func (tv TypeAndValue) IsNil() bool       { return tv.isNil }
func (tv TypeAndValue) Addressable() bool { return tv.addressable }
func (tv TypeAndValue) Assignable() bool  { return tv.assignable }
func (tv TypeAndValue) HasOk() bool       { return tv.hasOk }
func (tv TypeAndValue) IsBuiltin() bool   { return tv.isBuiltin }

// Initializer describes a package-level variable initialization.
type Initializer struct {
	Lhs []*Var
	Rhs ast.Expr
}

// Instance reports the LLGo type arguments and instantiated type for a generic
// identifier use (mirroring go/types.Instance).
type Instance struct {
	TypeArgs *TypeList
	Type     Type
}

// Info holds type-checked LLGo semantic information for a package.
// Every map and helper returns native LLGo Types and Objects with zero
// go/types.Type leakage.
type Info struct {
	Context    *Context
	Types      map[ast.Expr]TypeAndValue
	Defs       map[*ast.Ident]Object
	Uses       map[*ast.Ident]Object
	Implicits  map[ast.Node]Object
	Selections map[*ast.SelectorExpr]*Selection
	Scopes     map[ast.Node]*Scope
	InitOrder  []*Initializer
	Instances  map[*ast.Ident]Instance

	goInfo *gotypes.Info
}

// TypeOf returns the LLGo Type of expression e, or nil if unknown.
func (info *Info) TypeOf(e ast.Expr) Type {
	if info == nil || e == nil {
		return nil
	}
	if tv, ok := info.Types[e]; ok {
		return tv.Type
	}
	if id, ok := e.(*ast.Ident); ok {
		if obj := info.ObjectOf(id); obj != nil {
			return obj.Type()
		}
	}
	if info.goInfo != nil && info.Context != nil {
		if t := info.goInfo.TypeOf(e); t != nil {
			return info.Context.FromGo(t)
		}
	}
	return nil
}

// ObjectOf returns the LLGo Object denoted by ident (checking Defs then Uses).
func (info *Info) ObjectOf(id *ast.Ident) Object {
	if info == nil || id == nil {
		return nil
	}
	if obj := info.Defs[id]; obj != nil {
		return obj
	}
	if obj := info.Uses[id]; obj != nil {
		return obj
	}
	if info.goInfo != nil && info.Context != nil {
		if obj := info.goInfo.ObjectOf(id); obj != nil {
			return info.Context.FromGoObject(obj)
		}
	}
	return nil
}
