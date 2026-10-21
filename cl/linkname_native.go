package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

const stdcallVariableTypeSuffix = "$stdcall"

func nativeLinkTarget(target string) (string, llssa.CallConv, bool) {
	if strings.HasPrefix(target, "C.") {
		return strings.TrimPrefix(target, "C."), llssa.CDecl, true
	}
	if strings.HasPrefix(target, "stdcall.") {
		return strings.TrimPrefix(target, "stdcall."), llssa.Stdcall, true
	}
	return "", 0, false
}

func collectNativeLinkDefinitions(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, aliases map[string]ast.Expr) error {
	pkgPath := llssa.PathOf(pkg)
	for _, file := range files {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				name, _ := astFuncName(pkgPath, decl, aliases)
				if prog.LinknameDirective(name) == "llgo:link" && decl.Body == nil {
					return fmt.Errorf("%s: llgo:link requires a placeholder function body", fset.Position(decl.Pos()))
				}
				target, _ := prog.Linkname(name)
				native, cc, ok := nativeLinkTarget(target)
				if !ok {
					continue
				}
				if native == "" {
					return fmt.Errorf("%s: native linkname requires a symbol name", fset.Position(decl.Pos()))
				}
				if decl.Body == nil || prog.LinknameDirective(name) != "go:linkname" {
					continue
				}
				if native == name {
					return fmt.Errorf("%s: native export symbol %q conflicts with its Go implementation", fset.Position(decl.Pos()), native)
				}
				if decl.Recv != nil {
					return fmt.Errorf("%s: native linkname export requires a function, not a method", fset.Position(decl.Pos()))
				}
				if previous, ok := prog.PackageExport(name); ok && previous != native {
					return fmt.Errorf("%s: conflicting native export names %q and %q", fset.Position(decl.Pos()), previous, native)
				}
				if decl.Type.TypeParams != nil {
					return fmt.Errorf("%s: native export %s has an unsupported signature", fset.Position(decl.Pos()), decl.Name.Name)
				}
				prog.SetPackageExport(name, native)
				prog.SetNativeExport(name, llssa.NativeExportInfo{CallConv: cc})
			case *ast.GenDecl:
				if decl.Tok != token.VAR {
					continue
				}
				for _, spec := range decl.Specs {
					value := spec.(*ast.ValueSpec)
					for _, ident := range value.Names {
						name := pkgPath + "." + ident.Name
						target, _ := prog.Linkname(name)
						if strings.HasPrefix(target, "llgo.") {
							return fmt.Errorf("%s: llgo. linkname namespace applies only to functions", fset.Position(ident.Pos()))
						}
						native, cc, ok := nativeLinkTarget(target)
						if !ok {
							continue
						}
						if native == "" {
							return fmt.Errorf("%s: native linkname requires a symbol name", fset.Position(ident.Pos()))
						}
						if len(value.Values) != 0 {
							return fmt.Errorf("%s: native extern variable %s cannot have a Go initializer", fset.Position(ident.Pos()), ident.Name)
						}
						if cc == llssa.Stdcall {
							prog.SetTypeCallConv(name+stdcallVariableTypeSuffix, llssa.Stdcall)
						}
					}
				}
			}
		}
	}
	return nil
}

// A stdcall data symbol has ordinary native storage. Only a function value
// stored in it needs stdcall type metadata, which also survives loads and
// conversions to an ordinary Go function value.
func (p *context) nativeVariableType(v *ssa.Global) types.Type {
	typ := p.patchType(v.Type())
	name := llssa.FullName(v.Pkg.Pkg, v.Name())
	target, _ := p.prog.Linkname(name)
	if strings.HasPrefix(target, "stdcall.") {
		elem := typ.Underlying().(*types.Pointer).Elem()
		if sig, ok := elem.Underlying().(*types.Signature); ok {
			named := types.NewNamed(types.NewTypeName(v.Pos(), v.Pkg.Pkg, v.Name()+stdcallVariableTypeSuffix, nil), sig, nil)
			return types.NewPointer(named)
		}
	}
	return typ
}
