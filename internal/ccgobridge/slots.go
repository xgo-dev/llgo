package ccgobridge

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

const slotPrefix = "__ccgo_abiInternal_"

type slotRewrite struct {
	function *ast.FuncDecl
	closure  *ast.FuncLit
	variable string
	receiver string
	dest     string
}

// RewriteSlots adapts explicitly selected CCGo callback slots, not arbitrary
// unsafe conversions. It requires the complete type-checked package and the
// musl TLS stack lifecycle. Every source shape is validated before any edit.
// Callers must separately adapt consumers to load a complete typed function
// from the slot address; these addresses are neither C nor ABI0 code pointers.
func RewriteSlots(fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File) (int, error) {
	var plans []slotRewrite
	var tls *types.Named
	for _, file := range files {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, slotPrefix) {
				continue
			}
			plan, owner, err := slotValidateFunction(fset, pkg, info, fn)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", fset.Position(fn.Pos()), err)
			}
			if tls != nil && tls != owner {
				return 0, fmt.Errorf("CCGo callback slots have different TLS owners")
			}
			tls = owner
			plans = append(plans, plan)
		}
	}
	if len(plans) == 0 {
		return 0, nil
	}
	ownerFile, structure, free, receiver, err := slotValidateTLS(fset, pkg, info, files, tls)
	if err != nil {
		return 0, err
	}
	helper, err := parser.ParseFile(fset, "ccgo-slot-helpers.go", "package "+pkg.Name()+"\n"+slotHelpers, 0)
	if err != nil {
		return 0, err
	}

	for _, plan := range plans {
		guard, _ := parser.ParseExprFrom(fset, "ccgo-slot-size.go", "[16 - unsafe.Sizeof("+plan.variable+")]byte", 0)
		check := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("_")}, Type: guard},
		}}}
		keep, _ := parser.ParseExprFrom(fset, "ccgo-slot-keep.go", fmt.Sprintf(
			"%s.__llgoCCGoKeep(%s, unsafe.Sizeof(%s), unsafe.Alignof(%s), %s)",
			plan.receiver, plan.dest, plan.variable, plan.variable, plan.variable), 0)
		address, _ := parser.ParseExprFrom(fset, "ccgo-slot-address.go", "unsafe.Pointer("+plan.dest+")", 0)
		store := &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.StarExpr{X: &ast.CallExpr{
				Fun: &ast.ParenExpr{X: &ast.StarExpr{X: plan.closure.Type}}, Args: []ast.Expr{address},
			}}},
			Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent(plan.variable)},
		}
		plan.function.Body.List = []ast.Stmt{plan.function.Body.List[0], check, &ast.ExprStmt{X: keep}, store}
	}
	structure.Fields.List = append(structure.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("__llgoCCGoRoots")},
		Type:  &ast.ArrayType{Elt: ast.NewIdent("__llgoCCGoRoot")},
	})
	release, _ := parser.ParseExprFrom(fset, "ccgo-slot-release.go", receiver+".__llgoCCGoRelease()", 0)
	free.Body.List = append([]ast.Stmt{free.Body.List[0], &ast.ExprStmt{X: release}}, free.Body.List[1:]...)
	ownerFile.Decls = append(ownerFile.Decls, helper.Decls...)
	return len(plans), nil
}

