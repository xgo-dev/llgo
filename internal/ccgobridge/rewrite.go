// Package ccgobridge adapts an explicitly selected CCGo source graph to full,
// typed function boxes. It is not part of the ordinary unsafe conversion path.
package ccgobridge

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// Stats counts adaptations, not successful compilation or execution.
type Stats struct {
	StaticBoxes int
	Exports     int
	Imports     int
	Slots       int
}

type exportSite struct {
	call *ast.CallExpr
	file *ast.File
	fn   *types.Func
}

// ExportedFunctions reports the unique static declarations exported by a
// selected package without changing its syntax. A graph-level caller must
// reject functions exported by multiple packages: package-local boxes cannot
// preserve the original cross-package function-pointer identity.
// Instantiated generic functions, methods, and function variables are not
// supported by the shared export-site validation.
func ExportedFunctions(fset *token.FileSet, info *types.Info, files []*ast.File) ([]*types.Func, error) {
	sites, err := staticExportSites(fset, info, files)
	if err != nil {
		return nil, err
	}
	var functions []*types.Func
	seen := make(map[*types.Func]bool)
	for _, site := range sites {
		if !seen[site.fn] {
			seen[site.fn] = true
			functions = append(functions, site.fn)
		}
	}
	return functions, nil
}

func staticExportSites(fset *token.FileSet, info *types.Info, files []*ast.File) ([]exportSite, error) {
	helpers := make(map[types.Object]bool)
	for _, file := range files {
		var reserved string
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && strings.HasPrefix(id.Name, "__llgo_ccgo_") {
				reserved = id.Name
			}
			return reserved == ""
		})
		if reserved != "" {
			return nil, fmt.Errorf("CCGo generated identifier %s already exists", reserved)
		}
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || (fd.Name.Name != "__ccgo_fp" && fd.Name.Name != "cFuncPointer") {
				continue
			}
			if !validExporter(fset, info, fd) {
				return nil, fmt.Errorf("%s: CCGo exporter %s has an unrecognized ABI shape", fset.Position(fd.Pos()), fd.Name.Name)
			}
			helpers[info.Defs[fd.Name]] = true
		}
	}

	var sites []exportSite
	used := make(map[*ast.Ident]bool)
	var invalid error
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || invalid != nil {
				return invalid == nil
			}
			id := calledIdent(call.Fun)
			if id == nil || !helpers[info.Uses[id]] {
				return true
			}
			used[id] = true
			var fn *types.Func
			if len(call.Args) == 1 {
				fn, _ = referencedObject(info, call.Args[0]).(*types.Func)
			}
			if fn == nil || fn.Type().(*types.Signature).Recv() != nil {
				invalid = fmt.Errorf("%s: CCGo export requires a static function declaration, not a closure, method, or variable", fset.Position(call.Pos()))
				return false
			}
			// An inferred instantiation can still be a plain identifier in the
			// AST. Reject the generic declaration, not only indexed syntax.
			if fn.Type().(*types.Signature).TypeParams().Len() != 0 {
				invalid = fmt.Errorf("%s: CCGo export does not support generic function instantiations", fset.Position(call.Pos()))
				return false
			}
			if info.Uses[id].Name() == "cFuncPointer" {
				// A static function passed through T=any becomes an interface:
				// the original helper then reads a type word, not a function
				// descriptor. Only a matching instantiated function type has
				// the representation qualified by this source adaptation.
				sig, ok := info.TypeOf(call.Fun).(*types.Signature)
				if !ok || !types.Identical(sig.Params().At(0).Type().Underlying(), fn.Type()) {
					invalid = fmt.Errorf("%s: CCGo generic exporter requires a matching function-type argument", fset.Position(call.Pos()))
					return false
				}
			}
			sites = append(sites, exportSite{call: call, file: file, fn: fn})
			return true
		})
	}
	if invalid != nil {
		return nil, invalid
	}
	for id, obj := range info.Uses {
		if helpers[obj] && !used[id] {
			return nil, fmt.Errorf("%s: CCGo exporter must be called directly, not stored or passed as a value", fset.Position(id.Pos()))
		}
	}
	return sites, nil
}

