// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by an Apache 2.0 license.

// Package midway specializes portable SIMD source before construction of Go SSA.
// The input is type-checked Go syntax. The output must be type-checked again.
package midway

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

const (
	simdPath   = "simd"
	bridgePath = "simd/internal/bridge"
	// Every target has this hardware variant as dispatchWidth's final fallback.
	fallbackWidth = 128
)

// Widths are a target property, not the build host's CPU capabilities.
// Keep selectable widths in sync with runtime/_patch/simd/selection_llgo.go.
func targetWidths(arch string) []int {
	if arch == "amd64" {
		return []int{0, fallbackWidth, 256}
	}
	return []int{0, fallbackWidth}
}

func suffix(width int) string {
	if width == 0 {
		return "@simd0"
	}
	return "@simd" + strconv.Itoa(width)
}

func elementBits(name string) int {
	switch name {
	case "Int8s", "Uint8s", "Mask8s":
		return 8
	case "Int16s", "Uint16s", "Mask16s":
		return 16
	case "Int32s", "Uint32s", "Float32s", "Mask32s":
		return 32
	case "Int64s", "Uint64s", "Float64s", "Mask64s":
		return 64
	}
	return 0
}

func baseType(obj types.Object) bool {
	_, ok := obj.(*types.TypeName)
	return ok && obj.Pkg() != nil && obj.Pkg().Path() == simdPath && elementBits(obj.Name()) != 0
}

func packageObject(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}

// An imported package retains its original declarations for the initial type
// check, and exports the variants beside them. Testing for the variant also
// recognizes a type whose only SIMD dependency is in one of its method bodies.
func exportedVariant(obj types.Object) bool {
	return packageObject(obj) && obj.Pkg().Scope().Lookup(obj.Name()+suffix(128)) != nil
}

type analysis struct {
	pkg   *types.Package
	info  *types.Info
	funcs map[*types.Func]*ast.FuncDecl
	deps  map[types.Object]bool

	// Propagate through reverse edges rather than memoizing a negative result
	// during DFS: a recursive type may reach SIMD through a later field.
	types   map[types.Type]bool
	parents map[types.Type][]types.Type
	seen    map[types.Type]bool
	queue   []types.Type

	typeRules   map[types.Type][]*dependencyRule
	objectRules map[types.Object][]*dependencyRule
	rules       []*dependencyRule
}

// A declaration becomes dependent when any referenced type or non-function
// object does. Each rule fires once, marking its objects and/or result type.
type dependencyRule struct {
	objects []types.Object
	typ     types.Type
	queued  bool
}

func analyze(pkg *types.Package, info *types.Info, files []*ast.File) *analysis {
	a := &analysis{
		pkg: pkg, info: info, funcs: make(map[*types.Func]*ast.FuncDecl),
		deps: make(map[types.Object]bool), types: make(map[types.Type]bool),
		parents: make(map[types.Type][]types.Type), seen: make(map[types.Type]bool),
		typeRules: make(map[types.Type][]*dependencyRule), objectRules: make(map[types.Object][]*dependencyRule),
	}
	for _, objects := range []map[*ast.Ident]types.Object{info.Defs, info.Uses} {
		for _, obj := range objects {
			if obj != nil {
				a.addType(obj.Type())
			}
		}
	}
	for _, tv := range info.Types {
		a.addType(tv.Type)
	}
	for _, inst := range info.Instances {
		a.addType(inst.Type)
		for i := 0; i < inst.TypeArgs.Len(); i++ {
			a.addType(inst.TypeArgs.At(i))
		}
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			// Blank functions and methods cannot be referenced or executed.
			// Their bodies were checked, but must not seed specialization.
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name != "_" {
				if obj, ok := info.Defs[fn.Name].(*types.Func); ok {
					a.funcs[obj] = fn
				}
			}
		}
	}
	// Record syntax dependencies once. Layout constants and array lengths can
	// depend on portable syntax even when their checked types are scalar.
	for _, file := range files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gd.Tok == token.CONST {
				rule := new(dependencyRule)
				for _, spec := range gd.Specs {
					for _, id := range spec.(*ast.ValueSpec).Names {
						if obj := info.Defs[id]; obj != nil {
							rule.objects = append(rule.objects, obj)
						}
					}
				}
				a.watchBody(rule, gd)
			}
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if obj := info.Defs[ts.Name]; obj != nil {
						a.watchBody(&dependencyRule{typ: obj.Type()}, ts.Type)
					}
				}
			}
		}
	}
	for obj, decl := range a.funcs {
		rule := &dependencyRule{objects: []types.Object{obj}}
		if recv := obj.Signature().Recv(); recv != nil {
			rule.typ = recv.Type()
			if p, ok := rule.typ.(*types.Pointer); ok {
				rule.typ = p.Elem()
			}
			if named, ok := rule.typ.(*types.Named); ok {
				// A method body applies to every instance of its receiver.
				rule.typ = named.Origin()
			}
		}
		a.watchType(rule, obj.Type())
		a.watchBody(rule, decl.Body)
	}
	a.propagate()
	for _, obj := range info.Defs {
		if packageObject(obj) && a.types[obj.Type()] {
			a.deps[obj] = true
		}
	}
	return a
}