func slotValidateFunction(fset *token.FileSet, pkg *types.Package, info *types.Info, fn *ast.FuncDecl) (slotRewrite, *types.Named, error) {
	bad := func() (slotRewrite, *types.Named, error) {
		return slotRewrite{}, nil, fmt.Errorf("unrecognized CCGo callback slot shape: %s", fn.Name.Name)
	}
	object, ok := info.Defs[fn.Name].(*types.Func)
	if !ok || fn.Recv != nil || fn.Body == nil || len(fn.Body.List) != 2 {
		return bad()
	}
	sig := object.Type().(*types.Signature)
	if sig.Params().Len() != 3 || sig.Results().Len() != 0 || sig.Variadic() ||
		!types.Identical(sig.Params().At(1).Type(), types.Typ[types.Uintptr]) ||
		!types.Identical(sig.Params().At(2).Type(), types.Typ[types.Uintptr]) {
		return bad()
	}
	pointer, ok := sig.Params().At(0).Type().(*types.Pointer)
	if !ok {
		return bad()
	}
	tls, ok := pointer.Elem().(*types.Named)
	if !ok || tls.Obj().Pkg() != pkg || tls.Obj().Name() != "TLS" {
		return bad()
	}
	assignment, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return bad()
	}
	variable, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok {
		return bad()
	}
	closure, ok := assignment.Rhs[0].(*ast.FuncLit)
	if !ok || len(closure.Body.List) != 1 {
		return bad()
	}
	closureSig, ok := info.TypeOf(closure).(*types.Signature)
	if !ok || closureSig.Variadic() {
		return bad()
	}
	var call *ast.CallExpr
	switch statement := closure.Body.List[0].(type) {
	case *ast.ReturnStmt:
		if len(statement.Results) == 1 && closureSig.Results().Len() > 0 {
			call, _ = statement.Results[0].(*ast.CallExpr)
		}
	case *ast.ExprStmt:
		if closureSig.Results().Len() == 0 {
			call, _ = statement.X.(*ast.CallExpr)
		}
	}
	if call == nil || call.Ellipsis.IsValid() || len(call.Args) != closureSig.Params().Len()+1 {
		return bad()
	}
	callee, ok := call.Fun.(*ast.Ident)
	if !ok || callee.Name != "__ccgo_abi0_"+strings.TrimPrefix(fn.Name.Name, slotPrefix) {
		return bad()
	}
	target, ok := info.Uses[callee].(*types.Func)
	if !ok || target.Pkg() != pkg {
		return bad()
	}
	for i, argument := range call.Args {
		identifier, ok := argument.(*ast.Ident)
		want := sig.Params().At(2)
		if i < closureSig.Params().Len() {
			want = closureSig.Params().At(i)
		}
		if !ok || info.Uses[identifier] != want {
			return bad()
		}
	}
	store, ok := fn.Body.List[1].(*ast.AssignStmt)
	if !ok || store.Tok != token.ASSIGN || len(store.Lhs) != 1 || len(store.Rhs) != 1 {
		return bad()
	}
	lhs := "*(*[2]uintptr)(unsafe.Pointer(" + sig.Params().At(1).Name() + "))"
	rhs := "*(*[2]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(&" + variable.Name + "))))"
	if !slotSameExpr(fset, store.Lhs[0], lhs) || !slotSameExpr(fset, store.Rhs[0], rhs) {
		return bad()
	}
	valid := true
	ast.Inspect(store, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		switch identifier.Name {
		case "unsafe":
			imported, ok := info.Uses[identifier].(*types.PkgName)
			valid = valid && ok && imported.Imported().Path() == "unsafe"
		case "uintptr":
			valid = valid && info.Uses[identifier] == types.Universe.Lookup("uintptr")
		}
		return true
	})
	if !valid {
		return bad()
	}
	return slotRewrite{fn, closure, variable.Name, sig.Params().At(0).Name(), sig.Params().At(1).Name()}, tls, nil
}

