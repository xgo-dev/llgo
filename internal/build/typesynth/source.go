// Package typesynth adds compiler-generated declarations to checked packages.
package typesynth

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
	"go/version"
	"maps"

	"github.com/xgo-dev/llgo/internal/packages"
)

type Importer map[string]*types.Package

func (i Importer) Import(path string) (*types.Package, error) {
	if pkg := i[path]; pkg != nil {
		return pkg, nil
	}
	return nil, fmt.Errorf("generated source cannot find package %s", path)
}

// AddFile checks generated declarations against the existing package scope and
// makes their type information available to the normal SSA builder.
func AddFile(p *packages.Package, name, source string, imports Importer, minLang string) error {
	file, err := parser.ParseFile(p.Fset, p.PkgPath+"/"+name, source, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object),
		Instances: make(map[*ast.Ident]types.Instance), Scopes: make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), FileVersions: make(map[*ast.File]string),
	}
	lang := p.Types.GoVersion()
	if minLang != "" && lang != "" && version.Compare(lang, minLang) < 0 {
		file.GoVersion = minLang
	}
	conf := types.Config{Importer: imports, Sizes: p.TypesSizes, GoVersion: lang}
	if err := types.NewChecker(&conf, p.Fset, p.Types, info).Files([]*ast.File{file}); err != nil {
		return fmt.Errorf("type-check %s for %s: %w", name, p.PkgPath, err)
	}
	maps.Copy(p.TypesInfo.Types, info.Types)
	maps.Copy(p.TypesInfo.Defs, info.Defs)
	maps.Copy(p.TypesInfo.Uses, info.Uses)
	maps.Copy(p.TypesInfo.Implicits, info.Implicits)
	maps.Copy(p.TypesInfo.Instances, info.Instances)
	maps.Copy(p.TypesInfo.Scopes, info.Scopes)
	maps.Copy(p.TypesInfo.Selections, info.Selections)
	maps.Copy(p.TypesInfo.FileVersions, info.FileVersions)
	p.Syntax = append(p.Syntax, file)
	return nil
}

// BindCall installs a checked function reference for the normal SSA builder.
func BindCall(info *types.Info, call *ast.CallExpr, fn *types.Func, inst types.Instance) {
	id := &ast.Ident{Name: fn.Name(), NamePos: call.Fun.Pos()}
	tv := info.Types[ast.Unparen(call.Fun)]
	tv.Type = inst.Type
	info.Uses[id], info.Types[id] = fn, tv
	if inst.TypeArgs != nil {
		if info.Instances == nil {
			info.Instances = make(map[*ast.Ident]types.Instance)
		}
		info.Instances[id] = inst
	}
	call.Fun = id
}
