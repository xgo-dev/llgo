// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"go/token"
)

// Object preserves package identity for private fields/methods as well as names.
// Type refers to a TypeGraph node; -1 is the only absent-type representation.
type Object struct {
	Name, Package, Position string
	Type                    int
	Embedded                bool   `json:",omitempty"`
	Tag                     []byte `json:",omitempty"`
}

type TypeNode struct {
	Kind                          string
	Key                           string
	Object                        Object
	Elem, Base                    int
	Basic, Direction              int
	CallConv, PtrKind             uint8 `json:",omitempty"`
	Length                        int64
	Variadic                      bool
	Implicit                      bool
	NativeLayout                  bool `json:",omitempty"`
	IsMask                        bool `json:",omitempty"`
	Universe                      bool
	Types, Params, Results        []int    `json:",omitempty"`
	TypeParams, RecvParams        []int    `json:",omitempty"`
	Receiver                      *Object  `json:",omitempty"`
	Parameters, Returns, Elements []Object `json:",omitempty"`
	Fields, Methods               []Object `json:",omitempty"`
	Tilde                         []bool   `json:",omitempty"`
}

// TypeGraph transports private and compiler-generated runtime types that are
// not necessarily reachable from gc's public export data. Cycles use node IDs;
// neither Go pointers nor LLVM-owned values cross a process boundary.
type TypeGraph struct {
	Nodes []TypeNode
	ids   map[types.Type]int
	types *Types
	fset  *token.FileSet
}

func NewTypeGraph(typs *Types, fset *token.FileSet) *TypeGraph {
	return &TypeGraph{ids: make(map[types.Type]int), types: typs, fset: fset}
}

func (g *TypeGraph) object(obj types.Object) Object {
	return Object{Name: obj.Name(), Package: g.types.PackageID(obj.Pkg()), Position: Position(g.fset, obj.Pos()), Type: g.Add(obj.Type())}
}

