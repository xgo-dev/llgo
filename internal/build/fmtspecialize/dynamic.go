package fmtspecialize

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

const builtinSprintf = "llgoSprintfBuiltins"

const allBuiltinKinds = (1 << (types.String - types.Bool + 1)) - 1

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
	usesTypeName := false
	for _, name := range names {
		fn := methods[name]
		fn.Name.Name = renamed[name]
		fn.Type.Params.List = append(fn.Recv.List, fn.Type.Params.List...)
		fn.Type.Params.List = append(fn.Type.Params.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent("llgoKinds")}, Type: ast.NewIdent("uint32"),
		})
		fn.Recv = nil
		unsupported := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.CallExpr:
				switch name {
				case "badVerb", "printArg", "doPrintf":
					if rewriteBuiltinTypeName(node) {
						usesTypeName = true
						return false
					}
				}
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok && identifier(sel.X, "p") {
					if replacement, ok := renamed[sel.Sel.Name]; ok {
						node.Fun = ast.NewIdent(replacement)
						node.Args = append(append([]ast.Expr{sel.X}, node.Args...), ast.NewIdent("llgoKinds"))
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
func llgoSprintfBuiltins(format string, llgoKinds uint32, a ...any) string {
	p := newPrinter()
	llgoBuiltin_doPrintf(p, format, a, llgoKinds)
	s := string(p.buf)
	p.free()
	return s
}
`)
	if usesTypeName {
		writeBuiltinTypeName(&out)
	}
	return out.Bytes()
}

// The builtin formatter is only selected when every operand is a builtin
// scalar. This includes aliases: %T reports uint8 for byte and int32 for rune.
func rewriteBuiltinTypeName(call *ast.CallExpr) bool {
	method, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || method.Sel.Name != "String" || len(call.Args) != 0 {
		return false
	}
	typeOf, ok := method.X.(*ast.CallExpr)
	if !ok || len(typeOf.Args) != 1 || !identifier(typeOf.Args[0], "arg") {
		return false
	}
	fn, ok := typeOf.Fun.(*ast.SelectorExpr)
	if !ok || fn.Sel.Name != "TypeOf" || !identifier(fn.X, "reflect") {
		return false
	}
	call.Fun = ast.NewIdent("llgoBuiltin_typeName")
	call.Args = typeOf.Args
	return true
}

func writeBuiltinTypeName(out *bytes.Buffer) {
	out.WriteString("\nfunc llgoBuiltin_typeName(arg any) string {\n\tswitch arg.(type) {\n")
	for kind := types.Bool; kind <= types.String; kind++ {
		name := types.Typ[kind].Name()
		fmt.Fprintf(out, "\tcase %s:\n\t\treturn %q\n", name, name)
	}
	out.WriteString("\t}\n\tpanic(\"fmt: unexpected builtin format argument\")\n}\n")
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
	var kinds uint32
	for _, item := range typesSwitch.Body.List {
		clause := item.(*ast.CaseClause)
		if len(clause.List) == 0 {
			continue
		}
		if len(clause.List) != 1 {
			return false
		}
		mask := builtinTypeMask(clause.List[0])
		if mask != 0 {
			kinds |= mask
			clause.Body = []ast.Stmt{guardBuiltinPrinter(mask, clause.Body)}
			cases = append(cases, clause)
		}
	}
	if kinds != allBuiltinKinds || len(cases) != int(types.String-types.Bool+1) {
		return false
	}
	typesSwitch.Body.List = cases
	return true
}

// A single mask is passed by all specialized calls in the executable, so
// Full LTO can discard printers that its callers do not use.
func guardBuiltinPrinter(mask uint32, body []ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X: ast.NewIdent("llgoKinds"), Op: token.AND,
				Y: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(mask)},
			},
			Op: token.NEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"},
		},
		Body: &ast.BlockStmt{List: body},
	}
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

func builtinTypeMask(expr ast.Expr) uint32 {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return 0
	}
	obj := types.Universe.Lookup(id.Name)
	if obj == nil {
		return 0
	}
	if basic, ok := types.Unalias(obj.Type()).(*types.Basic); ok {
		return builtinKindMask(basic.Kind())
	}
	return 0
}

func builtinKindMask(kind types.BasicKind) uint32 {
	if kind < types.Bool || kind > types.String {
		return 0
	}
	return 1 << (kind - types.Bool)
}

func builtinFormatKinds(call ssa.CallInstruction) uint32 {
	operands := scalarFormatOperands(call, call.Common().Args[1])
	var kinds uint32
	for _, kind := range operands {
		mask := builtinKindMask(kind)
		if mask == 0 {
			return 0
		}
		kinds |= mask
	}
	return kinds
}
