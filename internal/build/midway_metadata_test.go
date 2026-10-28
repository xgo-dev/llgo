//go:build !llgo

package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"

	"github.com/xgo-dev/llgo/internal/directive"
)

type midwayMetadataImporter struct{ simd *types.Package }

func (i midwayMetadataImporter) Import(string) (*types.Package, error) { return i.simd, nil }

func TestRewritePortableSIMDMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, source, diagnostic string
		bridge                   bool
	}{
		{"bridge edge", "var Value simd.Uint32s", "", true},
		{"missing bridge", "var Value simd.Uint32s", "missing dependency metadata for simd/internal/bridge", false},
		{"bodyless function", "func External(simd.Uint32s)", "cannot specialize bodyless SIMD-dependent function External", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			simd := types.NewPackage("simd", "simd")
			obj := types.NewTypeName(token.NoPos, simd, "Uint32s", nil)
			types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			simd.Scope().Insert(obj)
			simd.MarkComplete()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "client.go", "package client\nimport \"simd\"\n"+tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object)}
			cfg := types.Config{Importer: midwayMetadataImporter{simd}}
			typed, err := cfg.Check("client", fset, []*ast.File{file}, info)
			if err != nil {
				t.Fatal(err)
			}
			bridge := &packages.Package{ID: "simd/internal/bridge", PkgPath: "simd/internal/bridge"}
			simdMetadata := &packages.Package{ID: "simd", PkgPath: "simd", Types: simd, Imports: make(map[string]*packages.Package)}
			if tc.bridge {
				simdMetadata.Imports[bridge.PkgPath] = bridge
			}
			pkg := &packages.Package{ID: "client", PkgPath: "client", Fset: fset, Syntax: []*ast.File{file}, Types: typed, TypesInfo: info, Imports: map[string]*packages.Package{"simd": simdMetadata}}
			changed, err := rewritePortableSIMD(pkg, new(directive.Index))
			if tc.diagnostic != "" {
				if changed || err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
					t.Fatalf("changed=%v error=%v, want %q", changed, err, tc.diagnostic)
				}
				return
			}
			if !changed || err != nil {
				t.Fatalf("changed=%v error=%v", changed, err)
			}
			if pkg.Imports[bridge.PkgPath] != bridge || pkg.Imports["simd"] != simdMetadata {
				t.Fatal("rewrite did not preserve canonical dependency metadata")
			}
		})
	}
}
