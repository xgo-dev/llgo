package frontcache

import (
	"encoding/json"
	"go/types"
	"testing"
)

func TestRepresentationGraphRoundTripAndValidation(t *testing.T) {
	prog, pkg, index := fixture(t, `package p
type root struct { Fn func(); Tail uint64 }
type Copy root
type Alias = Copy
type Plain struct { Fn func(); Tail uint64 }
type Anonymous = struct { Value uintptr }
type Ordinary = struct { Value uintptr }
type Generic[T any] struct { Value T }
var Instance Generic[int]
`, false)
	root := pkg.Pkg.Scope().Lookup("root").Type()
	anonymous := types.Unalias(pkg.Pkg.Scope().Lookup("Anonymous").Type())
	ordinary := types.Unalias(pkg.Pkg.Scope().Lookup("Ordinary").Type())
	index = NewTypes(map[*types.Package]string{pkg.Pkg: pkg.Pkg.Path()}, prog.Fset)
	index.LayoutIdentity = func(typ types.Type) string {
		if typ == anonymous {
			return "native"
		}
		return ""
	}
	if !types.Identical(anonymous, ordinary) || index.Key(anonymous) == index.Key(ordinary) {
		t.Fatal("anonymous layout attribute did not distinguish equal Go types")
	}
	graph := NewTypeGraph(index, prog.Fset)
	ids := make(map[string]int)
	// Exporting Copy first also tests an origin added while exporting policy.
	for _, name := range []string{"Copy", "Alias", "Plain", "Anonymous", "Ordinary", "Instance"} {
		ids[name] = graph.Add(pkg.Pkg.Scope().Lookup(name).Type())
	}
	integer := graph.Add(types.Typ[types.Int])
	graph.ExportRepresentations(func(obj *types.TypeName) (int, types.Type) {
		switch obj.Name() {
		case "Copy", "Alias", "root":
			return 2, root
		}
		return 0, nil
	})
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"valid", "missing policy", "basic policy", "instance policy", "invalid owner reference", "owner key mismatch", "invalid origin reference", "nondeclaration origin", "Go origin", "ordinary origin", "origin ABI mismatch", "origin cycle", "unknown layout", "layout on declaration", "invalid layout field", "layout key mismatch"} {
		t.Run(variant, func(t *testing.T) {
			var decoded TypeGraph
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			copyNode := &decoded.Nodes[ids["Copy"]]
			layoutID := decoded.Nodes[ids["Anonymous"]].Elem
			layoutNode := &decoded.Nodes[layoutID]
			switch variant {
			case "missing policy":
				copyNode.Representation = nil
			case "basic policy":
				decoded.Nodes[integer].Representation = &DeclarationRepresentation{Origin: -1}
			case "instance policy":
				decoded.Nodes[ids["Instance"]].Representation = &DeclarationRepresentation{Origin: -1}
			case "invalid owner reference":
				copyNode.Elem = len(decoded.Nodes)
			case "owner key mismatch":
				copyNode.Key = index.Key(types.Typ[types.Int])
			case "invalid origin reference":
				copyNode.Representation.Origin = len(decoded.Nodes)
			case "nondeclaration origin":
				copyNode.Representation.Origin = integer
			case "Go origin":
				copyNode.Representation.ABI = 0
			case "ordinary origin":
				copyNode.Representation.Origin = ids["Plain"]
			case "origin ABI mismatch":
				copyNode.Representation.ABI = 4
			case "origin cycle":
				decoded.Nodes[copyNode.Representation.Origin].Representation.Origin = ids["Copy"]
			case "unknown layout":
				layoutNode.Layout = "future-layout"
			case "layout on declaration":
				copyNode.Layout = "native"
			case "invalid layout field":
				layoutNode.Fields[0].Type = len(decoded.Nodes)
			case "layout key mismatch":
				layoutNode.Key = index.Key(types.Typ[types.Int])
			}
			fresh := types.NewPackage(pkg.Pkg.Path(), pkg.Pkg.Name())
			importer := &TypeImporter{Graph: &decoded, Packages: map[string]*types.Package{fresh.Path(): fresh}}
			if variant == "owner key mismatch" || variant == "layout key mismatch" {
				importer.Existing = NewTypes(nil)
				importer.Existing.Add(types.Typ[types.Int])
			}
			representations, repErr := importer.ImportRepresentations()
			layouts, layoutErr := importer.ImportLayouts()
			if variant != "valid" {
				if repErr == nil && layoutErr == nil {
					t.Fatal("invalid cached representation accepted")
				}
				return
			}
			if repErr != nil || layoutErr != nil {
				t.Fatalf("valid metadata rejected: %v / %v", repErr, layoutErr)
			}
			seen := make(map[string]bool)
			for obj, policy := range representations {
				seen[obj.Name()] = true
				switch obj.Name() {
				case "Copy", "Alias", "root":
					if policy.ABI != 2 || policy.Origin == nil || policy.Origin.Name() != "root" || policy.Origin.Pkg() != fresh {
						t.Fatalf("lost private native origin: %s %+v", obj, policy)
					}
				default:
					if policy.ABI != 0 || policy.Origin != nil {
						t.Fatalf("ordinary declaration acquired native storage: %s %+v", obj, policy)
					}
				}
			}
			for _, name := range []string{"root", "Copy", "Alias", "Plain", "Anonymous", "Ordinary", "Generic"} {
				if !seen[name] {
					t.Fatalf("missing declaration policy: %s", name)
				}
			}
			if len(layouts) != 1 {
				t.Fatalf("anonymous layout count = %d", len(layouts))
			}
			for st, attribute := range layouts {
				if attribute != "native" || st.NumFields() != 1 || st.Field(0).Name() != "Value" {
					t.Fatalf("anonymous layout changed: %s %q", st, attribute)
				}
			}
		})
	}
}
