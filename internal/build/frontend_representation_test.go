package build

import (
	"encoding/json"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/frontcache"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestFrontendRepresentationPolicyValidation(t *testing.T) {
	source := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer source.Dispose()
	pkg := types.NewPackage("example.com/layout", "layout")
	st := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Fn", llssa.NoArgsNoRet, false)}, nil)
	named := func(name string) *types.Named {
		n := types.NewNamed(types.NewTypeName(0, pkg, name, nil), st, nil)
		pkg.Scope().Insert(n.Obj())
		return n
	}
	native, copyType := named("Native"), named("Copy")
	named("Go")
	pkg.MarkComplete()
	source.SetTypeCallConv(pkg.Path()+".Native", llssa.CDecl)
	source.SetTypeDefinition(copyType.Obj(), native)
	index := frontcache.NewTypes(map[*types.Package]string{pkg: pkg.Path()})
	exported := frontcache.ExportPackage(pkg, index, token.NewFileSet())
	exportFrontendRepresentation(source, &exported.Types)
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"valid", "missing policy", "unknown ABI", "invalid origin", "unknown layout"} {
		t.Run(variant, func(t *testing.T) {
			var record frontcache.PackageExport
			if err := json.Unmarshal(data, &record); err != nil {
				t.Fatal(err)
			}
			for i := range record.Types.Nodes {
				node := &record.Types.Nodes[i]
				if node.Object.Name != "Copy" {
					continue
				}
				switch variant {
				case "missing policy":
					node.Representation = nil
				case "unknown ABI":
					node.Representation.ABI = 999
				case "invalid origin":
					node.Representation.Origin = len(record.Types.Nodes)
				case "unknown layout":
					node.Layout = "unknown"
				}
			}
			importer := &frontcache.TypeImporter{Packages: map[string]*types.Package{pkg.Path(): types.NewPackage(pkg.Path(), pkg.Name())}}
			loaded, err := frontcache.ImportPackage(&record, importer)
			if err != nil {
				t.Fatal(err)
			}
			backend := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer backend.Dispose()
			err = importFrontendRepresentation(backend, importer)
			if variant != "valid" {
				if err == nil {
					t.Fatal("invalid representation metadata accepted")
				}
				if backend.RepresentationOf(loaded.Scope().Lookup("Copy").Type()).Storage != llssa.GoRepresentation {
					t.Fatal("invalid metadata partially published")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			// An opaque package has no checker bindings. Explicit Go facts
			// must bypass any attempt to reconstruct its declaration source.
			backend.SetTypeDefinitionSource(loaded, func(*types.TypeName) types.Type { panic("opaque source queried") })
			if backend.RepresentationOf(loaded.Scope().Lookup("Go").Type()).Storage != llssa.GoRepresentation {
				t.Fatal("ordinary Go policy changed")
			}
			if rep := backend.RepresentationOf(loaded.Scope().Lookup("Copy").Type()); rep.Storage != llssa.NativeRepresentation || rep.Origin.Name() != "Native" {
				t.Fatalf("inherited native policy = %+v", rep)
			}
		})
	}
}
