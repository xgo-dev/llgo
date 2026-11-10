package cl

import (
	"encoding/json"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/frontcache"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestNativeRepresentationExportMetadata(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, _ := checkRepresentationPackage(t, prog, "example.com/native", `package native
//llgo:cdecl
type private struct { Fn func(); Tail uint64 }
type First private
type Exported First
type Alias = private
type ThroughAlias Alias
//llgo:cdecl
type Generic[T any] struct { Fn func(); Tail T }
type Derived[T any] Generic[T]
type Instance Derived[uint64]
//llgo:cdecl
type Anonymous = struct { Fn func(); Tail uint64 }
type FromAnonymous Anonymous
`, nil, true)
	index := frontcache.NewTypes(map[*types.Package]string{pkg: pkg.Path()})
	index.LayoutIdentity = prog.LayoutIdentity
	index.AddPackage(pkg, nil)
	exported := frontcache.ExportPackage(pkg, index, token.NewFileSet())
	exported.Types.ExportRepresentations(func(obj *types.TypeName) (int, types.Type) {
		r := prog.RepresentationOf(obj.Type())
		if r.Storage == llssa.NativeRepresentation {
			return int(r.NativeABI), r.Origin.Type()
		}
		return 0, nil
	})
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	var restored frontcache.PackageExport
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	importer := &frontcache.TypeImporter{Packages: map[string]*types.Package{pkg.Path(): types.NewPackage(pkg.Path(), pkg.Name())}}
	loaded, err := frontcache.ImportPackage(&restored, importer)
	if err != nil {
		t.Fatal(err)
	}
	representations, err := importer.ImportRepresentations()
	if err != nil {
		t.Fatal(err)
	}
	backend := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer backend.Dispose()
	backend.TypeSizes(types.SizesFor("gc", "amd64"))
	for _, name := range []string{"private", "Generic"} {
		backend.SetTypeCallConv(pkg.Path()+"."+name, llssa.CDecl)
	}
	for obj, imported := range representations {
		r := llssa.TypeRepresentation{}
		if imported.ABI != 0 {
			r.Storage, r.NativeABI, r.Origin = llssa.NativeRepresentation, llssa.CallConv(imported.ABI), imported.Origin
		}
		backend.SetImportedRepresentation(obj, r)
	}
	layouts, err := importer.ImportLayouts()
	if err != nil {
		t.Fatal(err)
	}
	for typ := range layouts {
		backend.SetNativeAnonymousLayout(typ)
	}
	backend.SetTypeCallConv(pkg.Path()+".Anonymous", llssa.CDecl)
	for _, name := range []string{"First", "Exported", "ThroughAlias", "Instance", "FromAnonymous"} {
		typ := loaded.Scope().Lookup(name).Type()
		if backend.RepresentationOf(typ).Storage != llssa.NativeRepresentation {
			t.Fatalf("restored %s lost native representation", name)
		}
		raw := backend.Type(typ, llssa.GoDecl)
		if backend.SizeOf(raw) != 16 || backend.OffsetOf(raw, 1) != 8 {
			t.Fatalf("restored %s lost native storage", name)
		}
	}
	// Malformed/missing origins must fail before any policy is installed.
	for i := range restored.Types.Nodes {
		if r := restored.Types.Nodes[i].Representation; r != nil && r.ABI != 0 {
			r.Origin = len(restored.Types.Nodes)
			if _, err := importer.ImportRepresentations(); err == nil {
				t.Fatal("invalid origin reference accepted")
			}
			break
		}
	}
}
