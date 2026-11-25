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

// Package ssa defines a pure-Go SSA intermediate representation whose core
// interfaces, types, methods, and 37 instruction kinds align 1:1 with
// golang.org/x/tools/go/ssa while operating on github.com/xgo-dev/llgo/types
// and requiring only Go 1.20+ with zero CGo, LLVM, or x/tools dependencies.
package ssa

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"io"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llgo/types/typeutil"
)

// BuilderMode is a bitmask of options for diagnostics and checking, matching
// golang.org/x/tools/go/ssa.BuilderMode.
type BuilderMode uint

const (
	PrintPackages BuilderMode = 1 << iota
	PrintFunctions
	LogSource
	SanityCheckFunctions
	NaiveForm
	BuildSerially
	GlobalDebug
	BareInits
	InstantiateGenerics
)

// BuilderModeDoc documents BuilderMode flag letters (matching x/tools/go/ssa).
const BuilderModeDoc = `Options controlling the SSA builder.
The value is a sequence of zero or more of these letters:
C	perform sanity [C]hecking of the SSA form.
D	include [D]ebug info for every function.
P	print [P]ackage inventory.
F	print [F]unction SSA code.
S	log [S]ource locations as SSA builder progresses.
L	build distinct packages seria[L]ly instead of in parallel.
N	build [N]aive SSA form: don't replace local loads/stores with registers.
I	build bare [I]nit functions: no init guards or calls to dependent inits.
G	instantiate [G]eneric function bodies via monomorphization.
`

func (m BuilderMode) String() string {
	var b strings.Builder
	if m&GlobalDebug != 0 {
		b.WriteByte('D')
	}
	if m&PrintPackages != 0 {
		b.WriteByte('P')
	}
	if m&PrintFunctions != 0 {
		b.WriteByte('F')
	}
	if m&LogSource != 0 {
		b.WriteByte('S')
	}
	if m&SanityCheckFunctions != 0 {
		b.WriteByte('C')
	}
	if m&NaiveForm != 0 {
		b.WriteByte('N')
	}
	if m&BuildSerially != 0 {
		b.WriteByte('L')
	}
	if m&BareInits != 0 {
		b.WriteByte('I')
	}
	if m&InstantiateGenerics != 0 {
		b.WriteByte('G')
	}
	return b.String()
}

func (m BuilderMode) Get() interface{} { return m }

func (m *BuilderMode) Set(s string) error {
	var mode BuilderMode
	for _, c := range s {
		switch c {
		case 'D':
			mode |= GlobalDebug
		case 'P':
			mode |= PrintPackages
		case 'F':
			mode |= PrintFunctions
		case 'S':
			mode |= LogSource
		case 'C':
			mode |= SanityCheckFunctions
		case 'N':
			mode |= NaiveForm
		case 'L':
			mode |= BuildSerially
		case 'I':
			mode |= BareInits
		case 'G':
			mode |= InstantiateGenerics
		default:
			return fmt.Errorf("unknown BuilderMode option: %q", c)
		}
	}
	*m = mode
	return nil
}

// Node is the common interface of Values and Instructions.
type Node interface {
	String() string
	Pos() token.Pos
	Parent() *Function
	Operands(rands []*Value) []*Value
	Referrers() *[]Instruction
}

// Value is an SSA value that can be referenced by an instruction.
type Value interface {
	Name() string
	String() string
	Type() types.Type
	Parent() *Function
	Referrers() *[]Instruction
	Pos() token.Pos
}

// Instruction is an SSA instruction that belongs to a BasicBlock.
type Instruction interface {
	String() string
	Parent() *Function
	Block() *BasicBlock
	Operands(rands []*Value) []*Value
	Pos() token.Pos
}

// CallInstruction is the common interface of Call, Defer, and Go.
type CallInstruction interface {
	Instruction
	Common() *CallCommon
	Value() *Call
}

// Member is a member of a Go package (Function, Global, NamedConst, or Type).
type Member interface {
	Name() string
	String() string
	RelString(*types.Package) string
	Object() types.Object
	Pos() token.Pos
	Type() types.Type
	Token() token.Token
	Package() *Package
}