func (g *TypeGraph) Add(typ types.Type) int {
	if typ == nil {
		return -1
	}
	if id, ok := g.ids[typ]; ok {
		return id
	}
	id := len(g.Nodes)
	g.ids[typ] = id
	g.Nodes = append(g.Nodes, TypeNode{})
	n := TypeNode{Key: g.types.Key(typ), Elem: -1, Base: -1}
	addParams := func(list *types.TypeParamList) []int {
		var result []int
		for i := 0; i < list.Len(); i++ {
			result = append(result, g.Add(list.At(i)))
		}
		return result
	}
	addTuple := func(tuple *types.Tuple) []int {
		var result []int
		for i := 0; i < tuple.Len(); i++ {
			result = append(result, g.Add(tuple.At(i).Type()))
		}
		return result
	}
	switch v := typ.(type) {
	case *types.Basic:
		n.Kind, n.Basic = "basic", int(v.Kind())
	case *types.Pointer:
		n.Kind, n.Elem, n.PtrKind = "pointer", g.Add(v.Elem()), uint8(v.Kind())
	case *types.Slice:
		n.Kind, n.Elem = "slice", g.Add(v.Elem())
	case *types.Array:
		n.Kind, n.Elem, n.Length = "array", g.Add(v.Elem()), v.Len()
	case *types.Chan:
		n.Kind, n.Elem, n.Direction = "chan", g.Add(v.Elem()), int(v.Dir())
	case *types.Map:
		n.Kind, n.Elem, n.Base = "map", g.Add(v.Elem()), g.Add(v.Key())
	case *types.Named:
		n.Kind, n.CallConv = "named", uint8(v.CallConv())
		n.Universe = types.Universe.Lookup(v.Obj().Name()) == v.Obj()
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
		if v.Origin() != v {
			n.Base = g.Add(v.Origin())
			for i := 0; i < v.TypeArgs().Len(); i++ {
				n.Types = append(n.Types, g.Add(v.TypeArgs().At(i)))
			}
		} else {
			n.TypeParams, n.Elem = addParams(v.TypeParams()), g.Add(v.Underlying())
			for i := 0; i < v.NumMethods(); i++ {
				n.Methods = append(n.Methods, g.object(v.Method(i)))
			}
		}
	case *types.Alias:
		// Alias identity is its target's identity; retaining a separate node
		// nevertheless preserves declarations used by the type importer.
		n.Kind, n.CallConv = "alias", uint8(v.CallConv())
		n.Universe = types.Universe.Lookup(v.Obj().Name()) == v.Obj()
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
		if v.Origin() != v {
			n.Base = g.Add(v.Origin())
			for i := 0; i < v.TypeArgs().Len(); i++ {
				n.Types = append(n.Types, g.Add(v.TypeArgs().At(i)))
			}
		} else {
			n.TypeParams, n.Elem = addParams(v.TypeParams()), g.Add(v.Rhs())
		}
	case *types.TypeParam:
		n.Kind, n.Elem = "parameter", g.Add(v.Constraint())
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
	case *types.Signature:
		n.Kind, n.Variadic, n.CallConv = "func", v.Variadic(), uint8(v.CallConv())
		n.TypeParams, n.RecvParams = addParams(v.TypeParams()), addParams(v.RecvTypeParams())
		n.Params, n.Results = addTuple(v.Params()), addTuple(v.Results())
		for i := 0; i < v.Params().Len(); i++ {
			n.Parameters = append(n.Parameters, g.object(v.Params().At(i)))
		}
		for i := 0; i < v.Results().Len(); i++ {
			n.Returns = append(n.Returns, g.object(v.Results().At(i)))
		}
		if v.Recv() != nil {
			recv := g.object(v.Recv())
			n.Receiver = &recv
		}
	case *types.Tuple:
		n.Kind, n.Types = "tuple", addTuple(v)
		for i := 0; i < v.Len(); i++ {
			n.Elements = append(n.Elements, g.object(v.At(i)))
		}
	case *types.Struct:
		n.Kind, n.NativeLayout = "struct", v.NativeLayout()
		for i := 0; i < v.NumFields(); i++ {
			f := g.object(v.Field(i))
			f.Embedded, f.Tag = v.Field(i).Embedded(), []byte(v.Tag(i))
			n.Fields = append(n.Fields, f)
		}
	case *types.SIMDVec:
		n.Kind, n.Basic, n.Length, n.IsMask = "simdvec", int(v.Elem().Kind()), int64(v.Len()), v.IsMask()
	case *types.Interface:
		n.Kind, n.Implicit = "interface", v.IsImplicit()
		for i := 0; i < v.NumExplicitMethods(); i++ {
			n.Methods = append(n.Methods, g.object(v.ExplicitMethod(i)))
		}
		for i := 0; i < v.NumEmbeddeds(); i++ {
			n.Types = append(n.Types, g.Add(v.EmbeddedType(i)))
		}
	case *types.Union:
		n.Kind = "union"
		for i := 0; i < v.Len(); i++ {
			n.Types = append(n.Types, g.Add(v.Term(i).Type()))
			n.Tilde = append(n.Tilde, v.Term(i).Tilde())
		}
	default:
		panic(fmt.Sprintf("unsupported frontend type %T", typ))
	}
	g.Nodes[id] = n
	return id
}

type TypeImporter struct {
	Graph    *TypeGraph
	Packages map[string]*types.Package
	Existing *Types
	Position func(string) token.Pos
	values   map[int]types.Type
	active   map[int]bool
	failed   error
}

func (p *TypeImporter) Import(id int) (typ types.Type, err error) {
	if p.failed != nil {
		return nil, p.failed
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			typ, err = nil, fmt.Errorf("invalid frontend type: %v", recovered)
		}
		if err != nil {
			p.failed = err
		}
	}()
	if p.values == nil {
		p.values, p.active = make(map[int]types.Type), make(map[int]bool)
	}
	return p.importType(id, 0)
}

