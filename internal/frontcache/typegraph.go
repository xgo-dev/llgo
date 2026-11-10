// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"go/token"
	"go/types"
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
	Layout                        string `json:",omitempty"`
	Object                        Object
	Elem, Base                    int
	Representation                *DeclarationRepresentation `json:",omitempty"`
	Basic, Direction              int
	Length                        int64
	Variadic                      bool
	Implicit                      bool
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
	decls map[int]*types.TypeName
}

func NewTypeGraph(typs *Types, fset *token.FileSet) *TypeGraph {
	return &TypeGraph{ids: make(map[types.Type]int), types: typs, fset: fset, decls: make(map[int]*types.TypeName)}
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
	if g.types.LayoutIdentity != nil {
		n.Layout = g.types.LayoutIdentity(typ)
	}
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
		n.Kind, n.Elem = "pointer", g.Add(v.Elem())
	case *types.Slice:
		n.Kind, n.Elem = "slice", g.Add(v.Elem())
	case *types.Array:
		n.Kind, n.Elem, n.Length = "array", g.Add(v.Elem()), v.Len()
	case *types.Chan:
		n.Kind, n.Elem, n.Direction = "chan", g.Add(v.Elem()), int(v.Dir())
	case *types.Map:
		n.Kind, n.Elem, n.Base = "map", g.Add(v.Elem()), g.Add(v.Key())
	case *types.Named:
		n.Kind = "named"
		n.Universe = types.Universe.Lookup(v.Obj().Name()) == v.Obj()
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
		if v.Origin() != v {
			n.Base = g.Add(v.Origin())
			for i := 0; i < v.TypeArgs().Len(); i++ {
				n.Types = append(n.Types, g.Add(v.TypeArgs().At(i)))
			}
		} else {
			g.decls[id] = v.Obj()
			n.TypeParams, n.Elem = addParams(v.TypeParams()), g.Add(v.Underlying())
			for i := 0; i < v.NumMethods(); i++ {
				n.Methods = append(n.Methods, g.object(v.Method(i)))
			}
		}
	case *types.Alias:
		// Alias identity is its target's identity; retaining a separate node
		// nevertheless preserves declarations used by the type importer.
		n.Kind = "alias"
		n.Universe = types.Universe.Lookup(v.Obj().Name()) == v.Obj()
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
		if v.Origin() != v {
			n.Base = g.Add(v.Origin())
			for i := 0; i < v.TypeArgs().Len(); i++ {
				n.Types = append(n.Types, g.Add(v.TypeArgs().At(i)))
			}
		} else {
			g.decls[id] = v.Obj()
			n.TypeParams, n.Elem = addParams(v.TypeParams()), g.Add(v.Rhs())
		}
	case *types.TypeParam:
		n.Kind, n.Elem = "parameter", g.Add(v.Constraint())
		n.Object = Object{Name: v.Obj().Name(), Package: g.types.PackageID(v.Obj().Pkg()), Position: Position(g.fset, v.Obj().Pos())}
	case *types.Signature:
		n.Kind, n.Variadic = "func", v.Variadic()
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
		n.Kind = "struct"
		for i := 0; i < v.NumFields(); i++ {
			f := g.object(v.Field(i))
			f.Embedded, f.Tag = v.Field(i).Embedded(), []byte(v.Tag(i))
			n.Fields = append(n.Fields, f)
		}
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

type DeclarationRepresentation struct {
	ABI    int
	Origin int
}

type ImportedRepresentation struct {
	ABI    int
	Origin *types.TypeName
}

// ExportRepresentations retains checked storage policy, including explicit Go
// facts. Source RHS bodies cannot be compared with lowered closure bodies.
func (g *TypeGraph) ExportRepresentations(resolve func(*types.TypeName) (int, types.Type)) {
	for id := 0; id < len(g.Nodes); id++ {
		if obj := g.decls[id]; obj != nil {
			abi, origin := resolve(obj)
			r := &DeclarationRepresentation{ABI: abi, Origin: -1}
			if origin != nil {
				r.Origin = g.Add(origin)
			}
			g.Nodes[id].Representation = r
		}
	}
}

// ImportRepresentations validates the full metadata set before publishing it.
func (p *TypeImporter) ImportRepresentations() (map[*types.TypeName]ImportedRepresentation, error) {
	result := make(map[*types.TypeName]ImportedRepresentation)
	object := func(t types.Type) *types.TypeName {
		switch t := t.(type) {
		case *types.Named:
			return t.Origin().Obj()
		case *types.Alias:
			return t.Origin().Obj()
		}
		return nil
	}
	for id, node := range p.Graph.Nodes {
		if node.Representation == nil {
			if (node.Kind == "named" || node.Kind == "alias") && node.Base == -1 && !node.Universe {
				return nil, fmt.Errorf("missing declaration representation %d", id)
			}
			continue
		}
		if node.Kind != "named" && node.Kind != "alias" || node.Base != -1 {
			return nil, fmt.Errorf("invalid representation owner %d", id)
		}
		owner, err := p.Import(id)
		if err != nil {
			return nil, err
		}
		obj := object(owner)
		if obj == nil {
			return nil, fmt.Errorf("invalid representation type %T", owner)
		}
		r := node.Representation
		entry := ImportedRepresentation{ABI: r.ABI}
		if r.ABI != 0 {
			origin, err := p.Import(r.Origin)
			if err != nil {
				return nil, err
			}
			entry.Origin = object(origin)
			if entry.Origin == nil {
				return nil, fmt.Errorf("invalid native representation origin %T", origin)
			}
		} else if r.Origin != -1 {
			return nil, fmt.Errorf("Go representation has a native origin")
		}
		result[obj] = entry
	}
	return result, nil
}

// ImportLayouts validates attributes before callers install them. Unknown
// layout protocols require source checking rather than silently using gc layout.
func (p *TypeImporter) ImportLayouts() (map[*types.Struct]string, error) {
	layouts := make(map[*types.Struct]string)
	for id, node := range p.Graph.Nodes {
		if node.Layout == "" {
			continue
		}
		if node.Layout != "native" || node.Kind != "struct" {
			return nil, fmt.Errorf("unsupported cached type layout %q", node.Layout)
		}
		typ, err := p.Import(id)
		if err != nil {
			return nil, err
		}
		st, ok := typ.(*types.Struct)
		if !ok {
			return nil, fmt.Errorf("invalid cached layout owner %T", typ)
		}
		layouts[st] = node.Layout
	}
	return layouts, nil
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
		result = types.NewPointer(get(n.Elem))
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
			named.SetTypeParams(parameters(n.TypeParams))
			named.SetUnderlying(get(n.Elem))
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
			recv = types.NewVar(pos(obj), owner(obj), obj.Name, get(obj.Type))
		}
		result = types.NewSignatureType(recv, parameters(n.RecvParams), parameters(n.TypeParams), namedTuple(n.Params, n.Parameters), namedTuple(n.Results, n.Returns), n.Variadic)
	case "tuple":
		result = namedTuple(n.Types, n.Elements)
	case "struct":
		var fields []*types.Var
		var tags []string
		for _, f := range n.Fields {
			fields, tags = append(fields, variable(f)), append(tags, string(f.Tag))
		}
		result = types.NewStruct(fields, tags)
	case "interface":
		var methods []*types.Func
		var embeds []types.Type
		for _, m := range n.Methods {
			methods = append(methods, method(m))
		}
		for _, ref := range n.Types {
			embeds = append(embeds, get(ref))
		}
		iface := types.NewInterfaceType(methods, embeds)
		if n.Implicit {
			iface.MarkImplicit()
		}
		result = iface.Complete()
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