func (a *analysis) seed(t types.Type) {
	if t != nil && !a.types[t] {
		a.types[t] = true
		a.queue = append(a.queue, t)
	}
}

func (a *analysis) propagate() {
	for len(a.queue) != 0 || len(a.rules) != 0 {
		for len(a.queue) != 0 {
			t := a.queue[0]
			a.queue = a.queue[1:]
			for _, parent := range a.parents[t] {
				a.seed(parent)
			}
			for _, rule := range a.typeRules[t] {
				a.enqueue(rule)
			}
		}
		if len(a.rules) == 0 {
			break
		}
		rule := a.rules[0]
		a.rules = a.rules[1:]
		a.seed(rule.typ)
		for _, obj := range rule.objects {
			if !a.deps[obj] {
				a.deps[obj] = true
				for _, user := range a.objectRules[obj] {
					a.enqueue(user)
				}
			}
		}
	}
}

func (a *analysis) enqueue(rule *dependencyRule) {
	if !rule.queued {
		rule.queued = true
		a.rules = append(a.rules, rule)
	}
}

func (a *analysis) watchType(rule *dependencyRule, t types.Type) {
	if t == nil {
		return
	}
	if a.types[t] {
		a.enqueue(rule)
	} else {
		a.typeRules[t] = append(a.typeRules[t], rule)
	}
}

func (a *analysis) watchBody(rule *dependencyRule, node ast.Node) {
	if node == nil {
		return
	}
	if body, ok := node.(*ast.BlockStmt); ok && body == nil {
		return
	}
	seenTypes := make(map[types.Type]bool)
	seenObjects := make(map[types.Object]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		if expr, ok := n.(ast.Expr); ok {
			if typ := a.info.TypeOf(expr); typ != nil && !seenTypes[typ] {
				seenTypes[typ] = true
				a.watchType(rule, typ)
			}
		}
		if id, ok := n.(*ast.Ident); ok {
			obj := a.info.ObjectOf(id)
			// Scalar function calls stay at the dispatch boundary; they do
			// not make their callers require specialized signatures/bodies.
			if _, fn := obj.(*types.Func); obj != nil && !fn && !seenObjects[obj] {
				seenObjects[obj] = true
				if a.dependent(obj) {
					a.enqueue(rule)
				} else {
					a.objectRules[obj] = append(a.objectRules[obj], rule)
				}
			}
		}
		return true
	})
}

func (a *analysis) addType(t types.Type) {
	if t == nil || a.seen[t] {
		return
	}
	a.seen[t] = true
	edge := func(child types.Type) {
		if child != nil {
			a.parents[child] = append(a.parents[child], t)
			a.addType(child)
		}
	}
	params := func(params *types.TypeParamList) {
		for i := 0; i < params.Len(); i++ {
			edge(params.At(i))
		}
	}
	switch t := t.(type) {
	case *types.Named:
		if t.Origin() != t {
			edge(t.Origin())
		}
		if baseType(t.Obj()) || t.Obj().Pkg() != a.pkg && exportedVariant(t.Obj()) {
			a.seed(t)
		}
		edge(t.Underlying())
		params(t.TypeParams())
		for i := 0; i < t.TypeArgs().Len(); i++ {
			edge(t.TypeArgs().At(i))
		}
		for i := 0; i < t.NumMethods(); i++ {
			edge(t.Method(i).Type())
		}
	case *types.Alias:
		edge(t.Rhs())
		params(t.TypeParams())
		for i := 0; i < t.TypeArgs().Len(); i++ {
			edge(t.TypeArgs().At(i))
		}
	case *types.Pointer:
		edge(t.Elem())
	case *types.Slice:
		edge(t.Elem())
	case *types.Array:
		edge(t.Elem())
	case *types.Map:
		edge(t.Key())
		edge(t.Elem())
	case *types.Chan:
		edge(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			edge(t.Field(i).Type())
		}
	case *types.Tuple:
		for i := 0; i < t.Len(); i++ {
			edge(t.At(i).Type())
		}
	case *types.Signature:
		edge(t.Params())
		edge(t.Results())
		if t.Recv() != nil {
			edge(t.Recv().Type())
		}
		params(t.TypeParams())
		params(t.RecvTypeParams())
	case *types.Interface:
		for i := 0; i < t.NumExplicitMethods(); i++ {
			edge(t.ExplicitMethod(i).Type())
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			edge(t.EmbeddedType(i))
		}
	case *types.TypeParam:
		edge(t.Constraint())
	case *types.Union:
		for i := 0; i < t.Len(); i++ {
			edge(t.Term(i).Type())
		}
	}
}

func (a *analysis) bodyDependent(node ast.Node) bool {
	if node == nil {
		return false
	}
	if body, ok := node.(*ast.BlockStmt); ok && body == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		if expr, ok := n.(ast.Expr); ok && a.types[a.info.TypeOf(expr)] {
			found = true
		}
		if id, ok := n.(*ast.Ident); ok {
			obj := a.info.ObjectOf(id)
			if _, fn := obj.(*types.Func); !fn && a.dependent(obj) {
				found = true
			}
		}
		return !found
	})
	return found
}

func (a *analysis) dependent(obj types.Object) bool {
	return obj != nil && (a.deps[obj] || obj.Pkg() != a.pkg && exportedVariant(obj))
}