func (p *TypeImporter) importType(id, depth int) (types.Type, error) {
	if id < 0 || id >= len(p.Graph.Nodes) || depth > 512 {
		return nil, fmt.Errorf("invalid frontend type reference %d", id)
	}
	if typ := p.values[id]; typ != nil {
		return typ, nil
	}
	if p.active[id] {
		return nil, fmt.Errorf("invalid anonymous frontend type cycle")
	}
	n := &p.Graph.Nodes[id]
	if n.Universe {
		obj, ok := types.Universe.Lookup(n.Object.Name).(*types.TypeName)
		if !ok {
			return nil, fmt.Errorf("unknown frontend universe type %s", n.Object.Name)
		}
		p.values[id] = obj.Type()
		return obj.Type(), nil
	}
	if p.Existing != nil && n.Kind != "parameter" && n.Kind != "alias" && n.Kind != "func" && n.Kind != "tuple" {
		if typ, ok := p.Existing.Lookup(n.Key); ok {
			p.values[id] = typ
			return typ, nil
		}
	}
	p.active[id] = true
	defer delete(p.active, id)
	get := func(ref int) types.Type {
		typ, err := p.importType(ref, depth+1)
		if err != nil {
			panic(err)
		}
		return typ
	}
	owner := func(obj Object) *types.Package {
		if obj.Package == "" {
			return nil
		}
		pkg := p.Packages[obj.Package]
		if pkg == nil {
			panic("missing frontend type package " + obj.Package)
		}
		return pkg
	}
	pos := func(obj Object) token.Pos {
		if p.Position != nil {
			return p.Position(obj.Position)
		}
		return token.NoPos
	}
	variable := func(obj Object) *types.Var {
		return types.NewField(pos(obj), owner(obj), obj.Name, get(obj.Type), obj.Embedded)
	}
	tuple := func(refs []int) *types.Tuple {
		vars := make([]*types.Var, len(refs))
		for i, ref := range refs {
			vars[i] = types.NewVar(token.NoPos, nil, "", get(ref))
		}
		return types.NewTuple(vars...)
	}
	namedTuple := func(refs []int, objects []Object) *types.Tuple {
		if len(objects) == 0 {
			return tuple(refs)
		}
		if len(refs) != len(objects) {
			panic("invalid exported parameter list")
		}
		vars := make([]*types.Var, len(objects))
		for i, obj := range objects {
			if obj.Type != refs[i] {
				panic("invalid exported parameter type")
			}
			vars[i] = types.NewVar(pos(obj), owner(obj), obj.Name, get(obj.Type))
		}
		return types.NewTuple(vars...)
	}
	parameters := func(refs []int) []*types.TypeParam {
		var result []*types.TypeParam
		for _, ref := range refs {
			result = append(result, get(ref).(*types.TypeParam))
		}
		return result
	}
	method := func(obj Object) *types.Func {
		return types.NewFunc(pos(obj), owner(obj), obj.Name, get(obj.Type).(*types.Signature))
	}
	var result types.Type
	switch n.Kind {
	case "basic":
		if n.Basic < 0 || n.Basic >= len(types.Typ) || types.Typ[n.Basic] == nil {
			return nil, fmt.Errorf("invalid frontend basic kind")
		}
		result = types.Typ[n.Basic]
	case "pointer":
		if types.PtrKind(n.PtrKind) != types.GoPtr {
			result = types.NewPointerWithKind(get(n.Elem), types.PtrKind(n.PtrKind))
		} else {
			result = types.NewPointer(get(n.Elem))
		}
	case "slice":
		result = types.NewSlice(get(n.Elem))
	case "array":
		if n.Length < 0 {
			return nil, fmt.Errorf("negative frontend array length")
		}
		result = types.NewArray(get(n.Elem), n.Length)
	case "chan":
		if n.Direction < int(types.SendRecv) || n.Direction > int(types.RecvOnly) {
			return nil, fmt.Errorf("invalid frontend channel direction")
		}
		result = types.NewChan(types.ChanDir(n.Direction), get(n.Elem))
	case "map":
		result = types.NewMap(get(n.Base), get(n.Elem))
	case "named":
		if n.Base >= 0 {
			var args []types.Type
			for _, ref := range n.Types {
				args = append(args, get(ref))
			}
			var err error
			result, err = types.Instantiate(nil, get(n.Base), args, false)
			if err != nil {
				return nil, err
			}
		} else {
			obj := types.NewTypeName(pos(n.Object), owner(n.Object), n.Object.Name, nil)
			named := types.NewNamed(obj, nil, nil)
			p.values[id] = named
			// Once the named shell is published in p.values, recursive
			// references through it are valid named cycles rather than
			// anonymous cycles (even when pointer types are interned).
			savedActive := p.active
			p.active = make(map[int]bool)
			defer func() { p.active = savedActive }()
			named.SetTypeParams(parameters(n.TypeParams))
			named.SetUnderlying(get(n.Elem))
			if types.CallConv(n.CallConv) != types.Go {
				named.SetCallConv(types.CallConv(n.CallConv))
			}
			for _, m := range n.Methods {
				named.AddMethod(method(m))
			}
			result = named
		}
	case "alias":
		if n.Base >= 0 {
			var args []types.Type
			for _, ref := range n.Types {
				args = append(args, get(ref))
			}
			var err error
			result, err = types.Instantiate(nil, get(n.Base), args, false)
			if err != nil {
				return nil, err
			}
		} else {
			obj := types.NewTypeName(pos(n.Object), owner(n.Object), n.Object.Name, nil)
			alias := types.NewAlias(obj, get(n.Elem))
			if types.CallConv(n.CallConv) != types.Go {
				alias.SetCallConv(types.CallConv(n.CallConv))
			}
			alias.SetTypeParams(parameters(n.TypeParams))
			result = alias
		}
	case "parameter":
		obj := types.NewTypeName(pos(n.Object), owner(n.Object), n.Object.Name, nil)
		param := types.NewTypeParam(obj, nil)
		p.values[id] = param
		param.SetConstraint(get(n.Elem))
		result = param
	case "func":
		var recv *types.Var
		if n.Receiver != nil {
			obj := *n.Receiver
			recvType := get(obj.Type)
			if existing := p.values[id]; existing != nil {
				return existing, nil
			}
			recv = types.NewVar(pos(obj), owner(obj), obj.Name, recvType)
		}
		sig := types.NewSignatureType(recv, parameters(n.RecvParams), parameters(n.TypeParams), namedTuple(n.Params, n.Parameters), namedTuple(n.Results, n.Returns), n.Variadic)
		if types.CallConv(n.CallConv) != types.Go {
			sig = sig.WithCallConv(types.CallConv(n.CallConv))
		}
		result = sig
	case "tuple":
		result = namedTuple(n.Types, n.Elements)
	case "struct":
		var fields []*types.Var
		var tags []string
		for _, f := range n.Fields {
			fields, tags = append(fields, variable(f)), append(tags, string(f.Tag))
		}
		st := types.NewStruct(fields, tags)
		if n.NativeLayout {
			st.SetNativeLayout(true)
		}
		result = st
	case "simdvec":
		if n.Basic < 0 || n.Basic >= len(types.Typ) || types.Typ[n.Basic] == nil || n.Length <= 0 {
			return nil, fmt.Errorf("invalid frontend simdvec")
		}
		result = types.NewSIMDVec(types.Typ[n.Basic], int(n.Length), n.IsMask)
	case "interface":
		var methods []*types.Func
		var methodNodes []int
		var embeds []types.Type
		for _, m := range n.Methods {
			if m.Type >= 0 && m.Type < len(p.Graph.Nodes) {
				fnNode := &p.Graph.Nodes[m.Type]
				if fnNode.Kind == "func" && fnNode.Receiver != nil && fnNode.Receiver.Type == id {
					// Explicit methods of an anonymous interface carry a
					// synthetic receiver pointing back to the interface itself.
					// Build the signature without the self-receiver so
					// NewInterfaceType can re-bind it after constructing iface.
					sig := types.NewSignatureType(nil, parameters(fnNode.RecvParams), parameters(fnNode.TypeParams), namedTuple(fnNode.Params, fnNode.Parameters), namedTuple(fnNode.Results, fnNode.Returns), fnNode.Variadic)
					if types.CallConv(fnNode.CallConv) != types.Go {
						sig = sig.WithCallConv(types.CallConv(fnNode.CallConv))
					}
					methods = append(methods, types.NewFunc(pos(m), owner(m), m.Name, sig))
					methodNodes = append(methodNodes, m.Type)
					continue
				}
			}
			methods = append(methods, method(m))
			methodNodes = append(methodNodes, -1)
		}
		for _, ref := range n.Types {
			embeds = append(embeds, get(ref))
		}
		iface := types.NewInterfaceType(methods, embeds)
		if n.Implicit {
			iface.MarkImplicit()
		}
		result = iface.Complete()
		for i, ref := range methodNodes {
			if ref >= 0 {
				sig := methods[i].Type()
				p.values[ref] = sig
				if p.Existing != nil && p.Graph.Nodes[ref].Key != "" && !p.Existing.Bind(p.Graph.Nodes[ref].Key, sig) {
					return nil, fmt.Errorf("conflicting frontend type identity %s", p.Graph.Nodes[ref].Key)
				}
			}
		}
	case "union":
		if len(n.Types) != len(n.Tilde) {
			return nil, fmt.Errorf("invalid frontend union terms")
		}
		var terms []*types.Term
		for i, ref := range n.Types {
			terms = append(terms, types.NewTerm(n.Tilde[i], get(ref)))
		}
		result = types.NewUnion(terms)
	default:
		return nil, fmt.Errorf("invalid frontend type kind %q", n.Kind)
	}
	p.values[id] = result
	if p.Existing != nil && n.Key != "" && !p.Existing.Bind(n.Key, types.Unalias(result)) {
		return nil, fmt.Errorf("conflicting frontend type identity %s", n.Key)
	}
	return result, nil
}
