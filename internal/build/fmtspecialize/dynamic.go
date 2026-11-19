package fmtspecialize

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

const builtinSprintf = "llgoSprintfBuiltins"

// BuiltinFormatSource reuses the selected Go toolchain's parser and flag
// handling. The generated parser dispatches only to builtin scalar printers.
// A different print.go shape leaves ordinary fmt in use.
func BuiltinFormatSource(source []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "print.go", source, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}

	names := []string{"doPrintf", "printArg", "badVerb", "fmtBool", "fmtInteger", "fmtFloat", "fmtComplex", "fmtString"}
	methods := make(map[string]*ast.FuncDecl, len(names))
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		recv := fn.Recv.List[0]
		ptr, ok := recv.Type.(*ast.StarExpr)
		if ok && identifier(ptr.X, "pp") && len(recv.Names) == 1 && recv.Names[0].Name == "p" {
			methods[fn.Name.Name] = fn
		}
	}
	for _, name := range names {
		if methods[name] == nil {
			return nil
		}
	}
	if !restrictBuiltinPrintArg(methods["printArg"]) || !restrictBuiltinBadVerb(methods["badVerb"]) {
		return nil
	}

	renamed := make(map[string]string, len(names))
	for _, name := range names {
		renamed[name] = "llgoBuiltin_" + name
	}
	var out bytes.Buffer
	out.WriteString("package fmt\nimport (\"reflect\"; \"unicode/utf8\")\n")
	for _, name := range names {
		fn := methods[name]
		fn.Name.Name = renamed[name]
		fn.Type.Params.List = append(fn.Recv.List, fn.Type.Params.List...)
		fn.Recv = nil
		unsupported := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.CallExpr:
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok && identifier(sel.X, "p") {
					if replacement, ok := renamed[sel.Sel.Name]; ok {
						node.Fun = ast.NewIdent(replacement)
						node.Args = append([]ast.Expr{sel.X}, node.Args...)
					}
				}
			case *ast.SelectorExpr:
				if identifier(node.X, "p") {
					switch node.Sel.Name {
					case "printValue", "fmtPointer", "handleMethods", "catchPanic", "fmtBytes":
						unsupported = true
					}
					if _, renamed := renamed[node.Sel.Name]; renamed {
						unsupported = true
					}
				}
			}
			return true
		})
		if unsupported || format.Node(&out, fset, fn) != nil {
			return nil
		}
		out.WriteByte('\n')
	}
	out.WriteString(`
func llgoSprintfBuiltins(format string, a ...any) string {
	p := newPrinter()
	llgoBuiltin_doPrintf(p, format, a)
	s := string(p.buf)
	p.free()
	return s
}
`)
	return out.Bytes()
}

func identifier(node ast.Expr, name string) bool {
	id, ok := node.(*ast.Ident)
	return ok && id.Name == name
}

func restrictBuiltinPrintArg(fn *ast.FuncDecl) bool {
	params := fn.Type.Params.List
	if len(params) != 2 || !identifier(params[0].Type, "any") || !identifier(params[1].Type, "rune") {
		return false
	}
	var typesSwitch *ast.TypeSwitchStmt
	var foundPointer bool
	for _, stmt := range fn.Body.List {
		switch stmt := stmt.(type) {
		case *ast.SwitchStmt:
			if !identifier(stmt.Tag, "verb") {
				continue
			}
			for i, item := range stmt.Body.List {
				clause := item.(*ast.CaseClause)
				if len(clause.List) != 1 {
					continue
				}
				if lit, ok := clause.List[0].(*ast.BasicLit); ok && lit.Kind == token.CHAR && lit.Value == "'p'" {
					// Scalar printers reject %p via their usual bad-verb
					// path, producing the same type and value diagnostic.
					stmt.Body.List = append(stmt.Body.List[:i], stmt.Body.List[i+1:]...)
					foundPointer = true
					break
				}
			}
		case *ast.TypeSwitchStmt:
			typesSwitch = stmt
		}
	}
	if !foundPointer || typesSwitch == nil {
		return false
	}
	var cases []ast.Stmt
	for _, item := range typesSwitch.Body.List {
		clause := item.(*ast.CaseClause)
		if len(clause.List) == 0 {
			continue
		}
		if len(clause.List) != 1 {
			return false
		}
		if id, ok := clause.List[0].(*ast.Ident); ok && builtinScalar(types.Universe.Lookup(id.Name)) {
			cases = append(cases, clause)
		}
	}
	if len(cases) != 17 {
		return false
	}
	typesSwitch.Body.List = cases
	return true
}

func restrictBuiltinBadVerb(fn *ast.FuncDecl) bool {
	params := fn.Type.Params.List
	if len(params) != 3 {
		return false
	}
	for _, stmt := range fn.Body.List {
		switchStmt, ok := stmt.(*ast.SwitchStmt)
		if !ok || switchStmt.Tag != nil || len(switchStmt.Body.List) != 3 {
			continue
		}
		clause := switchStmt.Body.List[1].(*ast.CaseClause)
		if len(clause.List) != 1 {
			continue
		}
		call, ok := clause.List[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && identifier(sel.X, "value") && sel.Sel.Name == "IsValid" {
			switchStmt.Body.List = []ast.Stmt{switchStmt.Body.List[0], switchStmt.Body.List[2]}
			return true
		}
	}
	return false
}

func builtinScalar(obj types.Object) bool {
	if obj == nil {
		return false
	}
	basic, ok := types.Unalias(obj.Type()).(*types.Basic)
	return ok && basic.Kind() >= types.Bool && basic.Kind() <= types.String
}

func builtinFormatOperands(call ssa.CallInstruction) bool {
	operands := scalarFormatOperands(call, call.Common().Args[1])
	if len(operands) == 0 {
		return false
	}
	for _, kind := range operands {
		if kind < types.Bool || kind > types.String {
			return false
		}
	}
	return true
}