// Program is a partial or complete Go program converted to SSA form.
type Program struct {
	Fset       *token.FileSet
	MethodSets typeutil.MethodSetCache
	Target     target.Spec // LLGo extension: target ABI specification

	mode         BuilderMode
	packages     map[*types.Package]*Package
	imported     map[string]*Package
	funcsByObj   map[*types.Func]*Function
	methods      map[string]*Function // "<type>|<pkg>|<name>" -> *Function
	runtimeTypes typeutil.Map
	noReturn     func(*types.Func) bool
}

// NewProgram creates a new SSA Program (matching x/tools/go/ssa.NewProgram).
func NewProgram(fset *token.FileSet, mode BuilderMode) *Program {
	if fset == nil {
		fset = token.NewFileSet()
	}
	return &Program{
		Fset:       fset,
		Target:     target.ResolveSpec(target.Spec{}),
		mode:       mode,
		packages:   make(map[*types.Package]*Package),
		imported:   make(map[string]*Package),
		funcsByObj: make(map[*types.Func]*Function),
		methods:    make(map[string]*Function),
	}
}

func (prog *Program) CreatePackage(pkg *types.Package, files []*ast.File, info *types.Info, importable bool) *Package {
	if pkg == nil {
		return nil
	}
	if existing, ok := prog.packages[pkg]; ok {
		return existing
	}
	p := &Package{
		Prog:    prog,
		Pkg:     pkg,
		Members: make(map[string]Member),
	}
	prog.packages[pkg] = p
	if importable && pkg.Path() != "" {
		prog.imported[pkg.Path()] = p
	}
	return p
}

func (prog *Program) Package(pkg *types.Package) *Package {
	if prog == nil || pkg == nil {
		return nil
	}
	return prog.packages[pkg]
}

func (prog *Program) ImportedPackage(path string) *Package {
	if prog == nil {
		return nil
	}
	return prog.imported[path]
}

func (prog *Program) AllPackages() []*Package {
	if prog == nil {
		return nil
	}
	out := make([]*Package, 0, len(prog.packages))
	for _, p := range prog.packages {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out
}

func (prog *Program) Build() {
	for _, p := range prog.AllPackages() {
		p.Build()
	}
}

func (prog *Program) NewFunction(name string, sig *types.Signature, provenance string) *Function {
	return &Function{
		name:      name,
		Signature: sig,
		Synthetic: provenance,
		Prog:      prog,
	}
}

func (prog *Program) FuncValue(obj *types.Func) *Function {
	if prog == nil || obj == nil {
		return nil
	}
	return prog.funcsByObj[obj]
}

func methodKey(T types.Type, pkg *types.Package, name string) string {
	pkgPath := ""
	if pkg != nil && !token.IsExported(name) {
		pkgPath = pkg.Path()
	}
	return T.String() + "|" + pkgPath + "|" + name
}

func (prog *Program) LookupMethod(T types.Type, pkg *types.Package, name string) *Function {
	if prog == nil || T == nil {
		return nil
	}
	if fn := prog.methods[methodKey(T, pkg, name)]; fn != nil {
		return fn
	}
	if sel := prog.MethodSets.MethodSet(T).Lookup(pkg, name); sel != nil {
		return prog.MethodValue(sel)
	}
	return nil
}

func (prog *Program) MethodValue(sel *types.Selection) *Function {
	if prog == nil || sel == nil {
		return nil
	}
	fnObj, _ := sel.Obj().(*types.Func)
	if fnObj != nil {
		if fn := prog.methods[methodKey(sel.Recv(), fnObj.Pkg(), fnObj.Name())]; fn != nil {
			return fn
		}
		if fn := prog.funcsByObj[fnObj]; fn != nil {
			return fn
		}
	}
	return nil
}

func (prog *Program) ConstValue(obj *types.Const) *Const {
	if obj == nil {
		return nil
	}
	return NewConst(obj.Val(), obj.Type())
}

func (prog *Program) RuntimeTypes() []types.Type {
	if prog == nil {
		return nil
	}
	return prog.runtimeTypes.Keys()
}

func (prog *Program) SetNoReturn(noReturn func(*types.Func) bool) {
	if prog != nil {
		prog.noReturn = noReturn
	}
}

func (prog *Program) VarValue(obj *types.Var, pkg *Package, ref []ast.Node) (value Value, isAddr bool) {
	if obj == nil || pkg == nil {
		return nil, false
	}
	if g := pkg.Var(obj.Name()); g != nil {
		return g, true
	}
	return nil, false
}

// Package is a single analyzed Go package containing Members.
type Package struct {
	Prog    *Program
	Pkg     *types.Package
	Members map[string]Member
	Kind    directive.PackageKind // LLGo extension: package linkage kind

	debug bool
}

func (p *Package) Build() {
	if p == nil {
		return
	}
	for _, m := range p.Members {
		if fn, ok := m.(*Function); ok {
			fn.Finish()
		}
	}
}

func (p *Package) SetDebugMode(debug bool) {
	if p != nil {
		p.debug = debug
	}
}

func (p *Package) Func(name string) *Function {
	if p == nil {
		return nil
	}
	f, _ := p.Members[name].(*Function)
	return f
}

func (p *Package) Var(name string) *Global {
	if p == nil {
		return nil
	}
	g, _ := p.Members[name].(*Global)
	return g
}

func (p *Package) Type(name string) *Type {
	if p == nil {
		return nil
	}
	t, _ := p.Members[name].(*Type)
	return t
}

func (p *Package) Const(name string) *NamedConst {
	if p == nil {
		return nil
	}
	c, _ := p.Members[name].(*NamedConst)
	return c
}

func (p *Package) String() string {
	if p == nil || p.Pkg == nil {
		return "package <nil>"
	}
	return "package " + p.Pkg.Path()
}

func (p *Package) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	WritePackage(&buf, p)
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// WritePackage writes a human-readable summary of p to buf.
func WritePackage(buf *bytes.Buffer, p *Package) {
	if p == nil || buf == nil {
		return
	}
	fmt.Fprintf(buf, "%s:\n", p.String())
	names := make([]string, 0, len(p.Members))
	for name := range p.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m := p.Members[name]
		fmt.Fprintf(buf, "  %-6s %-32s %s\n", m.Token(), name, m.Type())
	}
}