// Rewrite validates all export sites before changing syntax. Only a statically
// named, non-method function declaration can acquire a process-lifetime box.
// Closures require separately owned roots, never this static path. Callers
// adapting multiple packages must first check ExportedFunctions for duplicate
// declaration owners across the entire selected graph.
func Rewrite(fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File) (Stats, error) {
	var stats Stats
	sites, err := staticExportSites(fset, info, files)
	if err != nil {
		return stats, err
	}
	boxes := make(map[*types.Func]string)
	for _, site := range sites {
		name, ok := boxes[site.fn]
		if !ok {
			name = fmt.Sprintf("__llgo_ccgo_box_%d", len(boxes))
			if pkg.Scope().Lookup(name) != nil {
				return Stats{}, fmt.Errorf("CCGo generated name %s already exists", name)
			}
			boxes[site.fn] = name
		}
	}
	stats.Slots, err = RewriteSlots(fset, pkg, info, files)
	if err != nil {
		return Stats{}, err
	}
	if len(sites) > 0 {
		file := sites[0].file
		unsafeName := unsafeImport(fset, file)
		syncName := packageImport(fset, file, "sync", "sync")
		helper, err := parser.ParseFile(fset, "ccgo-static.go", fmt.Sprintf(`package bridge
type __llgo_ccgo_staticBox[T any] struct {
	once %s.Once
	fn T
}
func __llgo_ccgo_export[T any](box *__llgo_ccgo_staticBox[T], fn T) uintptr {
	box.once.Do(func() { box.fn = fn })
	return uintptr(%s.Pointer(&box.fn))
}
`, syncName, unsafeName), 0)
		if err != nil {
			return Stats{}, err
		}
		file.Decls = append(file.Decls, helper.Decls...)
	}
	declared := make(map[string]bool)
	for _, site := range sites {
		name := boxes[site.fn]
		if !declared[name] {
			// Lazy initialization avoids an initialization dependency cycle when
			// a function exports itself. The typed root and address are stable;
			// sync.Once also covers exports during concurrent package use.
			typeText := types.TypeString(site.fn.Type(), func(dep *types.Package) string {
				if dep.Path() == pkg.Path() {
					return ""
				}
				return packageImport(fset, site.file, dep.Path(), dep.Name())
			})
			fnType, err := parser.ParseExpr(typeText)
			if err != nil {
				return Stats{}, err
			}
			site.file.Decls = append(site.file.Decls, &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{&ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdent(name)},
					Type:  &ast.IndexExpr{X: ast.NewIdent("__llgo_ccgo_staticBox"), Index: fnType},
				}},
			})
			declared[name] = true
		}
		pos := site.call.Pos()
		ident := func(name string) *ast.Ident { return &ast.Ident{NamePos: pos, Name: name} }
		fn := site.call.Args[0]
		site.call.Fun = ident("__llgo_ccgo_export")
		site.call.Args = []ast.Expr{&ast.UnaryExpr{OpPos: pos, Op: token.AND, X: ident(name)}, fn}
		stats.Exports++
	}
	stats.StaticBoxes = len(boxes)

	var importFile *ast.File
	for _, file := range files {
		astutil.Apply(file, nil, func(cursor *astutil.Cursor) bool {
			load, ok := cursor.Node().(*ast.StarExpr)
			if !ok {
				return true
			}
			call, ok := unparen(load.X).(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			ptr, ok := info.TypeOf(call).(*types.Pointer)
			if !ok {
				return true
			}
			if _, ok := ptr.Elem().Underlying().(*types.Signature); !ok {
				return true
			}
			pointerType, ok := unparen(call.Fun).(*ast.StarExpr)
			if !ok {
				return true
			}
			cast, ok := unparen(call.Args[0]).(*ast.CallExpr)
			if !ok || !unsafePointerCall(info, cast) {
				return true
			}
			address, ok := unparen(cast.Args[0]).(*ast.UnaryExpr)
			if !ok || address.Op != token.AND {
				return true
			}
			box, ok := unparen(address.X).(*ast.CompositeLit)
			if !ok || len(box.Elts) != 1 {
				return true
			}
			st, ok := info.TypeOf(box).Underlying().(*types.Struct)
			if !ok || st.NumFields() != 1 || !st.Field(0).Embedded() || !types.Identical(st.Field(0).Type(), types.Typ[types.Uintptr]) {
				return true
			}
			if !types.Identical(info.TypeOf(box.Elts[0]), types.Typ[types.Uintptr]) {
				return true
			}
			// Read the complete function value, not a native code pointer.
			// A zero descriptor represents a nil function, not an address to read.
			// Keep the caller's unsafe conversion and import in its original file;
			// a shared helper must not leave another file with an unused import.
			cast.Args = []ast.Expr{box.Elts[0]}
			cursor.Replace(&ast.CallExpr{
				Fun: &ast.IndexExpr{
					X:      &ast.Ident{NamePos: load.Pos(), Name: "__llgo_ccgo_import"},
					Lbrack: pointerType.X.Pos(),
					Index:  pointerType.X,
					Rbrack: pointerType.X.End(),
				},
				Lparen: call.Lparen,
				Args:   []ast.Expr{cast},
				Rparen: call.Rparen,
			})
			if importFile == nil {
				importFile = file
			}
			stats.Imports++
			return true
		})
	}
	if importFile != nil {
		unsafeName := unsafeImport(fset, importFile)
		helper, err := parser.ParseFile(fset, "ccgo-import.go", fmt.Sprintf(`package bridge
func __llgo_ccgo_import[T any](p %s.Pointer) T {
	if p == nil {
		var zero T
		return zero
	}
	return *(*T)(p)
}
`, unsafeName), 0)
		if err != nil {
			return Stats{}, err
		}
		importFile.Decls = append(importFile.Decls, helper.Decls...)
	}
	return stats, nil
}

