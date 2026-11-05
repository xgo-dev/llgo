package packages

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/packages"
)

// gc requires a body (or assembly/linkname) when producing export data. LLGo
// owns llvm.* bindings itself. Discard only that driver's missing-body error
// at an explicitly linked declaration; all other diagnostics remain intact.
func allowLLVMIntrinsicDeclarations(pkg *Package, fset *token.FileSet) {
	var positions []token.Position
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body != nil {
				continue
			}
			for _, d := range directive.ParseGroup(fn.Doc) {
				local, target := directive.ParseLinknameArgs(d.Args)
				if d.Name == "llgo:link" && local == fn.Name.Name && strings.HasPrefix(target, "llvm.") {
					positions = append(positions, fset.Position(fn.Name.Pos()))
				}
			}
		}
	}
	if len(positions) == 0 {
		return
	}
	filtered := pkg.Errors[:0]
	for _, err := range pkg.Errors {
		if err.Kind == packages.ListError && strings.HasPrefix(err.Msg, "# ") {
			lines := strings.Split(strings.TrimSpace(err.Msg), "\n")
			kept := lines[:1]
			for _, line := range lines[1:] {
				matched := false
				if pos, ok := strings.CutSuffix(line, ": missing function body"); ok {
					for _, decl := range positions {
						if strings.HasSuffix(pos, fmt.Sprintf(":%d:%d", decl.Line, decl.Column)) && sameDiagnosticLine(pos, decl) {
							matched = true
							break
						}
					}
				}
				if !matched {
					kept = append(kept, line)
				}
			}
			if len(kept) == 1 && len(lines) > 1 {
				continue
			}
			err.Msg = strings.Join(kept, "\n")
		}
		filtered = append(filtered, err)
	}
	pkg.Errors = filtered
}