// WriteFunction writes a human-readable disassembly of f to buf.
func WriteFunction(buf *bytes.Buffer, f *Function) {
	if f == nil || buf == nil {
		return
	}
	fmt.Fprintf(buf, "func %s %s:\n", f.String(), f.Signature)
	for _, b := range f.Blocks {
		if b.Comment != "" {
			fmt.Fprintf(buf, "%d:                                                                # %s\n", b.Index, b.Comment)
		} else {
			fmt.Fprintf(buf, "%d:\n", b.Index)
		}
		for _, ins := range b.Instrs {
			if v, ok := ins.(Value); ok {
				fmt.Fprintf(buf, "\t%s = %s\n", v.Name(), ins.String())
			} else {
				fmt.Fprintf(buf, "\t%s\n", ins.String())
			}
		}
	}
}

// Function represents the parameters, results, and code of a function or method.
type Function struct {
	Signature *types.Signature
	Synthetic string
	Pkg       *Package
	Prog      *Program
	Params    []*Parameter
	FreeVars  []*FreeVar
	Locals    []*Alloc
	Blocks    []*BasicBlock
	Recover   *BasicBlock
	AnonFuncs []*Function

	// LLGo extensions: symbol linkage and directive attributes.
	LinkName   string
	ExportName string
	NoInline   bool
	WasmImport *directive.WasmImport

	name       string
	object     types.Object
	pos        token.Pos
	parent     *Function
	syntax     ast.Node
	origin     *Function
	typeParams *types.TypeParamList
	typeArgs   []types.Type
	referrers  []Instruction
	nextReg    int
}

func (f *Function) Name() string        { return f.name }
func (f *Function) Object() types.Object { return f.object }
func (f *Function) Package() *Package   { return f.Pkg }
func (f *Function) Parent() *Function   { return f.parent }
func (f *Function) Pos() token.Pos      { return f.pos }
func (f *Function) Syntax() ast.Node    { return f.syntax }
func (f *Function) Token() token.Token  { return token.FUNC }
func (f *Function) Type() types.Type    { return f.Signature }
func (f *Function) Origin() *Function   { return f.origin }
func (f *Function) TypeArgs() []types.Type {
	return f.typeArgs
}
func (f *Function) TypeParams() *types.TypeParamList {
	return f.typeParams
}
func (f *Function) Operands(rands []*Value) []*Value { return rands }
func (f *Function) Referrers() *[]Instruction {
	if f.parent != nil {
		return &f.referrers
	}
	return nil
}

