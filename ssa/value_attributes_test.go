package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestValidateDirectiveContracts(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, conflict := range []bool{false, true} {
			prog := NewProgram(nil)
			fset := token.NewFileSet()
			var records []*directive.Package
			order := []string{"a", "b"}
			if reverse {
				order[0], order[1] = order[1], order[0]
			}
			for _, name := range order {
				upper := "64"
				if conflict && name == "b" {
					upper = "32"
				}
				file, err := parser.ParseFile(fset, name+".go", "package "+name+"\n//llgo:link F shared\n//llgo:result range(0,"+upper+")\nfunc F() int\n", parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
				pkg, err := (&types.Config{}).Check(name, fset, []*ast.File{file}, info)
				if err != nil {
					t.Fatal(err)
				}
				r := directive.Collect(prog.Directives().Files([]*ast.File{file}), false, false)
				r.Bind(info)
				prog.SetPackageDirectives(pkg, r)
				records = append(records, r)
			}
			prog.Directives().Freeze()
			err := prog.ValidateDirectiveContracts(fset)
			if conflict {
				if err == nil || !strings.Contains(err.Error(), "conflicting range") || !strings.Contains(err.Error(), ".go:3:") {
					t.Fatalf("conflict = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if records[0].Names["F"] == records[1].Names["F"] {
				t.Fatal("aliased declarations were merged")
			}
			prog.Dispose()
		}
	}
}