func validExporter(fset *token.FileSet, info *types.Info, fd *ast.FuncDecl) bool {
	if fd.Recv != nil || fd.Body == nil {
		return false
	}
	fn, ok := info.Defs[fd.Name].(*types.Func)
	if !ok {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 1 ||
		!types.Identical(sig.Results().At(0).Type(), types.Typ[types.Uintptr]) {
		return false
	}
	parameter := sig.Params().At(0)
	var typeParameter *types.TypeParam
	if fd.Name.Name == "cFuncPointer" {
		if sig.TypeParams().Len() != 1 {
			return false
		}
		typeParameter = sig.TypeParams().At(0)
		if !types.Identical(parameter.Type(), typeParameter) || !emptyInterface(typeParameter.Constraint()) {
			return false
		}
	} else if sig.TypeParams().Len() != 0 || !emptyInterface(parameter.Type()) {
		return false
	}

	var body bytes.Buffer
	if format.Node(&body, fset, fd.Body) != nil {
		return false
	}
	expected := "{\n\ttype iface [2]uintptr\n\treturn (*iface)(unsafe.Pointer(&f))[1]\n}"
	if fd.Name.Name == "cFuncPointer" {
		expected = "{\n\treturn *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))\n}"
	}
	if syntaxTokens(body.Bytes()) != syntaxTokens([]byte(expected)) {
		return false
	}
	validUnsafe, validIdentity := false, true
	ast.Inspect(fd.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && unsafePointerCall(info, call) {
			validUnsafe = true
		}
		if id, ok := node.(*ast.Ident); ok {
			if used := info.Uses[id]; used != nil {
				switch id.Name {
				case "f":
					validIdentity = validIdentity && used == parameter
				case "T":
					validIdentity = validIdentity && typeParameter != nil && used == typeParameter.Obj()
				case "uintptr":
					validIdentity = validIdentity && used == types.Universe.Lookup("uintptr")
				}
			}
		}
		return true
	})
	return validUnsafe && validIdentity
}

func emptyInterface(typ types.Type) bool {
	iface, ok := typ.Underlying().(*types.Interface)
	return ok && iface.Empty()
}

func syntaxTokens(source []byte) string {
	set := token.NewFileSet()
	var scan scanner.Scanner
	scan.Init(set.AddFile("shape", -1, len(source)), source, nil, 0)
	var result strings.Builder
	for {
		_, tok, literal := scan.Scan()
		if tok == token.EOF {
			return result.String()
		}
		fmt.Fprintf(&result, "%d:%q;", tok, literal)
	}
}

func unsafePointerCall(info *types.Info, call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	obj := referencedObject(info, call.Fun)
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "unsafe" && obj.Name() == "Pointer"
}

func referencedObject(info *types.Info, expr ast.Expr) types.Object {
	switch expr := unparen(expr).(type) {
	case *ast.Ident:
		return info.ObjectOf(expr)
	case *ast.SelectorExpr:
		return info.ObjectOf(expr.Sel)
	}
	return nil
}

func calledIdent(expr ast.Expr) *ast.Ident {
	switch expr := unparen(expr).(type) {
	case *ast.Ident:
		return expr
	case *ast.IndexExpr:
		return calledIdent(expr.X)
	}
	return nil
}

func unparen(expr ast.Expr) ast.Expr {
	return ast.Unparen(expr)
}

func unsafeImport(fset *token.FileSet, file *ast.File) string {
	return packageImport(fset, file, "unsafe", "unsafe")
}

func packageImport(fset *token.FileSet, file *ast.File, path, name string) string {
	for _, spec := range file.Imports {
		if spec.Path.Value == fmt.Sprintf("%q", path) {
			if spec.Name == nil {
				return name
			}
			return spec.Name.Name
		}
	}
	alias := "__llgo_ccgo_" + name
	astutil.AddNamedImport(fset, file, alias, path)
	return alias
}