func (f *Function) String() string {
	return f.RelString(nil)
}

func (f *Function) RelString(from *types.Package) string {
	if f == nil {
		return "<nil>"
	}
	if f.parent != nil {
		return f.parent.RelString(from) + "$" + f.name
	}
	if f.Signature != nil && f.Signature.Recv() != nil {
		recv := types.TypeString(f.Signature.Recv().Type(), types.RelativeTo(from))
		return fmt.Sprintf("(%s).%s", recv, f.name)
	}
	if f.Pkg != nil && f.Pkg.Pkg != nil && f.Pkg.Pkg != from {
		return f.Pkg.Pkg.Path() + "." + f.name
	}
	return f.name
}

func (f *Function) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	WriteFunction(&buf, f)
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

func (f *Function) ValueForExpr(e ast.Expr) (value Value, isAddr bool) {
	if f == nil || e == nil {
		return nil, false
	}
	for _, b := range f.Blocks {
		for _, ins := range b.Instrs {
			if dr, ok := ins.(*DebugRef); ok && dr.Expr == e {
				return dr.X, dr.IsAddr
			}
		}
	}
	return nil, false
}

func (f *Function) DomPreorder() []*BasicBlock {
	if f == nil || len(f.Blocks) == 0 {
		return nil
	}
	var out []*BasicBlock
	var walk func(*BasicBlock)
	walk = func(b *BasicBlock) {
		out = append(out, b)
		for _, child := range b.dominees {
			walk(child)
		}
	}
	walk(f.Blocks[0])
	return out
}

func (f *Function) DomPostorder() []*BasicBlock {
	if f == nil || len(f.Blocks) == 0 {
		return nil
	}
	var out []*BasicBlock
	var walk func(*BasicBlock)
	walk = func(b *BasicBlock) {
		for _, child := range b.dominees {
			walk(child)
		}
		out = append(out, b)
	}
	walk(f.Blocks[0])
	return out
}

// EnclosingFunction returns the Function enclosing path, if present.
func EnclosingFunction(pkg *Package, path []ast.Node) *Function {
	if pkg == nil {
		return nil
	}
	for _, n := range path {
		if fnDecl, ok := n.(*ast.FuncDecl); ok && fnDecl.Name != nil {
			if fn := pkg.Func(fnDecl.Name.Name); fn != nil {
				return fn
			}
		}
	}
	return nil
}

// HasEnclosingFunction reports whether path has an enclosing Function in pkg.
func HasEnclosingFunction(pkg *Package, path []ast.Node) bool {
	return EnclosingFunction(pkg, path) != nil
}

// BasicBlock represents an SSA basic block.
type BasicBlock struct {
	Index   int
	Comment string
	Instrs  []Instruction
	Preds   []*BasicBlock
	Succs   []*BasicBlock

	parent   *Function
	idom     *BasicBlock
	dominees []*BasicBlock
	domPre   int
	domPost  int
}

func (b *BasicBlock) Parent() *Function       { return b.parent }
func (b *BasicBlock) String() string          { return fmt.Sprintf("%d", b.Index) }
func (b *BasicBlock) Idom() *BasicBlock       { return b.idom }
func (b *BasicBlock) Dominees() []*BasicBlock { return b.dominees }
func (b *BasicBlock) Dominates(c *BasicBlock) bool {
	if b == nil || c == nil {
		return false
	}
	if b == c {
		return true
	}
	return b.domPre > 0 && b.domPre <= c.domPre && c.domPost <= b.domPost
}

// Global is a package-level variable whose SSA Value is its address (*T).
type Global struct {
	Pkg      *Package
	LinkName string             // LLGo extension
	Locality directive.Locality // LLGo extension

	name   string
	object types.Object
	typ    types.Type
	pos    token.Pos
}

func (g *Global) Name() string                      { return g.name }
func (g *Global) Object() types.Object              { return g.object }
func (g *Global) Package() *Package                 { return g.Pkg }
func (g *Global) Parent() *Function                 { return nil }
func (g *Global) Pos() token.Pos                    { return g.pos }
func (g *Global) Referrers() *[]Instruction         { return nil }
func (g *Global) Token() token.Token                { return token.VAR }
func (g *Global) Type() types.Type                  { return g.typ }
func (g *Global) Operands(rands []*Value) []*Value  { return rands }
func (g *Global) String() string                    { return g.RelString(nil) }
func (g *Global) RelString(from *types.Package) string {
	if g.Pkg != nil && g.Pkg.Pkg != nil && g.Pkg.Pkg != from {
		return g.Pkg.Pkg.Path() + "." + g.name
	}
	return g.name
}

