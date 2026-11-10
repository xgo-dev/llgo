package build

import (
	"encoding/json"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/frontcache"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestFrontendLinkExportRoundTripAndValidation(t *testing.T) {
	fset := token.NewFileSet()
	file := fset.AddFile("pkg.go", -1, 100)
	pkg := types.NewPackage("example/p", "p")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(file.Pos(12), pkg, "input", types.NewPointer(types.Typ[types.Int]))), types.NewTuple(types.NewVar(file.Pos(20), pkg, "answer", types.Typ[types.Int])), false)
	ids := map[*types.Package]string{pkg: pkg.Path()}
	index := frontcache.NewTypes(ids, fset)
	index.Add(sig)
	f := &frontendCache{ctx: &context{progSSA: ssa.NewProgram(fset, 0)}, ids: ids, packages: map[string]*types.Package{pkg.Path(): pkg}, types: index}
	original := &packageLinkSnapshot{needAbiInit: 2, methodByIndex: map[int]none{3: {}}, methodByName: map[string]none{"Private": {}}, abiSymbols: map[string]none{"type.p": {}}, abiTypes: []llssa.AbiTypeInfo{{Name: "type.p", Raw: types.Typ[types.Int]}, {Name: "not-owned", Raw: types.Typ[types.String]}}, funcInfo: []funcInfoRecord{{symbol: "p.F", name: "F", file: "pkg.go", line: 12}}, pcLineInfo: []pcLineRecord{{id: 1, symbol: "p.F", file: "pkg.go", line: 12}}, hasLocalExports: true, cExports: []cExport{{goName: "p.F", cName: "external", sig: sig}, {goName: "p.Raw", cName: "raw"}}}
	original.cExports[0].callConv = llssa.CDecl
	original.cExports = append(original.cExports, cExport{goName: "p.Win", cName: "win", sig: sig, callConv: llssa.Stdcall})
	data, err := f.exportLink(original)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var decoded frontendLinkExport
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	restored, err := f.importLink(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.abiTypes) != 1 || restored.abiTypes[0].Name != "type.p" || len(restored.cExports) != 3 || restored.cExports[1].sig != nil {
		t.Fatal("ABI ownership or raw export changed")
	}
	for i, export := range original.cExports {
		if restored.cExports[i].callConv != export.callConv {
			t.Fatalf("export %s lost its calling convention", export.cName)
		}
	}
	if !types.Identical(restored.cExports[0].sig, sig) || restored.cExports[0].sig.Params().At(0).Name() != "input" || !reflect.DeepEqual(original.funcInfo, restored.funcInfo) || !reflect.DeepEqual(original.pcLineInfo, restored.pcLineInfo) || !reflect.DeepEqual(original.methodByIndex, restored.methodByIndex) {
		t.Fatal("link contract changed")
	}
	for _, key := range []string{"invalid", "pkg.go:no", "pkg.go:-1", "pkg.go:101", "missing.go:1"} {
		if f.position(key) != token.NoPos {
			t.Fatalf("invalid position accepted: %s", key)
		}
	}
	if f.position("pkg.go:12") != file.Pos(12) {
		t.Fatal("source position lost")
	}
	if _, err := f.importLink(nil); err == nil {
		t.Fatal("nil link export accepted")
	}
	for _, tc := range []struct {
		name   string
		change func(*frontendLinkExport)
	}{
		{"method index", func(d *frontendLinkExport) { d.MethodIndexes = []int{-1} }},
		{"ABI type", func(d *frontendLinkExport) { d.ABI = []frontendABIType{{Name: "type.p", Type: -1}} }},
		{"C signature", func(d *frontendLinkExport) {
			d.CExports = []frontendCExport{{GoName: "p.F", CName: "bad", Signature: 0}}
			d.Types.Nodes[0].Kind = "invalid"
		}},
		{"C identity", func(d *frontendLinkExport) { d.CExports = []frontendCExport{{GoName: "", CName: "bad", Signature: -1}} }},
		{"C calling convention", func(d *frontendLinkExport) { d.CExports[0].CallConv = llssa.PyDecl }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var bad frontendLinkExport
			if err := json.Unmarshal(wire, &bad); err != nil {
				t.Fatal(err)
			}
			tc.change(&bad)
			if _, err := f.importLink(&bad); err == nil {
				t.Fatal("invalid link contract accepted")
			}
		})
	}
}
