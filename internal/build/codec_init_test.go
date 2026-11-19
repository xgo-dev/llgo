package build

import (
	"go/token"
	"go/types"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

type codecInitTestImporter map[string]*types.Package

func (i codecInitTestImporter) Import(path string) (*types.Package, error) {
	return i[path], nil
}

func TestConfigureCodecInits(t *testing.T) {
	const codecPath = "encoding/json"
	const entrySource = `package main
import "encoding/json"
import _ "io"
func init() {}
func main() {}
`
	const codecSource = `package json
var values = [2]int{1, 2}
func Marshal(_ any) ([]byte, error) { return nil, nil }
`

	for _, test := range []struct {
		name     string
		entry    string
		codec    string
		change   func(*context, *linkPruningInputs)
		wantDrop bool
	}{
		{name: "private initialization", wantDrop: true},
		{name: "generic codec call", entry: `package main
import "encoding/json"
import _ "io"
func main() { json.Marshal(0) }
`},
		{name: "explicit side-effect import", entry: `package main
import _ "encoding/json"
import _ "io"
func main() {}
`},
		{name: "linkname reference", entry: entrySource + `
//go:linkname external encoding/json.Marshal
func external()
`},
		{name: "initializer can panic", codec: codecSource + `func init() { panic("effect") }`},
		{name: "untrusted codec", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.pkgByID[codecPath].Dir = ""
		}},
		{name: "native root", change: func(_ *context, in *linkPruningInputs) {
			in.roots = []string{"encoding/json.Marshal"}
		}},
		{name: "opaque assembly", change: func(_ *context, in *linkPruningInputs) {
			in.opaqueAsmPkg = "example.com/asm"
		}},
		{name: "open wasm program", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.wasmClosedProgram = false
		}},
		{name: "no specialization applied", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.jsonCodecPackages = nil
		}},
		{name: "dependency uses JSON", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.pkgByID["io"].Imports = map[string]*packages.Package{codecPath: ctx.pkgByID[codecPath].Package}
		}},
		{name: "alternate JSON source", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.pkgByID[codecPath].AltPkg = &packages.Cached{}
		}},
		{name: "missing dependency types", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.pkgByID["io"].Types = nil
		}},
		{name: "untyped entry", change: func(ctx *context, _ *linkPruningInputs) {
			ctx.initial[0].TypesInfo = nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			goroot := t.TempDir()
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			pkgs := make(map[string]*aPackage)
			imports := make(codecInitTestImporter)
			add := func(path, source string) *packages.Package {
				// An import can remain in the package after its typed call has been
				// rewritten. Keep that dependency in the SSA initializer.
				pkg := checkFormatTestPackage(t, prog.Fset, path, source, types.Config{
					Importer: imports, DisableUnusedImportCheck: true,
				})
				pkg.ID = path
				pkg.Dir = filepath.Join(goroot, "src", filepath.FromSlash(path))
				imports[path] = pkg.Types
				pkgs[path] = &aPackage{Package: pkg}
				prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
				return pkg
			}
			if test.codec == "" {
				test.codec = codecSource
			}
			if test.entry == "" {
				test.entry = entrySource
			}
			codec := add(codecPath, test.codec)
			other := add("io", "package io")
			entry := add("main", test.entry)
			entry.Imports = map[string]*packages.Package{codecPath: codec, "io": other}
			prog.Build()

			ctx := &context{
				initial:           []*packages.Package{entry},
				progSSA:           prog,
				pkgByID:           pkgs,
				buildConf:         &Config{Goarch: "wasm", Verbose: true},
				jsonCodecPackages: map[*types.Package]bool{entry.Types: true},
				wasmClosedProgram: true,
			}
			in := &linkPruningInputs{goroot: goroot}
			codecInit := prog.Package(codec.Types).Func("init")
			otherInit := prog.Package(other.Types).Func("init")
			if test.change != nil {
				test.change(ctx, in)
			}
			configureCodecInits(ctx, in)
			if got := ctx.codecInitDrops[llssa.FullName(codec.Types, "init")]; got != test.wantDrop {
				t.Fatalf("omitted codec init = %v, want %v", got, test.wantDrop)
			}
			calls := make(map[*ssa.Function]bool)
			for _, block := range prog.Package(entry.Types).Func("init").Blocks {
				for _, instr := range block.Instrs {
					if call, ok := instr.(*ssa.Call); ok {
						calls[call.Common().StaticCallee()] = true
					}
				}
			}
			if calls[codecInit] == test.wantDrop || !calls[otherInit] {
				t.Fatalf("initializer calls: JSON=%v, io=%v", calls[codecInit], calls[otherInit])
			}
		})
	}
}
