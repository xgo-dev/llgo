// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"math/big"
	"strconv"
)

type PackageObject struct {
	Object
	Kind             string
	Value, Imaginary string
}

type PackageExport struct {
	ID, Path, Name string
	Types          TypeGraph
	Objects        []PackageObject
}

// ExportPackage includes private declarations: code generation and layout
// metadata need more than the public API accepted by a source importer.
func ExportPackage(pkg *types.Package, index *Types, fset *token.FileSet) *PackageExport {
	graph := NewTypeGraph(index, fset)
	out := &PackageExport{ID: index.PackageID(pkg), Path: pkg.Path(), Name: pkg.Name()}
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		item := PackageObject{Object: graph.object(obj)}
		switch v := obj.(type) {
		case *types.TypeName:
			item.Kind = "type"
		case *types.Var:
			item.Kind = "var"
		case *types.Func:
			item.Kind = "func"
		case *types.Const:
			item.Kind = "const"
			switch v.Val().Kind() {
			case constant.String:
				item.Value = strconv.Quote(constant.StringVal(v.Val()))
			case constant.Complex:
				item.Value = constant.Real(v.Val()).ExactString()
				item.Imaginary = constant.Imag(v.Val()).ExactString()
			default:
				item.Value = v.Val().ExactString()
			}
		default:
			panic(fmt.Sprintf("unsupported package object %T", obj))
		}
		out.Objects = append(out.Objects, item)
	}
	out.Types = *graph
	return out
}

func rational(value string) (constant.Value, error) {
	v, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid exported constant %q", value)
	}
	return constant.Make(v), nil
}

// ImportPackage populates an empty package using this invocation's dependency
// identities. Callers publish it only after the complete graph validates.
func ImportPackage(data *PackageExport, importer *TypeImporter) (*types.Package, error) {
	if data == nil {
		return nil, fmt.Errorf("missing type export")
	}
	pkg := importer.Packages[data.ID]
	if pkg == nil || pkg.Path() != data.Path || pkg.Name() != data.Name || pkg.Scope().Len() != 0 {
		return nil, fmt.Errorf("invalid exported package identity")
	}
	importer.Graph = &data.Types
	// Materialize named declarations before inserting other package objects.
	for _, item := range data.Objects {
		if item.Kind != "type" {
			continue
		}
		typ, err := importer.Import(item.Type)
		if err != nil {
			return nil, err
		}
		var obj *types.TypeName
		switch v := typ.(type) {
		case *types.Named:
			obj = v.Obj()
		case *types.Alias:
			obj = v.Obj()
		default:
			pos := token.NoPos
			if importer.Position != nil {
				pos = importer.Position(item.Position)
			}
			obj = types.NewTypeName(pos, pkg, item.Name, typ)
		}
		if obj.Name() != item.Name || obj.Pkg() != pkg || pkg.Scope().Insert(obj) != nil {
			return nil, fmt.Errorf("invalid type declaration %s", item.Name)
		}
	}
	for _, item := range data.Objects {
		if item.Kind == "type" {
			continue
		}
		typ, err := importer.Import(item.Type)
		if err != nil {
			return nil, err
		}
		pos := token.NoPos
		if importer.Position != nil {
			pos = importer.Position(item.Position)
		}
		var obj types.Object
		switch item.Kind {
		case "func":
			sig, ok := typ.(*types.Signature)
			if !ok {
				return nil, fmt.Errorf("invalid function %s", item.Name)
			}
			obj = types.NewFunc(pos, pkg, item.Name, sig)
		case "var":
			obj = types.NewVar(pos, pkg, item.Name, typ)
		case "const":
			basic, ok := typ.Underlying().(*types.Basic)
			if !ok {
				return nil, fmt.Errorf("invalid constant type")
			}
			var v constant.Value
			switch {
			case basic.Info()&types.IsBoolean != 0:
				if item.Value != "true" && item.Value != "false" {
					return nil, fmt.Errorf("invalid bool constant")
				}
				v = constant.MakeBool(item.Value == "true")
			case basic.Info()&types.IsString != 0:
				text, err := strconv.Unquote(item.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid exported string constant: %w", err)
				}
				v = constant.MakeString(text)
			case basic.Info()&types.IsInteger != 0:
				integer, ok := new(big.Int).SetString(item.Value, 10)
				if !ok {
					return nil, fmt.Errorf("invalid integer constant")
				}
				v = constant.Make(integer)
			case basic.Info()&types.IsComplex != 0:
				real, err := rational(item.Value)
				if err != nil {
					return nil, err
				}
				imag, err := rational(item.Imaginary)
				if err != nil {
					return nil, err
				}
				v = constant.BinaryOp(real, token.ADD, constant.MakeImag(imag))
			case basic.Info()&types.IsFloat != 0:
				v, err = rational(item.Value)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid constant kind")
			}
			obj = types.NewConst(pos, pkg, item.Name, typ, v)
		default:
			return nil, fmt.Errorf("invalid package declaration kind %q", item.Kind)
		}
		if item.Name == "" || pkg.Scope().Insert(obj) != nil {
			return nil, fmt.Errorf("duplicate exported declaration")
		}
	}
	pkg.MarkComplete()
	return pkg, nil
}