func slotValidateTLS(fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File, tls *types.Named) (*ast.File, *ast.StructType, *ast.FuncDecl, string, error) {
	bad := func() (*ast.File, *ast.StructType, *ast.FuncDecl, string, error) {
		return nil, nil, nil, "", fmt.Errorf("unrecognized CCGo TLS callback ownership lifecycle")
	}
	for _, name := range []string{"__llgoCCGoRoot", "__llgoCCGoKeep", "__llgoCCGoRelease"} {
		if pkg.Scope().Lookup(name) != nil {
			return bad()
		}
	}
	structure, ok := tls.Underlying().(*types.Struct)
	if !ok || !types.Identical(slotField(structure, "sp"), types.Typ[types.Int]) {
		return bad()
	}
	for _, name := range []string{"__llgoCCGoRoots", "__llgoCCGoKeep", "__llgoCCGoRelease"} {
		if slotField(structure, name) != nil {
			return bad()
		}
	}
	stack, ok := slotField(structure, "stack").(*types.Slice)
	if !ok {
		return bad()
	}
	entry, ok := stack.Elem().Underlying().(*types.Struct)
	if !ok || !types.Identical(slotField(entry, "p"), types.Typ[types.Uintptr]) {
		return bad()
	}
	size := slotField(entry, "sz")
	if size == nil || !types.Identical(size.Underlying(), types.Typ[types.Uint64]) {
		return bad()
	}
	var ownerFile *ast.File
	var syntax *ast.StructType
	var free, close *ast.FuncDecl
	var receiver string
	for _, file := range files {
		for _, declaration := range file.Decls {
			switch decl := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					if spec, ok := spec.(*ast.TypeSpec); ok && info.Defs[spec.Name] == tls.Obj() {
						syntax, _ = spec.Type.(*ast.StructType)
						ownerFile = file
					}
				}
			case *ast.FuncDecl:
				object, ok := info.Defs[decl.Name].(*types.Func)
				if !ok {
					continue
				}
				sig := object.Type().(*types.Signature)
				if sig.Recv() == nil || !types.Identical(sig.Recv().Type(), types.NewPointer(tls)) {
					continue
				}
				switch decl.Name.Name {
				case "__llgoCCGoKeep", "__llgoCCGoRelease":
					return bad()
				case "Free":
					if sig.Params().Len() != 1 || !types.Identical(sig.Params().At(0).Type(), types.Typ[types.Int]) ||
						sig.Results().Len() != 0 || decl.Body == nil || len(decl.Body.List) != 3 {
						return bad()
					}
					body := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}}, Body: decl.Body}
					want := fmt.Sprintf("func() { %[1]s.sp--; if !%[1]s.checkSignals { return }; %[1]s.checkSignal() }", sig.Recv().Name())
					if !slotSameExpr(fset, body, want) {
						return bad()
					}
					free, receiver = decl, sig.Recv().Name()
				case "Close":
					if sig.Params().Len() != 0 || sig.Results().Len() != 0 ||
						decl.Body == nil || len(decl.Body.List) == 0 {
						return bad()
					}
					cleanup, ok := decl.Body.List[0].(*ast.DeferStmt)
					if !ok || !slotSameExpr(fset, cleanup.Call, "func() { *"+sig.Recv().Name()+" = TLS{} }()") {
						return bad()
					}
					close = decl
				}
			}
		}
	}
	if ownerFile == nil || syntax == nil || free == nil || close == nil {
		return bad()
	}
	return ownerFile, syntax, free, receiver, nil
}

func slotField(structure *types.Struct, name string) types.Type {
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if field.Name() == name && !field.Embedded() {
			return field.Type()
		}
	}
	return nil
}

func slotSameExpr(_ *token.FileSet, expression ast.Expr, want string) bool {
	parsed, err := parser.ParseExpr(want)
	if err != nil {
		return false
	}
	var actual, expected bytes.Buffer
	return format.Node(&actual, token.NewFileSet(), expression) == nil &&
		format.Node(&expected, token.NewFileSet(), parsed) == nil && bytes.Equal(actual.Bytes(), expected.Bytes())
}

const slotHelpers = `
type __llgoCCGoRoot struct {
	depth int
	slot uintptr
	fn interface{}
}

func (tls *TLS) __llgoCCGoKeep(dest, size, align uintptr, fn interface{}) {
	if tls.sp <= 0 || tls.sp > len(tls.stack) {
		panic("ccgo: callback requires an active TLS allocation")
	}
	if align == 0 || dest%align != 0 {
		panic("ccgo: callback slot is not aligned")
	}
	frame := tls.stack[tls.sp-1]
	if dest < frame.p || size > uintptr(frame.sz) || dest-frame.p > uintptr(frame.sz)-size {
		panic("ccgo: callback slot is outside the current TLS allocation")
	}
	for i := len(tls.__llgoCCGoRoots)-1; i >= 0 && tls.__llgoCCGoRoots[i].depth == tls.sp; i-- {
		if tls.__llgoCCGoRoots[i].slot == dest {
			tls.__llgoCCGoRoots[i].fn = fn
			return
		}
	}
	tls.__llgoCCGoRoots = append(tls.__llgoCCGoRoots, __llgoCCGoRoot{tls.sp, dest, fn})
}

func (tls *TLS) __llgoCCGoRelease() {
	for len(tls.__llgoCCGoRoots) != 0 {
		last := len(tls.__llgoCCGoRoots)-1
		if tls.__llgoCCGoRoots[last].depth <= tls.sp {
			return
		}
		tls.__llgoCCGoRoots[last] = __llgoCCGoRoot{}
		tls.__llgoCCGoRoots = tls.__llgoCCGoRoots[:last]
	}
}
`