// NamedConst is a package-level constant Member.
type NamedConst struct {
	Value *Const

	pkg    *Package
	object *types.Const
}

func (c *NamedConst) Name() string         { return c.object.Name() }
func (c *NamedConst) Object() types.Object { return c.object }
func (c *NamedConst) Package() *Package    { return c.pkg }
func (c *NamedConst) Pos() token.Pos       { return c.object.Pos() }
func (c *NamedConst) Token() token.Token   { return token.CONST }
func (c *NamedConst) Type() types.Type     { return c.object.Type() }
func (c *NamedConst) String() string       { return c.RelString(nil) }
func (c *NamedConst) RelString(from *types.Package) string {
	if c.pkg != nil && c.pkg.Pkg != nil && c.pkg.Pkg != from {
		return c.pkg.Pkg.Path() + "." + c.Name()
	}
	return c.Name()
}

// Type is a package-level type Member.
type Type struct {
	pkg    *Package
	object *types.TypeName
}

func (t *Type) Name() string         { return t.object.Name() }
func (t *Type) Object() types.Object { return t.object }
func (t *Type) Package() *Package    { return t.pkg }
func (t *Type) Pos() token.Pos       { return t.object.Pos() }
func (t *Type) Token() token.Token   { return token.TYPE }
func (t *Type) Type() types.Type     { return t.object.Type() }
func (t *Type) String() string       { return t.RelString(nil) }
func (t *Type) RelString(from *types.Package) string {
	if t.pkg != nil && t.pkg.Pkg != nil && t.pkg.Pkg != from {
		return t.pkg.Pkg.Path() + "." + t.Name()
	}
	return t.Name()
}

// Parameter is an input parameter of a Function.
type Parameter struct {
	name      string
	object    types.Object
	typ       types.Type
	parent    *Function
	referrers []Instruction
}

func (p *Parameter) Name() string                      { return p.name }
func (p *Parameter) Object() types.Object              { return p.object }
func (p *Parameter) Parent() *Function                 { return p.parent }
func (p *Parameter) Type() types.Type                  { return p.typ }
func (p *Parameter) Referrers() *[]Instruction         { return &p.referrers }
func (p *Parameter) Operands(rands []*Value) []*Value  { return rands }
func (p *Parameter) Pos() token.Pos {
	if p.object != nil {
		return p.object.Pos()
	}
	return token.NoPos
}
func (p *Parameter) String() string { return fmt.Sprintf("parameter %s : %s", p.name, p.typ) }

// FreeVar is a captured free variable of an anonymous Function.
type FreeVar struct {
	name      string
	typ       types.Type
	pos       token.Pos
	parent    *Function
	referrers []Instruction
}

func (v *FreeVar) Name() string                     { return v.name }
func (v *FreeVar) Parent() *Function                { return v.parent }
func (v *FreeVar) Pos() token.Pos                   { return v.pos }
func (v *FreeVar) Type() types.Type                 { return v.typ }
func (v *FreeVar) Referrers() *[]Instruction        { return &v.referrers }
func (v *FreeVar) Operands(rands []*Value) []*Value { return rands }
func (v *FreeVar) String() string                   { return fmt.Sprintf("freevar %s : %s", v.name, v.typ) }

// Builtin represents a Go built-in function callee.
type Builtin struct {
	name string
	sig  *types.Signature
}

func (b *Builtin) Name() string                     { return b.name }
func (b *Builtin) Object() types.Object             { return types.Universe.Lookup(b.name) }
func (b *Builtin) Parent() *Function                { return nil }
func (b *Builtin) Pos() token.Pos                   { return token.NoPos }
func (b *Builtin) Referrers() *[]Instruction        { return nil }
func (b *Builtin) Type() types.Type                 { return b.sig }
func (b *Builtin) Operands(rands []*Value) []*Value { return rands }
func (b *Builtin) String() string                   { return "builtin " + b.name }

// Const represents the value of a constant expression.
type Const struct {
	Value constant.Value
	typ   types.Type
}

// NewConst returns a new Const of the specified value and type.
func NewConst(val constant.Value, typ types.Type) *Const {
	return &Const{Value: val, typ: typ}
}

func (c *Const) Type() types.Type                 { return c.typ }
func (c *Const) Parent() *Function                { return nil }
func (c *Const) Pos() token.Pos                   { return token.NoPos }
func (c *Const) Referrers() *[]Instruction        { return nil }
func (c *Const) Operands(rands []*Value) []*Value { return rands }
func (c *Const) IsNil() bool                      { return c.Value == nil }
func (c *Const) Name() string                     { return c.RelString(nil) }
func (c *Const) String() string                   { return c.RelString(nil) }

func (c *Const) RelString(from *types.Package) string {
	var p string
	if c.Value == nil {
		p = "nil"
	} else if c.Value.Kind() == constant.String {
		p = constant.StringVal(c.Value)
		if len(p) > 20 {
			p = p[:17] + "..."
		}
		p = fmt.Sprintf("%q", p)
	} else {
		p = c.Value.String()
	}
	return fmt.Sprintf("%s:%s", p, types.TypeString(c.typ, types.RelativeTo(from)))
}

func (c *Const) Int64() int64 {
	if c.Value == nil {
		return 0
	}
	switch c.Value.Kind() {
	case constant.Int:
		if v, ok := constant.Int64Val(c.Value); ok {
			return v
		}
		return 0
	case constant.Float:
		f, _ := constant.Float64Val(c.Value)
		return int64(f)
	}
	return 0
}

func (c *Const) Uint64() uint64 {
	if c.Value == nil {
		return 0
	}
	switch c.Value.Kind() {
	case constant.Int:
		if v, ok := constant.Uint64Val(c.Value); ok {
			return v
		}
		return 0
	case constant.Float:
		f, _ := constant.Float64Val(c.Value)
		return uint64(f)
	}
	return 0
}

func (c *Const) Float64() float64 {
	if c.Value == nil {
		return 0
	}
	f, _ := constant.Float64Val(c.Value)
	return f
}

func (c *Const) Complex128() complex128 {
	if c.Value == nil {
		return 0
	}
	re, _ := constant.Float64Val(constant.Real(c.Value))
	im, _ := constant.Float64Val(constant.Imag(c.Value))
	return complex(re, im)
}

// CallCommon is embedded by Call, Go, and Defer instructions.
type CallCommon struct {
	Value  Value
	Method *types.Func
	Args   []Value

	pos token.Pos
}

func (c *CallCommon) IsInvoke() bool { return c.Method != nil }
func (c *CallCommon) Pos() token.Pos { return c.pos }

func (c *CallCommon) Signature() *types.Signature {
	if c.Method != nil {
		sig, _ := c.Method.Type().(*types.Signature)
		return sig
	}
	if c.Value != nil && c.Value.Type() != nil {
		sig, _ := c.Value.Type().Underlying().(*types.Signature)
		return sig
	}
	return nil
}

func (c *CallCommon) StaticCallee() *Function {
	if c.Method != nil {
		return nil
	}
	fn, _ := c.Value.(*Function)
	return fn
}

func (c *CallCommon) Description() string {
	switch {
	case c.Method != nil:
		return "dynamic method call"
	case c.StaticCallee() != nil:
		return "static function call"
	default:
		return "dynamic function call"
	}
}

func (c *CallCommon) Operands(rands []*Value) []*Value {
	rands = append(rands, &c.Value)
	for i := range c.Args {
		rands = append(rands, &c.Args[i])
	}
	return rands
}

func (c *CallCommon) String() string {
	var b strings.Builder
	if c.Method != nil {
		fmt.Fprintf(&b, "invoke %s.%s(", valName(c.Value), c.Method.Name())
	} else {
		fmt.Fprintf(&b, "%s(", valName(c.Value))
	}
	for i, a := range c.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(valName(a))
	}
	b.WriteByte(')')
	return b.String()
}

// SelectState describes a single case of a Select instruction.
type SelectState struct {
	Dir       types.ChanDir
	Chan      Value
	Send      Value
	Pos       token.Pos
	DebugNode ast.Node
}

func valName(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return v.Name()
}
